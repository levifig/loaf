package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCursorInstructionNativeProjection(t *testing.T) {
	projections, err := nativeCursorHookProjections(filepath.Join(testRepositoryRoot(t), "config", "hooks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, projection := range projections {
		if projection.hook.instruction == "" {
			continue
		}
		found[projection.hookID] = true
		wantEvent := "beforeShellExecution"
		wantCommand := "git push origin test"
		if projection.hookID != "workflow-pre-push" {
			wantCommand = "gh pr merge 123 --squash"
		}
		if projection.hookID == "workflow-post-merge" {
			wantEvent = "postToolUse"
		}
		if projection.event != wantEvent || projection.entry.If != "" || !strings.HasPrefix(projection.entry.Command, "loaf check --hook "+projection.hookID+" ") || !strings.Contains(projection.entry.Command, "--cursor-hook") {
			t.Errorf("instruction projection = %#v", projection)
		}
		if wantEvent == "beforeShellExecution" {
			matcher, err := regexp.Compile(projection.entry.Matcher)
			if err != nil || !matcher.MatchString(wantCommand) || matcher.MatchString("pwd") {
				t.Errorf("native shell matcher = %q: %v", projection.entry.Matcher, err)
			}
			if err == nil {
				commands := []string{"gh pr merge; echo done", "gh pr merge&&echo done", "/opt/homebrew/bin/gh pr merge", `"gh" pr merge`}
				if projection.hookID == "workflow-pre-push" {
					commands = []string{"git push; git status", "git push&&echo done", "/usr/bin/git push", `"git" push`}
				}
				for _, command := range commands {
					if !matcher.MatchString(command) || !cursorInstructionCommandMatches(command, cursorInstructionSpecs[projection.hookID]) {
						t.Errorf("native projection misses executable command %q: %q", command, projection.entry.Matcher)
					}
				}
			}
		} else if projection.entry.Matcher != "Shell" {
			t.Errorf("post-tool matcher = %q", projection.entry.Matcher)
		}
	}
	for _, id := range []string{"workflow-pre-merge", "workflow-pre-push", "workflow-post-merge"} {
		if !found[id] {
			t.Errorf("missing %s", id)
		}
	}
}

func TestCursorInstructionNativeResponses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	mkdirAll(t, filepath.Join(home, ".cursor", "hooks", "instructions"))
	for _, name := range []string{"pre-merge", "pre-push", "post-merge"} {
		writeFile(t, filepath.Join(home, ".cursor", "hooks", "instructions", name+".md"), "guidance "+name+"\n")
	}
	for _, tc := range []struct{ name, id, payload, want, channel string }{
		{"merge", "workflow-pre-merge", `{"command":"gh pr merge 123 --squash"}`, "guidance pre-merge", "agent_message"},
		{"push", "workflow-pre-push", `{"command":"git push origin test"}`, "guidance pre-push", "agent_message"},
		{"push delimiter", "workflow-pre-push", `{"command":"git push; git status"}`, "guidance pre-push", "agent_message"},
		{"push and", "workflow-pre-push", `{"command":"git push&&echo done"}`, "guidance pre-push", "agent_message"},
		{"push path", "workflow-pre-push", `{"command":"/usr/bin/git push"}`, "guidance pre-push", "agent_message"},
		{"push quoted executable", "workflow-pre-push", `{"command":"\"git\" push"}`, "guidance pre-push", "agent_message"},
		{"merge delimiter", "workflow-pre-merge", `{"command":"gh pr merge; echo done"}`, "guidance pre-merge", "agent_message"},
		{"merge and", "workflow-pre-merge", `{"command":"gh pr merge&&echo done"}`, "guidance pre-merge", "agent_message"},
		{"merge path", "workflow-pre-merge", `{"command":"/opt/homebrew/bin/gh pr merge"}`, "guidance pre-merge", "agent_message"},
		{"merge quoted executable", "workflow-pre-merge", `{"command":"\"gh\" pr merge"}`, "guidance pre-merge", "agent_message"},
		{"benign", "workflow-pre-push", `{"command":"pwd"}`, "", "agent_message"},
		{"quoted mention", "workflow-pre-push", `{"command":"echo 'git push'"}`, "", "agent_message"},
		{"command lookup", "workflow-pre-push", `{"command":"command -v git push"}`, "", "agent_message"},
		{"comment mention", "workflow-pre-push", `{"command":"echo ok # ; git push"}`, "", "agent_message"},
		{"verbose lookup", "workflow-pre-push", `{"command":"command -V git push"}`, "", "agent_message"},
		{"combined lookup flag", "workflow-pre-push", `{"command":"command -pv git push"}`, "", "agent_message"},
		{"wrapped lookup", "workflow-pre-push", `{"command":"env EXAMPLE=1 command -v git push"}`, "", "agent_message"},
		{"real command builtin", "workflow-pre-push", `{"command":"command git push"}`, "guidance pre-push", "agent_message"},
		{"command default path", "workflow-pre-push", `{"command":"command -p git push"}`, "guidance pre-push", "agent_message"},
		{"env wrapped execution", "workflow-pre-push", `{"command":"env EXAMPLE=1 command -p git push"}`, "guidance pre-push", "agent_message"},
		{"comment newline execution", "workflow-pre-push", `{"command":"echo ok # ; git status\ngit push"}`, "guidance pre-push", "agent_message"},
		{"quoted hash", "workflow-pre-push", `{"command":"echo 'hash # text'; git push"}`, "guidance pre-push", "agent_message"},
		{"embedded hash", "workflow-pre-push", `{"command":"echo foo#bar; git push"}`, "guidance pre-push", "agent_message"},
		{"escaped hash", "workflow-pre-push", `{"command":"echo \\#; git push"}`, "guidance pre-push", "agent_message"},
		{"compound", "workflow-pre-push", `{"command":"git status && git push origin test"}`, "guidance pre-push", "agent_message"},
		{"post merge", "workflow-post-merge", `{"tool_name":"Shell","tool_input":{"command":"gh pr merge 123"}}`, "guidance post-merge", "additional_context"},
		{"post benign", "workflow-post-merge", `{"tool_name":"Shell","tool_input":{"command":"pwd"}}`, "", "additional_context"},
		{"post wrong tool", "workflow-post-merge", `{"tool_name":"Read","tool_input":{"command":"gh pr merge 123"}}`, "", "additional_context"},
		{"malformed", "workflow-pre-push", `{`, "unavailable", "agent_message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := (Runner{WorkingDir: t.TempDir(), Stdin: strings.NewReader(tc.payload), Stdout: &out}).Run([]string{"check", "--hook", tc.id, "--cursor-hook"})
			if err != nil {
				t.Errorf("advisory returned %v", err)
			}
			var result map[string]any
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatalf("invalid native JSON %q: %v", out.String(), err)
			}
			if tc.channel == "agent_message" && result["permission"] != "allow" {
				t.Errorf("permission = %#v", result)
			}
			message, _ := result[tc.channel].(string)
			if tc.want == "" && message != "" || tc.want != "" && !strings.Contains(message, tc.want) {
				t.Errorf("output %s, want %q in %s", out.String(), tc.want, tc.channel)
			}
			if tc.channel == "additional_context" && result["agent_message"] != nil {
				t.Errorf("post result uses permission channel: %s", out.String())
			}
		})
	}
	if err := os.Remove(filepath.Join(home, ".cursor", "hooks", "instructions", "pre-push.md")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := (Runner{WorkingDir: t.TempDir(), Stdin: strings.NewReader(`{"command":"git push"}`), Stdout: &out}).Run([]string{"check", "--hook", "workflow-pre-push", "--cursor-hook"})
	var result map[string]any
	if json.Unmarshal(out.Bytes(), &result) != nil || err != nil || result["permission"] != "allow" || !strings.Contains(out.String(), "unavailable") {
		t.Fatalf("missing instruction: %s; %v", out.String(), err)
	}
}

func TestCursorInstructionMigrationPreservesIntent(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "historical present", true: "recorded disabled"}[disabled], func(t *testing.T) {
			fixture := newCursorHookFixture(t)
			fixture.writeHooks(t, string(testHookFixture(t, "cursor-hooks-live.json")))
			fixture.writeInstalledManifest(t, testHookPriorVersion)
			value := fixture.decodeHooks(t)
			value["hooks"].(map[string]any)["beforeShellExecution"] = []any{map[string]any{"command": "foreign-command", "matcher": "curl", "custom": true}}
			fixture.writeHooksValue(t, value)
			if disabled {
				if _, err := fixture.store.SetHookEnablement(t.Context(), "cursor", "preToolUse", "workflow-pre-push", false); err != nil {
					t.Fatal(err)
				}
			}
			before := testHookDocumentSnapshot(t, []byte(fixture.readHooks(t)), fixture.recognition(t))
			fixture.apply(t)
			after := testHookDocumentSnapshot(t, []byte(fixture.readHooks(t)), fixture.recognition(t))
			if !bytes.Equal(mustCursorInstructionJSON(t, before), mustCursorInstructionJSON(t, after)) {
				t.Fatal("foreign hooks or document shape changed")
			}
			ids := fixture.hookIDsByEvent(t)
			for _, id := range ids["preToolUse"] {
				if id == "workflow-pre-merge" || id == "workflow-pre-push" {
					t.Fatalf("historical instruction left active in old event: %s", id)
				}
			}
			for _, id := range []string{"workflow-pre-merge", "workflow-pre-push"} {
				present := false
				for _, actual := range ids["beforeShellExecution"] {
					present = present || actual == id
				}
				if present != !(disabled && id == "workflow-pre-push") {
					t.Fatalf("instruction %s present=%v disabled=%v", id, present, disabled)
				}
			}
			if actions := fixture.apply(t); len(actions) != 0 {
				t.Fatalf("migration not idempotent: %#v", actions)
			}
		})
	}
}

func mustCursorInstructionJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestCursorInstructionGeneratedCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	mkdirAll(t, filepath.Join(home, ".cursor", "hooks", "instructions"))
	projections, err := nativeCursorHookProjections(filepath.Join(testRepositoryRoot(t), "config", "hooks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, projection := range projections {
		if projection.hook.instruction == "" {
			continue
		}
		file := filepath.Base(projection.hook.instruction)
		writeFile(t, filepath.Join(home, ".cursor", "hooks", "instructions", file), "native guidance "+file+"\nquoted \"text\"\n")
		payload := `{"command":"gh pr merge 123"}`
		channel := "agent_message"
		if projection.hookID == "workflow-pre-push" {
			payload = `{"command":"git push origin test"}`
		}
		if projection.event == "postToolUse" {
			payload = `{"tool_name":"Shell","tool_input":{"command":"gh pr merge 123"}}`
			channel = "additional_context"
		}
		var out bytes.Buffer
		words := strings.Fields(projection.entry.Command)
		err := (Runner{WorkingDir: t.TempDir(), Stdin: strings.NewReader(payload), Stdout: &out}).Run(words[1:])
		var result map[string]any
		if json.Unmarshal(out.Bytes(), &result) != nil || err != nil {
			t.Fatalf("generated %q: %s; %v", projection.entry.Command, out.String(), err)
		}
		message, _ := result[channel].(string)
		if !strings.Contains(message, "native guidance "+file) {
			t.Fatalf("generated %q has no native context: %s", projection.entry.Command, out.String())
		}
	}
}
