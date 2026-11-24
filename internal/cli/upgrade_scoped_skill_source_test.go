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
					cursor := "---\nname: loaf-core\ndescription: Test loaf-core.\n---\n\n# Loaf Core\n"
					openCode := "---\nname: loaf-core\ndescription: Test loaf-core.\nsubtask: false\n---\n\n# Loaf Core\n"
					writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "loaf-core", "SKILL.md"), cursor)
					writeInstallFile(t, filepath.Join(root, "dist", "opencode", "skills", "loaf-core", "SKILL.md"), openCode)
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
						if err := syncSelectedManagedSkills(source, dest, []string{"loaf-core"}); err != nil {
							t.Fatal(err)
						}
					}
					expected, err := hashInstallSkillTree(filepath.Join(source, "loaf-core"))
					if err != nil {
						t.Fatal(err)
					}
					args := []string{"upgrade", "--select", "skills/skill:loaf-core"}
					if withHook {
						args = append(args, "--select", "cursor/hook:preToolUse/validate-infra-safety")
					}
					preview := func() artifactPlanDecision {
						plan := parseInstallPlanJSON(t, runInstallCapture(t, root, append(append([]string{}, args...), "--dry-run", "--json")...))
						return findScopedSkill(t, plan, "skill:loaf-core")
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
					actual, err := hashInstallSkillTree(filepath.Join(dest, "loaf-core"))
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
