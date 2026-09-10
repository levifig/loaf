package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScopedUpgradeUsesInstalledCanonicalSkillSource(t *testing.T) {
	for _, withHook := range []bool{false, true} {
		for _, alreadyCanonical := range []bool{false, true} {
			for _, openCodeInstalled := range []bool{false, true} {
				name := map[bool]string{false: "skills-only", true: "cursor-hook"}[withHook] + "/" + map[bool]string{false: "update", true: "preserve"}[alreadyCanonical] + "/" + map[bool]string{false: "cursor-only", true: "opencode-installed"}[openCodeInstalled]
				t.Run(name, func(t *testing.T) {
					root, home := setupScopedUpgradeFixture(t)
					cursor := "---\nname: foundations\ndescription: Test foundations.\n---\n\n# Foundations\n"
					openCode := "---\nname: foundations\ndescription: Test foundations.\nsubtask: false\n---\n\n# Foundations\n"
					writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "foundations", "SKILL.md"), cursor)
					writeInstallFile(t, filepath.Join(root, "dist", "opencode", "skills", "foundations", "SKILL.md"), openCode)
					target := "cursor"
					if openCodeInstalled {
						target = "opencode"
					} else {
						if err := os.RemoveAll(filepath.Join(home, ".config", "opencode")); err != nil {
							t.Fatal(err)
						}
						if err := os.Remove(installRecordPath(home, "opencode")); err != nil {
							t.Fatal(err)
						}
					}
					source := filepath.Join(root, "dist", target, "skills")
					dest := filepath.Join(home, ".agents", "skills")
					if alreadyCanonical {
						if err := syncSelectedManagedSkills(source, dest, []string{"foundations"}); err != nil {
							t.Fatal(err)
						}
					}
					expected, err := hashInstallSkillTree(filepath.Join(source, "foundations"))
					if err != nil {
						t.Fatal(err)
					}
					args := []string{"upgrade", "--select", "skills/skill:foundations"}
					if withHook {
						args = append(args, "--select", "cursor/hook:preToolUse/validate-infra-safety")
					}
					preview := func() artifactPlanDecision {
						plan := parseInstallPlanJSON(t, runInstallCapture(t, root, append(append([]string{}, args...), "--dry-run", "--json")...))
						return findScopedSkill(t, plan, "skill:foundations")
					}
					first := preview()
					wantAction := planActionUpdate
					if alreadyCanonical {
						wantAction = planActionPreserve
					}
					if first.Action != wantAction || first.DesiredSHA256 != expected {
						t.Fatalf("first action=%s desired=%s, want %s %s", first.Action, first.DesiredSHA256, wantAction, expected)
					}
					runInstallCapture(t, root, args...)
					actual, err := hashInstallSkillTree(filepath.Join(dest, "foundations"))
					if err != nil {
						t.Fatal(err)
					}
					if actual != expected {
						t.Fatalf("applied hash=%s want planned %s", actual, expected)
					}
					second := preview()
					if second.Action != planActionPreserve || second.LiveSHA256 != expected || second.DesiredSHA256 != expected {
						t.Fatalf("second plan did not converge: %#v", second)
					}
				})
			}
		}
	}
}
