package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/state"
)

func TestJournalCommandsRefuseUnknownMovedCheckoutAndPreserveIdentity(t *testing.T) {
	parent := t.TempDir()
	oldPath := filepath.Join(parent, "old checkout's path")
	newPath := filepath.Join(parent, "new checkout with spaces'")
	if err := os.MkdirAll(oldPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(old) error = %v", err)
	}
	if err := os.MkdirAll(newPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(new) error = %v", err)
	}
	oldPath = realpath(t, oldPath)
	newPath = realpath(t, newPath)
	databasePath := filepath.Join(t.TempDir(), "isolated journal.sqlite")
	t.Setenv("LOAF_DB", databasePath)

	var stdout bytes.Buffer
	oldRunner := Runner{Stdout: &stdout, WorkingDir: oldPath}
	if err := oldRunner.Run([]string{"state", "init", "--json"}); err != nil {
		t.Fatalf("state init --json error = %v\n%s", err, stdout.String())
	}
	stdout.Reset()
	if err := oldRunner.Run([]string{"journal", "log", "decision(move): preserve continuity", "--json"}); err != nil {
		t.Fatalf("journal log --json error = %v\n%s", err, stdout.String())
	}
	var logged state.JournalLogResult
	if err := json.Unmarshal(stdout.Bytes(), &logged); err != nil {
		t.Fatalf("decode journal log = %v\n%s", err, stdout.String())
	}
	if logged.ProjectID == "" || logged.ID == "" {
		t.Fatalf("logged = %#v, want project and entry IDs", logged)
	}

	countRows := func(query string) int {
		db := openCLITestDB(t, databasePath)
		defer closeCLITestDB(t, db)
		return sqliteCount(t, db, query)
	}
	projectsBefore := countRows(`SELECT COUNT(*) FROM projects`)
	pathsBefore := countRows(`SELECT COUNT(*) FROM project_paths`)
	entriesBefore := countRows(`SELECT COUNT(*) FROM journal_entries`)

	if err := os.RemoveAll(newPath); err != nil {
		t.Fatalf("RemoveAll(new) error = %v", err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatalf("Rename(old,new) error = %v", err)
	}
	newPath = realpath(t, newPath)
	moveCommand := fmt.Sprintf("loaf project move --from '%s' --to '%s' --dry-run", strings.ReplaceAll(oldPath, "'", "'\\''"), strings.ReplaceAll(newPath, "'", "'\\''"))

	commands := [][]string{
		{"journal", "recent"},
		{"journal", "search", "continuity"},
		{"journal", "context"},
		{"journal", "log", "decision(move): must refuse"},
	}
	for _, args := range commands {
		var human bytes.Buffer
		err := (Runner{Stdout: &human, WorkingDir: newPath}).Run(args)
		if err == nil {
			t.Fatalf("Run(%v) error = nil, want unregistered project identity", args)
		}
		for _, want := range []string{newPath, moveCommand, "loaf state init"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("Run(%v) error = %q, want %q", args, err, want)
			}
		}

		jsonArgs := append(append([]string{}, args...), "--json")
		var machine bytes.Buffer
		err = (Runner{Stdout: &machine, WorkingDir: newPath}).Run(jsonArgs)
		if err == nil {
			t.Fatalf("Run(%v) error = nil, want JSON command error", jsonArgs)
		}
		var output struct {
			ContractVersion   int      `json:"contract_version"`
			Command           string   `json:"command"`
			Error             string   `json:"error"`
			Code              string   `json:"code"`
			CurrentPath       string   `json:"current_path"`
			KnownCurrentPaths []string `json:"known_current_paths"`
			Suggestions       []struct {
				Action  string `json:"action"`
				Command string `json:"command"`
			} `json:"suggestions"`
		}
		if err := json.Unmarshal(machine.Bytes(), &output); err != nil {
			t.Fatalf("decode Run(%v) JSON error = %v\n%s", jsonArgs, err, machine.String())
		}
		if output.ContractVersion != state.StateJSONContractVersion || output.Code != state.ProjectIdentityUnregisteredCode || output.CurrentPath != newPath {
			t.Fatalf("Run(%v) JSON error = %#v, want contract/code/path", jsonArgs, output)
		}
		if len(output.KnownCurrentPaths) != 1 || output.KnownCurrentPaths[0] != oldPath {
			t.Fatalf("Run(%v) known paths = %#v, want [%q]", jsonArgs, output.KnownCurrentPaths, oldPath)
		}
		if len(output.Suggestions) != 2 || output.Suggestions[0].Action != "move-project" || output.Suggestions[0].Command != moveCommand || output.Suggestions[1].Action != "initialize-project" || output.Suggestions[1].Command != "loaf state init" {
			t.Fatalf("Run(%v) suggestions = %#v, want move then initialize", jsonArgs, output.Suggestions)
		}
		if output.Error == "" || !strings.Contains(output.Error, moveCommand) {
			t.Fatalf("Run(%v) JSON error = %#v, want deterministic human diagnostic", jsonArgs, output)
		}
		if got := countRows(`SELECT COUNT(*) FROM projects`); got != projectsBefore {
			t.Fatalf("projects after Run(%v) = %d, want %d", jsonArgs, got, projectsBefore)
		}
		if got := countRows(`SELECT COUNT(*) FROM project_paths`); got != pathsBefore {
			t.Fatalf("project_paths after Run(%v) = %d, want %d", jsonArgs, got, pathsBefore)
		}
		if got := countRows(`SELECT COUNT(*) FROM journal_entries`); got != entriesBefore {
			t.Fatalf("journal_entries after Run(%v) = %d, want %d", jsonArgs, got, entriesBefore)
		}
	}

	stdout.Reset()
	if err := (Runner{Stdout: &stdout, WorkingDir: newPath}).Run([]string{"project", "move", "--from", oldPath, "--to", newPath, "--json"}); err != nil {
		t.Fatalf("project move --json error = %v\n%s", err, stdout.String())
	}
	var moved state.ProjectMoveResult
	if err := json.Unmarshal(stdout.Bytes(), &moved); err != nil {
		t.Fatalf("decode project move = %v\n%s", err, stdout.String())
	}
	if moved.Project.ID != logged.ProjectID || moved.Project.CurrentPath != newPath {
		t.Fatalf("moved project = %#v, want ID %q at %q", moved.Project, logged.ProjectID, newPath)
	}

	stdout.Reset()
	if err := (Runner{Stdout: &stdout, WorkingDir: newPath}).Run([]string{"journal", "recent", "--json"}); err != nil {
		t.Fatalf("journal recent after move --json error = %v\n%s", err, stdout.String())
	}
	var recent state.JournalRecent
	if err := json.Unmarshal(stdout.Bytes(), &recent); err != nil {
		t.Fatalf("decode recent after move = %v\n%s", err, stdout.String())
	}
	if recent.ProjectID != logged.ProjectID || len(recent.Entries) != 1 || recent.Entries[0].ID != logged.ID {
		t.Fatalf("recent after move = %#v, want original project and entry", recent)
	}

	db := openCLITestDB(t, databasePath)
	defer closeCLITestDB(t, db)
	if got := sqliteCount(t, db, `SELECT COUNT(*) FROM projects`); got != 1 {
		t.Fatalf("projects after move = %d, want 1", got)
	}
	if got := sqliteCount(t, db, `SELECT COUNT(*) FROM project_paths WHERE is_current = 1`); got != 1 {
		t.Fatalf("current project mappings after move = %d, want 1", got)
	}
}

// TestJournalSearchReturnsHitsWithoutScanningDocs proves `loaf journal search`
// (SPEC-056 M1) returns journal hits without touching the docs index: a broken
// (invalid UTF-8) docs fixture that would fail the docs scan must not affect the
// journal search command.
func TestJournalSearchReturnsHitsWithoutScanningDocs(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)

	logRun := Runner{Stdout: &bytes.Buffer{}, WorkingDir: workingDir, StateHome: stateHome}
	if err := logRun.Run([]string{"journal", "log", "decision(search): sphinx searchable entry"}); err != nil {
		t.Fatalf("journal log error = %v", err)
	}

	// A docs file whose bytes are not valid UTF-8 makes any docs-index scan fail.
	// If journal search routed through the global docs-indexing path, it would
	// error on this fixture.
	writeBrokenCLIDocsFile(t, workingDir, "docs/broken.md")

	var stdout bytes.Buffer
	err := Runner{Stdout: &stdout, WorkingDir: workingDir, StateHome: stateHome}.Run([]string{"journal", "search", "sphinx"})
	if err != nil {
		t.Fatalf("journal search error = %v (must not scan docs)", err)
	}
	out := stdout.String()
	if strings.Contains(out, "docs") && strings.Contains(out, "UTF-8") {
		t.Fatalf("journal search stdout = %q, must not report a docs scan error", out)
	}
	for _, want := range []string{"loaf journal search", "results: 1", "sphinx searchable entry"} {
		if !strings.Contains(out, want) {
			t.Fatalf("journal search stdout = %q, want %q", out, want)
		}
	}
}

// TestJournalSearchJSONReturnsOnlyJournalHits confirms the JSON output shape
// carries the journal hits the command emits today.
func TestJournalSearchJSONReturnsOnlyJournalHits(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	logRun := Runner{Stdout: &bytes.Buffer{}, WorkingDir: workingDir, StateHome: stateHome}
	if err := logRun.Run([]string{"journal", "log", "finding(search): obsidian json hit"}); err != nil {
		t.Fatalf("journal log error = %v", err)
	}
	writeBrokenCLIDocsFile(t, workingDir, "docs/broken.md")

	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "search", "--json", "obsidian"}); err != nil {
		t.Fatalf("journal search --json error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"source": "journal_entry"`) {
		t.Fatalf("journal search --json stdout = %q, want journal_entry hit", out)
	}
	if strings.Contains(out, `"source": "docs_index"`) || strings.Contains(out, `"source": "artifact_body"`) {
		t.Fatalf("journal search --json stdout = %q, want only journal hits", out)
	}
}

// TestJournalSearchErrorsOnMissingStateNonHook confirms the interactive search
// path still errors when state is missing (M2 leaves non-hook behavior intact).
func TestJournalSearchErrorsOnMissingStateNonHook(t *testing.T) {
	workingDir := realpath(t, t.TempDir())
	stateHome := t.TempDir() // no state init: database absent

	var stdout bytes.Buffer
	err := Runner{Stdout: &stdout, WorkingDir: workingDir, StateHome: stateHome}.Run([]string{"journal", "search", "anything"})
	if err == nil {
		t.Fatal("journal search error = nil, want uninitialized-state error on missing DB")
	}
	if !strings.Contains(err.Error(), "SQLite state database is not initialized") {
		t.Fatalf("journal search error = %v, want uninitialized-state sentinel", err)
	}
}

func writeBrokenCLIDocsFile(t *testing.T, rootPath string, relativePath string) {
	t.Helper()
	path := filepath.Join(rootPath, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte{0xff, 0xfe, 0xfd}, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", relativePath, err)
	}
}
