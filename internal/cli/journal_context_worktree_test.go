package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// Project identity normalizes a linked worktree to the main checkout, but the
// journal context must scope its branch layers to the checkout it describes.
// These fixtures keep main and a linked "feature" worktree of one project, each
// with its own branch-tagged entry.

type journalContextWorktreeFixture struct {
	primary, worktree, stateHome string
}

func setupJournalContextWorktree(t *testing.T) journalContextWorktreeFixture {
	t.Helper()
	fixture := journalContextWorktreeFixture{primary: initCLIGitRepo(t), stateHome: t.TempDir()}
	if err := (Runner{Stdout: &bytes.Buffer{}, WorkingDir: fixture.primary, StateHome: fixture.stateHome}).Run([]string{"state", "init", "--json"}); err != nil {
		t.Fatal(err)
	}
	fixture.worktree = linkedWorktree(t, fixture.primary, "feature")
	for dir, entry := range map[string]string{
		fixture.primary:  "discover(context): main-only marker",
		fixture.worktree: "discover(context): feature-only marker",
	} {
		if err := (Runner{Stdout: &bytes.Buffer{}, WorkingDir: dir, StateHome: fixture.stateHome}).Run([]string{"journal", "log", entry}); err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}

func assertJournalContextScopedToFeature(t *testing.T, digest string) {
	t.Helper()
	if !strings.Contains(digest, "  branch: feature\n") || !strings.Contains(digest, "feature-only marker") {
		t.Fatalf("digest is not scoped to the feature worktree:\n%s", digest)
	}
	if strings.Contains(digest, "branch: main") || strings.Contains(digest, "main-only marker") {
		t.Fatalf("digest carries the main checkout's branch layers:\n%s", digest)
	}
}

func runJournalContextFixture(t *testing.T, workingDir, stateHome, input string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	runner := Runner{Stdout: &out, WorkingDir: workingDir, StateHome: stateHome}
	if input != "" {
		runner.Stdin = strings.NewReader(input)
	}
	if err := runner.Run(append([]string{"journal", "context"}, args...)); err != nil {
		t.Fatalf("journal context %v error = %v\n%s", args, err, out.String())
	}
	return out.String()
}

func TestJournalContextScopesToInvokingLinkedWorktree(t *testing.T) {
	fixture := setupJournalContextWorktree(t)
	subdir := filepath.Join(fixture.worktree, "docs")
	mkdirAll(t, subdir)

	t.Run("worktree root", func(t *testing.T) {
		assertJournalContextScopedToFeature(t, runJournalContextFixture(t, fixture.worktree, fixture.stateHome, ""))
	})
	t.Run("worktree subdirectory", func(t *testing.T) {
		assertJournalContextScopedToFeature(t, runJournalContextFixture(t, subdir, fixture.stateHome, ""))
	})
	t.Run("json", func(t *testing.T) {
		var result journalContextCLIResult
		output := runJournalContextFixture(t, fixture.worktree, fixture.stateHome, "", "--json")
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("output = %q: %v", output, err)
		}
		if result.Branch != "feature" {
			t.Fatalf("branch = %q, want feature", result.Branch)
		}
		if result.ProjectCurrentPath != fixture.primary {
			t.Fatalf("project path = %q, want the main checkout %q", result.ProjectCurrentPath, fixture.primary)
		}
	})
	t.Run("main checkout keeps main", func(t *testing.T) {
		digest := runJournalContextFixture(t, fixture.primary, fixture.stateHome, "")
		if !strings.Contains(digest, "  branch: main\n") || strings.Contains(digest, "feature-only marker") {
			t.Fatalf("main checkout digest:\n%s", digest)
		}
	})
	t.Run("explicit branch wins", func(t *testing.T) {
		digest := runJournalContextFixture(t, fixture.worktree, fixture.stateHome, "", "--branch", "main")
		if !strings.Contains(digest, "  branch: main\n") || !strings.Contains(digest, "main-only marker") {
			t.Fatalf("explicit --branch main digest:\n%s", digest)
		}
	})
}

func TestJournalContextHooksScopeToInvokingLinkedWorktree(t *testing.T) {
	fixture := setupJournalContextWorktree(t)
	claude := func(cwd string) string {
		return `{"hook_event_name":"SessionStart","source":"startup","session_id":"s1","cwd":` + jsonString(cwd) + `}`
	}
	codex := func(cwd string) string {
		return `{"cwd":` + jsonString(cwd) + `,"hook_event_name":"SessionStart","model":"gpt-5.6","permission_mode":"default","session_id":"codex-session","source":"startup","transcript_path":null}`
	}
	generic := func(cwd string) string {
		return `{"session_id":"s1","cwd":` + jsonString(cwd) + `}`
	}
	digest := func(t *testing.T, args []string, output string) string {
		t.Helper()
		switch args[len(args)-1] {
		case "--claude-code":
			var payload claudeSessionStartOutput
			if err := json.Unmarshal([]byte(output), &payload); err != nil || payload.HookSpecificOutput == nil {
				t.Fatalf("output = %q, want Claude Code additionalContext (%v)", output, err)
			}
			return payload.HookSpecificOutput.AdditionalContext + "\n"
		case "--codex-hook":
			var payload codexSessionStartOutput
			if err := json.Unmarshal([]byte(output), &payload); err != nil || payload.HookSpecificOutput == nil {
				t.Fatalf("output = %q, want Codex additionalContext (%v)", output, err)
			}
			return payload.HookSpecificOutput.AdditionalContext + "\n"
		case "--cursor-hook":
			var payload cursorSessionStartOutput
			if err := json.Unmarshal([]byte(output), &payload); err != nil {
				t.Fatalf("output = %q, want Cursor additional_context (%v)", output, err)
			}
			return payload.AdditionalContext + "\n"
		}
		return output
	}

	otherProject := initCLIGitRepo(t)
	subdir := filepath.Join(fixture.worktree, "docs")
	mkdirAll(t, subdir)
	for _, tc := range []struct {
		name, processDir, input string
		args                    []string
	}{
		{name: "claude code in worktree", processDir: fixture.worktree, input: claude(fixture.worktree), args: []string{"--from-hook", "--claude-code"}},
		{name: "codex in worktree", processDir: fixture.worktree, input: codex(fixture.worktree), args: []string{"--from-hook", "--codex-hook"}},
		{name: "cursor in worktree", processDir: fixture.worktree, input: `{"hook_event_name":"sessionStart","session_id":"s1","composer_mode":"agent","is_background_agent":false,"cursor_version":"2026.05.09-0afadcc","workspace_roots":[]}`, args: []string{"--from-hook", "--cursor-hook"}},
		{name: "opencode in worktree", processDir: fixture.worktree, input: openCodeSessionStartPayload, args: []string{"--from-hook", "--opencode-hook"}},
		{name: "generic hook in worktree", processDir: fixture.worktree, input: generic(fixture.worktree), args: []string{"--from-hook"}},
		{name: "post-compaction resumption in worktree", processDir: fixture.worktree, input: generic(fixture.worktree), args: []string{"for-resumption"}},
		// The harness may spawn the hook from the main checkout while the
		// session works in the linked worktree its payload names.
		{name: "claude code payload names worktree", processDir: fixture.primary, input: claude(fixture.worktree), args: []string{"--from-hook", "--claude-code"}},
		{name: "codex payload names worktree", processDir: fixture.primary, input: codex(fixture.worktree), args: []string{"--from-hook", "--codex-hook"}},
		{name: "generic payload names worktree subdirectory", processDir: fixture.primary, input: generic(subdir), args: []string{"--from-hook"}},
		// A payload directory that cannot describe this project falls back to
		// the hook process's own worktree instead of failing the hook.
		{name: "payload cwd missing on disk", processDir: fixture.worktree, input: codex("/workspace/project"), args: []string{"--from-hook", "--codex-hook"}},
		{name: "payload cwd relative", processDir: fixture.worktree, input: claude("."), args: []string{"--from-hook", "--claude-code"}},
		{name: "payload cwd in another project", processDir: fixture.worktree, input: claude(otherProject), args: []string{"--from-hook", "--claude-code"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := runJournalContextFixture(t, tc.processDir, fixture.stateHome, tc.input, tc.args...)
			assertJournalContextScopedToFeature(t, digest(t, tc.args, output))
		})
	}
}

func jsonString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
