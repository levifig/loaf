package cli

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/auth"
	"github.com/levifig/loaf/internal/state"
)

func TestWorktreeStorageGateUsesOperationAndStoragePaths(t *testing.T) {
	t.Setenv("LOAF_DB", filepath.Join(t.TempDir(), "loaf.sqlite"))
	main := initCLIGitRepo(t)
	mkdirAll(t, filepath.Join(main, ".agents", "tasks"))
	writeFile(t, filepath.Join(main, ".agents", "loaf.json"), "{}\n")
	writeFile(t, filepath.Join(main, ".agents", "tasks", "legacy.md"), "# Original\n")
	gitCLI(t, main, "add", ".agents")
	gitCLI(t, main, "-c", "user.name=Loaf Test", "-c", "user.email=loaf@example.test", "-c", "commit.gpgsign=false", "commit", "-m", "agent configuration")
	linked := filepath.Join(main, ".claude", "worktrees", "storage")
	gitCLI(t, main, "worktree", "add", "-b", "storage", linked)
	nested := filepath.Join(linked, "src", "nested")
	mkdirAll(t, nested)
	writeFile(t, filepath.Join(main, ".agents", "loaf.json"), "{\"changed\":true}\n")
	mkdirAll(t, filepath.Join(linked, ".agents", "research"))
	writeFile(t, filepath.Join(linked, ".agents", "research", "note.md"), "ordinary artifact\n")
	for _, args := range [][]string{{"report", "list"}, {"task", "list"}, {"migrate", "markdown"}} {
		if shouldRefuseCommandNative(args, nested) {
			t.Errorf("ordinary configuration/artifact difference blocked %v", args)
		}
	}
	writeFile(t, filepath.Join(linked, ".agents", "tasks", "legacy.md"), "# Changed\n")
	for _, args := range [][]string{{"journal", "recent"}, {"journal", "context", "--from-hook"}, {"journal", "log", "decision(test): isolated"}, {"plan", "list"}, {"handoff", "list"}, {"doctor"}, {"report", "list"}, {"not-a-command"}, {"migrate", "schema"}} {
		if shouldRefuseCommandNative(args, nested) {
			t.Errorf("unrelated task storage blocked %v", args)
		}
	}
	for _, args := range [][]string{{"task", "list"}, {"task", "refresh"}, {"migrate", "markdown"}, {"state", "migrate", "markdown"}} {
		if !shouldRefuseCommandNative(args, nested) {
			t.Errorf("divergent legacy task did not block %v", args)
		}
	}
	for _, root := range []string{main, linked} {
		if _, err := os.Lstat(filepath.Join(root, ".agents", worktreeBackPointerFile)); !os.IsNotExist(err) {
			t.Errorf("classification created pointer: %v", err)
		}
	}
}

func TestWorktreeStorageRefusalNamesConflict(t *testing.T) {
	main := initCLIGitRepo(t)
	linked := addCLILinkedWorktree(t, main, "storage-conflict")
	mkdirAll(t, filepath.Join(linked, ".agents", "reports"))
	path := filepath.Join(linked, ".agents", "reports", "local.md")
	writeFile(t, path, "# Local report\n")
	stateHome := t.TempDir()
	db := filepath.Join(stateHome, "loaf.sqlite")
	t.Setenv("LOAF_DB", db)
	var stderr bytes.Buffer
	err := (Runner{WorkingDir: linked, StateHome: stateHome, Stderr: &stderr}).Run([]string{"report", "list"})
	if exitErr, ok := err.(ExitError); !ok || exitErr.Code != 2 {
		t.Fatalf("error = %v, want refusal", err)
	}
	for _, want := range []string{path, "local-only", "loaf migrate worktree-storage", "preview"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want %q", stderr.String(), want)
		}
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "# Local report\n" {
		t.Fatalf("source changed: %q %v", got, err)
	}
	for _, path := range []string{db, filepath.Join(linked, ".agents", worktreeBackPointerFile)} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("refusal created %s: %v", path, err)
		}
	}
}

func TestWorktreeStorageConflictLayouts(t *testing.T) {
	for _, scenario := range []string{"identical", "local-only", "divergent", "valid-pointer", "stale-pointer", "empty-pointer", "marker-symlink", "local-root-symlink", "local-file-symlink", "main-root-symlink", "main-file-symlink", "canonical-only-symlink", "partial", "nested-partial", "canonical-only-partial", "ordinary-symlink"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			main := filepath.Join(root, "main")
			local := filepath.Join(root, "linked")
			for _, dir := range []string{main, local} {
				mkdirAll(t, filepath.Join(dir, "reports"))
			}
			path := filepath.Join(local, "reports", "report.md")
			writeFile(t, path, "# Report\n")
			writeFile(t, filepath.Join(main, "reports", "report.md"), "# Report\n")
			symlink := func(target, link string) {
				t.Helper()
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
			}
			switch scenario {
			case "local-only":
				writeFile(t, filepath.Join(local, "reports", "local.md"), "local\n")
			case "divergent":
				writeFile(t, path, "divergent\n")
			case "valid-pointer":
				writeFile(t, filepath.Join(local, worktreeBackPointerFile), root+"\n")
			case "stale-pointer":
				writeFile(t, filepath.Join(local, worktreeBackPointerFile), root+"/missing\n")
			case "empty-pointer":
				writeFile(t, filepath.Join(local, worktreeBackPointerFile), "\n")
			case "marker-symlink":
				symlink(path, filepath.Join(local, worktreeBackPointerFile))
			case "local-root-symlink":
				renamed := local + "-real"
				if err := os.Rename(local, renamed); err != nil {
					t.Fatal(err)
				}
				symlink(renamed, local)
			case "local-file-symlink":
				symlink(path, filepath.Join(local, "reports", "link.md"))
			case "main-root-symlink":
				renamed := main + "-real"
				if err := os.Rename(main, renamed); err != nil {
					t.Fatal(err)
				}
				symlink(renamed, main)
			case "main-file-symlink", "canonical-only-symlink":
				symlink(path, filepath.Join(main, "reports", "link.md"))
			case "partial":
				writeFile(t, filepath.Join(main, "reports"+worktreePartialSuffix), "staging\n")
			case "nested-partial", "canonical-only-partial":
				writeFile(t, filepath.Join(main, "reports", "report.md"+worktreePartialSuffix), "staging\n")
			case "ordinary-symlink":
				symlink(path, filepath.Join(local, "instructions.md"))
			}
			if strings.HasPrefix(scenario, "canonical-only-") {
				if err := os.RemoveAll(filepath.Join(local, "reports")); err != nil {
					t.Fatal(err)
				}
			}
			before := snapshotStorageTree(t, root)
			conflicts := worktreeStorageConflicts(local, main, root, []string{"reports"})
			wantConflict := scenario != "identical" && scenario != "valid-pointer" && scenario != "ordinary-symlink"
			if (len(conflicts) > 0) != wantConflict {
				t.Fatalf("conflicts = %v, want conflict %v", conflicts, wantConflict)
			}
			after := snapshotStorageTree(t, root)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("classification changed storage: before=%v after=%v", before, after)
			}
		})
	}
}

func snapshotStorageTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			result[path] = "directory"
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			result[path] = "symlink:" + target
			return err
		}
		raw, err := os.ReadFile(path)
		result[path] = string(raw)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestWorktreeStorageMissingMainPreservesCanonicalIdentity(t *testing.T) {
	for _, relative := range []bool{false, true} {
		t.Run(fmt.Sprint("relative-pointer-", relative), func(t *testing.T) {
			main := initCLIGitRepo(t)
			linked := addCLILinkedWorktree(t, main, "missing-main")
			if relative {
				raw, err := os.ReadFile(filepath.Join(linked, ".git"))
				if err != nil {
					t.Fatal(err)
				}
				target := strings.TrimSpace(strings.TrimPrefix(string(raw), "gitdir:"))
				rel, err := filepath.Rel(linked, target)
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(linked, ".git"), "gitdir: "+rel+"\n")
			}
			moved := main + "-offline"
			if err := os.Rename(main, moved); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Rename(moved, main); err != nil {
					t.Error(err)
				}
			})
			stateHome := t.TempDir()
			db := filepath.Join(stateHome, "loaf.sqlite")
			t.Setenv("LOAF_DB", db)
			for _, args := range [][]string{{"journal", "log", "decision(test): no fork"}, {"state", "path"}, {"auth", "attach"}, {"report", "list"}, {"migrate", "worktree-storage"}} {
				var stderr bytes.Buffer
				err := (Runner{WorkingDir: linked, StateHome: stateHome, Stderr: &stderr}).Run(args)
				if exitErr, ok := err.(ExitError); !ok || exitErr.Code != 2 || !strings.Contains(stderr.String(), main) {
					t.Fatalf("%v: %v; stderr: %s", args, err, stderr.String())
				}
			}
			for _, args := range [][]string{{"journal", "context", "for-prompt"}, {"journal", "context", "for-compact"}, {"version"}} {
				var stdout bytes.Buffer
				if err := (Runner{WorkingDir: linked, StateHome: stateHome, Stdout: &stdout}).Run(args); err != nil {
					t.Fatalf("%v: %v", args, err)
				}
			}
			if _, err := os.Lstat(db); !os.IsNotExist(err) {
				t.Fatalf("missing main created DB: %v", err)
			}
		})
	}
}

func TestWorktreeStorageBypassPreservesAttachGate(t *testing.T) {
	main := initCLIGitRepo(t)
	linked := addCLILinkedWorktree(t, main, "attach-gate")
	seedCLIWorktreeAgents(t, linked)
	stateHome := testAuthStateHome(t)
	db := filepath.Join(stateHome, "loaf.sqlite")
	t.Setenv("LOAF_DB", db)
	authDir, err := (state.PathResolver{StateHome: stateHome}).AuthDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Setup(context.Background(), auth.NewStore(authDir), auth.SetupInput{Endpoint: "http://127.0.0.1:8080", ServerDB: filepath.Join(t.TempDir(), "sync.sqlite")}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	err = (Runner{WorkingDir: linked, StateHome: stateHome, Stderr: &stderr}).Run([]string{"journal", "recent"})
	if exitErr, ok := err.(ExitError); !ok || exitErr.Code != 1 || !strings.Contains(stderr.String(), "not attached") {
		t.Fatalf("error = %v; stderr = %s", err, stderr.String())
	}
	if _, err := os.Lstat(db); !os.IsNotExist(err) {
		t.Fatalf("attach refusal created DB: %v", err)
	}
}

func TestWorktreeStorageOperationConsumers(t *testing.T) {
	for _, sqlite := range []bool{false, true} {
		t.Run(fmt.Sprint("sqlite-", sqlite), func(t *testing.T) {
			main := initCLIGitRepo(t)
			linked := addCLILinkedWorktree(t, main, "consumers")
			stateHome := t.TempDir()
			t.Setenv("LOAF_DB", filepath.Join(stateHome, "loaf.sqlite"))
			runner := Runner{WorkingDir: linked, StateHome: stateHome}
			if sqlite {
				var out bytes.Buffer
				runner.Stdout = &out
				if err := runner.Run([]string{"state", "init"}); err != nil {
					t.Fatal(err)
				}
			}
			for _, testcase := range []struct {
				path   string
				args   []string
				refuse bool
			}{
				{"reports/local.md", []string{"report", "list"}, true},
				{"reports/local.md", []string{"report", "generate", "--json", "release-readiness"}, true},
				{"reports/local.md", []string{"report", "generate", "--format", "markdown", "release-readiness"}, true},
				{"reports/local.md", []string{"report", "generate", "triage"}, false},
				{"reports/local.md", []string{"state", "export", "release-readiness", "--format", "markdown"}, true},
				{"reports/local.md", []string{"search", "query"}, true},
				{"reports/local.md", []string{"search", "--all-projects", "query"}, false},
				{"reports/local.md", []string{"housekeeping"}, true},
				{"drafts/local.md", []string{"housekeeping"}, true},
				{"drafts/local.md", []string{"trace", "local"}, true},
				{"councils/local.md", []string{"council", "list"}, true},
				{"councils/local.md", []string{"council", "link", "local", "another"}, false},
				{"specs/local.md", []string{"render", "sweep"}, true},
				{"tasks/local.md", []string{"task", "refresh"}, !sqlite},
				{"tasks/local.md", []string{"task", "sync"}, !sqlite},
				{"tasks/local.md", []string{"task", "list"}, !sqlite},
				{"tasks/local.md", []string{"housekeeping"}, !sqlite},
				{"sessions/local.md", []string{"journal", "recent"}, false},
				{"sessions/local.md", []string{"migrate", "markdown"}, true},
				{"sessions/local.md", []string{"state", "migrate", "markdown"}, true},
				{"sessions/local.md", []string{"state", "restore-ephemerals", "--target", "all"}, true},
				{"plans/local.md", []string{"plan", "list"}, false},
				{"handoffs/local.md", []string{"handoff", "list"}, false},
			} {
				path := filepath.Join(linked, ".agents", filepath.FromSlash(testcase.path))
				mkdirAll(t, filepath.Dir(path))
				writeFile(t, path, "# Local storage\n")
				if got := runner.worktreeStorageRefusal(testcase.args, linked); (got != "") != testcase.refuse {
					t.Errorf("%v against %s: refusal=%q, want %v", testcase.args, testcase.path, got, testcase.refuse)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
