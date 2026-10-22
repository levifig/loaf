package state

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levifig/loaf/internal/project"
)

func TestJournalContextMarksOnlyAnOldOvertakenSynthesisStale(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 22, 0, 0, 1, 0, time.UTC)
	for _, tc := range []struct {
		name         string
		synthesisAt  string
		entriesAfter int
		wantStale    bool
	}{
		{name: "old and overtaken", synthesisAt: "2026-07-30T17:25:00Z", entriesAfter: JournalContextSynthesisStaleEntries, wantStale: true},
		{name: "exactly thirty days old", synthesisAt: "2026-09-22T00:00:00Z", entriesAfter: 200, wantStale: false},
		{name: "one calendar day past the limit", synthesisAt: "2026-09-21T23:59:59Z", entriesAfter: JournalContextSynthesisStaleEntries, wantStale: true},
		{name: "old but dormant", synthesisAt: "2026-07-30T17:25:00Z", entriesAfter: JournalContextSynthesisStaleEntries - 1, wantStale: false},
		{name: "fresh during a busy stretch", synthesisAt: "2026-10-20T09:00:00Z", entriesAfter: 500, wantStale: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := projectRoot(t)
			stateHome := t.TempDir()
			if _, err := Initialize(ctx, root, PathResolver{StateHome: stateHome}); err != nil {
				t.Fatalf("Initialize() error = %v", err)
			}
			store := openTestStore(t, root, stateHome)
			defer store.Close()
			projectID := projectIDForTest(t, store, root)
			synthesis := seedJournalEntry(t, store, projectID, "wrap", "project", "the project synthesis", "main", tc.synthesisAt)
			created, _ := time.Parse(time.RFC3339, tc.synthesisAt)
			for i := 0; i < tc.entriesAfter; i++ {
				seedJournalEntry(t, store, projectID, "discover", "noise", fmt.Sprintf("after-%03d", i), "main", created.Add(time.Duration(i+1)*time.Second).Format(time.RFC3339))
			}

			digest, err := store.JournalContext(ctx, root, JournalContextOptions{Branch: "main", Now: now})
			if err != nil {
				t.Fatalf("JournalContext() error = %v", err)
			}
			layer := digest.ProjectSynthesis
			if len(layer.Items) != 1 || layer.Items[0].ID != synthesis {
				t.Fatalf("project synthesis = %#v, want %s", layer.Items, synthesis)
			}
			if layer.Stale != tc.wantStale || layer.EntriesSince != tc.entriesAfter {
				t.Fatalf("stale = %t entries since = %d, want %t and %d", layer.Stale, layer.EntriesSince, tc.wantStale, tc.entriesAfter)
			}
		})
	}
}

func TestJournalContextShowsANewerScopedWrapBesideAStaleSynthesis(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	setup := func(t *testing.T, scopedAt string) (*Store, project.Root, string) {
		t.Helper()
		root := projectRoot(t)
		stateHome := t.TempDir()
		if _, err := Initialize(ctx, root, PathResolver{StateHome: stateHome}); err != nil {
			t.Fatalf("Initialize() error = %v", err)
		}
		store := openTestStore(t, root, stateHome)
		t.Cleanup(func() { store.Close() })
		projectID := projectIDForTest(t, store, root)
		seedJournalEntry(t, store, projectID, "wrap", "project", "July synthesis", "main", "2026-07-30T17:25:00Z")
		scoped := seedJournalEntry(t, store, projectID, "wrap", "digest", "scoped wrap", "main", scopedAt)
		for i := 0; i < JournalContextSynthesisStaleEntries; i++ {
			seedJournalEntry(t, store, projectID, "discover", "noise", fmt.Sprintf("n-%02d", i), "main", fmt.Sprintf("2026-10-01T10:%02d:00Z", i))
		}
		return store, root, scoped
	}

	store, root, scoped := setup(t, "2026-10-12T21:42:00Z")
	digest, err := store.JournalContext(ctx, root, JournalContextOptions{Branch: "main", Now: now})
	if err != nil {
		t.Fatalf("JournalContext() error = %v", err)
	}
	if !digest.ProjectSynthesis.Stale || len(digest.LatestCheckpoint.Items) != 1 || digest.LatestCheckpoint.Items[0].Entry.ID != scoped {
		t.Fatalf("stale synthesis %t, checkpoint %#v, want the newer scoped wrap %s", digest.ProjectSynthesis.Stale, digest.LatestCheckpoint.Items, scoped)
	}
	for _, entry := range digest.BranchRecency.Items {
		if entry.ID == scoped {
			t.Fatalf("branch recency repeats the scoped wrap shown as a checkpoint")
		}
	}

	store, root, _ = setup(t, "2026-07-20T09:00:00Z")
	digest, err = store.JournalContext(ctx, root, JournalContextOptions{Branch: "main", Now: now})
	if err != nil {
		t.Fatalf("JournalContext() error = %v", err)
	}
	if !digest.ProjectSynthesis.Stale || len(digest.LatestCheckpoint.Items) != 0 {
		t.Fatalf("checkpoint = %#v, want none when the scoped wrap is older than the stale synthesis", digest.LatestCheckpoint.Items)
	}
}

func TestJournalContextStampsEveryEntryWithItsAge(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(ctx, root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()
	projectID := projectIDForTest(t, store, root)
	synthesis := seedJournalEntry(t, store, projectID, "wrap", "project", "synthesis", "main", "2026-07-30T17:25:00Z")
	lineage := seedJournalEntry(t, store, projectID, "decision", "lineage/digest", "lineage decision", "main", "2026-08-10T10:00:00Z")
	for i := 0; i < JournalContextSynthesisStaleEntries; i++ {
		seedJournalEntry(t, store, projectID, "discover", "noise", fmt.Sprintf("n-%02d", i), "other", fmt.Sprintf("2026-09-01T10:%02d:00Z", i))
	}
	scoped := seedJournalEntry(t, store, projectID, "wrap", "digest", "newer scoped wrap", "main", "2026-10-12T21:42:00Z")
	block := seedJournalEntry(t, store, projectID, "block", "ci", "runner offline", "main", "2026-10-21T23:59:59Z")
	recent := seedJournalEntry(t, store, projectID, "discover", "x", "late last night", "main", "2026-10-21T23:30:00Z")
	today := seedJournalEntry(t, store, projectID, "discover", "x", "just now", "main", "2026-10-22T00:00:00Z")
	skewed := seedJournalEntry(t, store, projectID, "discover", "x", "from a fast clock", "main", "2026-10-23T08:00:00Z")

	digest, err := store.JournalContext(ctx, root, JournalContextOptions{Branch: "main", LineageKeys: []string{"digest"}, Now: time.Date(2026, 10, 22, 0, 0, 1, 0, time.UTC)})
	if err != nil {
		t.Fatalf("JournalContext() error = %v", err)
	}
	ages := map[string]int{}
	collect := func(layer string, entry JournalEntryRecord) {
		if entry.AgeDays == nil {
			t.Fatalf("%s entry %s has no age", layer, entry.ID)
		}
		ages[entry.ID] = *entry.AgeDays
	}
	for _, entry := range digest.ProjectSynthesis.Items {
		collect("synthesis", entry)
	}
	for _, item := range digest.LatestCheckpoint.Items {
		collect("checkpoint", item.Entry)
	}
	for _, entry := range digest.ActiveLineage.Items {
		collect("lineage", entry)
	}
	for _, item := range digest.UnresolvedBlockers.Items {
		collect("blockers", item.Block)
	}
	for _, entry := range digest.BranchRecency.Items {
		collect("branch recency", entry)
	}
	for _, entry := range digest.BranchEntries {
		collect("branch entries", entry)
	}
	if digest.LatestWrap == nil {
		t.Fatalf("latest wrap is missing")
	}
	collect("latest wrap", *digest.LatestWrap)
	for id, want := range map[string]int{synthesis: 84, scoped: 10, lineage: 73, block: 1, recent: 1, today: 0, skewed: 0} {
		if got, ok := ages[id]; !ok || got != want {
			t.Fatalf("age of %s = %d (present %t), want %d (ages %v)", id, got, ok, want, ages)
		}
	}

	encoded, err := json.Marshal(digest.ProjectSynthesis)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"age_days":84`) {
		t.Fatalf("synthesis JSON = %s, want age_days", encoded)
	}
	recentRead, err := store.RecentJournal(ctx, root, JournalRecentOptions{})
	if err != nil {
		t.Fatalf("RecentJournal() error = %v", err)
	}
	if encoded, _ := json.Marshal(recentRead); strings.Contains(string(encoded), "age_days") {
		t.Fatalf("journal recent JSON = %s, want no age outside context reads", encoded)
	}
}
