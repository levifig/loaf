package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopedCodexPolicyPreservesLateEdits(t *testing.T) {
	for _, action := range []string{"update", "create"} {
		t.Run(action, func(t *testing.T) {
			root, home, rulePath, _ := setupScopedCodexPolicyFixture(t)
			if err := copyDirContentsForInstall(filepath.Join(root, "dist", "cursor", "skills"), filepath.Join(root, "dist", "codex", "skills")); err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(home, ".codex", "rules", codexJournalRuleManifest)
			if action == "create" {
				if err := os.Remove(rulePath); err != nil {
					t.Fatal(err)
				}
			}
			oldManifest := readFileString(t, manifestPath)
			skillPath := filepath.Join(home, ".agents", "skills", "foundations", "SKILL.md")
			oldSkill := readFileString(t, skillPath)
			late := "# concurrent user policy\n"
			t.Cleanup(func() { scopedApplyTestFault = nil })
			scopedApplyTestFault = func(phase string) error {
				if phase == "skills" {
					writeInstallFile(t, rulePath, late)
				}
				return nil
			}
			var out strings.Builder
			err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{"upgrade", "--select", "skills/skill:foundations", "--select", "codex/codex-rule:loaf.rules"})
			if err == nil {
				t.Fatal("scoped policy publication accepted a concurrent user edit")
			}
			if got := readFileString(t, rulePath); got != late {
				t.Fatalf("concurrent policy overwritten: %q", got)
			}
			if readFileString(t, manifestPath) != oldManifest || readFileString(t, skillPath) != oldSkill {
				t.Fatal("refused policy publication failed to roll back ownership and the earlier skill update")
			}
		})
	}
}

func TestScopedCodexPolicyCompletesInterruptedOwnedPublication(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	options := f.options(false, true)
	options.ConfigDir = f.codexHome
	decision := artifactPlanDecision{ID: "codex-rule:loaf.rules", Kind: "codex-rule", Destination: f.dest(), Action: planActionUpdate}
	desired, err := desiredCodexPolicyContent(options, decision, "")
	if err != nil {
		t.Fatal(err)
	}
	writeInstallFile(t, f.dest(), string(desired))
	manifest := codexManagedRuleManifest{Version: 1}
	manifest.set(codexJournalRuleRelativePath, sha256Bytes([]byte("old owned policy")))
	if err := writeCodexManagedRuleManifest(f.manifest(), manifest); err != nil {
		t.Fatal(err)
	}
	if err := publishSelectedCodexPolicyArtifacts(options, []artifactPlanDecision{decision}); err != nil {
		t.Fatal(err)
	}
	updated := readCodexRuleManifestTest(t, f.manifest())
	if digest, ok := updated.ownedDigest(codexJournalRuleRelativePath); !ok || digest != sha256Bytes(desired) {
		t.Fatal("interrupted owned publication did not converge its ownership digest")
	}
}

func TestScopedCodexPolicyPublisherRefusesNewForeignFile(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	options := f.options(false, true)
	options.ConfigDir = f.codexHome
	writeInstallFile(t, f.dest(), "# foreign policy\n")
	decision := artifactPlanDecision{ID: "codex-rule:loaf.rules", Kind: "codex-rule", Destination: f.dest(), Action: planActionCreate}
	if err := publishSelectedCodexPolicyArtifacts(options, []artifactPlanDecision{decision}); err == nil {
		t.Fatal("publisher overwrote a foreign policy created after planning")
	}
	if got := readFileString(t, f.dest()); got != "# foreign policy\n" {
		t.Fatalf("foreign policy changed: %q", got)
	}
}
