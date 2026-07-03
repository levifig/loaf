package cli

import (
	"bytes"
	"strings"
	"testing"
)

// setupJournalHookRunner initializes a temp project with SQLite state so the
// journal hook commands (SessionStart digest, --from-hook logging, guidance
// emitters) can run against an isolated database.
func setupJournalHookRunner(t *testing.T) (workingDir string, stateHome string) {
	t.Helper()
	workingDir = realpath(t, t.TempDir())
	stateHome = t.TempDir()
	writeCLIAgentsFile(t, workingDir, "specs/SPEC-001-active.md", "---\nid: SPEC-001\ntitle: Active Spec\nstatus: implementing\n---\n# Active Spec\n")
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"state", "migrate", "markdown", "--apply"}); err != nil {
		t.Fatalf("state migrate markdown --apply error = %v\n%s", err, stdout.String())
	}
	return workingDir, stateHome
}

func TestJournalContextFromHookExitsSilentlyForSubagent(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1","agent_id":"subagent"}`),
	}.Run([]string{"journal", "context", "--from-hook"})
	if err != nil {
		t.Fatalf("journal context --from-hook (subagent) error = %v", err)
	}
	if stdout.String() != "" {
		t.Fatalf("journal context --from-hook (subagent) stdout = %q, want empty (silent exit)", stdout.String())
	}
}

func TestJournalContextFromHookEmitsDigestForPrimarySession(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	logRun := Runner{Stdout: &bytes.Buffer{}, WorkingDir: workingDir, StateHome: stateHome}
	if err := logRun.Run([]string{"journal", "log", "wrap(hooks): checkpoint one"}); err != nil {
		t.Fatalf("journal log wrap error = %v", err)
	}

	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1"}`),
	}.Run([]string{"journal", "context", "--from-hook"})
	if err != nil {
		t.Fatalf("journal context --from-hook error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"loaf journal context", "latest wrap:", "wrap(hooks): checkpoint one"} {
		if !strings.Contains(out, want) {
			t.Fatalf("journal context --from-hook stdout = %q, want %q", out, want)
		}
	}
}

func TestJournalLogFromHookDerivesTaskCompletedEntry(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1","hook_event_name":"TaskCompleted","task_description":"wire journal hooks"}`),
	}.Run([]string{"journal", "log", "--from-hook"})
	if err != nil {
		t.Fatalf("journal log --from-hook (task) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "task(completed): wire journal hooks") {
		t.Fatalf("journal log --from-hook (task) stdout = %q, want task(completed) entry", stdout.String())
	}
}

func TestJournalLogFromHookExitsSilentlyForSubagent(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1","agent_id":"sub","hook_event_name":"TaskCompleted","task_description":"x"}`),
	}.Run([]string{"journal", "log", "--from-hook"})
	if err != nil {
		t.Fatalf("journal log --from-hook (subagent) error = %v", err)
	}
	if stdout.String() != "" {
		t.Fatalf("journal log --from-hook (subagent) stdout = %q, want empty (no write)", stdout.String())
	}
}

func TestJournalLogFromHookNoOpsOnUnrecognizedPayload(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1"}`),
	}.Run([]string{"journal", "log", "--from-hook"})
	if err != nil {
		t.Fatalf("journal log --from-hook (empty) error = %v", err)
	}
	if stdout.String() != "" {
		t.Fatalf("journal log --from-hook (empty) stdout = %q, want empty (no derivable entry)", stdout.String())
	}
}

func TestJournalContextForPromptInjectsJournalFirstPrinciples(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1"}`),
	}.Run([]string{"journal", "context", "for-prompt"})
	if err != nil {
		t.Fatalf("journal context for-prompt error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "loaf journal log") {
		t.Fatalf("for-prompt stdout = %q, want journal-first log guidance", out)
	}
	if strings.Contains(out, "loaf session") {
		t.Fatalf("for-prompt stdout = %q, must not reference the deleted session command", out)
	}
}

func TestJournalContextForPromptExitsSilentlyForSubagent(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"agent_id":"sub"}`),
	}.Run([]string{"journal", "context", "for-prompt"})
	if err != nil {
		t.Fatalf("journal context for-prompt (subagent) error = %v", err)
	}
	if stdout.String() != "" {
		t.Fatalf("for-prompt (subagent) stdout = %q, want empty", stdout.String())
	}
}

func TestJournalContextForCompactInjectsFlushGuidance(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1"}`),
	}.Run([]string{"journal", "context", "for-compact"})
	if err != nil {
		t.Fatalf("journal context for-compact error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"COMPACTION IMMINENT", "loaf journal log", "wrap(scope):"} {
		if !strings.Contains(out, want) {
			t.Fatalf("for-compact stdout = %q, want %q", out, want)
		}
	}
	// Journal-first: no session-state snapshot language.
	for _, forbidden := range []string{"loaf session", "Current State", "session journal"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("for-compact stdout = %q, must not contain session-lifecycle language %q", out, forbidden)
		}
	}
}

func TestJournalContextForResumptionEmitsDigest(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	logRun := Runner{Stdout: &bytes.Buffer{}, WorkingDir: workingDir, StateHome: stateHome}
	if err := logRun.Run([]string{"journal", "log", "wrap(hooks): resumption checkpoint"}); err != nil {
		t.Fatalf("journal log wrap error = %v", err)
	}
	var stdout bytes.Buffer
	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
		StateHome:  stateHome,
		Stdin:      strings.NewReader(`{"session_id":"s1"}`),
	}.Run([]string{"journal", "context", "for-resumption"})
	if err != nil {
		t.Fatalf("journal context for-resumption error = %v", err)
	}
	if !strings.Contains(stdout.String(), "wrap(hooks): resumption checkpoint") {
		t.Fatalf("for-resumption stdout = %q, want the layered digest", stdout.String())
	}
}
