package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

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
		open.ID + " [",
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
	if envelope.Code != state.JournalUnblockNotBlockCode || envelope.Key != decision.ID || !strings.Contains(envelope.Error, "it is a decision entry, not a block") {
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
	block := func(id, scope, message, createdAt string) state.JournalContextBlockerItem {
		return state.JournalContextBlockerItem{Key: scope, Block: state.JournalEntryRecord{ID: id, EntryType: "block", Scope: scope, Message: message, CreatedAt: createdAt}}
	}
	recent := block("01a1aaaa-0000-7000-8000-000000000001", "ci", "runner offline", "2026-10-20T09:00:00Z")
	staleReview := block("01a1aaaa-0000-7000-8000-000000000002", "review", "PR 253 review never started", "2026-09-11T09:00:00Z")
	staleReview.Stale = true
	staleRebase := block("01a1aaaa-0000-7000-8000-000000000003", "pr-212", "rebase needed", "2026-09-01T09:00:00Z")
	staleRebase.Stale = true
	digest := state.JournalContext{UnresolvedBlockers: state.JournalContextBlockerLayer{
		Available: true, AvailableCount: 3, ShownCount: 3, StaleCount: 2,
		ExpandCommand: "loaf journal context --layer unresolved-blockers --limit 10",
		Items:         []state.JournalContextBlockerItem{recent, staleReview, staleRebase},
	}}
	render := func(selected string) string {
		var output bytes.Buffer
		writeJournalContextHuman(&output, composeJournalContextCLIResult(digest, activeChangeLayer{}, selected))
		return output.String()
	}

	defaultDigest := render("")
	for _, want := range []string{
		"unresolved-blockers: showing 1 of 1 from the last 30 days",
		`resolve one: loaf journal log "unblock(<id>): how it was resolved"`,
		recent.Block.ID + " [2026-10-20 09:00] block(ci): runner offline",
		"older than 30 days, not listed: 2 (list every blocker: loaf journal context --layer unresolved-blockers --limit 10)",
	} {
		if !strings.Contains(defaultDigest, want) {
			t.Fatalf("default digest = %q, want %q", defaultDigest, want)
		}
	}
	for _, item := range []state.JournalContextBlockerItem{staleReview, staleRebase} {
		if strings.Contains(defaultDigest, item.Block.ID) {
			t.Fatalf("default digest = %q, lists stale blocker %s", defaultDigest, item.Block.ID)
		}
	}

	listed := render(journalContextLayerBlockers)
	if !strings.Contains(listed, "unresolved-blockers: showing 3 of 3") || strings.Contains(listed, "not listed") {
		t.Fatalf("blocker layer = %q, want every blocker listed", listed)
	}
	for _, item := range []state.JournalContextBlockerItem{recent, staleReview, staleRebase} {
		if !strings.Contains(listed, item.Block.ID+" [") {
			t.Fatalf("blocker layer = %q, want %s with its id", listed, item.Block.ID)
		}
	}

	onlyStale := digest
	onlyStale.UnresolvedBlockers.AvailableCount, onlyStale.UnresolvedBlockers.ShownCount = 2, 2
	onlyStale.UnresolvedBlockers.Items = []state.JournalContextBlockerItem{staleReview, staleRebase}
	var output bytes.Buffer
	writeJournalContextHuman(&output, composeJournalContextCLIResult(onlyStale, activeChangeLayer{}, ""))
	if got := output.String(); !strings.Contains(got, "unresolved-blockers: none from the last 30 days") || !strings.Contains(got, "older than 30 days, not listed: 2") || strings.Contains(got, "resolve one:") {
		t.Fatalf("all-stale default digest = %q, want a none line and the stale count", got)
	}
}
