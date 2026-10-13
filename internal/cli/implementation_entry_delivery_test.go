package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are delivery/ownership regressions. They do not claim that the presence
// of prose proves an agent follows it; native behavioral replay supplies that evidence.
func TestImplementationEntryGuidanceReachesInstalledSkills(t *testing.T) {
	root := setupBuildCommandLoafRoot(t)
	for _, dir := range []string{"content", "config"} {
		if err := copyDirContentsForInstall(filepath.Join(testRepositoryRoot(t), dir), filepath.Join(root, dir)); err != nil {
			t.Fatal(err)
		}
	}
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatalf("build: %v\n%s", err, stdout.String())
	}
	selected := []string{"implement", "loaf-reference", "orchestration", "project-management"}
	for _, target := range defaultBuildTargets {
		for _, upgrade := range []bool{false, true} {
			name := target + "/fresh"
			if upgrade {
				name = target + "/scoped-upgrade"
			}
			t.Run(name, func(t *testing.T) {
				source := nativeBuildSkillTreeDir(root, target)
				dest := filepath.Join(t.TempDir(), "skills")
				if upgrade {
					old := filepath.Join(t.TempDir(), "previous")
					for _, skill := range selected {
						writeInstallFile(t, filepath.Join(old, skill, "SKILL.md"), "Previous Loaf guidance.\n")
					}
					if err := syncSelectedManagedSkills(old, dest, selected); err != nil {
						t.Fatal(err)
					}
				}
				foreign := filepath.Join(dest, "operator-skill", "SKILL.md")
				writeInstallFile(t, foreign, "# Operator skill\nKeep these exact bytes.\n")
				if err := syncSelectedManagedSkills(source, dest, selected); err != nil {
					t.Fatal(err)
				}
				flow := string(readFileBytes(t, filepath.Join(dest, "loaf-reference", "references", "flow-semantics.md")))
				if flow != string(readFileBytes(t, filepath.Join(source, "loaf-reference", "references", "flow-semantics.md"))) {
					t.Fatal("installed canonical entry guidance differs from the distribution")
				}
				for _, phrase := range []string{"## Before Entering Implementation", "Before announcing implementation", "loading or invoking Implement for a concrete request", "selected workflow state", "without repeating Shape", "explicit scoped override"} {
					if !strings.Contains(flow, phrase) {
						t.Errorf("%s misses entry guidance %q", target, phrase)
					}
				}
				delegation := string(readFileBytes(t, filepath.Join(dest, "orchestration", "references", "delegation.md")))
				if !strings.Contains(delegation, "selected-state readback") || !strings.Contains(delegation, "before dispatch") {
					t.Fatal("optional delegation lacks verified selection context before dispatch")
				}
				implement := string(readFileBytes(t, filepath.Join(dest, "implement", "SKILL.md")))
				if !strings.Contains(implement, "Before announcing implementation") || !strings.Contains(implement, "#before-entering-implementation") {
					t.Fatal("installed Implement does not route execution through the canonical early-entry rule")
				}
				management := string(readFileBytes(t, filepath.Join(dest, "project-management", "SKILL.md")))
				if !strings.Contains(management, "before loading implement") || !strings.Contains(management, "#before-entering-implementation") {
					t.Fatal("installed Project Management does not route concrete implementation requests before Implement")
				}
				assertInstallFile(t, foreign, "# Operator skill\nKeep these exact bytes.\n")
				before, err := hashInstallSkillTree(dest)
				if err != nil {
					t.Fatal(err)
				}
				if err := syncSelectedManagedSkills(source, dest, selected); err != nil {
					t.Fatal(err)
				}
				after, err := hashInstallSkillTree(dest)
				if err != nil || after != before {
					t.Fatalf("repeat scoped refresh changed the installed tree: %v", err)
				}
			})
		}
	}
}

func TestImplementationEntryRefreshRefusesModifiedOwnedGuidance(t *testing.T) {
	source, dest := t.TempDir(), filepath.Join(t.TempDir(), "skills")
	path := filepath.Join("implement", "SKILL.md")
	writeInstallFile(t, filepath.Join(source, path), "Original owned instructions.\n")
	if err := syncSelectedManagedSkills(source, dest, []string{"implement"}); err != nil {
		t.Fatal(err)
	}
	writeInstallFile(t, filepath.Join(dest, path), "Operator modified instructions.\n")
	writeInstallFile(t, filepath.Join(source, path), "New early-entry instructions.\n")
	manifest := readFileBytes(t, filepath.Join(dest, loafSkillManifestFile))
	if err := syncSelectedManagedSkills(source, dest, []string{"implement"}); err == nil {
		t.Fatal("modified owned guidance was overwritten without an ownership decision")
	}
	assertInstallFile(t, filepath.Join(dest, path), "Operator modified instructions.\n")
	if got, err := os.ReadFile(filepath.Join(dest, loafSkillManifestFile)); err != nil || !bytes.Equal(got, manifest) {
		t.Fatalf("refused refresh changed ownership: %v", err)
	}
}
