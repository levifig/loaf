package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopedCodexRetirementChecksOwnership(t *testing.T) {
	for _, name := range []string{codexJournalRuleRelativePath, codexJournalGuidanceRelativePath} {
		for _, ownership := range []string{"unchanged", "modified", "foreign"} {
			t.Run(name+"/"+ownership, func(t *testing.T) {
				root, home, rulePath, _ := setupScopedCodexPolicyFixture(t)
				if err := os.Remove(filepath.Join(root, "dist", "codex", ".codex", "rules", codexJournalRuleTemplateRelativePath)); err != nil {
					t.Fatal(err)
				}
				// Legitimate retirement does not need a runnable Loaf binary.
				t.Setenv("PATH", t.TempDir())
				path := rulePath
				if name == codexJournalGuidanceRelativePath {
					path = filepath.Join(home, ".codex", "AGENTS.md")
					writeInstallFile(t, path, "# User instructions\n"+readFileString(t, path))
				}
				manifestPath := filepath.Join(home, ".codex", "rules", codexJournalRuleManifest)
				if ownership == "foreign" {
					manifest := readCodexRuleManifestTest(t, manifestPath)
					manifest.remove(name)
					if err := writeCodexManagedRuleManifest(manifestPath, manifest); err != nil {
						t.Fatal(err)
					}
				} else if ownership == "modified" {
					writeInstallFile(t, path, strings.Replace(readFileString(t, path), "loaf", "changed-loaf", 1))
				}
				before, manifestBefore := readFileString(t, path), readFileString(t, manifestPath)
				args := []string{"upgrade", "--select", "codex/codex-rule:" + name}
				plan := parseInstallPlanJSON(t, runInstallCapture(t, root, append(args, "--dry-run", "--json")...))
				decision := findScopedTargetArtifact(t, plan, "codex", "codex-rule:"+name)
				wantAction := map[string]string{"unchanged": planActionRetire, "modified": planActionConflict, "foreign": planActionPreserve}[ownership]
				if name == codexJournalGuidanceRelativePath && ownership == "modified" {
					wantAction = planActionPreserve
				}
				if decision.Action != wantAction {
					t.Errorf("plan action = %s, want %s", decision.Action, wantAction)
				}
				var out strings.Builder
				err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(args)
				if wantAction == planActionConflict && err == nil {
					t.Error("modified owned policy was accepted")
				} else if wantAction != planActionConflict && err != nil {
					t.Fatal(err)
				}
				if ownership != "unchanged" {
					assertInstallFile(t, path, before)
					assertInstallFile(t, manifestPath, manifestBefore)
					return
				}
				if name == codexJournalRuleRelativePath {
					assertInstallPathMissing(t, path)
				} else {
					assertInstallFile(t, path, "# User instructions\n")
				}
				manifest := readCodexRuleManifestTest(t, manifestPath)
				if _, owned := manifest.ownedDigest(name); owned || len(manifest.Files) != 1 {
					t.Fatalf("retirement did not remove only the selected ownership: %#v", manifest)
				}
			})
		}
	}
}

func TestScopedCodexPolicyMissingRuntimePreservesFiles(t *testing.T) {
	for _, name := range []string{codexJournalRuleRelativePath} {
		t.Run(name, func(t *testing.T) {
			root, home, rulePath, _ := setupScopedCodexPolicyFixture(t)
			t.Setenv("PATH", t.TempDir())
			guidancePath := filepath.Join(home, ".codex", "AGENTS.md")
			manifestPath := filepath.Join(home, ".codex", "rules", codexJournalRuleManifest)
			before := map[string]string{rulePath: readFileString(t, rulePath), guidancePath: readFileString(t, guidancePath), manifestPath: readFileString(t, manifestPath)}
			var out strings.Builder
			err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{"upgrade", "--select", "codex/codex-rule:" + name})
			if err == nil || !strings.Contains(err.Error(), "PATH") || !strings.Contains(err.Error(), "brew install loaf") {
				t.Errorf("error = %v, want missing runtime refusal with install guidance", err)
			}
			for path, body := range before {
				assertInstallFile(t, path, body)
			}
		})
	}
}

func TestSelectedCodexRetirementRechecksDigest(t *testing.T) {
	for _, name := range []string{codexJournalRuleRelativePath, codexJournalGuidanceRelativePath} {
		t.Run(name, func(t *testing.T) {
			f := newCodexRuleInstallFixture(t)
			if err := f.install(t, true, false); err != nil {
				t.Fatal(err)
			}
			path, kind := f.dest(), "codex-rule"
			if name == codexJournalGuidanceRelativePath {
				path, kind = filepath.Join(f.codexHome, name), "codex-guidance"
				seedLegacyCodexGuidance(t, f, "")
			}
			manifest := readCodexRuleManifestTest(t, f.manifest())
			changed := strings.Replace(readFileString(t, path), "loaf", "changed-loaf", 1)
			writeInstallFile(t, path, changed)
			decision := artifactPlanDecision{ID: "codex-rule:" + name, Kind: kind, Destination: path, Action: planActionRetire}
			if err := retireSelectedCodexPolicyArtifact(f.options(false, true), decision, &manifest); err == nil {
				t.Error("writer accepted a stale retirement decision")
			}
			assertInstallFile(t, path, changed)
		})
	}
}
