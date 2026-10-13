package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestObservedHarnessFromEnvironment(t *testing.T) {
	clearAdditionalHarnessMarkers(t)
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CODEX_VERSION", "")
	t.Setenv("CLAUDECODE", "")
	t.Setenv("CLAUDE_CODE_VERSION", "")
	if harness, version := observedHarnessFromEnvironment(); harness != "unknown" || version != "unknown" {
		t.Fatalf("no marker = %q %q", harness, version)
	}
	t.Setenv("CODEX_THREAD_ID", "thread-1")
	t.Setenv("CODEX_VERSION", "1.2.3")
	if harness, version := observedHarnessFromEnvironment(); harness != "codex" || version != "1.2.3" {
		t.Fatalf("Codex marker = %q %q", harness, version)
	}
	t.Setenv("CODEX_VERSION", "secret payload with spaces")
	if harness, version := observedHarnessFromEnvironment(); harness != "codex" || version != "unknown" {
		t.Fatalf("untrusted version marker = %q %q", harness, version)
	}
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CLAUDECODE", "1")
	if harness, version := observedHarnessFromEnvironment(); harness != "claude-code" || version != "unknown" {
		t.Fatalf("Claude marker = %q %q", harness, version)
	}
}

func clearAdditionalHarnessMarkers(t *testing.T) {
	t.Helper()
	for _, key := range []string{"AGENT", "AMP_VERSION", "OPENCODE", "OPENCODE_VERSION", "LOAF_HOOK_HARNESS", "LOAF_HOOK_TYPE"} {
		t.Setenv(key, "")
	}
}

func committedOriginFixture(t *testing.T, slug, date string) (string, string, []byte) {
	t.Helper()
	repo := t.TempDir()
	if err := originGitCLI(repo, "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	if err := originGitCLI(repo, "config", "user.name", "Loaf Test"); err != nil {
		t.Fatal(err)
	}
	if err := originGitCLI(repo, "config", "user.email", "loaf@example.test"); err != nil {
		t.Fatal(err)
	}
	changeFile := filepath.Join(repo, "docs", "changes", date+"-"+slug, "change.md")
	content := []byte("---\nslug: " + slug + "\n---\ncommitted bytes\n")
	if err := os.MkdirAll(filepath.Dir(changeFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changeFile, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := originGitCLI(repo, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := originGitCLI(repo, "-c", "commit.gpgsign=false", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	return repo, changeFile, content
}

func originGitCLI(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(output)))
	}
	return nil
}

func mustOriginGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(output)
}
