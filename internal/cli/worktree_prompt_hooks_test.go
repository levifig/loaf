package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticJournalGuidanceInNestedLinkedWorktree(t *testing.T) {
	for _, scenario := range []string{"identical-checkout", "main-config-edited", "branch-config-diff", "divergent-task", "local-only-task"} {
		t.Run(scenario, func(t *testing.T) {
			main := initCLIGitRepo(t)
			mkdirAll(t, filepath.Join(main, ".agents", "tasks"))
			writeFile(t, filepath.Join(main, ".agents", "loaf.json"), "{\"version\":\"0.4.0\"}\n")
			writeFile(t, filepath.Join(main, ".agents", "tasks", "legacy.md"), "# Original task\n")
			gitCLI(t, main, "add", ".agents")
			gitCLI(t, main, "-c", "user.name=Loaf Test", "-c", "user.email=loaf@example.test", "-c", "commit.gpgsign=false", "commit", "-m", "tracked agent configuration")
			linked := filepath.Join(main, ".claude", "worktrees", "review")
			gitCLI(t, main, "worktree", "add", "-b", "review", linked)
			switch scenario {
			case "main-config-edited", "branch-config-diff":
				writeFile(t, filepath.Join(main, ".agents", "loaf.json"), "{\"version\":\"0.5.0\"}\n")
				if scenario == "branch-config-diff" {
					gitCLI(t, main, "add", ".agents/loaf.json")
					gitCLI(t, main, "-c", "user.name=Loaf Test", "-c", "user.email=loaf@example.test", "-c", "commit.gpgsign=false", "commit", "-m", "configuration on another branch")
				}
			case "divergent-task":
				writeFile(t, filepath.Join(linked, ".agents", "tasks", "legacy.md"), "# Changed task\n")
			case "local-only-task":
				writeFile(t, filepath.Join(linked, ".agents", "tasks", "local.md"), "# Worktree-only task\n")
			}
			if got := findMainWorktreeRootNative(linked); got != main {
				t.Fatalf("main root = %q, want %q", got, main)
			}
			before := map[string]string{}
			for _, root := range []string{main, linked} {
				for _, rel := range enumerateWorktreeAgentFiles(filepath.Join(root, ".agents")) {
					path := filepath.Join(root, ".agents", filepath.FromSlash(rel))
					raw, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					before[path] = string(raw)
				}
			}
			stateHome := t.TempDir()
			db := filepath.Join(stateHome, "loaf.sqlite")
			t.Setenv("LOAF_DB", db)
			for _, selector := range []string{"for-prompt", "for-compact"} {
				for _, payload := range []string{`{"session_id":"review"}`, `{"session_id":"review","agent_id":"subagent"}`} {
					var stdout, stderr bytes.Buffer
					err := (Runner{WorkingDir: linked, StateHome: stateHome, Stdin: strings.NewReader(payload), Stdout: &stdout, Stderr: &stderr}).Run([]string{"journal", "context", selector})
					if err != nil {
						t.Fatalf("%s (%s): %v; stderr: %s", selector, payload, err, stderr.String())
					}
					if stderr.Len() != 0 {
						t.Fatalf("%s stderr = %q", selector, stderr.String())
					}
					if strings.Contains(payload, "agent_id") {
						if stdout.Len() != 0 {
							t.Fatalf("subagent guidance = %q, want empty", stdout.String())
						}
					} else if !strings.Contains(stdout.String(), "loaf journal log") {
						t.Fatalf("%s stdout = %q, want guidance", selector, stdout.String())
					}
				}
			}
			for path, want := range before {
				raw, err := os.ReadFile(path)
				if err != nil || string(raw) != want {
					t.Fatalf("guidance changed %s: %q, %v", path, raw, err)
				}
			}
			for _, path := range []string{db, filepath.Join(linked, ".agents", worktreeBackPointerFile)} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("guidance created %s: %v", path, err)
				}
			}
			var initOut bytes.Buffer
			if err := (Runner{WorkingDir: linked, StateHome: stateHome, Stdout: &initOut}).Run([]string{"state", "init"}); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{
				{"journal", "recent"}, {"journal", "log", "decision(test): use canonical SQLite"},
				{"journal", "context", "--from-hook"}, {"journal", "context", "for-resumption"},
			} {
				var stdout, stderr bytes.Buffer
				err := (Runner{WorkingDir: linked, StateHome: stateHome, Stdin: strings.NewReader(`{}`), Stdout: &stdout, Stderr: &stderr}).Run(args)
				if err != nil {
					t.Fatalf("%v: %v; stderr: %s", args, err, stderr.String())
				}
				if strings.Contains(stderr.String(), "loaf migrate worktree-storage") {
					t.Fatalf("SQLite command refused: %s", stderr.String())
				}
			}
			var mainOut bytes.Buffer
			if err := (Runner{WorkingDir: main, StateHome: stateHome, Stdout: &mainOut}).Run([]string{"journal", "recent"}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(mainOut.String(), "use canonical SQLite") {
				t.Fatalf("main journal does not share linked identity: %s", mainOut.String())
			}

			for path, want := range before {
				raw, err := os.ReadFile(path)
				if err != nil || string(raw) != want {
					t.Fatalf("SQLite operation changed %s: %q, %v", path, raw, err)
				}
			}
			if _, err := os.Lstat(filepath.Join(linked, ".agents", worktreeBackPointerFile)); !os.IsNotExist(err) {
				t.Fatalf("SQLite command created marker: %v", err)
			}

		})
	}
}

func TestWorktreeRefusalRecognizesCurrentCommands(t *testing.T) {
	for _, command := range cliReferenceCommands() {
		if got := unknownTopLevelCommandNative([]string{command.Name}); got != "" {
			t.Errorf("documented command %q reported unknown", got)
		}
	}
	var help bytes.Buffer
	writeRootHelp(&help)
	for command := range parseHelpSectionNames(t, help.String(), "Commands:") {
		if got := unknownTopLevelCommandNative([]string{command}); got != "" {
			t.Errorf("root command %q reported unknown", got)
		}
	}
	for _, command := range []string{"session", "spec", "change", "not-a-command"} {
		if got := unknownTopLevelCommandNative([]string{command}); got != command {
			t.Errorf("removed or unknown command %q reported known", command)
		}
	}
}
