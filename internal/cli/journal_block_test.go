package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

func TestJournalLogUnmatchedUnblockHumanAndJSON(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	runner := Runner{Stdout: &bytes.Buffer{}, WorkingDir: workingDir, StateHome: stateHome}

	err := runner.Run([]string{"journal", "log", "unblock(human-key): never matched"})
	var unmatched *state.JournalUnmatchedUnblockError
	if !errors.As(err, &unmatched) || unmatched.Code != state.JournalUnmatchedUnblockCode || unmatched.Key != "human-key" {
		t.Fatalf("human journal log error = %T %#v, want typed unmatched error", err, err)
	}

	var blockOut bytes.Buffer
	if err := (Runner{Stdout: &blockOut, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", "block(json-key): open", "--json"}); err != nil {
		t.Fatalf("block --json error = %v\n%s", err, blockOut.String())
	}
	var block state.JournalLogResult
	if err := json.Unmarshal(blockOut.Bytes(), &block); err != nil {
		t.Fatalf("decode block result: %v\n%s", err, blockOut.String())
	}
	var unblockOut bytes.Buffer
	if err := (Runner{Stdout: &unblockOut, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", "unblock(json-key): resolved", "--json"}); err != nil {
		t.Fatalf("unblock --json error = %v\n%s", err, unblockOut.String())
	}
	var unblock state.JournalLogResult
	if err := json.Unmarshal(unblockOut.Bytes(), &unblock); err != nil {
		t.Fatalf("decode unblock result: %v\n%s", err, unblockOut.String())
	}

	var output bytes.Buffer
	err = (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", "unblock(json-key): duplicate", "--json"})
	var exitErr interface {
		ExitCode() int
		Silent() bool
	}
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !exitErr.Silent() {
		t.Fatalf("JSON duplicate error = %v, want silent exit 1", err)
	}
	var envelope journalUnmatchedUnblockJSON
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatalf("decode unmatched envelope: %v\n%s", err, output.String())
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		t.Fatalf("unmatched JSON output = %q, want exactly one object (trailing decode error %v)", output.String(), err)
	}
	if envelope.ContractVersion != state.StateJSONContractVersion || envelope.Command != "journal log" || envelope.Code != state.JournalUnmatchedUnblockCode || envelope.Key != "json-key" || envelope.LatestSourceID != unblock.ID || envelope.LatestSourceType != "unblock" || envelope.Error == "" {
		t.Fatalf("unmatched envelope = %#v, want typed fields and latest unblock", envelope)
	}
	if block.ID == "" || unblock.ID == "" {
		t.Fatalf("successful IDs block=%q unblock=%q, want nonempty", block.ID, unblock.ID)
	}
}

func TestJournalLogFromHookUnmatchedUnblockWarnsWithoutWriting(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	log := func(entry string) state.JournalLogResult {
		t.Helper()
		var output bytes.Buffer
		if err := (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", entry, "--json"}); err != nil {
			t.Fatalf("journal log %q error = %v\n%s", entry, err, output.String())
		}
		var result state.JournalLogResult
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatalf("decode journal log %q: %v\n%s", entry, err, output.String())
		}
		return result
	}
	log("block(hook-key): open")
	resolved := log("unblock(hook-key): resolved")

	root, err := project.ResolveRoot(workingDir)
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	databasePath, err := (state.PathResolver{StateHome: stateHome}).DatabasePath(root)
	if err != nil {
		t.Fatalf("DatabasePath() error = %v", err)
	}
	db := openCLITestDB(t, databasePath)
	defer closeCLITestDB(t, db)
	canonicalBefore := sqliteCount(t, db, `SELECT COUNT(*) FROM journal_entries`)
	searchBefore := sqliteCount(t, db, `SELECT COUNT(*) FROM journal_search`)
	originsBefore := sqliteCount(t, db, `SELECT COUNT(*) FROM journal_origins`)

	var output bytes.Buffer
	err = (Runner{
		Stdout:     &output,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"hook-session"}`),
	}).Run([]string{"journal", "log", "--from-hook", "unblock(hook-key): duplicate adapter event"})
	if err != nil {
		t.Fatalf("journal log --from-hook unmatched error = %v, want nonblocking nil\n%s", err, output.String())
	}
	var warning journalHookWarning
	if err := json.Unmarshal(output.Bytes(), &warning); err != nil {
		t.Fatalf("decode hook warning: %v\n%s", err, output.String())
	}
	if warning.ContractVersion != state.StateJSONContractVersion || warning.Kind != "journal-diagnostic" || warning.Severity != "warning" || warning.Code != state.JournalUnmatchedUnblockCode || warning.Key != "hook-key" || warning.LatestSourceID != resolved.ID || warning.LatestSourceType != "unblock" || !warning.NonBlocking || warning.Message == "" {
		t.Fatalf("hook warning = %#v, want structured nonblocking diagnostic", warning)
	}
	if got := sqliteCount(t, db, `SELECT COUNT(*) FROM journal_entries`); got != canonicalBefore {
		t.Fatalf("canonical rows after hook warning = %d, want %d", got, canonicalBefore)
	}
	if got := sqliteCount(t, db, `SELECT COUNT(*) FROM journal_search`); got != searchBefore {
		t.Fatalf("search rows after hook warning = %d, want %d", got, searchBefore)
	}
	if got := sqliteCount(t, db, `SELECT COUNT(*) FROM journal_origins`); got != originsBefore {
		t.Fatalf("origin rows after hook warning = %d, want %d", got, originsBefore)
	}
}

func TestJournalLogUnblockByEntryIDAndBlockerLayerIDs(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	log := func(entry string) state.JournalLogResult {
		t.Helper()
		var output bytes.Buffer
		if err := (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", entry, "--json"}); err != nil {
			t.Fatalf("journal log %q error = %v\n%s", entry, err, output.String())
		}
		var result state.JournalLogResult
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatalf("decode journal log %q: %v\n%s", entry, err, output.String())
		}
		return result
	}
	layer := func() string {
		t.Helper()
		var output bytes.Buffer
		if err := (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "context", "--layer", "unresolved-blockers"}); err != nil {
			t.Fatalf("journal context --layer unresolved-blockers error = %v\n%s", err, output.String())
		}
		return output.String()
	}
	landed := log("block(ship): PR one waits on review")
	open := log("block(ship): PR two waits on CI")
	log("unblock(" + landed.ID + "): PR one landed")

	listed := layer()
	for _, want := range []string{
		"unresolved-blockers: showing 1 of 1",
		`resolve one: loaf journal log "unblock(<id>): how it was resolved"`,
		"] " + open.ID + " block(ship): PR two waits on CI",
		"block(ship): PR two waits on CI",
	} {
		if !strings.Contains(listed, want) {
			t.Fatalf("blocker layer = %q, want %q", listed, want)
		}
	}
	if strings.Contains(listed, landed.ID) {
		t.Fatalf("blocker layer = %q, still lists the block resolved by id", listed)
	}

	decision := log("decision(blockers): resolve by id")
	var output bytes.Buffer
	err := (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", "unblock(" + decision.ID + "): wrong entry", "--json"})
	var exitErr interface{ ExitCode() int }
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("not-a-block --json error = %v, want exit 1", err)
	}
	var envelope journalUnmatchedUnblockJSON
	if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
		t.Fatalf("decode not-a-block envelope: %v\n%s", err, output.String())
	}
	if envelope.Code != state.JournalUnblockNotBlockCode || envelope.Key != decision.ID || !strings.Contains(envelope.Error, "its type is decision, not block") {
		t.Fatalf("not-a-block envelope = %#v", envelope)
	}

	err = (Runner{Stdout: &bytes.Buffer{}, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", "unblock(01a1aaaa-bbbb-7ccc-8ddd-eeeeffff0000): typo"})
	var unknown *state.JournalUnmatchedUnblockError
	if !errors.As(err, &unknown) || unknown.Code != state.JournalUnblockUnknownEntryCode || !strings.Contains(err.Error(), "no journal entry with that id") {
		t.Fatalf("unknown id error = %v, want the unknown-entry rejection", err)
	}

	log("unblock(ship): the remaining work landed")
	if listed := layer(); !strings.Contains(listed, "unresolved-blockers: none") {
		t.Fatalf("blocker layer after scope unblock = %q, want none", listed)
	}
}

func TestJournalContextDefaultDigestCountsStaleBlockersInsteadOfListingThem(t *testing.T) {
	block := func(id, scope, message, createdAt string, stale bool) state.JournalContextBlockerItem {
		return state.JournalContextBlockerItem{Key: scope, Stale: stale, Block: state.JournalEntryRecord{ID: id, EntryType: "block", Scope: scope, Message: message, CreatedAt: createdAt}}
	}
	recent := block("01a1aaaa-0000-7000-8000-000000000001", "ci", "runner offline", "2026-10-20T09:00:00Z", false)
	staleReview := block("01a1aaaa-0000-7000-8000-000000000002", "review", "PR 253 review never started", "2026-09-11T09:00:00Z", true)
	staleRebase := block("01a1aaaa-0000-7000-8000-000000000003", "pr-212", "rebase needed", "2026-09-01T09:00:00Z", true)
	digest := func(items ...state.JournalContextBlockerItem) state.JournalContext {
		stale := 0
		for _, item := range items {
			if item.Stale {
				stale++
			}
		}
		return state.JournalContext{ProjectName: "loaf", Branch: "main", UnresolvedBlockers: state.JournalContextBlockerLayer{
			Available: true, AvailableCount: len(items), ShownCount: len(items), StaleCount: stale, Items: items,
		}}
	}
	render := func(result state.JournalContext) string {
		return renderJournalContextDigest(composeJournalContextCLIResult(result, activeChangeLayer{}, ""), journalContextDigestBudget)
	}

	mixed := render(digest(recent, staleReview, staleRebase))
	for _, want := range []string{
		`unresolved-blockers: showing 1 of 3 (more: loaf journal context --layer unresolved-blockers --limit 3; 2 older than 30 days not listed; resolve one: loaf journal log "unblock(<id>): how it was resolved")`,
		"  [2026-10-20 09:00] " + recent.Block.ID + " block(ci): runner offline",
	} {
		if !strings.Contains(mixed, want) {
			t.Fatalf("default digest = %q, want %q", mixed, want)
		}
	}
	for _, item := range []state.JournalContextBlockerItem{staleReview, staleRebase} {
		if strings.Contains(mixed, item.Block.ID) {
			t.Fatalf("default digest = %q, lists stale blocker %s", mixed, item.Block.ID)
		}
	}
	if strings.Contains(mixed, "left out to stay within the injection budget") {
		t.Fatalf("default digest = %q, counts aged-out blockers as budget omissions", mixed)
	}

	onlyStale := render(digest(staleReview, staleRebase))
	if want := "unresolved-blockers: showing 0 of 2 (more: loaf journal context --layer unresolved-blockers --limit 2; 2 older than 30 days not listed)\n"; !strings.Contains(onlyStale, want) || strings.Contains(onlyStale, "resolve one:") {
		t.Fatalf("all-stale default digest = %q, want %q and no resolve hint", onlyStale, want)
	}

	var listed bytes.Buffer
	writeJournalContextHuman(&listed, composeJournalContextCLIResult(digest(recent, staleReview, staleRebase), activeChangeLayer{}, journalContextLayerBlockers))
	if got := listed.String(); !strings.Contains(got, "unresolved-blockers: showing 3 of 3") || strings.Contains(got, "not listed") {
		t.Fatalf("blocker layer = %q, want every blocker listed", got)
	}
	for _, item := range []state.JournalContextBlockerItem{recent, staleReview, staleRebase} {
		if !strings.Contains(listed.String(), "] "+item.Block.ID+" block("+item.Key+"): ") {
			t.Fatalf("blocker layer = %q, want %s with its id", listed.String(), item.Block.ID)
		}
	}
}

func TestJournalContextDefaultPathsAgeRealBlockersByTheReadClock(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	ids := []string{}
	for _, entry := range []string{"block(ci): runner offline", "block(review): waiting on review"} {
		var output bytes.Buffer
		if err := (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", entry, "--json"}); err != nil {
			t.Fatalf("journal log %q error = %v\n%s", entry, err, output.String())
		}
		var result state.JournalLogResult
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatalf("decode journal log %q: %v", entry, err)
		}
		ids = append(ids, result.ID)
	}
	later := func() time.Time { return time.Now().AddDate(0, 0, state.JournalContextBlockerStaleDays+2) }
	run := func(now func() time.Time, input string, args ...string) string {
		t.Helper()
		var output bytes.Buffer
		runner := Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome, Now: now}
		if input != "" {
			runner.Stdin = strings.NewReader(input)
		}
		if err := runner.Run(append([]string{"journal", "context"}, args...)); err != nil {
			t.Fatalf("journal context %v error = %v\n%s", args, err, output.String())
		}
		return output.String()
	}

	today := run(nil, "")
	if !strings.Contains(today, "unresolved-blockers: showing 2 of 2 (resolve one:") {
		t.Fatalf("default digest today = %q, want both blockers listed", today)
	}
	stale := "unresolved-blockers: showing 0 of 2 (more: loaf journal context --layer unresolved-blockers --limit 2; 2 older than 30 days not listed)"
	var hook claudeSessionStartOutput
	if err := json.Unmarshal([]byte(run(later, `{"hook_event_name":"SessionStart","source":"startup","session_id":"s1"}`, "--from-hook", "--claude-code")), &hook); err != nil || hook.HookSpecificOutput == nil {
		t.Fatalf("claude hook output did not decode: %v", err)
	}
	for name, digest := range map[string]string{"cli": run(later, ""), "claude hook": hook.HookSpecificOutput.AdditionalContext} {
		if !strings.Contains(digest, stale) {
			t.Fatalf("%s digest a month later = %q, want %q", name, digest, stale)
		}
		for _, id := range ids {
			if strings.Contains(digest, id) {
				t.Fatalf("%s digest a month later lists stale blocker %s", name, id)
			}
		}
	}
	listed := run(later, "", "--layer", "unresolved-blockers", "--limit", "2")
	for _, id := range ids {
		if !strings.Contains(listed, id) {
			t.Fatalf("blocker layer a month later = %q, want %s", listed, id)
		}
	}
}
