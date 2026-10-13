package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectRepairPlanningRefusesSwappedLegacyDirectory(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	writeInstallFile(t, filepath.Join(dir, "AGENTS.md"), "canonical")
	writeInstallFile(t, filepath.Join(outside, "AGENTS.md"), "outside secret")
	if err := os.Symlink(outside, filepath.Join(dir, ".agents")); err != nil {
		t.Fatal(err)
	}
	action, detail, _ := planRootInstallAgentsFile(dir, true)
	if action != "error" {
		t.Fatalf("action = %s (%s), want refusal rather than outside read", action, detail)
	}
}

func TestProjectRepairUnreadableClaudePlanAndDoctor(t *testing.T) {
	skipWithoutEnforcedPermissions(t)
	dir := t.TempDir()
	writeInstallFile(t, filepath.Join(dir, "AGENTS.md"), "canonical")
	writeInstallFile(t, filepath.Join(dir, ".claude", "CLAUDE.md"), "original bytes")
	chmodForTest(t, filepath.Join(dir, ".claude"), 0)
	entries := planInstallProjectFiles(dir, []string{"cursor"}, true, true, "test")
	var refused, skipped bool
	for _, entry := range entries {
		if entry.Path == claudeInstructionsPath {
			refused = entry.Action == "error"
		}
		if entry.Path == "AGENTS.md" {
			skipped = entry.Action == "skipped"
		}
	}
	if !refused || !skipped {
		t.Fatalf("plan = %#v, want error followed by skipped fenced write", entries)
	}
	result := checkClaudeInstructions().Run(doctorContext{projectRoot: dir})
	if result.Status != doctorFail || result.Fixable {
		t.Fatalf("doctor = %#v, want unfixable inspection error", result)
	}
	applied := ensureInstallClaudeInstructions(dir, installSymlinkOptions{AssumeYes: true})
	if applied.Error == "" {
		t.Fatalf("apply = %#v, want refusal", applied)
	}
}

func TestProjectRepairMoveRefusesExistingDestination(t *testing.T) {
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			if kind == "file" {
				writeInstallFile(t, filepath.Join(dir, "original"), "original bytes")
			} else if err := os.Symlink("target", filepath.Join(dir, "original")); err != nil {
				t.Fatal(err)
			}
			writeInstallFile(t, filepath.Join(dir, "backup"), "concurrent bytes")
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			err = renameProjectFileNoReplace(root, "original", "backup")
			if err == nil {
				t.Fatal("move overwrote the preseeded destination")
			}
			if kind == "file" {
				assertInstallFile(t, filepath.Join(dir, "original"), "original bytes")
			} else {
				assertRawSymlink(t, filepath.Join(dir, "original"), "target")
			}
			assertInstallFile(t, filepath.Join(dir, "backup"), "concurrent bytes")
		})
	}
}

func TestProjectRepairRollbackRefusesRecreatedSource(t *testing.T) {
	dir := t.TempDir()
	writeInstallFile(t, filepath.Join(dir, "original"), "recreated bytes")
	writeInstallFile(t, filepath.Join(dir, "backup"), "original bytes")
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	err = restoreProjectFile(root, "backup", "original", os.ErrPermission)
	if !strings.Contains(err.Error(), "recovery retained at backup") {
		t.Fatalf("rollback = %v", err)
	}
	assertInstallFile(t, filepath.Join(dir, "original"), "recreated bytes")
	assertInstallFile(t, filepath.Join(dir, "backup"), "original bytes")
}

func TestProjectRepairMoveRefusesSwappedParent(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	writeInstallFile(t, filepath.Join(outside, "original"), "outside bytes")
	if err := os.Symlink(outside, filepath.Join(dir, ".agents")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := renameProjectFileNoReplace(root, ".agents/original", ".agents/backup"); err == nil {
		t.Fatal("moved an outside file")
	}
	assertInstallFile(t, filepath.Join(outside, "original"), "outside bytes")
	assertInstallPathMissing(t, filepath.Join(outside, "backup"))
}

func TestProjectRepairDoctorRefusesSwappedCursorDirectory(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	writeInstallFile(t, filepath.Join(outside, "rules", "loaf.mdc"), "outside instructions")
	if err := os.Symlink(outside, filepath.Join(dir, ".cursor")); err != nil {
		t.Fatal(err)
	}
	ctx := doctorContext{projectRoot: dir}
	result := checkStaleCursorMdc().Run(ctx)
	if result.Status != doctorFail || result.Fixable {
		t.Fatalf("doctor = %#v, want safe inspection refusal", result)
	}
	fixed := checkStaleCursorMdc().Fix(ctx, result)
	if fixed.Fixed {
		t.Fatalf("fixed = %#v, want refusal", fixed)
	}
	assertInstallFile(t, filepath.Join(outside, "rules", "loaf.mdc"), "outside instructions")
}

func TestProjectRepairRemovalKeepsChangedEntry(t *testing.T) {
	dir := t.TempDir()
	writeInstallFile(t, filepath.Join(dir, "original"), "checked bytes")
	before, err := os.Lstat(filepath.Join(dir, "original"))
	if err != nil {
		t.Fatal(err)
	}
	writeInstallFile(t, filepath.Join(dir, "replacement"), "concurrent bytes")
	if err := os.Rename(filepath.Join(dir, "replacement"), filepath.Join(dir, "original")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := removeProjectEntry(root, "original", before); err == nil {
		t.Fatal("removed an entry that changed after inspection")
	}
	assertInstallFile(t, filepath.Join(dir, "original"), "concurrent bytes")
}
