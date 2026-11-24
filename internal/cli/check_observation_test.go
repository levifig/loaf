package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

// newHookStateProject returns a project directory with initialized local state.
func newHookStateProject(t *testing.T) (string, project.Root, state.PathResolver, state.Status) {
	t.Helper()
	base := t.TempDir()
	workingDir := filepath.Join(base, "project")
	if err := os.Mkdir(workingDir, 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := project.ResolveRoot(workingDir)
	if err != nil {
		t.Fatal(err)
	}
	resolver := state.PathResolver{StateHome: filepath.Join(base, "state")}
	status, err := state.Initialize(context.Background(), root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	return workingDir, root, resolver, status
}

func TestGatewayObservationsRecordAllowAndBlockWithoutPayload(t *testing.T) {
	t.Setenv("CURSOR_VERSION", "")
	workingDir, _, resolver, initialized := newHookStateProject(t)
	path := initialized.DatabasePath
	secret := "AKIA" + strings.Repeat("Z", 16)
	for _, test := range []struct {
		payload  string
		blocked  bool
		advisory bool
	}{
		{payload: `{"hook_event_name":"preToolUse","tool_name":"Shell","tool_input":{"command":"pwd"}}`},
		{payload: `{"hook_event_name":"preToolUse","tool_name":"Shell","tool_input":{"command":"echo ` + secret + `"}}`, blocked: true},
		{payload: `{"hook_event_name":"preToolUse","tool_name":"Shell","tool_input":{"command":"echo ` + secret + `"}}`, advisory: true},
	} {
		var output bytes.Buffer
		args := []string{"check", "--hook", "check-secrets", "--cursor-hook"}
		if test.advisory {
			args = append(args, "--advisory")
		}
		err := (Runner{WorkingDir: workingDir, StateHome: resolver.StateHome, Stdin: strings.NewReader(test.payload), Stdout: &output}).Run(args)
		assertCursorCheckDecision(t, output.Bytes(), err, test.blocked)
	}
	store, err := state.OpenStoreReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := store.QueryContext(context.Background(), `
SELECT tool_name, decision, reason_class, observed_harness, observed_harness_version, count
FROM hook_gateway_counts ORDER BY decision`)
	if err != nil {
		t.Fatal(err)
	}
	var decisions []string
	for rows.Next() {
		var tool, decision, reason, harness, version string
		var count int
		if err := rows.Scan(&tool, &decision, &reason, &harness, &version, &count); err != nil {
			t.Fatal(err)
		}
		if tool != "Bash" || harness != "cursor" || version != "unknown" || count != 1 {
			t.Fatalf("unsafe or incomplete observation: tool=%q decision=%q reason=%q harness=%q version=%q count=%d", tool, decision, reason, harness, version, count)
		}
		decisions = append(decisions, decision+":"+reason)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	store.Close()
	if strings.Join(decisions, ",") != "allow:passed,block:policy,warn:advisory" {
		t.Fatalf("decisions = %v", decisions)
	}
	for _, suffix := range []string{"", "-wal"} {
		body, err := os.ReadFile(path + suffix)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(body, []byte(secret)) {
			t.Fatalf("secret appeared in local state file %s", suffix)
		}
	}
	var statusJSON bytes.Buffer
	if err := (Runner{WorkingDir: workingDir, StateHome: resolver.StateHome, Stdout: &statusJSON}).Run([]string{"telemetry", "status", "--json"}); err != nil {
		t.Fatal(err)
	}
	var status telemetryStatusOutput
	if err := json.Unmarshal(statusJSON.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if len(status.Rows) != 3 || status.Rows[0].Count != 1 || status.Rows[1].Count != 1 || status.Rows[2].Count != 1 {
		t.Fatalf("status = %#v", status)
	}
}

// A hook records its outcome without the full state validation, whose integrity
// and parity scans grow with the database and would run on every tool call.
func TestGatewayObservationSkipsFullStateValidation(t *testing.T) {
	workingDir, root, resolver, status := newHookStateProject(t)
	path := status.DatabasePath
	store, err := state.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	// A journal origin without its entry fails the operational invariants.
	if _, err := store.DB().Exec(`
INSERT INTO journal_origins (journal_entry_id, project_id, envelope_version, capture_mechanism, created_at)
VALUES ('missing-entry', ?, 1, 'test', '2026-11-01T00:00:00Z')`, status.ProjectID); err != nil {
		t.Fatal(err)
	}
	store.Close()
	if invalid, err := state.OpenProjectStoreForWrite(context.Background(), root, resolver); err == nil {
		invalid.Close()
		t.Fatal("fixture must fail full state validation")
	}

	payload := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`
	var output bytes.Buffer
	if err := (Runner{WorkingDir: workingDir, StateHome: resolver.StateHome, Stdin: strings.NewReader(payload), Stdout: &output}).Run([]string{"check", "--hook", "validate-sql-safety"}); err != nil {
		t.Fatal(err)
	}
	store, err = state.OpenStoreReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var count int
	if err := store.QueryRowContext(context.Background(), `
SELECT COALESCE(SUM(count), 0) FROM hook_gateway_counts
WHERE hook_id = 'validate-sql-safety' AND tool_name = 'Bash' AND decision = 'allow'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("gateway count = %d, want 1", count)
	}
}

// A long write elsewhere must not stall the tool call: the hook gives up its
// count instead of waiting out the database's normal five-second busy timeout.
func TestGatewayObservationDoesNotWaitOnABusyDatabase(t *testing.T) {
	workingDir, _, resolver, status := newHookStateProject(t)
	store, err := state.OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tx, err := store.DB().Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
INSERT INTO hook_gateway_counts (project_id, month, hook_id, tool_name, decision, reason_class,
                                 observed_harness, observed_harness_version, count)
VALUES (?, '2026-01', 'held', 'Bash', 'allow', 'passed', 'unknown', 'unknown', 1)`, status.ProjectID); err != nil {
		t.Fatal(err)
	}

	payload := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`
	var output bytes.Buffer
	start := time.Now()
	err = (Runner{WorkingDir: workingDir, StateHome: resolver.StateHome, Stdin: strings.NewReader(payload), Stdout: &output}).Run([]string{"check", "--hook", "validate-sql-safety"})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("hook waited %v on a busy database", elapsed)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM hook_gateway_counts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("gateway rows = %d, want 0: the held write lock did not block the hook's count", count)
	}
}
