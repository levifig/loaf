package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
				if (hook.section != "pre-tool" && hook.section != "post-tool") || hook.instruction != "" || hook.typeName == "prompt" {
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

func TestCursorArtifactNamesAllowsRepairCommands(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "changes", "demo", "research", "pr-12-notes.md")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("draft notes"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	for _, explicit := range []bool{false, true} {
		for _, tc := range []struct {
			command string
			deny    bool
		}{
			{"pwd", false},
			{"git status", false},
			{"git mv docs/changes/demo/research/pr-12-notes.md docs/changes/demo/research/notes.md", false},
			{"git rm --cached docs/changes/demo/research/pr-12-notes.md", false},
			{"echo git commit", false},
			{`echo "notes; git commit"`, false},
			{"git config example.key commit", false},
			{"git commit -m notes", true},
			{"git add CHANGELOG.md && git commit -m notes", true},
			{"FOO=bar git commit -m notes", true},
			{"git -c commit.gpgsign=true commit -m notes", true},
			{"git -C . commit -m notes", true},
		} {
			command := tc.command
			t.Run(fmt.Sprintf("explicit=%t/%s", explicit, command), func(t *testing.T) {
				payload, err := json.Marshal(map[string]any{"hook_event_name": "preToolUse", "cwd": root, "tool_name": "Shell", "tool_input": map[string]string{"command": command}})
				if err != nil {
					t.Fatal(err)
				}
				args := []string{"check", "--hook", "artifact-names"}
				if explicit {
					args = append(args, "--cursor-hook")
				}
				var stdout bytes.Buffer
				err = (Runner{WorkingDir: t.TempDir(), Stdout: &stdout, Stdin: bytes.NewReader(payload)}).Run(args)
				assertCursorCheckDecision(t, stdout.Bytes(), err, tc.deny)
			})
		}
	}
	var stdout bytes.Buffer
	err := (Runner{WorkingDir: root, Stdout: &stdout, Stdin: strings.NewReader("")}).Run([]string{"check", "--hook", "artifact-names", "--json"})
	var result checkJSONOutput
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil || !result.Blocked || err == nil {
		t.Fatalf("manual scan must still deny: %s; %v; %v", stdout.Bytes(), err, decodeErr)
	}
}

func TestCursorCheckKeepsOriginalPayloadSizeBoundary(t *testing.T) {
	content := strings.Repeat("<>&", int((projectFileReadLimit-1024)/3))
	payload := `{"hook_event_name":"preToolUse","tool_name":"Write","tool_input":{"file_path":"example.html","content":"` + content + `"}}`
	if int64(len(payload)) >= projectFileReadLimit {
		t.Fatal("fixture must fit the original payload limit")
	}
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit=%t", explicit), func(t *testing.T) {
			args := []string{"check", "--hook", "check-secrets"}
			if explicit {
				args = append(args, "--cursor-hook")
			}
			var stdout bytes.Buffer
			err := (Runner{WorkingDir: t.TempDir(), Stdout: &stdout, Stdin: strings.NewReader(payload)}).Run(args)
			assertCursorCheckDecision(t, stdout.Bytes(), err, false)
		})
	}
	var stdout bytes.Buffer
	err := (Runner{WorkingDir: t.TempDir(), Stdout: &stdout, Stdin: strings.NewReader(payload + strings.Repeat(" ", 2048))}).Run([]string{"check", "--hook", "check-secrets", "--cursor-hook"})
	assertCursorCheckDecision(t, stdout.Bytes(), err, true)
}

func TestCursorKbNudgeIsAdvisoryAndPreservesNativePayload(t *testing.T) {
	root := writeKbCheckFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("TOOL_INPUT", "")
	for attempt, conversation := range []string{"one", "one", "two"} {
		payload, err := json.Marshal(map[string]any{"hook_event_name": "postToolUse", "conversation_id": conversation, "cwd": root, "tool_name": "Write", "tool_input": map[string]string{"file_path": "src/nested/main.go"}})
		if err != nil {
			t.Fatal(err)
		}
		var stdout bytes.Buffer
		err = (Runner{WorkingDir: t.TempDir(), Stdin: bytes.NewReader(payload), Stdout: &stdout}).Run([]string{"check", "--hook", "kb-staleness-nudge", "--json", "--cursor-hook"})
		assertCursorCheckDecision(t, stdout.Bytes(), err, false)
		var result cursorCheckOutput
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		wantWarnings := 1
		if attempt == 1 {
			wantWarnings = 0
		}
		if len(result.Warnings) != wantWarnings {
			t.Fatalf("warnings = %v, want %d", result.Warnings, wantWarnings)
		}
		if wantWarnings > 0 {
			if !strings.Contains(result.AgentMessage, "docs/knowledge/go.md") {
				t.Fatalf("missing repository knowledge warning: %s", stdout.Bytes())
			}
		}
	}
	for _, payload := range []string{"{", "", `{"cwd":"relative"}`} {
		var stdout bytes.Buffer
		err := (Runner{WorkingDir: t.TempDir(), Stdin: strings.NewReader(payload), Stdout: &stdout}).Run([]string{"check", "--hook", "kb-staleness-nudge", "--cursor-hook"})
		assertCursorCheckDecision(t, stdout.Bytes(), err, false)
	}
}

func TestArtifactNamesChecksCompoundCommandsAcrossHarnesses(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "research", "pr-12-notes.md")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("notes"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	var stdout bytes.Buffer
	err := (Runner{WorkingDir: root, Stdin: strings.NewReader(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"git add . && git commit -m notes"}}`), Stdout: &stdout}).Run([]string{"check", "--hook", "artifact-names", "--json"})
	var result checkJSONOutput
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil || !result.Blocked || err == nil {
		t.Fatalf("compound commit must deny: %s; %v; %v", stdout.Bytes(), err, decodeErr)
	}
}

func TestCursorValidateCommitIgnoresLiteralMentions(t *testing.T) {
	for _, tc := range []struct {
		name, command string
		deny          bool
	}{
		{"quoted mention", `echo "git commit -m wip" >> notes.md`, false},
		{"pull request body", "gh pr create --title \"fix: guard\" --body \"$(cat <<'EOF'\nRepro: git commit -m wip\nEOF\n)\"", false},
		{"heredoc quoted phrase", "git commit -m \"$(cat <<'EOF'\nfix: explain \"permission denied\" errors\n\nBody.\nEOF\n)\"", false},
		{"heredoc unmatched quote", "git commit -m \"$(cat <<'EOF'\nfix: support 5\" disks\nEOF\n)\"", false},
		{"short message heredoc invalid", "git commit -am \"$(cat <<'EOF'\nwip\nEOF\n)\"", true},
		{"long message heredoc invalid", "git commit --message \"$(cat <<'EOF'\nwip\nEOF\n)\"", true},
		{"assigned long message heredoc invalid", "git commit --message=\"$(cat <<'EOF'\nwip\nEOF\n)\"", true},
		{"compound commit", "git add . && git commit -m wip", true},
		{"git options", "git -C . commit -m wip", true},
		{"valid option commit with mention", `git -C . commit -m "feat: document git commit -m wip"`, false},
		{"config argument is not message", `git -c example.key="-m wip" commit -m "feat: change"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]any{"hook_event_name": "preToolUse", "tool_name": "Shell", "tool_input": map[string]string{"command": tc.command}})
			if err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			err = (Runner{WorkingDir: t.TempDir(), Stdout: &stdout, Stdin: bytes.NewReader(payload)}).Run([]string{"check", "--hook", "validate-commit", "--json", "--cursor-hook"})
			assertCursorCheckDecision(t, stdout.Bytes(), err, tc.deny)
		})
	}
}
