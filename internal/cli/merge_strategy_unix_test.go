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
	addOriginTrackingMain(t, repo)
	gitCLI(t, repo, "commit", "--allow-empty", "-m", "chore: unpushed base")
	gitCLI(t, repo, "switch", "-c", "feat/fifo-config")
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), workflowChangelog("- FIFO fixture."))
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
	context := checkHookContext{ToolName: "Bash", ToolInput: checkHookInput{Command: "gh pr create --fill"}}
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
