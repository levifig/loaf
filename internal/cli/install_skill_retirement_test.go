package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetiredDistributionRefusesBeforeInstallOrUpgradeWrites(t *testing.T) {
	for _, payload := range []string{"skills/wrap/SKILL.md", "commands/wrap.md"} {
		for _, action := range []string{"install", "upgrade", "preview", "scoped", "scoped-preview"} {
			t.Run(payload+"/"+action, func(t *testing.T) {
				root, home := setupInstallCommandFixture(t)
				writeInstallDeprecationManifest(t, root, readBuildFileString(t, filepath.Join(testRepositoryRoot(t), installDeprecationManifestPath)))
				writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "foundations", "SKILL.md"), "# Foundations\n")
				installTestHookDistribution(t, root, "cursor")
				writeInstallFile(t, filepath.Join(home, ".cursor", loafInstallMarkerFile), "old\n")
				// Cleanup would remove this before installing from the stale source.
				seedOwnedManagedSkill(t, filepath.Join(home, ".cursor", "skills"), "wrap", "# Owned legacy Wrap\n")
				writeInstallFile(t, filepath.Join(root, "dist", "cursor", filepath.FromSlash(payload)), "# Stale Wrap\n")
				args := []string{"install", "--to", "cursor", "--yes"}
				switch action {
				case "upgrade":
					args = []string{"upgrade", "--to", "cursor", "--yes"}
				case "preview":
					args = []string{"upgrade", "--to", "cursor", "--dry-run", "--json"}
				case "scoped":
					args = []string{"upgrade", "--select", "skills/skill:foundations"}
				case "scoped-preview":
					args = []string{"upgrade", "--select", "skills/skill:foundations", "--dry-run", "--json"}
				}
				before := hashInstallFixtureTrees(t, home)
				var out bytes.Buffer
				err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(args)
				if err == nil || !strings.Contains(err.Error()+out.String(), "retired skill") || !strings.Contains(err.Error()+out.String(), "loaf build") {
					t.Fatalf("stale payload result = %v; output %s", err, out.String())
				}
				if after := hashInstallFixtureTrees(t, home); after != before {
					t.Fatal("stale distribution refusal changed installed content, receipts, or user files")
				}
			})
		}
	}
}

func TestCursorReinstallAndUpgradeKeepCanonicalSkillsAndUserCopies(t *testing.T) {
	root, home := setupInstallCommandFixture(t)
	writeInstallDeprecationManifest(t, root, readBuildFileString(t, filepath.Join(testRepositoryRoot(t), installDeprecationManifestPath)))
	for _, target := range []string{"cursor", "opencode"} {
		writeInstallFile(t, filepath.Join(root, "dist", target, "skills", "foundations", "SKILL.md"), "# Foundations\n")
	}
	installTestHookDistribution(t, root, "cursor")
	personal := filepath.Join(home, ".cursor", "skills", "personal", "SKILL.md")
	userCopy := filepath.Join(home, ".cursor", "skills", "foundations", "SKILL.md")
	retired := filepath.Join(home, ".cursor", "skills", "resume-session", "SKILL.md")
	writeInstallFile(t, personal, "# My skill\n")
	writeInstallFile(t, userCopy, "# My Foundations\n")
	writeInstallFile(t, retired, "# My resume skill\n")
	// A recorded old destination must not redirect later writes there.
	record := installTargetRecord{Target: "cursor", Version: "old", ConfigDir: filepath.Join(home, ".cursor"), SkillsDir: filepath.Join(home, ".cursor", "skills")}
	body, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallFile(t, installRecordPath(home, "cursor"), string(body))
	for _, args := range [][]string{
		{"install", "--to", "cursor", "--yes"},
		{"install", "--to", "opencode", "--yes"},
		{"install", "--to", "cursor", "--yes"},
		{"upgrade", "--dry-run", "--json"},
		{"upgrade", "--yes"},
		{"upgrade", "--yes"},
		{"upgrade", "--select", "skills/skill:foundations"},
	} {
		var out bytes.Buffer
		if err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(args); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out.String())
		}
		assertInstallFile(t, personal, "# My skill\n")
		assertInstallFile(t, userCopy, "# My Foundations\n")
		assertInstallFile(t, retired, "# My resume skill\n")
		assertInstallFile(t, filepath.Join(home, ".agents", "skills", "foundations", "SKILL.md"), "# Foundations\n")
		assertInstallPathMissing(t, filepath.Join(home, ".claude", "skills", "foundations"))
		for _, name := range []string{"wrap", "breakdown", "cleanup", "council-session", "reference-session", "resume-session"} {
			assertInstallPathMissing(t, filepath.Join(home, ".agents", "skills", name))
		}
	}
	// Removal of a redundant copy is respected by later whole and scoped updates.
	if err := os.RemoveAll(filepath.Dir(userCopy)); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"install", "--to", "cursor", "--yes"}, {"upgrade", "--yes"}, {"upgrade", "--select", "skills/skill:foundations"}} {
		runInstallCapture(t, root, args...)
		assertInstallPathMissing(t, filepath.Dir(userCopy))
	}
}

func TestNameOnlySkillReceiptCannotAuthorizeOverwriteOrRetirement(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "full", true: "selected"}[selected], func(t *testing.T) {
			root := t.TempDir()
			src := filepath.Join(root, "dist", "cursor", "skills")
			dest := filepath.Join(root, "home", ".agents", "skills")
			writeInstallFile(t, filepath.Join(src, "foundations", "SKILL.md"), "# Current\n")
			writeInstallFile(t, filepath.Join(dest, "foundations", "SKILL.md"), "# My custom skill\n")
			writeInstallFile(t, filepath.Join(dest, "wrap", "SKILL.md"), "# My custom Wrap\n")
			writeInstallFile(t, filepath.Join(dest, loafSkillManifestFile), `{"version":1,"skills":["foundations","wrap"]}`)
			before := hashInstallFixtureTrees(t, dest)
			var err error
			if selected {
				err = syncSelectedManagedSkills(src, dest, []string{"foundations", "wrap"})
			} else {
				err = syncManagedSkillsDirIfExists(src, dest)
			}
			if err == nil || !strings.Contains(err.Error(), "name-only") {
				t.Fatalf("unproven receipt result = %v", err)
			}
			if hashInstallFixtureTrees(t, dest) != before {
				t.Fatal("name-only ownership changed user content or receipt")
			}
		})
	}
}

func TestUpgradeRelocationCannotPromoteNameOnlySkillOwnership(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "apply", true: "preview"}[dryRun], func(t *testing.T) {
			root, home := setupInstallCommandFixture(t)
			writeInstallDeprecationManifest(t, root, readBuildFileString(t, filepath.Join(testRepositoryRoot(t), installDeprecationManifestPath)))
			writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "foundations", "SKILL.md"), "# Current Foundations\n")
			installTestHookDistribution(t, root, "cursor")
			writeInstallFile(t, filepath.Join(home, ".cursor", loafInstallMarkerFile), "old\n")
			shared := filepath.Join(home, ".agents", "skills")
			writeInstallFile(t, filepath.Join(shared, "foundations", "SKILL.md"), "# My customized Foundations\n")
			writeInstallFile(t, filepath.Join(shared, loafSkillManifestFile), `{"version":1,"skills":["foundations"]}`)
			oldHome := filepath.Join(home, ".config", "opencode", "skills")
			seedOwnedManagedSkill(t, oldHome, "idea", "# Owned legacy Idea\n")
			before := hashInstallFixtureTrees(t, shared, oldHome)
			args := []string{"upgrade", "--to", "cursor", "--yes"}
			if dryRun {
				args = append(args, "--dry-run", "--json")
			}
			var out bytes.Buffer
			err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(args)
			if !strings.Contains(out.String()+fmt.Sprint(err), "name-only") || (!dryRun && err == nil) {
				t.Fatalf("legacy relocation result = %v; output %s", err, out.String())
			}
			if hashInstallFixtureTrees(t, shared, oldHome) != before {
				t.Fatal("relocation promoted legacy ownership or changed custom content, source, or receipts")
			}
		})
	}
}

func TestClaimManagedSkillDigestCannotPromoteNameOnlyOwnership(t *testing.T) {
	home := t.TempDir()
	writeInstallFile(t, filepath.Join(home, "foundations", "SKILL.md"), "# My customized Foundations\n")
	writeInstallFile(t, filepath.Join(home, loafSkillManifestFile), `{"version":1,"skills":["foundations"]}`)
	before := hashInstallFixtureTrees(t, home)
	if err := claimManagedSkillDigest(home, "idea", strings.Repeat("a", 64)); err == nil || !strings.Contains(err.Error(), "name-only") {
		t.Fatalf("legacy claim result = %v", err)
	}
	if hashInstallFixtureTrees(t, home) != before {
		t.Fatal("claim changed unrelated content or legacy ownership receipt")
	}
}

func TestLegacyRetirementManifestAndGeneratedOutputs(t *testing.T) {
	root := setupIsolatedRepositoryBuildRoot(t)
	manifest, found, err := loadInstallDeprecationManifest(root)
	if err != nil || !found {
		t.Fatalf("manifest: found=%v, err=%v", found, err)
	}
	names := []string{"wrap", "breakdown", "cleanup", "council-session", "reference-session", "resume-session"}
	for _, name := range names {
		declared := false
		for _, retired := range manifest.RetiredSkills {
			if retired.Skill == name {
				declared = true
			}
		}
		if !declared {
			t.Fatalf("missing retirement declaration for %s", name)
		}
	}
	if err := (Runner{WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatal(err)
	}
	for _, target := range defaultBuildTargets {
		for _, name := range names {
			assertInstallPathMissing(t, filepath.Join(nativeBuildSkillTreeDir(root, target), name))
			assertInstallPathMissing(t, filepath.Join(nativeBuildTargetOutputDir(root, target), "commands", name+".md"))
		}
	}
}

func TestRetiredSkillsPreserveNameOnlyAndSymlinkedUserContent(t *testing.T) {
	for _, name := range []string{"wrap", "breakdown", "cleanup", "council-session", "reference-session", "resume-session"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "home")
			legacyHome := filepath.Join(home, ".cursor", "skills")
			legacyPath := filepath.Join(legacyHome, name, "SKILL.md")
			writeInstallFile(t, legacyPath, "# My customized legacy skill\n")
			writeInstallFile(t, filepath.Join(legacyHome, loafSkillManifestFile), `{"version":1,"skills":["`+name+`"]}`)
			userTarget := filepath.Join(root, "user-authored", name)
			writeInstallFile(t, filepath.Join(userTarget, "SKILL.md"), "# My linked skill\n")
			linkedHome := filepath.Join(home, ".claude", "skills")
			mkdirAll(t, linkedHome)
			linkedPath := filepath.Join(linkedHome, name)
			if err := os.Symlink(userTarget, linkedPath); err != nil {
				t.Fatal(err)
			}
			manifest, found, err := loadInstallDeprecationManifest(testRepositoryRoot(t))
			if err != nil || !found {
				t.Fatalf("manifest: %v", err)
			}
			paths := map[string]string{"HOME": home, "XDG_CONFIG_HOME": filepath.Join(home, ".config")}
			if _, err := applyInstallDeprecationCleanup(manifest, paths, false); err != nil {
				t.Fatal(err)
			}
			assertInstallFile(t, legacyPath, "# My customized legacy skill\n")
			if _, err := applyInstallDeprecationCleanup(manifest, paths, true); err != nil {
				t.Fatal(err)
			}
			assertInstallFile(t, legacyPath, "# My customized legacy skill\n")
			assertInstallFile(t, filepath.Join(userTarget, "SKILL.md"), "# My linked skill\n")
			if got, err := os.Readlink(linkedPath); err != nil || got != userTarget {
				t.Fatalf("user symlink = %q, %v", got, err)
			}
		})
	}
}
