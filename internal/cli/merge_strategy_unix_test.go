//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkflowPrePRDoesNotBlockOnMergeStrategyFIFO(t *testing.T) {
	repo := initCLIGitRepo(t)
	mkdirAll(t, filepath.Join(repo, ".github"))
	writeFile(t, filepath.Join(repo, ".github", "PULL_REQUEST_TEMPLATE.md"), coverageDeclaration)
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog("- Baseline note."))
	coverageCommit(t, repo)
	addOriginTrackingMain(t, repo)
	gitCLI(t, repo, "commit", "--allow-empty", "-m", "chore: unpushed base")
	gitCLI(t, repo, "switch", "-c", "feat/fifo-config")
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog("- Preserve nonblocking config reads "+coverageLink+"."))
	coverageCommit(t, repo)
	body := filepath.Join(t.TempDir(), "pr.md")
	writeFile(t, body, coverageDeclaration)
	if err := os.MkdirAll(filepath.Join(repo, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, ".agents", "loaf.json")
	mkfifoForTest(t, path)
	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "gh"), []byte("#!/bin/sh\nprintf 'main\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	context := checkHookContext{ToolName: "Bash", ToolInput: checkHookInput{Command: `gh pr create --repo levifig/loaf --base main --title "fix: preserve nonblocking config reads" --body-file "` + body + `"`}}
	done := make(chan checkResult, 1)
	go func() { done <- runNativeWorkflowPrePR(context, repo) }()
	select {
	case result := <-done:
		want := baseBranchAbsorptionWarning("main", "1", "")
		if result.Blocked || !strings.Contains(strings.Join(result.Warnings, "\n"), want) {
			t.Fatalf("result = %#v, want neutral unpushed-base warning %q", result, want)
		}
	case <-time.After(5 * time.Second):
		// Release a regressed blocking reader before the fixture is removed.
		file, err := os.OpenFile(path, os.O_RDWR, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.WriteString(`{"git":{"merge_strategy":"merge"}}`); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		t.Fatal("workflow-pre-pr blocked on a FIFO at .agents/loaf.json")
	}
}
