package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

func TestTelemetryReportsJournalOnlyHarnessUsage(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "test-thread")
	t.Setenv("CODEX_VERSION", "1.2.3")
	t.Setenv("CLAUDECODE", "")
	workingDir := t.TempDir()
	root, err := project.ResolveRoot(workingDir)
	if err != nil {
		t.Fatal(err)
	}
	resolver := state.PathResolver{StateHome: filepath.Join(t.TempDir(), "state")}
	if _, err := state.Initialize(context.Background(), root, resolver); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	runner := Runner{WorkingDir: workingDir, StateHome: resolver.StateHome, Stdout: &output, Stderr: &output}
	if err := runner.Run([]string{"journal", "log", "discover(test): journal-only usage"}); err != nil {
		t.Fatal(err)
	}
	if err := runner.Run([]string{"issue", "new", "A private question?", "--kind", "decision", "--json"}); err != nil {
		t.Fatalf("decision CLI: %v: %s", err, output.String())
	}
	output.Reset()
	if err := runner.Run([]string{"telemetry", "status", "--json"}); err != nil {
		t.Fatal(err)
	}
	var status struct {
		Rows       []json.RawMessage `json:"rows"`
		Continuity []struct {
			ProjectID string `json:"project_id"`
			Harness   string `json:"observed_harness"`
			Count     int    `json:"count"`
		} `json:"continuity_rows"`
	}
	if err := json.Unmarshal(output.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if len(status.Rows) != 0 || len(status.Continuity) != 1 || status.Continuity[0].Harness != "codex" || status.Continuity[0].Count != 2 || status.Continuity[0].ProjectID == "" {
		t.Fatalf("journal-only status=%s", output.String())
	}
}

func TestObservedHarnessRejectsConflictingMarkers(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "inherited-parent")
	t.Setenv("CODEX_VERSION", "1.2.3")
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_VERSION", "2.0.0")
	if harness, version := observedHarnessFromEnvironment(); harness != "unknown" || version != "unknown" {
		t.Fatalf("conflicting markers=%q/%q, want unknown/unknown", harness, version)
	}
}
