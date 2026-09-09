package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeCursorCheckHooksRequestJSONOutput(t *testing.T) {
	hooks, err := readNativeBuildHooks(filepath.Join(testRepositoryRoot(t), "config", "hooks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, hook := range hooks {
		command := nativeCursorHookCommand(hook)
		if !strings.HasPrefix(command, "loaf check --hook ") {
			continue
		}
		checked++
		if !strings.Contains(command, " --json") {
			t.Errorf("Cursor hook %s command = %q, want JSON output", hook.id, command)
		}
	}
	if checked == 0 {
		t.Fatal("no Cursor check hooks were exercised")
	}
}

func TestCursorExplicitCheckCommandsRequestJSONOnce(t *testing.T) {
	for _, command := range []string{
		"loaf check --hook check-secrets",
		"loaf check --hook validate-push --advisory",
		"loaf check --hook check-secrets --json",
	} {
		t.Run(command, func(t *testing.T) {
			got := nativeCursorHookCommand(nativeBuildHook{id: "explicit-check", command: command})
			if strings.Count(got, " --json") != 1 {
				t.Fatalf("command = %q, want exactly one JSON flag", got)
			}
		})
	}
	command := "loaf journal context --from-hook --cursor-hook"
	if got := nativeCursorHookCommand(nativeBuildHook{id: "session-start-loaf", command: command}); got != command {
		t.Fatalf("non-check command = %q, want %q", got, command)
	}
}

func TestCursorGeneratedCheckReturnsJSONWithoutWeakeningEnforcement(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "benign", true: "finding"}[blocked], func(t *testing.T) {
			content := "ordinary text"
			if blocked {
				content = "AKIA" + strings.Repeat("A", 16)
			}
			input, err := json.Marshal(map[string]any{"tool_name": "Write", "tool_input": map[string]string{"file_path": "example.txt", "content": content}})
			if err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			command := nativeCursorHookCommand(nativeBuildHook{id: "check-secrets", failClosed: true})
			err = (Runner{WorkingDir: t.TempDir(), Stdin: bytes.NewReader(input), Stdout: &stdout}).Run(strings.Fields(command)[1:])
			var exitErr interface{ ExitCode() int }
			if blocked {
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
					t.Fatalf("error = %v, want blocking exit 2", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var output checkJSONOutput
			if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
				t.Fatalf("invalid hook JSON %q: %v", stdout.String(), err)
			}
			if output.Blocked != blocked || output.Passed == blocked || output.Advisory {
				t.Fatalf("output = %#v, want blocked=%v without advisory", output, blocked)
			}
		})
	}
}

func TestCursorHookPairingPreservesIdentityAcrossJSONTransport(t *testing.T) {
	recognition := testHookRecognition(t, "cursor", testRepoHookCatalog(t, "cursor"))
	for _, command := range []string{"loaf check --hook check-secrets", "loaf check --hook check-secrets --json"} {
		outcome, err := pairHookEventEntries(recognition, "preToolUse", []map[string]any{{"command": command, "matcher": "Bash"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(outcome.paired) != 1 || outcome.paired[0].hookID != "check-secrets" || len(outcome.foreign) != 0 {
			t.Fatalf("command %q pairing = %#v, want the original managed identity", command, outcome)
		}
	}
}
