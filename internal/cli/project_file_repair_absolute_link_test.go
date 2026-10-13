package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRepairAbsoluteLegacyLinkMigrates(t *testing.T) {
	dir := realpath(t, t.TempDir())
	legacy := filepath.Join(dir, ".agents", "AGENTS.md")
	writeInstallFile(t, legacy, "# Legacy original\n")
	if err := os.Symlink(legacy, filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	action, detail, isError := planRootInstallAgentsFile(dir, true)
	if action != "migrated" || isError {
		t.Fatalf("plan = %s %s %v, want existing absolute legacy migration", action, detail, isError)
	}
	result := ensureRootInstallAgentsFile(dir, installSymlinkOptions{AssumeYes: true})
	if result.Action != "migrated" || result.Error != "" {
		t.Fatalf("apply = %#v, want existing absolute legacy migration", result)
	}
	assertInstallFile(t, filepath.Join(dir, "AGENTS.md"), "# Legacy original\n")
	assertInstallPathMissing(t, legacy)
}

func TestProjectRepairAbsoluteInsideLinkPreservesContent(t *testing.T) {
	dir := realpath(t, t.TempDir())
	target := filepath.Join(dir, "docs", "instructions.md")
	writeInstallFile(t, target, "# Preserved original\n")
	if err := os.Symlink(target, filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	action, detail, isError := planRootInstallAgentsFile(dir, true)
	if action != "replaced-file" || isError {
		t.Fatalf("plan = %s %s %v, want safe inside-root conversion", action, detail, isError)
	}
	result := ensureRootInstallAgentsFile(dir, installSymlinkOptions{AssumeYes: true})
	if result.Action != "replaced-file" || result.Error != "" {
		t.Fatalf("apply = %#v, want safe inside-root conversion", result)
	}
	assertInstallFile(t, filepath.Join(dir, "AGENTS.md"), "# Preserved original\n")
	assertInstallFile(t, target, "# Preserved original\n")
}

func TestProjectRepairAbsoluteOutsideLinkRefusesContentRead(t *testing.T) {
	dir := realpath(t, t.TempDir())
	outside := realpath(t, t.TempDir())
	target := filepath.Join(outside, "instructions.md")
	writeInstallFile(t, target, "# Outside original\n")
	if err := os.Symlink(target, filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	action, detail, isError := planRootInstallAgentsFile(dir, true)
	if action != "error" || !isError {
		t.Fatalf("plan = %s %s %v, want outside-root refusal", action, detail, isError)
	}
	result := ensureRootInstallAgentsFile(dir, installSymlinkOptions{AssumeYes: true})
	if result.Error == "" {
		t.Fatalf("apply = %#v, want outside-root refusal", result)
	}
	assertRawSymlink(t, filepath.Join(dir, "AGENTS.md"), target)
	assertInstallFile(t, target, "# Outside original\n")
}
