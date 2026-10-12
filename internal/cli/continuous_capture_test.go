package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContinuousCaptureReachesFreshAndUpgradedProjects(t *testing.T) {
	for _, upgrade := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "upgrade"}[upgrade], func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "AGENTS.md")
			prefix, suffix := "# Custom rules\nKeep my runtime.\n\n", "\n# Custom tail\nNever deploy.\n"
			if upgrade {
				writeInstallFile(t, path, prefix+"<!-- loaf:managed:start -->\nOld guidance\n"+fencedEndMarker+suffix)
			}
			results, err := installFencedSectionsForTargets(defaultBuildTargets, root, "0.6.0-rc.2", upgrade)
			if err != nil {
				t.Fatal(err)
			}
			body := readBuildFileString(t, path)
			for _, want := range []string{
				"Record significant decisions and discoveries as they happen",
				"Before completing substantial work, handing off, or compacting context",
				"useful reasoning is missing from existing records",
				"abandoned approaches and why",
				"unresolved uncertainty",
				"next useful step",
				"Routine interactions need no redundant checkpoint",
				"foundations",
			} {
				if !strings.Contains(body, want) {
					t.Errorf("managed instructions missing %q", want)
				}
			}
			if strings.Contains(body, "/wrap") || strings.Contains(body, "orchestration` skill for full details") {
				t.Error("ordinary capture still depends on a workflow invocation")
			}
			if upgrade && body != prefix+generateFencedContent()+suffix {
				t.Error("upgrade changed user instructions outside the managed section")
			}
			for _, target := range defaultBuildTargets {
				if results[target].Error != "" {
					t.Errorf("%s instruction delivery: %s", target, results[target].Error)
				}
			}
		})
	}
}

func TestWrapCommandRetirementRefusesBeforeInstallerMutations(t *testing.T) {
	for _, target := range []string{"opencode", "cursor"} {
		for _, body := range []string{"# Legacy Wrap\n", "# My customized command\n"} {
			t.Run(target+"/"+body, func(t *testing.T) {
				root := setupIsolatedRepositoryBuildRoot(t)
				if err := (Runner{WorkingDir: root}).Run([]string{"build", "--target", target}); err != nil {
					t.Fatal(err)
				}
				config := t.TempDir()
				wrap := filepath.Join(config, "commands", "wrap.md")
				other := filepath.Join(config, "commands", "personal.md")
				writeInstallFile(t, wrap, body)
				writeInstallFile(t, other, "# My other command\n")
				options := targetInstallOptions{Target: target, DistDir: nativeBuildTargetOutputDir(root, target), ConfigDir: config, Version: "0.6.0-rc.2", HookState: installTestHookState(t)}
				decisions, err := planTargetDistribution(options)
				if err != nil {
					t.Fatal(err)
				}
				var blocked bool
				for _, decision := range decisions {
					if decision.Destination == wrap && decision.Action == planActionConflict && strings.Contains(decision.Detail, "manually") {
						blocked = true
					}
				}
				if !blocked {
					t.Fatalf("plan omitted manual command retirement: %+v", decisions)
				}
				if target == "opencode" {
					err = installOpencodeTarget(options)
				} else {
					err = installCursorTarget(options)
				}
				if err == nil || !strings.Contains(err.Error(), "ownership receipt") || !strings.Contains(err.Error(), "manually") {
					t.Fatalf("retirement error = %v, want actionable ownership refusal", err)
				}
				assertInstallFile(t, wrap, body)
				assertInstallFile(t, other, "# My other command\n")
				assertInstallPathMissing(t, filepath.Join(config, "skills"))
				assertInstallPathMissing(t, filepath.Join(config, loafInstallMarkerFile))
			})
		}
	}
}

func TestCompactionCaptureIsSelectiveAndWritesNothing(t *testing.T) {
	root := t.TempDir()
	stateHome := filepath.Join(root, "uninitialized-state")
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root, StateHome: stateHome, Stdin: strings.NewReader(`{"session_id":"capture-test"}`)}).Run([]string{"journal", "context", "for-compact"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"useful reasoning is missing from existing records", "abandoned approaches and why", "unresolved uncertainty", "next useful step", "Routine interactions need no redundant checkpoint", "wrap(scope):"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("compaction output missing %q", want)
		}
	}
	if _, err := os.Stat(stateHome); !os.IsNotExist(err) {
		t.Fatalf("instruction-only compaction created state: %v", err)
	}
}

func TestContinuousCaptureShipsWithoutWrapSkill(t *testing.T) {
	root := setupIsolatedRepositoryBuildRoot(t)
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatalf("build: %v\n%s", err, stdout.String())
	}
	for _, target := range defaultBuildTargets {
		skills := nativeBuildSkillTreeDir(root, target)
		assertInstallPathMissing(t, filepath.Join(skills, "wrap"))
		body := readBuildFileString(t, filepath.Join(skills, "foundations", "references", "continuity.md"))
		if !strings.Contains(body, "Routine interactions need no redundant checkpoint") {
			t.Errorf("%s lacks selective continuity reference", target)
		}
		assertInstallPathMissing(t, filepath.Join(nativeBuildTargetOutputDir(root, target), "commands", "wrap.md"))
	}
	assertInstallPathMissing(t, filepath.Join(root, "dist", "skills", "wrap"))
}

func TestWrapRetirementUsesRealManifestAndPreservesCustomizations(t *testing.T) {
	root, home := setupInstallCommandFixture(t)
	manifest := readBuildFileString(t, filepath.Join(testRepositoryRoot(t), installDeprecationManifestPath))
	writeInstallDeprecationManifest(t, root, manifest)
	loaded, found, err := loadInstallDeprecationManifest(root)
	if err != nil || !found {
		t.Fatalf("load real retirement manifest: found=%v, err=%v", found, err)
	}
	var retirement *retiredInstallSkill
	for i := range loaded.RetiredSkills {
		if loaded.RetiredSkills[i].Skill == "wrap" {
			retirement = &loaded.RetiredSkills[i]
		}
	}
	if retirement == nil {
		t.Fatal("Wrap has no retirement declaration")
	}
	homes, err := retiredSkillHomes(*retirement, map[string]string{"HOME": home, "XDG_CONFIG_HOME": filepath.Join(home, ".config")})
	if err != nil || len(homes) < 3 {
		t.Fatalf("retired skill homes: %v, %v", homes, err)
	}
	for _, skillHome := range homes {
		seedOwnedManagedSkill(t, skillHome, "wrap", "# Original managed Wrap\n")
	}
	modified := filepath.Join(homes[0], "wrap", "SKILL.md")
	writeInstallFile(t, modified, "# My customized Wrap\n")
	foreign := filepath.Join(homes[1], "wrap", "SKILL.md")
	// Remove only the fixture's ownership claim: same name, no Loaf ownership.
	if err := unmanageManagedSkillClaim(homes[1], "wrap"); err != nil {
		t.Fatal(err)
	}
	writeInstallFile(t, foreign, "# My unrelated Wrap\n")
	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout, WorkingDir: root, Executable: distributionFixtureExecutable(root)}
	if err := runner.Run([]string{"upgrade", "--dry-run"}); err != nil {
		t.Fatalf("preview: %v\n%s", err, stdout.String())
	}
	for _, skillHome := range homes[2:] {
		assertInstallFile(t, filepath.Join(skillHome, "wrap", "SKILL.md"), "# Original managed Wrap\n")
	}
	stdout.Reset()
	if err := runner.Run([]string{"upgrade", "--yes"}); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, stdout.String())
	}
	assertInstallFile(t, modified, "# My customized Wrap\n")
	assertInstallFile(t, foreign, "# My unrelated Wrap\n")
	for _, skillHome := range homes[2:] {
		assertInstallPathMissing(t, filepath.Join(skillHome, "wrap"))
	}
	if !strings.Contains(stdout.String(), "wrap") || !strings.Contains(stdout.String(), "un-managed") {
		t.Fatalf("upgrade did not report retained customization: %s", stdout.String())
	}
}
