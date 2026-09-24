package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Root-based helpers are what keep a .claude swapped for a symlink after
// the early symlinkedClaudeDirTarget refusal from moving, reading, or removing
// anything outside the project. Calling them directly on a project whose
// .claude already points outside is the state such a swap produces, with the
// early refusal bypassed.
func TestClaudeInstructionsHelpersAreConfinedToTheProjectRoot(t *testing.T) {
	project := realpath(t, t.TempDir())
	canonical := filepath.Join(project, "AGENTS.md")
	writeInstallFile(t, canonical, "# Root\n")
	outside := realpath(t, t.TempDir())
	outsideFile := filepath.Join(outside, "CLAUDE.md")
	writeInstallFile(t, outsideFile, "# Outside\n")
	outsideLink := filepath.Join(outside, "linked", "CLAUDE.md")
	mkdirAll(t, filepath.Dir(outsideLink))
	if err := os.Symlink("../CLAUDE.md", outsideLink); err != nil {
		t.Fatalf("Symlink error = %v", err)
	}

	for _, tc := range []struct {
		name      string
		claudeDir string
	}{
		{name: "real file outside", claudeDir: outside},
		{name: "link outside", claudeDir: filepath.Dir(outsideLink)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claudeDir := filepath.Join(project, ".claude")
			_ = os.Remove(claudeDir)
			if err := os.Symlink(tc.claudeDir, claudeDir); err != nil {
				t.Fatalf("Symlink(.claude) error = %v", err)
			}
			root, err := os.OpenRoot(project)
			if err != nil {
				t.Fatalf("OpenRoot error = %v", err)
			}
			defer root.Close()

			if _, _, err := lstatClaudeInstructions(root); err == nil {
				t.Error("lstatClaudeInstructions succeeded through an escaping .claude")
			}
			if _, err := claudeInstructionsLinkTarget(root, project); err == nil {
				t.Error("claudeInstructionsLinkTarget succeeded through an escaping .claude")
			}
			if err := removeClaudeInstructionsLink(root); err == nil {
				t.Error("removeClaudeInstructionsLink succeeded through an escaping .claude")
			}
			if _, _, err := retireClaudeInstructionsFile(root, canonical); err == nil {
				t.Error("retireClaudeInstructionsFile succeeded through an escaping .claude")
			}
			if _, err := readRootRegularFile(root, claudeInstructionsName, projectFileReadLimit); err == nil {
				t.Error("readRootRegularFile succeeded through an escaping .claude")
			}

			assertInstallFile(t, outsideFile, "# Outside\n")
			assertRawSymlink(t, outsideLink, "../CLAUDE.md")
			assertInstallPathMissing(t, outsideFile+".bak")
			assertInstallPathMissing(t, outsideLink+".bak")
			assertInstallFile(t, canonical, "# Root\n")
		})
	}
}

// The real-file migration moves the file aside before reading it, so the bytes
// merged into AGENTS.md are the backup's bytes; a read refusal after the move
// puts the file back.
func TestRetireClaudeInstructionsFileMergesTheBackupAndRestoresOnRefusal(t *testing.T) {
	project := realpath(t, t.TempDir())
	canonical := filepath.Join(project, "AGENTS.md")
	writeInstallFile(t, canonical, "# Root\n")
	claude := filepath.Join(project, ".claude", "CLAUDE.md")
	writeInstallFile(t, claude, "# Claude Notes\n")
	root, err := os.OpenRoot(project)
	if err != nil {
		t.Fatalf("OpenRoot error = %v", err)
	}
	defer root.Close()

	backup, merged, err := retireClaudeInstructionsFile(root, canonical)
	if err != nil || !merged || backup != ".claude/CLAUDE.md.bak" {
		t.Fatalf("retire = %q, %v, %v; want merged into .claude/CLAUDE.md.bak", backup, merged, err)
	}
	assertInstallPathMissing(t, claude)
	assertInstallFile(t, claude+".bak", "# Claude Notes\n")
	assertInstallFile(t, canonical, "# Root\n\n## Migrated from .claude/CLAUDE.md\n\n# Claude Notes\n")

	// Oversized content is refused after the move and restored in place.
	big := make([]byte, projectFileReadLimit+1)
	writeInstallFile(t, claude, string(big))
	if _, _, err := retireClaudeInstructionsFile(root, canonical); err == nil {
		t.Fatal("retire accepted a file over the project read limit")
	}
	if info, err := os.Lstat(claude); err != nil || info.Size() != int64(len(big)) {
		t.Fatalf("oversized .claude/CLAUDE.md not restored: info=%v err=%v", info, err)
	}
	assertInstallPathMissing(t, claude+".bak.1")
	assertInstallFile(t, canonical, "# Root\n\n## Migrated from .claude/CLAUDE.md\n\n# Claude Notes\n")
}

// symlinkedClaudeDirWithFile points .claude at an outside directory holding a
// CLAUDE.md, the layout both apply and the plan refuse.
func symlinkedClaudeDirWithFile(t *testing.T, root string) string {
	t.Helper()
	outside := realpath(t, t.TempDir())
	writeInstallFile(t, filepath.Join(outside, "CLAUDE.md"), "# Outside\n")
	if err := os.Symlink(outside, filepath.Join(root, ".claude")); err != nil {
		t.Fatalf("Symlink(.claude) error = %v", err)
	}
	return outside
}

// The install-shaped plan (project files planned unconditionally) must not
// promise the fenced write that apply never reaches after a layout error.
func TestPlanInstallProjectFilesSkipsFencedWritesAfterALayoutError(t *testing.T) {
	root := realpath(t, t.TempDir())
	stale := "# Project\n\n<!-- loaf:managed:start -->\nPrevious Loaf guidance\n" + fencedEndMarker + "\n"
	writeInstallFile(t, filepath.Join(root, "AGENTS.md"), stale)
	symlinkedClaudeDirWithFile(t, root)

	entries := planInstallProjectFiles(root, []string{"cursor"}, true, true, "2.0.0-test.1")
	sawLayoutError := false
	fenced := 0
	for _, entry := range entries {
		switch {
		case entry.Path == claudeInstructionsPath:
			sawLayoutError = entry.Action == "error"
		case entry.Target == "claude-code" || entry.Target == "cursor":
			fenced++
			if entry.Action != "skipped" || !strings.Contains(entry.Detail, claudeInstructionsPath+" failed first") {
				t.Fatalf("fenced entry = %#v, want skipped naming the layout error", entry)
			}
		}
	}
	if !sawLayoutError || fenced != 2 {
		t.Fatalf("entries = %#v, want the layout error and both fenced targets skipped", entries)
	}
	assertInstallFile(t, filepath.Join(root, "AGENTS.md"), stale)
}
