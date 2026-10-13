package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectRepairRootImportReportsUnreadableFiles(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		unreadable bool
	}{
		{name: "oversized root instructions", path: "CLAUDE.md"},
		{name: "unreadable local instructions", path: "CLAUDE.local.md", unreadable: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			body := strings.Repeat("x", projectFileReadLimit+1)
			if tc.unreadable {
				skipWithoutEnforcedPermissions(t)
				body = "@AGENTS.md\n"
			}
			path := filepath.Join(dir, tc.path)
			writeInstallFile(t, path, body)
			if tc.unreadable {
				chmodForTest(t, path, 0)
			}
			result := checkClaudeRootImport().Run(doctorContext{projectRoot: dir})
			if result.Status != doctorFail || result.Fixable || !strings.Contains(result.Detail, tc.path) {
				t.Fatalf("doctor = %#v, want actionable inspection failure rather than missing-import claim", result)
			}
		})
	}
}

func TestProjectRepairRootImportKeepsNonFollowingSymlinkGuidance(t *testing.T) {
	dir := t.TempDir()
	writeInstallFile(t, filepath.Join(dir, "other.md"), "@AGENTS.md\n")
	if err := os.Symlink("other.md", filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	result := checkClaudeRootImport().Run(doctorContext{projectRoot: dir})
	if result.Status != doctorWarn {
		t.Fatalf("doctor = %#v, want existing non-follow warning", result)
	}
}
