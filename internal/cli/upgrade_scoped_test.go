package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseUpgradeSelectRequiresQualifiedTargetAndID(t *testing.T) {
	if _, err := parseUpgradeArgs([]string{"--select", "skill:foundations"}); err == nil || !strings.Contains(err.Error(), "target/id") {
		t.Fatalf("unqualified id error = %v, want target/id", err)
	}
	if _, err := parseUpgradeArgs([]string{"--select", "cursor/hook:postToolUse/kb-staleness-nudge", "--to", "cursor"}); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("--select + --to error = %v, want rejection", err)
	}
	options, err := parseUpgradeArgs([]string{"--select", "skills/skill:foundations", "--select", "cursor/hook:preToolUse/validate-sql-safety"})
	if err != nil {
		t.Fatalf("parseUpgradeArgs error = %v", err)
	}
	if len(options.selections) != 2 || options.selections[0].String() != "cursor/hook:preToolUse/validate-sql-safety" {
		t.Fatalf("selections = %#v", options.selections)
	}
}

func TestScopedUpgradeDryRunExactnessIdempotencePathLoafAndPreservation(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	args := scopedUpgradeArgs()

	beforeDry := hashScopedSurfaces(t, home)
	plan := parseInstallPlanJSON(t, runInstallCapture(t, root, append([]string{"upgrade"}, append(args, "--dry-run", "--json")...)...))
	if hashScopedSurfaces(t, home) != beforeDry {
		t.Fatal("dry-run mutated scoped surfaces")
	}
	if plan.VersionStamp != "none" {
		t.Fatalf("version_stamp = %q, want none", plan.VersionStamp)
	}
	if strings.Contains(runInstallCapture(t, root, append([]string{"upgrade"}, append(args, "--dry-run", "--json")...)...), `"pin_executable"`) {
		t.Fatal("scoped plan still emits pin_executable")
	}
	if plan.ProjectPart != nil || len(plan.Deprecations) != 0 {
		t.Fatalf("scoped plan leaked whole-target surfaces: %#v %#v", plan.ProjectPart, plan.Deprecations)
	}
	foundations := findScopedSkill(t, plan, "skill:foundations")
	if foundations.Action != planActionUpdate || !strings.Contains(foundations.Diff, "+# Foundations v2") || !strings.Contains(foundations.Diff, "-# Foundations v1") {
		t.Fatalf("foundations decision = %#v", foundations)
	}
	kb := findScopedTargetArtifact(t, plan, "cursor", "hook:postToolUse/kb-staleness-nudge")
	if kb.Action != hookActionUpdate || !strings.Contains(kb.Diff, `"command": "loaf check --hook kb-staleness-nudge"`) {
		t.Fatalf("kb-staleness decision = %#v", kb)
	}
	if strings.Contains(kb.Diff, filepath.Join("loaf", "pinned", "loaf")) || strings.Contains(kb.Diff, "/pinned/loaf") {
		t.Fatalf("kb-staleness desired command is still an absolute pin:\n%s", kb.Diff)
	}

	output := runInstallCapture(t, root, append([]string{"upgrade"}, args...)...)
	if strings.Contains(output, ".loaf-version") && !strings.Contains(output, "left unchanged") {
		t.Fatalf("apply claimed version convergence:\n%s", output)
	}

	assertScopedApplyConverged(t, home, plan)
	if got := strings.TrimSpace(readFileString(t, filepath.Join(home, ".cursor", loafInstallMarkerFile))); got != "0.5.0" {
		t.Fatalf("cursor .loaf-version = %q, want 0.5.0", got)
	}
	if got := strings.TrimSpace(readFileString(t, filepath.Join(home, ".config", "opencode", loafInstallMarkerFile))); got != "0.5.0" {
		t.Fatalf("opencode .loaf-version = %q, want 0.5.0", got)
	}

	afterFirst := hashScopedSurfaces(t, home)
	secondPlan := parseInstallPlanJSON(t, runInstallCapture(t, root, append([]string{"upgrade"}, append(args, "--dry-run", "--json")...)...))
	for _, skill := range secondPlan.Skills {
		if skill.Action != planActionPreserve {
			t.Fatalf("second dry-run skill %s action = %s", skill.ID, skill.Action)
		}
	}
	for _, target := range secondPlan.Targets {
		for _, artifact := range target.Artifacts {
			if artifact.Action != planActionPreserve {
				t.Fatalf("second dry-run %s/%s action = %s", target.Target, artifact.ID, artifact.Action)
			}
		}
	}
	runInstallCapture(t, root, append([]string{"upgrade"}, args...)...)
	if hashScopedSurfaces(t, home) != afterFirst {
		t.Fatal("second apply was not a no-op")
	}
}

func TestScopedUpgradeSelectedConflictAbortsWithoutTouchingPitch(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	pitchPath := filepath.Join(home, ".agents", "skills", "pitch", "SKILL.md")
	foundationsPath := filepath.Join(home, ".agents", "skills", "foundations", "SKILL.md")
	writeInstallFile(t, pitchPath, "# Pitch tampered\n")
	writeInstallFile(t, foundationsPath, "# Foundations tampered\n")
	pitchBefore := readFileString(t, pitchPath)
	foundationsBefore := readFileString(t, foundationsPath)
	hooksBefore := readFileString(t, filepath.Join(home, ".cursor", "hooks.json"))

	var stdout strings.Builder
	err := Runner{Stdout: &stdout, WorkingDir: root, Executable: distributionFixtureExecutable(root)}.Run(append([]string{"upgrade"}, scopedUpgradeArgs()...))
	if err == nil || !strings.Contains(err.Error(), "selected conflict") {
		t.Fatalf("error = %v\n%s, want selected conflict", err, stdout.String())
	}
	if readFileString(t, pitchPath) != pitchBefore {
		t.Fatal("selected conflict rewrote unrelated pitch")
	}
	if readFileString(t, foundationsPath) != foundationsBefore {
		t.Fatal("selected conflict overwrote the conflicted skill")
	}
	if readFileString(t, filepath.Join(home, ".cursor", "hooks.json")) != hooksBefore {
		t.Fatal("selected conflict mutated cursor hooks")
	}
}

func TestScopedUpgradeIgnoresUnrelatedPitchConflict(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	pitchPath := filepath.Join(home, ".agents", "skills", "pitch", "SKILL.md")
	writeInstallFile(t, pitchPath, "# Pitch tampered\n")
	pitchBefore := readFileString(t, pitchPath)

	runInstallCapture(t, root, append([]string{"upgrade"}, scopedUpgradeArgs()...)...)
	if readFileString(t, pitchPath) != pitchBefore {
		t.Fatal("unrelated pitch drift was rewritten")
	}
	if readFileString(t, filepath.Join(home, ".agents", "skills", "foundations", "SKILL.md")) != "# Foundations v2\n" {
		t.Fatal("selected foundations was not replaced")
	}
}

func TestScopedUpgradeRecoversMidApplyFailure(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	before := hashScopedSurfaces(t, home)
	t.Cleanup(func() { scopedApplyTestFault = nil })
	scopedApplyTestFault = func(phase string) error {
		if phase == "cursor-adapters" {
			return errors.New("injected scoped apply failure")
		}
		return nil
	}

	var stdout strings.Builder
	err := Runner{Stdout: &stdout, WorkingDir: root, Executable: distributionFixtureExecutable(root)}.Run(append([]string{"upgrade"}, scopedUpgradeArgs()...))
	if err == nil || !strings.Contains(err.Error(), "injected scoped apply failure") {
		t.Fatalf("error = %v\n%s, want injected failure", err, stdout.String())
	}
	if hashScopedSurfaces(t, home) != before {
		t.Fatal("mid-apply failure left hooks, files, or manifests inconsistent")
	}
}

func TestScopedUpgradeRejectsRetirementLeavingLiveCallers(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{
		"upgrade", "--select", "cursor/hook-file:hooks/post-tool/kb-staleness-nudge.sh",
	})
	if err == nil {
		t.Fatalf("retired hook-file without companions succeeded:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "cursor/hook:postToolUse/kb-staleness-nudge") {
		t.Fatalf("error = %v, want required companion hook id", err)
	}
	old := filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh")
	if _, statErr := os.Stat(old); statErr != nil {
		t.Fatalf("incomplete retirement deleted the script: %v", statErr)
	}
	hooks := readFileString(t, filepath.Join(home, ".cursor", "hooks.json"))
	if !strings.Contains(hooks, "kb-staleness-nudge.sh") {
		t.Fatal("incomplete retirement mutated hooks.json")
	}
}

func TestScopedUpgradeRejectsRetirementLeavingForeignCaller(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	hookPath := filepath.Join(home, ".cursor", "hooks.json")
	foreignCommand := "bash -c 'bash $HOME/.cursor/hooks/post-tool/kb-staleness-nudge.sh'"
	mutateCursorHooks(t, hookPath, func(file map[string]any) {
		hooks := file["hooks"].(map[string]any)
		hooks["postToolUse"] = append(asObjectSlice(hooks["postToolUse"]), map[string]any{
			"command": foreignCommand,
			"matcher": "Write",
			"timeout": 5,
		})
	})
	hooksBefore := readFileString(t, hookPath)
	old := filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh")
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{
		"upgrade",
		"--select", "cursor/hook-file:hooks/post-tool/kb-staleness-nudge.sh",
		"--select", "cursor/hook:postToolUse/kb-staleness-nudge",
	})
	if err == nil {
		t.Fatalf("retirement leaving a foreign caller succeeded:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "preserved hook entries still reference retiring hook files") {
		t.Fatalf("error = %v, want preserved-caller refusal", err)
	}
	if strings.Contains(err.Error(), "also select") {
		t.Fatalf("error = %v, claimed the foreign caller as a selectable companion", err)
	}
	if _, statErr := os.Stat(old); statErr != nil {
		t.Fatalf("foreign-caller retirement deleted the script: %v", statErr)
	}
	hooks := readFileString(t, hookPath)
	if hooks != hooksBefore {
		t.Fatal("foreign-caller retirement mutated hooks.json")
	}
	if !strings.Contains(hooks, foreignCommand) {
		t.Fatal("foreign caller was rewritten or removed")
	}
}

func TestScopedUpgradeRejectsSelectedHookStillReferencingRetirement(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	sources := scopedCursorHookSources()
	for i := range sources {
		if sources[i].hookID != "kb-staleness-nudge" {
			continue
		}
		sources[i].command = "bash $HOME/.cursor/hooks/post-tool/kb-staleness-nudge.sh"
		sources[i].template.(map[string]any)["command"] = sources[i].command
	}
	installTestHookDistribution(t, root, "cursor", sources...)
	hookPath := filepath.Join(home, ".cursor", "hooks.json")
	hooksBefore := readFileString(t, hookPath)
	old := filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh")
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{
		"upgrade",
		"--select", "cursor/hook-file:hooks/post-tool/kb-staleness-nudge.sh",
		"--select", "cursor/hook:postToolUse/kb-staleness-nudge",
	})
	if err == nil {
		t.Fatalf("selected hook still referencing the retiring script succeeded:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "selected cursor/hook:postToolUse/kb-staleness-nudge still references retiring hook files") {
		t.Fatalf("error = %v, want selected companion still-references refusal", err)
	}
	if _, statErr := os.Stat(old); statErr != nil {
		t.Fatalf("selected still-referencing retirement deleted the script: %v", statErr)
	}
	if readFileString(t, hookPath) != hooksBefore {
		t.Fatal("selected still-referencing retirement mutated hooks.json")
	}
}

func TestScopedUpgradeRejectsSelectedHookCreatingHooksFileThatReferencesRetirement(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	sources := scopedCursorHookSources()
	for i := range sources {
		if sources[i].hookID != "kb-staleness-nudge" {
			continue
		}
		sources[i].command = "bash $HOME/.cursor/hooks/post-tool/kb-staleness-nudge.sh"
		sources[i].template.(map[string]any)["command"] = sources[i].command
	}
	installTestHookDistribution(t, root, "cursor", sources...)
	hookPath := filepath.Join(home, ".cursor", "hooks.json")
	if err := os.Remove(hookPath); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh")
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{
		"upgrade",
		"--select", "cursor/hook-file:hooks/post-tool/kb-staleness-nudge.sh",
		"--select", "cursor/hook:postToolUse/kb-staleness-nudge",
	})
	if err == nil {
		t.Fatalf("absent hooks.json + script-backed companion succeeded:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "selected cursor/hook:postToolUse/kb-staleness-nudge still references retiring hook files") {
		t.Fatalf("error = %v, want selected companion still-references refusal", err)
	}
	if _, statErr := os.Stat(old); statErr != nil {
		t.Fatalf("absent-file still-referencing retirement deleted the script: %v", statErr)
	}
	if _, statErr := os.Stat(hookPath); !os.IsNotExist(statErr) {
		t.Fatal("absent-file still-referencing retirement created hooks.json")
	}
}

func TestScopedUpgradeRetiresHookFileWhenHooksFileIsAbsent(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	hookPath := filepath.Join(home, ".cursor", "hooks.json")
	if err := os.Remove(hookPath); err != nil {
		t.Fatal(err)
	}
	runInstallCapture(t, root, "upgrade", "--select", "cursor/hook-file:hooks/post-tool/kb-staleness-nudge.sh")
	assertInstallPathMissing(t, filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh"))
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatal("retire-only created hooks.json when no live callers existed")
	}
}

func TestScopedUpgradeAddsNativeCompanionWithoutReferencingRetiredFile(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	hookPath := filepath.Join(home, ".cursor", "hooks.json")
	if err := os.Remove(hookPath); err != nil {
		t.Fatal(err)
	}
	runInstallCapture(t, root,
		"upgrade",
		"--select", "cursor/hook-file:hooks/post-tool/kb-staleness-nudge.sh",
		"--select", "cursor/hook:postToolUse/kb-staleness-nudge",
	)
	assertInstallPathMissing(t, filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh"))
	hooks := readFileString(t, hookPath)
	if strings.Contains(hooks, "kb-staleness-nudge.sh") {
		t.Fatalf("native companion still references the retiring script:\n%s", hooks)
	}
	if !strings.Contains(hooks, "check --hook kb-staleness-nudge") {
		t.Fatalf("native companion was not written:\n%s", hooks)
	}
}

func TestScopedUpgradeDoesNotPublishPinnedRuntime(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	leftover := leftoverPinnedLoafPath(home)
	writeInstallFile(t, leftover, "old pinned runtime\n")
	t.Cleanup(func() { scopedApplyTestFault = nil })
	scopedApplyTestFault = func(phase string) error {
		if phase == "cursor-adapters" {
			return errors.New("review injected failure")
		}
		return nil
	}
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(append([]string{"upgrade"}, scopedUpgradeArgs()...))
	if err == nil || !strings.Contains(err.Error(), "review injected failure") {
		t.Fatalf("unexpected result: %v\n%s", err, out.String())
	}
	if got := readFileString(t, leftover); got != "old pinned runtime\n" {
		t.Fatalf("failed apply mutated leftover pin: %q", got)
	}
	scopedApplyTestFault = nil
	runInstallCapture(t, root, append([]string{"upgrade"}, scopedUpgradeArgs()...)...)
	if got := readFileString(t, leftover); got != "old pinned runtime\n" {
		t.Fatalf("successful apply mutated leftover pin: %q", got)
	}
	assertNoPublishedPinnedRuntime(t, home)
}

func TestScopedUpgradeSkillsOnlyDoesNotCreatePinnedRuntime(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	leftover := leftoverPinnedLoafPath(home)
	writeInstallFile(t, leftover, "unrelated pinned runtime\n")
	runInstallCapture(t, root, "upgrade", "--select", "skills/skill:foundations")
	if got := readFileString(t, leftover); got != "unrelated pinned runtime\n" {
		t.Fatalf("skills-only apply mutated leftover pin: %q", got)
	}
	assertNoPublishedPinnedRuntime(t, home)
}

func TestScopedUpgradeFailsWhenPathLoafMissing(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	t.Setenv("PATH", t.TempDir())
	hooksBefore := readFileString(t, filepath.Join(home, ".cursor", "hooks.json"))
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(append([]string{"upgrade"}, scopedUpgradeArgs()...))
	if err == nil || !strings.Contains(err.Error(), "PATH loaf is missing") || !strings.Contains(err.Error(), "brew install loaf") {
		t.Fatalf("error = %v\n%s, want missing PATH loaf guidance", err, out.String())
	}
	if readFileString(t, filepath.Join(home, ".cursor", "hooks.json")) != hooksBefore {
		t.Fatal("missing PATH loaf still rewrote hooks")
	}
	assertNoPublishedPinnedRuntime(t, home)
}

func TestScopedUpgradeFailsWhenPathLoafLacksRequiredHooks(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	writeOldPathLoaf(t, root)
	hooksBefore := readFileString(t, filepath.Join(home, ".cursor", "hooks.json"))
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(append([]string{"upgrade"}, scopedUpgradeArgs()...))
	if err == nil || !strings.Contains(err.Error(), "too old or lacks") || !strings.Contains(err.Error(), "validate-infra-safety") || !strings.Contains(err.Error(), "brew upgrade loaf") {
		t.Fatalf("error = %v\n%s, want unsupported PATH loaf guidance", err, out.String())
	}
	if readFileString(t, filepath.Join(home, ".cursor", "hooks.json")) != hooksBefore {
		t.Fatal("unsupported PATH loaf still rewrote hooks")
	}
	assertNoPublishedPinnedRuntime(t, home)
}

func TestScopedUpgradeIgnoresUnselectedForeignSkillSymlink(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	foreign := t.TempDir()
	writeInstallFile(t, filepath.Join(foreign, "SKILL.md"), "foreign skill")
	if err := os.Symlink(foreign, filepath.Join(home, ".agents", "skills", "foreign")); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{
		"upgrade", "--select", "skills/skill:foundations",
	})
	if err != nil {
		t.Fatalf("unselected foreign symlink blocked selected skill: %v\n%s", err, out.String())
	}
}

func TestScopedUpgradeRejectsUnknownRetiredID(t *testing.T) {
	root, _ := setupScopedUpgradeFixture(t)
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{
		"upgrade", "--select", "cursor/hook-file:hooks/never-existed.sh", "--dry-run", "--json",
	})
	if err == nil {
		t.Fatal("unknown artifact accepted as already retired")
	}
	combined := err.Error() + "\n" + out.String()
	if !strings.Contains(combined, "unknown scoped selection") && !strings.Contains(combined, "never-existed.sh") {
		t.Fatalf("error = %v\n%s, want unknown selection", err, out.String())
	}
}

func TestScopedUpgradeCleansSkillBackupsAfterSuccess(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	runInstallCapture(t, root, "upgrade", "--select", "skills/skill:foundations")
	assertNoScopedBackupDirs(t, home)
	runInstallCapture(t, root, "upgrade", "--select", "skills/skill:foundations")
	assertNoScopedBackupDirs(t, home)
}

func scopedUpgradeArgs() []string {
	return []string{
		"--select", "skills/skill:foundations",
		"--select", "cursor/hook-file:hooks/post-tool/kb-staleness-nudge.sh",
		"--select", "cursor/hook:postToolUse/kb-staleness-nudge",
		"--select", "cursor/hook:preToolUse/validate-infra-safety",
		"--select", "cursor/hook:preToolUse/validate-sql-safety",
		"--select", "opencode/hook-file:plugins/hooks/post-tool/kb-staleness-nudge.sh",
		"--select", "opencode/plugin:plugins/hooks.ts",
	}
}

func setupScopedUpgradeFixture(t *testing.T) (string, string) {
	t.Helper()
	root, home := setupUpgradeFixture(t)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data"))
	writeCapablePathLoaf(t, root)
	writeScopedDistributionBinaries(t, root)

	writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "foundations", "SKILL.md"), "# Foundations v1\n")
	writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "pitch", "SKILL.md"), "# Pitch v1\n")
	writeInstallFile(t, filepath.Join(root, "dist", "opencode", "skills", "foundations", "SKILL.md"), "# Foundations v1\n")
	writeInstallFile(t, filepath.Join(root, "dist", "opencode", "skills", "pitch", "SKILL.md"), "# Pitch v1\n")
	writeScopedPlugin(t, root, "opencode", "plugins/hooks.ts", "export const hooks = { kb: { script: 'post-tool/kb-staleness-nudge.sh' } }\n")
	installTestHookDistribution(t, root, "cursor", scopedCursorHookSources()...)

	runInstallFixture(t, root, "install", "--to", "cursor", "--yes")
	runInstallFixture(t, root, "install", "--to", "opencode", "--yes")
	for _, path := range []string{"AGENTS.md", ".agents", ".claude"} {
		if err := os.RemoveAll(filepath.Join(root, path)); err != nil {
			t.Fatalf("RemoveAll(%s) error = %v", path, err)
		}
	}

	writeInstallFile(t, filepath.Join(root, "dist", "cursor", "skills", "foundations", "SKILL.md"), "# Foundations v2\n")
	writeInstallFile(t, filepath.Join(root, "dist", "opencode", "skills", "foundations", "SKILL.md"), "# Foundations v2\n")
	writeScopedPlugin(t, root, "opencode", "plugins/hooks.ts", "export const hooks = { kb: { command: 'loaf check --hook kb-staleness-nudge' }, infra: { command: 'loaf check --hook validate-infra-safety' } }\n")

	writeInstallFile(t, filepath.Join(home, ".cursor", loafInstallMarkerFile), "0.5.0\n")
	writeInstallFile(t, filepath.Join(home, ".config", "opencode", loafInstallMarkerFile), "0.5.0\n")
	seedScopedLeftoversAndDrift(t, home)
	return root, home
}

func scopedCursorHookSources() []hookCatalogSource {
	return []hookCatalogSource{
		{
			event:    "postToolUse",
			hookID:   "kb-staleness-nudge",
			typeName: "command",
			command:  "loaf check --hook kb-staleness-nudge",
			template: map[string]any{"command": "loaf check --hook kb-staleness-nudge", "matcher": "Edit|Write", "timeout": 5, "loaf-managed": true},
		},
		{
			event:    "preToolUse",
			hookID:   "validate-infra-safety",
			typeName: "command",
			command:  "loaf check --hook validate-infra-safety",
			template: map[string]any{"command": "loaf check --hook validate-infra-safety", "matcher": "Bash", "failClosed": true, "timeout": 30, "loaf-managed": true},
		},
		{
			event:    "preToolUse",
			hookID:   "validate-sql-safety",
			typeName: "command",
			command:  "loaf check --hook validate-sql-safety",
			template: map[string]any{"command": "loaf check --hook validate-sql-safety", "matcher": "Bash", "failClosed": true, "timeout": 30, "loaf-managed": true},
		},
		{
			event:    "beforeShellExecution",
			hookID:   "validate-commit",
			typeName: "command",
			command:  "loaf check --hook validate-commit",
			template: map[string]any{"command": "loaf check --hook validate-commit", "matcher": "Bash", "loaf-managed": true},
		},
	}
}

func writeScopedDistributionBinaries(t *testing.T, root string) {
	t.Helper()
	writeInstallFile(t, filepath.Join(root, "bin", "loaf"), "#!/bin/sh\necho scoped-pin-launcher\n")
	if err := os.Chmod(filepath.Join(root, "bin", "loaf"), 0o755); err != nil {
		t.Fatalf("chmod launcher: %v", err)
	}
	native := filepath.Join(root, "bin", "native", runtime.GOOS+"-"+runtime.GOARCH, "loaf")
	writeInstallFile(t, native, "#!/bin/sh\necho scoped-pin-native\n")
	if err := os.Chmod(native, 0o755); err != nil {
		t.Fatalf("chmod native: %v", err)
	}
	testTarget := filepath.Join(root, "bin", "native", "test-target", "loaf")
	writeInstallFile(t, testTarget, "#!/bin/sh\necho fixture-executable\n")
	if err := os.Chmod(testTarget, 0o755); err != nil {
		t.Fatalf("chmod fixture executable: %v", err)
	}
}

func writeScopedPlugin(t *testing.T, root string, target string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, "dist", target, filepath.FromSlash(rel))
	writeInstallFile(t, path, body)
	writeTestTargetAdapterManifest(t, filepath.Join(root, "dist", target), target, []map[string]string{{
		"id":          "plugin:" + rel,
		"kind":        "plugin",
		"source_path": rel,
		"destination": rel,
		"sha256":      sha256Bytes([]byte(body)),
	}})
}

func seedScopedLeftoversAndDrift(t *testing.T, home string) {
	t.Helper()
	cursorLeftover := filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh")
	writeInstallFile(t, cursorLeftover, "#!/bin/sh\necho leftover-cursor\n")
	if err := os.Chmod(cursorLeftover, 0o755); err != nil {
		t.Fatalf("chmod cursor leftover: %v", err)
	}
	appendInstalledHookFile(t, filepath.Join(home, ".cursor", targetInstallManifestFile), "hook-file:hooks/post-tool/kb-staleness-nudge.sh", "hooks/post-tool/kb-staleness-nudge.sh", cursorLeftover)

	openLeftover := filepath.Join(home, ".config", "opencode", "plugins", "hooks", "post-tool", "kb-staleness-nudge.sh")
	writeInstallFile(t, openLeftover, "#!/bin/sh\necho leftover-opencode\n")
	if err := os.Chmod(openLeftover, 0o755); err != nil {
		t.Fatalf("chmod opencode leftover: %v", err)
	}
	appendInstalledHookFile(t, filepath.Join(home, ".config", "opencode", targetInstallManifestFile), "hook-file:plugins/hooks/post-tool/kb-staleness-nudge.sh", "plugins/hooks/post-tool/kb-staleness-nudge.sh", openLeftover)

	foreign := filepath.Join(home, ".cursor", "hooks", "pre-tool", "foundations-check-secrets.sh")
	writeInstallFile(t, foreign, "#!/bin/sh\necho march-2026-foreign\n")
	mutateCursorHooks(t, filepath.Join(home, ".cursor", "hooks.json"), func(file map[string]any) {
		hooks := file["hooks"].(map[string]any)
		post := asObjectSlice(hooks["postToolUse"])
		for _, entry := range post {
			if command, _ := entry["command"].(string); strings.Contains(command, "kb-staleness-nudge") {
				entry["command"] = "bash $HOME/.cursor/hooks/post-tool/kb-staleness-nudge.sh"
			}
		}
		hooks["postToolUse"] = post
		pre := asObjectSlice(hooks["preToolUse"])
		kept := pre[:0]
		for _, entry := range pre {
			command, _ := entry["command"].(string)
			if strings.Contains(command, "validate-infra-safety") || strings.Contains(command, "validate-sql-safety") {
				continue
			}
			kept = append(kept, entry)
		}
		kept = append(kept, map[string]any{
			"command": "bash $HOME/.cursor/hooks/pre-tool/foundations-check-secrets.sh",
			"matcher": "Bash",
			"timeout": 30,
		})
		hooks["preToolUse"] = kept
		if before, ok := hooks["beforeShellExecution"].([]any); ok {
			for _, raw := range before {
				entry, _ := raw.(map[string]any)
				if command, _ := entry["command"].(string); strings.Contains(command, "validate-commit") {
					entry["command"] = "loaf check --hook validate-commit --advisory"
				}
			}
		}
	})
}

func appendInstalledHookFile(t *testing.T, manifestPath string, id string, destination string, path string) {
	t.Helper()
	manifest, err := readTargetAdapterManifest(manifestPath)
	if err != nil {
		t.Fatalf("read installed manifest: %v", err)
	}
	mode := uint32(0o755)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read leftover: %v", err)
	}
	manifest.Artifacts = append(manifest.Artifacts, targetAdapterArtifact{
		ID:          id,
		Kind:        "hook-file",
		SourcePath:  destination,
		Destination: destination,
		SHA256:      sha256Bytes(body),
		Mode:        &mode,
	})
	if err := writeTargetAdapterManifest(manifestPath, manifest); err != nil {
		t.Fatalf("write installed manifest: %v", err)
	}
}

func mutateCursorHooks(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hooks: %v", err)
	}
	var file map[string]any
	if err := json.Unmarshal(body, &file); err != nil {
		t.Fatalf("unmarshal hooks: %v\n%s", err, body)
	}
	mutate(file)
	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatalf("marshal hooks: %v", err)
	}
	writeInstallFile(t, path, string(out)+"\n")
}

func asObjectSlice(value any) []map[string]any {
	raw, _ := value.([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if object, ok := item.(map[string]any); ok {
			out = append(out, object)
		}
	}
	return out
}

func assertScopedApplyConverged(t *testing.T, home string, plan installDryRunPlan) {
	t.Helper()
	foundations := filepath.Join(home, ".agents", "skills", "foundations")
	digest, err := hashInstallSkillTree(foundations)
	if err != nil {
		t.Fatalf("hash foundations: %v", err)
	}
	if digest != findScopedSkill(t, plan, "skill:foundations").DesiredSHA256 {
		t.Fatalf("applied foundations digest %s != dry-run desired %s", digest, findScopedSkill(t, plan, "skill:foundations").DesiredSHA256)
	}
	assertInstallPathMissing(t, filepath.Join(home, ".cursor", "hooks", "post-tool", "kb-staleness-nudge.sh"))
	assertInstallPathMissing(t, filepath.Join(home, ".config", "opencode", "plugins", "hooks", "post-tool", "kb-staleness-nudge.sh"))
	if _, err := os.Stat(filepath.Join(home, ".cursor", "hooks", "pre-tool", "foundations-check-secrets.sh")); err != nil {
		t.Fatalf("foreign wrapper missing: %v", err)
	}

	hooks := readFileString(t, filepath.Join(home, ".cursor", "hooks.json"))
	if !strings.Contains(hooks, `"command": "loaf check --hook kb-staleness-nudge"`) {
		t.Fatalf("cursor hooks were not PATH loaf:\n%s", hooks)
	}
	if !strings.Contains(hooks, `"command": "loaf check --hook validate-infra-safety"`) || !strings.Contains(hooks, `"command": "loaf check --hook validate-sql-safety"`) {
		t.Fatalf("cursor safety hooks missing PATH loaf:\n%s", hooks)
	}
	if strings.Contains(hooks, filepath.Join("loaf", "pinned", "loaf")) || strings.Contains(hooks, "/pinned/loaf") {
		t.Fatalf("cursor hooks were rewritten to an absolute pin:\n%s", hooks)
	}
	if !strings.Contains(hooks, "bash $HOME/.cursor/hooks/pre-tool/foundations-check-secrets.sh") {
		t.Fatal("foreign March wrapper entry was removed")
	}
	if !strings.Contains(hooks, "loaf check --hook validate-commit --advisory") {
		t.Fatal("unselected stale validate-commit was rewritten")
	}

	plugin := readFileString(t, filepath.Join(home, ".config", "opencode", "plugins", "hooks.ts"))
	if !strings.Contains(plugin, "loaf check --hook kb-staleness-nudge") {
		t.Fatalf("opencode plugin was not PATH loaf:\n%s", plugin)
	}
	if strings.Contains(plugin, filepath.Join("loaf", "pinned", "loaf")) || strings.Contains(plugin, "/pinned/loaf") {
		t.Fatalf("opencode plugin was rewritten to an absolute pin:\n%s", plugin)
	}

	cursorManifest, err := readTargetAdapterManifest(filepath.Join(home, ".cursor", targetInstallManifestFile))
	if err != nil {
		t.Fatalf("read cursor manifest: %v", err)
	}
	for _, artifact := range cursorManifest.Artifacts {
		if artifact.ID == "hook-file:hooks/post-tool/kb-staleness-nudge.sh" {
			t.Fatal("retired cursor leftover remained in the ownership manifest")
		}
	}
	openManifest, err := readTargetAdapterManifest(filepath.Join(home, ".config", "opencode", targetInstallManifestFile))
	if err != nil {
		t.Fatalf("read opencode manifest: %v", err)
	}
	for _, artifact := range openManifest.Artifacts {
		if artifact.ID == "hook-file:plugins/hooks/post-tool/kb-staleness-nudge.sh" {
			t.Fatal("retired opencode leftover remained in the ownership manifest")
		}
	}
}

func hashScopedSurfaces(t *testing.T, home string) string {
	t.Helper()
	roots := []string{
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, ".cursor", "hooks.json"),
		filepath.Join(home, ".cursor", "hooks"),
		filepath.Join(home, ".cursor", targetInstallManifestFile),
		filepath.Join(home, ".cursor", loafInstallMarkerFile),
		filepath.Join(home, ".config", "opencode", "plugins"),
		filepath.Join(home, ".config", "opencode", targetInstallManifestFile),
		filepath.Join(home, ".config", "opencode", loafInstallMarkerFile),
	}
	return hashInstallFixtureTrees(t, roots...)
}

func leftoverPinnedLoafPath(home string) string {
	return filepath.Join(home, "xdg-data", "loaf", "pinned", "loaf")
}

func assertNoPublishedPinnedRuntime(t *testing.T, home string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(home, "xdg-data", "loaf", ".pinned.loaf-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) > 0 {
		t.Fatalf("scoped apply published or staged a private pin: %v", matches)
	}
}

func writeCapablePathLoaf(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "path-bin")
	body := "#!/bin/sh\n" +
		"if [ \"$1\" = \"version\" ] || [ \"$1\" = \"--version\" ]; then\n" +
		"  echo \"loaf 0.5.0+gtest\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"check\" ]; then\n" +
		"  if [ \"$2\" = \"--help\" ] || [ \"$2\" = \"-h\" ]; then\n" +
		"    echo \"Usage: loaf check --hook <id>\"\n" +
		"    echo \"  --hook      Hook id: artifact-body-write, check-secrets, kb-staleness-nudge, validate-commit, validate-infra-safety, validate-sql-safety\"\n" +
		"    exit 0\n" +
		"  fi\n" +
		"  if [ \"$2\" = \"--hook\" ]; then\n" +
		"    echo ok\n" +
		"    exit 0\n" +
		"  fi\n" +
		"fi\n" +
		"if [ \"$1\" = \"journal\" ] && [ \"$2\" = \"context\" ] && { [ \"$3\" = \"--help\" ] || [ \"$3\" = \"-h\" ]; }; then\n" +
		"  echo \"Usage: loaf journal context\"\n" +
		"  echo \"  --from-hook\"\n" +
		"  echo \"  --codex-hook\"\n" +
		"  echo \"  --cursor-hook\"\n" +
		"  echo \"  --claude-code\"\n" +
		"  echo \"  --opencode-hook\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo \"unsupported $*\" >&2\n" +
		"exit 1\n"
	writeInstallFile(t, filepath.Join(dir, "loaf"), body)
	if err := os.Chmod(filepath.Join(dir, "loaf"), 0o755); err != nil {
		t.Fatalf("chmod path loaf: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeOldPathLoaf(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "old-path-bin")
	body := "#!/bin/sh\n" +
		"if [ \"$1\" = \"version\" ] || [ \"$1\" = \"--version\" ]; then\n" +
		"  echo \"loaf 0.5.0+g40baf53.dirty\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"check\" ]; then\n" +
		"  if [ \"$2\" = \"--help\" ] || [ \"$2\" = \"-h\" ]; then\n" +
		"    echo \"Usage: loaf check --hook <id>\"\n" +
		"    echo \"  --hook      Hook id: check-secrets, validate-commit\"\n" +
		"    exit 0\n" +
		"  fi\n" +
		"  if [ \"$2\" = \"--hook\" ]; then\n" +
		"    echo \"Unknown hook: $3\" >&2\n" +
		"    exit 1\n" +
		"  fi\n" +
		"fi\n" +
		"echo \"loaf 0.5.0+g40baf53.dirty\"\n" +
		"exit 0\n"
	writeInstallFile(t, filepath.Join(dir, "loaf"), body)
	if err := os.Chmod(filepath.Join(dir, "loaf"), 0o755); err != nil {
		t.Fatalf("chmod old path loaf: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func assertNoScopedBackupDirs(t *testing.T, home string) {
	t.Helper()
	patterns := []string{
		filepath.Join(home, ".agents", ".skills.loaf-scoped-bak-*"),
		filepath.Join(home, ".agents", "skills", ".*.loaf-scoped-bak-*"),
		filepath.Join(home, "xdg-data", "loaf", ".pinned.loaf-scoped-bak-*"),
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) > 0 {
			t.Fatalf("successful apply leaked scoped backups: %v", matches)
		}
	}
}

func findScopedSkill(t *testing.T, plan installDryRunPlan, id string) artifactPlanDecision {
	t.Helper()
	for _, skill := range plan.Skills {
		if skill.ID == id {
			return skill
		}
	}
	t.Fatalf("skill %s missing from plan", id)
	return artifactPlanDecision{}
}

func findScopedTargetArtifact(t *testing.T, plan installDryRunPlan, target string, id string) artifactPlanDecision {
	t.Helper()
	for _, item := range plan.Targets {
		if item.Target != target {
			continue
		}
		for _, artifact := range item.Artifacts {
			if artifact.ID == id {
				return artifact
			}
		}
	}
	t.Fatalf("%s/%s missing from plan", target, id)
	return artifactPlanDecision{}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(body)
}
