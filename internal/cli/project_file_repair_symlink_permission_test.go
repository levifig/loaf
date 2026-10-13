package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectRepairUnreadableSymlinkTargetIsNotAbsence(t *testing.T) {
	skipWithoutEnforcedPermissions(t)
	dir := realpath(t, t.TempDir())
	outside := realpath(t, t.TempDir())
	writeInstallFile(t, filepath.Join(dir, "AGENTS.md"), "# Canonical\n")
	writeInstallFile(t, filepath.Join(outside, "CLAUDE.md"), "# Outside preserved\n")
	if err := os.Symlink(outside, filepath.Join(dir, ".claude")); err != nil {
		t.Fatal(err)
	}
	chmodForTest(t, outside, 0)
	action, detail := planInstallClaudeInstructions(dir, true)
	if action != "error" || !strings.Contains(detail, "permission denied") {
		t.Fatalf("plan = %s %s, want metadata error instead of absence", action, detail)
	}
	result := checkClaudeInstructions().Run(doctorContext{projectRoot: dir})
	if result.Status != doctorFail || result.Fixable || !strings.Contains(result.Detail, "permission denied") {
		t.Fatalf("doctor = %#v, want metadata error instead of absence", result)
	}
	applied := ensureInstallClaudeInstructions(dir, installSymlinkOptions{AssumeYes: true})
	if applied.Error == "" || !strings.Contains(applied.Error, "permission denied") {
		t.Fatalf("apply = %#v, want metadata error instead of absence", applied)
	}
	chmodForTest(t, outside, 0o755)
	assertInstallFile(t, filepath.Join(outside, "CLAUDE.md"), "# Outside preserved\n")
}
