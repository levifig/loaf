package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/state"
)

func TestCursorCheckNativeDecisions(t *testing.T) {
	secret := "AKIA" + strings.Repeat("A", 16)
	cases := []struct {
		name, hook, tool, command, path, content string
		advisory, deny                           bool
	}{
		{name: "benign shell", hook: "check-secrets", tool: "Shell", command: "pwd"},
		{name: "shell secret", hook: "check-secrets", tool: "Shell", command: "echo " + secret, deny: true},
		{name: "write secret", hook: "check-secrets", tool: "Write", path: "example.txt", content: secret, deny: true},
		{name: "replace secret", hook: "check-secrets", tool: "StrReplace", path: "example.txt", content: secret, deny: true},
		{name: "destructive shell", hook: "validate-infra-safety", tool: "Shell", command: "kubectl delete namespace prod", deny: true},
		{name: "replace artifact", hook: "artifact-body-write", tool: "StrReplace", path: ".agents/reports/example.md", content: "body", deny: true},
		{name: "advisory secret", hook: "check-secrets", tool: "Shell", command: "echo " + secret, advisory: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(map[string]any{"hook_event_name": "preToolUse", "tool_name": tc.tool, "tool_input": map[string]string{"command": tc.command, "file_path": tc.path, "new_string": tc.content}})
			args := []string{"check", "--hook", tc.hook, "--cursor-hook"}
			if tc.advisory {
				args = append(args, "--advisory")
			}
			var stdout, stderr bytes.Buffer
			err := (Runner{WorkingDir: t.TempDir(), Stdin: bytes.NewReader(payload), Stdout: &stdout, Stderr: &stderr}).Run(args)
			assertCursorCheckDecision(t, stdout.Bytes(), err, tc.deny)
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func assertCursorCheckDecision(t *testing.T, body []byte, err error, deny bool) {
	t.Helper()
	var output struct {
		Permission   string `json:"permission"`
		AgentMessage string `json:"agent_message"`
	}
	if json.Unmarshal(body, &output) != nil {
		t.Fatalf("invalid JSON %q, error %v", body, err)
	}
	want := "allow"
	if deny {
		want = "deny"
	}
	if output.Permission != want {
		t.Fatalf("output = %s, want permission %s; error %v", body, want, err)
	}
	if deny {
		var exitErr interface{ ExitCode() int }
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 || output.AgentMessage == "" {
			t.Fatalf("denial must have exit 2 and diagnostic: %s; %v", body, err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
}

func TestCursorCheckFailuresRemainValidDenials(t *testing.T) {
	for _, tc := range []struct {
		name, hook, payload string
		readerError         bool
	}{
		{name: "malformed", hook: "check-secrets", payload: "{"},
		{name: "empty", hook: "check-secrets"},
		{name: "missing command", hook: "check-secrets", payload: `{"tool_name":"Shell","tool_input":{}}`},
		{name: "missing file path", hook: "artifact-body-write", payload: `{"tool_name":"StrReplace","tool_input":{"new_string":"body"}}`},
		{name: "missing tool", hook: "check-secrets", payload: "{}"},
		{name: "unknown hook", hook: "unknown", payload: `{"tool_name":"Shell","tool_input":{"command":"pwd"}}`},
		{name: "unreadable", hook: "check-secrets", readerError: true},
		{name: "relative cwd", hook: "check-secrets", payload: `{"cwd":"relative","tool_name":"Shell","tool_input":{"command":"pwd"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			runner := Runner{WorkingDir: t.TempDir(), Stdin: strings.NewReader(tc.payload), Stdout: &stdout}
			if tc.readerError {
				runner.Stdin = failReader{err: errors.New("closed")}
			}
			err := runner.Run([]string{"check", "--hook", tc.hook, "--cursor-hook"})
			assertCursorCheckDecision(t, stdout.Bytes(), err, true)
		})
	}
}

func TestCursorChecksDoNotDependOnContinuityDatabase(t *testing.T) {
	for _, mode := range []string{"missing", "incompatible", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			database := filepath.Join(t.TempDir(), "loaf.sqlite")
			t.Setenv("LOAF_DB", database)
			if mode == "incompatible" {
				store, err := state.OpenStore(database)
				if err != nil {
					t.Fatal(err)
				}
				if err = store.ApplyMigrations(t.Context()); err != nil {
					t.Fatal(err)
				}
				if _, err = store.DB().Exec("INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(999,'future','future','2026-09-29T00:00:00Z')"); err != nil {
					t.Fatal(err)
				}
				store.Close()
			} else if mode == "unavailable" {
				if err := os.WriteFile(database, []byte("not SQLite"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(database)
			hooks, err := readNativeBuildHooks(filepath.Join(testRepositoryRoot(t), "config", "hooks.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			for _, hook := range hooks {
				if hook.section != "pre-tool" || hook.instruction != "" || hook.typeName == "prompt" {
					continue
				}
				var stdout bytes.Buffer
				runner := Runner{WorkingDir: root, Stdin: strings.NewReader(`{"hook_event_name":"preToolUse","tool_name":"Shell","tool_input":{"command":"pwd"}}`), Stdout: &stdout}
				command := nativeCursorHookCommand(hook)
				err := runner.Run(strings.Fields(command)[1:])
				assertCursorCheckDecision(t, stdout.Bytes(), err, false)
			}
			var blocked bytes.Buffer
			err = (Runner{WorkingDir: root, Stdin: strings.NewReader(`{"tool_name":"Shell","tool_input":{"command":"kubectl delete namespace prod"}}`), Stdout: &blocked}).Run([]string{"check", "--hook", "validate-infra-safety", "--cursor-hook"})
			assertCursorCheckDecision(t, blocked.Bytes(), err, true)
			after, _ := os.ReadFile(database)
			if !bytes.Equal(before, after) {
				t.Fatal("hook changed database")
			}
			if mode == "missing" {
				if _, err := os.Stat(database); !os.IsNotExist(err) {
					t.Fatalf("missing database was created: %v", err)
				}
			}
		})
	}
}

func TestCursorNativeToolMatchers(t *testing.T) {
	entry := nativeCursorHookEntry(nativeBuildHook{id: "check-secrets", matcher: "Edit|Write|Bash", failClosed: true}, 30000, true)
	if entry.Matcher != "StrReplace|Write|Shell" || !entry.FailClosed {
		t.Fatalf("entry = %#v", entry)
	}
	if !strings.Contains(entry.Command, "--cursor-hook") {
		t.Fatalf("command = %q", entry.Command)
	}
}

func TestCursorCheckUsesPayloadWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents", "specs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "specs", "example.md"), []byte("invalid render\n<!-- loaf:render invalid -->"), 0600); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"cwd": root, "tool_name": "Shell", "tool_input": map[string]string{"command": "git push"}})
	var stdout bytes.Buffer
	err := (Runner{WorkingDir: t.TempDir(), Stdout: &stdout, Stdin: bytes.NewReader(payload)}).Run([]string{"check", "--hook", "render-drift", "--cursor-hook"})
	assertCursorCheckDecision(t, stdout.Bytes(), err, true)
}

func TestCursorLegacyCommandEmitsNativeJSON(t *testing.T) {
	for _, jsonFlag := range []bool{false, true} {
		t.Run(map[bool]string{false: "unflagged", true: "generic JSON flag"}[jsonFlag], func(t *testing.T) {
			var stdout bytes.Buffer
			args := []string{"check", "--hook", "check-secrets"}
			if jsonFlag {
				args = append(args, "--json")
			}
			err := (Runner{WorkingDir: t.TempDir(), Stdout: &stdout, Stdin: strings.NewReader(`{"hook_event_name":"preToolUse","tool_name":"Shell","tool_input":{"command":"pwd"}}`)}).Run(args)
			assertCursorCheckDecision(t, stdout.Bytes(), err, false)
		})
	}
}
