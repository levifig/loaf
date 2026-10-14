package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

func TestGatewayObservationsRecordAllowAndBlockWithoutPayload(t *testing.T) {
	t.Setenv("CURSOR_VERSION", "")
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
	if _, err := state.Initialize(context.Background(), root, resolver); err != nil {
		t.Fatal(err)
	}
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
	path, err := resolver.DatabasePath(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := state.OpenStoreReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := store.QueryContext(context.Background(), `
SELECT tool_name, decision, reason_class, observed_harness, observed_harness_version,
       command IS NULL, message IS NULL
FROM hook_events WHERE hook_phase = 'pre-tool' ORDER BY created_at, rowid`)
	if err != nil {
		t.Fatal(err)
	}
	var decisions []string
	for rows.Next() {
		var tool, decision, reason, harness, version string
		var noCommand, noMessage bool
		if err := rows.Scan(&tool, &decision, &reason, &harness, &version, &noCommand, &noMessage); err != nil {
			t.Fatal(err)
		}
		if tool != "Bash" || harness != "cursor" || version != "unknown" || !noCommand || !noMessage {
			t.Fatalf("unsafe or incomplete observation: tool=%q decision=%q reason=%q harness=%q version=%q noCommand=%t noMessage=%t", tool, decision, reason, harness, version, noCommand, noMessage)
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
