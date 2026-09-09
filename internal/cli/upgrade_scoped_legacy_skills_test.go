package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopedUpgradeRejectsPartialLegacySkillsWithoutWrites(t *testing.T) {
	for _, absent := range []bool{false, true} {
		for _, dryRun := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{false: "present", true: "absent"}[absent], map[bool]string{false: "apply", true: "preview"}[dryRun]}, "/"), func(t *testing.T) {
				root, home := setupScopedUpgradeFixture(t)
				dest := filepath.Join(home, ".agents", "skills")
				writeInstallFile(t, filepath.Join(dest, loafSkillManifestFile), `{"version":1,"skills":["foundations","pitch"]}`)
				if absent {
					if err := os.RemoveAll(filepath.Join(dest, "pitch")); err != nil {
						t.Fatal(err)
					}
				}
				before := hashScopedSurfaces(t, home)
				args := []string{"upgrade", "--select", "skills/skill:foundations"}
				if dryRun {
					args = append(args, "--dry-run", "--json")
				}
				var out strings.Builder
				err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(args)
				if err == nil || !strings.Contains(err.Error()+out.String(), "unselected legacy skills: pitch") {
					t.Fatalf("partial legacy upgrade error = %v, want actionable refusal; output: %s", err, out.String())
				}
				if hashScopedSurfaces(t, home) != before {
					t.Fatal("rejected partial legacy upgrade changed installed content or ownership")
				}
			})
		}
	}
}

func TestSelectedSkillsWriterRejectsPartialLegacyManifest(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	dest := filepath.Join(home, ".agents", "skills")
	writeInstallFile(t, filepath.Join(dest, loafSkillManifestFile), `{"version":1,"skills":["foundations","pitch"]}`)
	before := hashScopedSurfaces(t, home)
	err := syncSelectedManagedSkills(filepath.Join(root, "dist", "cursor", "skills"), dest, []string{"foundations"})
	if err == nil || !strings.Contains(err.Error(), "unselected legacy skills: pitch") {
		t.Fatalf("writer error = %v, want partial legacy refusal", err)
	}
	if hashScopedSurfaces(t, home) != before {
		t.Fatal("writer changed content before refusing partial legacy ownership")
	}
}

func TestScopedUpgradeCanSelectEveryLegacySkillIncludingRetirement(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	dest := filepath.Join(home, ".agents", "skills")
	writeInstallFile(t, filepath.Join(dest, loafSkillManifestFile), `{"version":1,"skills":["foundations","pitch","retired","absent"]}`)
	writeInstallFile(t, filepath.Join(dest, "retired", "SKILL.md"), "# Retired\n")
	args := []string{"upgrade", "--select", "skills/skill:foundations", "--select", "skills/skill:pitch", "--select", "skills/skill:retired", "--select", "skills/skill:absent"}
	runInstallCapture(t, root, append(append([]string{}, args...), "--dry-run", "--json")...)
	runInstallCapture(t, root, args...)
	state, err := readManagedSkillsState(dest)
	if err != nil || state.legacy || len(state.digests) != 2 || state.digests["foundations"] == "" || state.digests["pitch"] == "" {
		t.Fatalf("all-selected manifest = %#v, error %v", state, err)
	}
	assertInstallPathMissing(t, filepath.Join(dest, "retired"))
	runInstallCapture(t, root, "upgrade", "--select", "skills/skill:foundations", "--dry-run", "--json")
}

func TestScopedHookOnlyUpgradeLeavesLegacySkillsManifestUntouched(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	manifest := filepath.Join(home, ".agents", "skills", loafSkillManifestFile)
	body := `{"version":1,"skills":["foundations","pitch"]}`
	writeInstallFile(t, manifest, body)
	runInstallCapture(t, root, "upgrade", "--select", "cursor/hook:preToolUse/validate-infra-safety")
	if got := readFileString(t, manifest); got != body {
		t.Fatalf("hook-only upgrade changed legacy skills manifest: %s", got)
	}
}
