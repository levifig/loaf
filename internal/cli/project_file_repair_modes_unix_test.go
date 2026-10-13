//go:build darwin || linux

package cli

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestProjectRepairMergePreservesCanonicalMode(t *testing.T) {
	previous := syscall.Umask(0o022)
	defer syscall.Umask(previous)
	dir := t.TempDir()
	canonical := filepath.Join(dir, "AGENTS.md")
	writeInstallFile(t, canonical, "# Existing\n")
	if err := os.Chmod(canonical, 0o666); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := mergeRootProjectContent(root, "AGENTS.md", "# Legacy", ".agents/AGENTS.md"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o666 {
		t.Fatalf("canonical mode = %o, want original 666", info.Mode().Perm())
	}
}

func TestProjectRepairMergeKeepsLegacyHeadingForEmptyCanonical(t *testing.T) {
	dir := t.TempDir()
	writeInstallFile(t, filepath.Join(dir, "AGENTS.md"), "")
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := mergeRootProjectContent(root, "AGENTS.md", "# Legacy", ".agents/AGENTS.md"); err != nil {
		t.Fatal(err)
	}
	assertInstallFile(t, filepath.Join(dir, "AGENTS.md"), "\n\n## Migrated from .agents/AGENTS.md\n\n# Legacy\n")
}
