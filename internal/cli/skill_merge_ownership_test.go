package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// mergedSkills are the skills folded into loaf-core, architecture, and
// release, plus the pre-rename foundations name. None of them ships, and
// upgrade removes their installed copies.
var mergedSkills = []string{"foundations", "knowledge-base", "documentation-standards", "debugging", "security-compliance"}

func TestMergedSkillsAreRetiredAndAbsentFromBuildOutput(t *testing.T) {
	repo := testRepositoryRoot(t)
	manifest, found, err := loadInstallDeprecationManifest(repo)
	if err != nil || !found {
		t.Fatalf("manifest: found=%v, err=%v", found, err)
	}
	declared := map[string]bool{}
	for _, retired := range manifest.RetiredSkills {
		declared[retired.Skill] = true
	}
	for _, name := range mergedSkills {
		if !declared[name] {
			t.Errorf("config/deprecations.json does not retire %s", name)
		}
		assertInstallPathMissing(t, filepath.Join(repo, "content", "skills", name))
	}
	if len(manifest.Aliases) != 0 {
		t.Errorf("the merge is a clean cut; aliases = %#v", manifest.Aliases)
	}

	root := setupIsolatedRepositoryBuildRoot(t)
	if err := (Runner{WorkingDir: root}).Run([]string{"build"}); err != nil {
		t.Fatal(err)
	}
	for _, target := range defaultBuildTargets {
		skills := nativeBuildSkillTreeDir(root, target)
		if _, err := os.Stat(filepath.Join(skills, "loaf-core", "SKILL.md")); err != nil {
			t.Errorf("%s build lacks loaf-core: %v", target, err)
		}
		for _, name := range mergedSkills {
			assertInstallPathMissing(t, filepath.Join(skills, name))
			assertInstallPathMissing(t, filepath.Join(nativeBuildTargetOutputDir(root, target), "commands", name+".md"))
		}
	}
}

func TestUpgradeRemovesInstalledCopiesOfMergedSkills(t *testing.T) {
	root, home := setupInstallCommandFixture(t)
	writeInstallDeprecationManifest(t, root, readBuildFileString(t, filepath.Join(testRepositoryRoot(t), installDeprecationManifestPath)))
	manifest, found, err := loadInstallDeprecationManifest(root)
	if err != nil || !found {
		t.Fatalf("load real retirement manifest: found=%v, err=%v", found, err)
	}
	pathContext := map[string]string{"HOME": home, "XDG_CONFIG_HOME": filepath.Join(home, ".config")}
	var seeded []string
	for _, name := range mergedSkills {
		var retirement *retiredInstallSkill
		for i := range manifest.RetiredSkills {
			if manifest.RetiredSkills[i].Skill == name {
				retirement = &manifest.RetiredSkills[i]
			}
		}
		if retirement == nil {
			t.Fatalf("%s has no retirement declaration, so upgrade would leave its installed copies", name)
		}
		homes, err := retiredSkillHomes(*retirement, pathContext)
		if err != nil || len(homes) == 0 {
			t.Fatalf("retired skill homes for %s: %v, %v", name, homes, err)
		}
		for _, skillHome := range homes {
			seedOwnedManagedSkill(t, skillHome, name, "# Installed "+name+"\n")
			seeded = append(seeded, filepath.Join(skillHome, name))
		}
	}
	// An old foundations copy marks Cursor and OpenCode as installed, so the
	// upgrade also syncs their current content, which carries loaf-core.
	for _, target := range []string{"cursor", "opencode"} {
		writeInstallFile(t, filepath.Join(root, "dist", target, "skills", "loaf-core", "SKILL.md"), "# Loaf Core\n")
		installTestHookDistribution(t, root, target)
	}
	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout, WorkingDir: root, Executable: distributionFixtureExecutable(root)}
	if err := runner.Run([]string{"upgrade", "--yes"}); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, stdout.String())
	}
	for _, path := range seeded {
		assertInstallPathMissing(t, path)
	}
}

func TestEveryHookOwnerShips(t *testing.T) {
	repo := testRepositoryRoot(t)
	registry := readBuildFileString(t, filepath.Join(repo, "config", "hooks.yaml"))
	owners := regexp.MustCompile(`(?m)^\s+skill:\s*(\S+)\s*$`).FindAllStringSubmatch(registry, -1)
	if len(owners) == 0 {
		t.Fatal("config/hooks.yaml names no hook owners")
	}
	for _, owner := range owners {
		if _, err := os.Stat(filepath.Join(repo, "content", "skills", owner[1], "SKILL.md")); err != nil {
			t.Errorf("hook owner %q does not ship: %v", owner[1], err)
		}
	}
}

// changelogRulesOwner is the only delivered surface that states the changelog
// rules; every other surface names it in plain text.
const (
	changelogRulesOwner = "content/skills/release/references/changelog.md"
	changelogRulesRoute = "release skill's Changelog rules"
)

func TestChangelogRulesHaveOneOwner(t *testing.T) {
	repo := testRepositoryRoot(t)
	owner := readBuildFileString(t, filepath.Join(repo, filepath.FromSlash(changelogRulesOwner)))
	for _, rule := range []string{
		"Every implemented canonical work record gets a new or substantively updated Unreleased entry",
		"An unchanged, already released, unrelated, or formatting-only entry does not cover new implementation.",
		"run its coverage gate against the actual review base and candidate",
		"A release title, tag, or skip label alone is not proof.",
		"Common Changelog",
		"Never add or change a version heading",
	} {
		if !strings.Contains(owner, rule) {
			t.Errorf("%s lost the rule %q", changelogRulesOwner, rule)
		}
	}

	restatements := []string{
		"Common Changelog", "Keep a Changelog", "`Changed`, `Added`", "Changed | Added", "release-facing",
		// The required-entry rules.
		"substantively updated Unreleased", "formatting-only", "skip label", "regardless of change size or category",
	}
	err := filepath.WalkDir(filepath.Join(repo, "content"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		rel, _ := filepath.Rel(repo, path)
		rel = filepath.ToSlash(rel)
		if rel == changelogRulesOwner {
			return nil
		}
		body := readBuildFileString(t, path)
		for _, marker := range restatements {
			if strings.Contains(body, marker) {
				t.Errorf("%s restates changelog rules (%q); name the %s instead", rel, marker, changelogRulesRoute)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		"content/skills/implement/SKILL.md",
		"content/skills/ship/SKILL.md",
		"content/skills/loaf-core/SKILL.md",
		"content/skills/architecture/SKILL.md",
		"content/skills/git-workflow/references/commits.md",
	} {
		if !strings.Contains(readBuildFileString(t, filepath.Join(repo, filepath.FromSlash(rel))), changelogRulesRoute) {
			t.Errorf("%s mentions changelogs without naming the %s", rel, changelogRulesRoute)
		}
	}
	// Ship keeps its gate: coverage is defined by release but still blocks.
	if ship := readBuildFileString(t, filepath.Join(repo, "content", "skills", "ship", "SKILL.md")); !strings.Contains(ship, "Missing coverage blocks Approve and Done.") {
		t.Error("ship lost the rule that missing changelog coverage blocks Approve and Done")
	}
}

func TestLoafCoreOwnsDebuggingEvidenceAndSecurityChecks(t *testing.T) {
	repo := testRepositoryRoot(t)
	body := readBuildFileString(t, filepath.Join(repo, "content", "skills", "loaf-core", "SKILL.md"))
	debugging, ok := labeledHarnessSectionBody(body, "## Debugging")
	if !ok {
		t.Fatal("loaf-core has no Debugging section")
	}
	for _, rule := range []string{"**Reproduce first.**", "**A bug fix carries a regression test that fails with the fix reverted**", "Capture Deferred Work"} {
		if !strings.Contains(debugging, rule) {
			t.Errorf("loaf-core Debugging lost %q", rule)
		}
	}
	security, ok := labeledHarnessSectionBody(body, "## Security")
	if !ok {
		t.Fatal("loaf-core has no Security section")
	}
	for _, check := range []string{"`check-secrets` hook", "`security-audit` hook", "`loaf check secrets [dir]`", "`loaf check compliance <file.md>`"} {
		if !strings.Contains(security, check) {
			t.Errorf("loaf-core Security does not name %s", check)
		}
	}
	if got := strings.Count(body, "**Secrets rule:**"); got != 1 {
		t.Errorf("loaf-core carries %d secrets rules, want exactly one", got)
	}
	// Codex projects no pre-tool hooks, so the table must not promise them
	// everywhere, and the fallback must stay.
	if !strings.Contains(security, "on harnesses that run Loaf's pre-tool hooks") || !strings.Contains(security, "run `loaf check secrets` over the directories you changed before committing") {
		t.Error("loaf-core Security overstates where the hooks run or lost the manual fallback")
	}
	for _, projection := range nativeCodexHookProjections() {
		if projection.hookID == "check-secrets" || projection.hookID == "security-audit" {
			t.Errorf("Codex now projects %s; update the loaf-core Security harness table", projection.hookID)
		}
	}

	// A second secrets position anywhere in loaf-core's references would make
	// the rule ambiguous again; references point at the rule instead.
	refs, err := filepath.Glob(filepath.Join(repo, "content", "skills", "loaf-core", "references", "*.md"))
	if err != nil || len(refs) == 0 {
		t.Fatalf("loaf-core references: %v, %v", refs, err)
	}
	restatement := regexp.MustCompile(`(?i)secrets? manager|vault|env(ironment)? var(iable)?s? .*secret|secret.* env(ironment)? var`)
	for _, ref := range refs {
		for _, line := range strings.Split(readBuildFileString(t, ref), "\n") {
			if match := restatement.FindString(line); match != "" {
				t.Errorf("%s restates where secrets live (%q); point to the loaf-core skill's secrets rule", filepath.Base(ref), match)
			}
		}
	}
}

// TestFoundationsOnlyInstallIsDetectedAndUpgradedToLoafCore covers an install
// made before the rename, with no version marker or install record: its only
// trace is the foundations skill. Upgrade must still find the harness, retire
// foundations, and install loaf-core.
func TestFoundationsOnlyInstallIsDetectedAndUpgradedToLoafCore(t *testing.T) {
	root, home := setupInstallCommandFixture(t)
	writeInstallDeprecationManifest(t, root, readBuildFileString(t, filepath.Join(testRepositoryRoot(t), installDeprecationManifestPath)))
	cursorConfig := filepath.Join(home, ".cursor")
	if isLoafInstalledForTargetInstall("cursor", cursorConfig, home) {
		t.Fatal("empty Cursor home must not read as a Loaf install")
	}
	seedOwnedManagedSkill(t, filepath.Join(cursorConfig, "skills"), "foundations", "# Foundations\n")
	if !isLoafInstalledForTargetInstall("cursor", cursorConfig, home) {
		t.Fatal("a Cursor home whose only Loaf skill is foundations must read as a Loaf install")
	}
	writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "loaf-core", "SKILL.md"), "# Loaf Core\n")
	installTestHookDistribution(t, root, "cursor")

	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout, WorkingDir: root, Executable: distributionFixtureExecutable(root)}
	if err := runner.Run([]string{"upgrade", "--yes"}); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, stdout.String())
	}
	assertInstallPathMissing(t, filepath.Join(cursorConfig, "skills", "foundations"))
	assertInstallFile(t, filepath.Join(home, ".agents", "skills", "loaf-core", "SKILL.md"), "# Loaf Core\n")
}
