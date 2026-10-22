package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestJournalUnblockByEntryIDResolvesExactlyThatBlock(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(ctx, root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()
	projectID := projectIDForTest(t, store, root)
	log := func(entry string) JournalLogResult {
		t.Helper()
		result, err := store.LogJournal(ctx, root, JournalLogOptions{Entry: entry})
		if err != nil {
			t.Fatalf("LogJournal(%q) error = %v", entry, err)
		}
		return result
	}
	rejected := func(entry, code string) *JournalUnmatchedUnblockError {
		t.Helper()
		_, err := store.LogJournal(ctx, root, JournalLogOptions{Entry: entry})
		var unmatched *JournalUnmatchedUnblockError
		if !errors.As(err, &unmatched) || unmatched.Code != code {
			t.Fatalf("LogJournal(%q) error = %T %v, want code %s", entry, err, err, code)
		}
		return unmatched
	}
	openBlocks := func() []string {
		t.Helper()
		digest, err := store.JournalContext(ctx, root, JournalContextOptions{})
		if err != nil {
			t.Fatalf("JournalContext() error = %v", err)
		}
		if len(digest.Diagnostics) != 0 {
			t.Fatalf("diagnostics = %#v, want none for validated writes", digest.Diagnostics)
		}
		ids := []string{}
		for _, item := range digest.UnresolvedBlockers.Items {
			if item.Key != item.Block.Scope {
				t.Fatalf("blocker key %q != block scope %q", item.Key, item.Block.Scope)
			}
			ids = append(ids, item.Block.ID)
		}
		return ids
	}

	first := log("block(ship): PR one waits on review")
	second := log("block(ship): PR two waits on CI")
	review := log("block(review): reviewer unavailable")
	assertIDs(t, openBlocks(), review.ID, second.ID, first.ID)

	byID := log("unblock(" + first.ID + "): PR one landed")
	assertIDs(t, openBlocks(), review.ID, second.ID)

	again := rejected("unblock("+first.ID+"): landed twice", JournalUnblockResolvedCode)
	if again.LatestSourceID != byID.ID || again.LatestSourceType != "unblock" || again.Error() == "" {
		t.Fatalf("already-resolved error = %#v, want the resolving unblock %s", again, byID.ID)
	}

	scope := log("unblock(ship): the remaining ship work landed")
	assertIDs(t, openBlocks(), review.ID)
	unmatched := rejected("unblock(ship): nothing left", JournalUnmatchedUnblockCode)
	if unmatched.LatestSourceID != scope.ID || unmatched.LatestSourceType != "unblock" {
		t.Fatalf("scope unmatched error = %#v, want latest outcome %s", unmatched, scope.ID)
	}
	resolvedByScope := rejected("unblock("+second.ID+"): already gone", JournalUnblockResolvedCode)
	if resolvedByScope.LatestSourceID != scope.ID {
		t.Fatalf("id after scope unblock = %#v, want resolved by %s", resolvedByScope, scope.ID)
	}

	decision := log("decision(blockers): resolve by entry id")
	notBlock := rejected("unblock("+decision.ID+"): not a block", JournalUnblockNotBlockCode)
	if notBlock.EntryType != "decision" {
		t.Fatalf("not-a-block error = %#v, want entry type decision", notBlock)
	}
	rejected("unblock(01a1aaaa-bbbb-7ccc-8ddd-eeeeffff0000): unknown id", JournalUnblockUnknownEntryCode)
	rejected("unblock(journal:0123456789abcdef01234567): unknown legacy id", JournalUnblockUnknownEntryCode)

	legacy := seedJournalEntry(t, store, projectID, "block", "migration", "imported block", "main", "2026-07-01T10:00:00Z")
	assertIDs(t, openBlocks(), review.ID, legacy)
	log("unblock(" + legacy + "): imported block resolved by its legacy id")
	assertIDs(t, openBlocks(), review.ID)

	var rows int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM journal_entries WHERE entry_type IN ('block', 'unblock')`).Scan(&rows); err != nil {
		t.Fatalf("count blocker rows: %v", err)
	}
	if rows != 7 {
		t.Fatalf("blocker rows = %d, want 7 (rejections write nothing)", rows)
	}
}

func TestJournalUnblockByIDThenScopeLeavesNoStrayDiagnostics(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(ctx, root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()
	projectID := projectIDForTest(t, store, root)
	older := seedJournalEntry(t, store, projectID, "block", "db", "lock contention", "main", "2026-10-01T10:00:00Z")
	seedJournalEntry(t, store, projectID, "unblock", older, "lock fixed", "main", "2026-10-02T10:00:00Z")
	newer := seedJournalEntry(t, store, projectID, "block", "db", "migration stuck", "main", "2026-10-03T10:00:00Z")
	lone := seedJournalEntry(t, store, projectID, "block", "db", "a third db block", "main", "2026-10-04T10:00:00Z")
	seedJournalEntry(t, store, projectID, "unblock", lone, "third resolved", "main", "2026-10-05T10:00:00Z")

	digest, err := store.JournalContext(ctx, root, JournalContextOptions{Now: time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("JournalContext() error = %v", err)
	}
	if digest.UnresolvedBlockers.AvailableCount != 1 || digest.UnresolvedBlockers.Items[0].Block.ID != newer {
		t.Fatalf("blockers = %#v, want only %s open", digest.UnresolvedBlockers.Items, newer)
	}
	if got := digest.UnresolvedBlockers.Items[0].PreviousUnblockID; got == "" {
		t.Fatalf("reopened block has no previous unblock")
	}
	if len(digest.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", digest.Diagnostics)
	}
	if _, err := store.LogJournal(ctx, root, JournalLogOptions{Entry: "unblock(db): migration finished"}); err != nil {
		t.Fatalf("scope unblock after id unblocks error = %v", err)
	}
	digest, err = store.JournalContext(ctx, root, JournalContextOptions{})
	if err != nil {
		t.Fatalf("JournalContext() error = %v", err)
	}
	if digest.UnresolvedBlockers.AvailableCount != 0 || len(digest.Diagnostics) != 0 {
		t.Fatalf("after scope unblock blockers = %#v diagnostics = %#v, want none", digest.UnresolvedBlockers.Items, digest.Diagnostics)
	}
}

func TestJournalContextFlagsBlockersOlderThanTheAgeLimit(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(ctx, root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()
	projectID := projectIDForTest(t, store, root)
	recent := seedJournalEntry(t, store, projectID, "block", "ci", "runner offline", "main", "2026-10-20T09:00:00Z")
	// Thirty calendar days before the read date: at the limit, still listed.
	boundary := seedJournalEntry(t, store, projectID, "block", "review", "waiting on review", "main", "2026-09-22T00:00:00Z")
	// One second earlier is the previous calendar day: past the limit.
	pastDay := seedJournalEntry(t, store, projectID, "block", "review", "older review wait", "main", "2026-09-21T23:59:59Z")
	ancient := seedJournalEntry(t, store, projectID, "block", "naming", "report name collision", "main", "2026-07-25T00:10:40Z")

	digest, err := store.JournalContext(ctx, root, JournalContextOptions{BlockerLimit: 2, Now: time.Date(2026, 10, 22, 0, 0, 1, 0, time.UTC)})
	if err != nil {
		t.Fatalf("JournalContext() error = %v", err)
	}
	layer := digest.UnresolvedBlockers
	if layer.AvailableCount != 4 || layer.ShownCount != 2 || layer.StaleCount != 2 || !layer.Truncated {
		t.Fatalf("layer = %#v, want 4 available, 2 shown, 2 stale", layer)
	}
	assertIDs(t, []string{layer.Items[0].Block.ID, layer.Items[1].Block.ID}, recent, boundary)
	if layer.Items[0].Stale || layer.Items[1].Stale {
		t.Fatalf("shown items = %#v, want neither stale", layer.Items)
	}

	all, err := store.JournalContext(ctx, root, JournalContextOptions{BlockerLimit: 10, Now: time.Date(2026, 10, 22, 0, 0, 1, 0, time.UTC)})
	if err != nil {
		t.Fatalf("JournalContext(all) error = %v", err)
	}
	stale := map[string]bool{}
	for _, item := range all.UnresolvedBlockers.Items {
		stale[item.Block.ID] = item.Stale
	}
	if stale[recent] || stale[boundary] || !stale[pastDay] || !stale[ancient] || all.UnresolvedBlockers.StaleCount != 2 {
		t.Fatalf("stale flags = %#v count = %d, want only the two older than 30 days", stale, all.UnresolvedBlockers.StaleCount)
	}

	fresh, err := store.JournalContext(ctx, root, JournalContextOptions{Now: time.Date(2026, 7, 25, 23, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("JournalContext(fresh clock) error = %v", err)
	}
	if fresh.UnresolvedBlockers.StaleCount != 0 {
		t.Fatalf("stale count on the ancient block's own day = %d, want 0", fresh.UnresolvedBlockers.StaleCount)
	}
}

func TestJournalContextAgeDaysCountsUTCCalendarDays(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 30, 0, 0, time.FixedZone("WEST", 3600))
	for _, tc := range []struct {
		createdAt string
		want      int
	}{
		{"2026-10-21T23:30:00Z", 0},
		{"2026-10-21T23:29:59Z", 0},
		{"2026-10-20T23:59:59.999Z", 1},
		{"2026-09-21T00:00:00Z", 30},
		{"2026-09-20T23:59:59Z", 31},
	} {
		got, ok := journalContextAgeDays(tc.createdAt, now)
		if !ok || got != tc.want {
			t.Fatalf("journalContextAgeDays(%q) = %d, %t; want %d", tc.createdAt, got, ok, tc.want)
		}
	}
	if _, ok := journalContextAgeDays("not a timestamp", now); ok {
		t.Fatalf("unparseable timestamp reported an age")
	}
}

func assertIDs(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	}
}
