package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopedUpgradeRejectsInteractiveTargetSelection(t *testing.T) {
	for _, flag := range []string{"-i", "--interactive"} {
		if _, err := parseUpgradeArgs([]string{"--select", "skills/skill:foundations", flag}); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
			t.Fatalf("--select %s error = %v, want conflicting selection modes rejected", flag, err)
		}
	}
}

type forbiddenScopedClaudeCLI struct{ t *testing.T }

func (c forbiddenScopedClaudeCLI) run(args ...string) (string, error) {
	c.t.Fatalf("scoped upgrade consulted unselected Claude plugin state: %v", args)
	return "", nil
}

func TestScopedUpgradeDoesNotConsultUnselectedClaude(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "preview", false: "apply"}[dryRun], func(t *testing.T) {
			root, _ := setupScopedUpgradeFixture(t)
			writeClaudeMarketplaceManifest(t, root)
			claudePath := filepath.Join(root, "bin", "claude")
			writeInstallFile(t, claudePath, "#!/bin/sh\nexit 1\n")
			if err := os.Chmod(claudePath, 0o755); err != nil {
				t.Fatal(err)
			}
			if !installCommandExists("claude") {
				t.Fatal("fixture did not expose claude on PATH")
			}
			args := []string{"upgrade", "--select", "skills/skill:foundations"}
			if dryRun {
				args = append(args, "--dry-run", "--json")
			}
			err := (Runner{Stdout: io.Discard, WorkingDir: root, Executable: distributionFixtureExecutable(root), ClaudePluginCLI: forbiddenScopedClaudeCLI{t}}).Run(args)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestScopedUpgradePreservesUnselectedAmpModesAndOwnership(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	dist := filepath.Join(root, "dist", "amp")
	config := filepath.Join(home, ".config", "amp")
	hookID := "plugin:.amp/plugins/loaf.ts"
	writeDistribution := func(hooks, modes string) {
		t.Helper()
		writeInstallFile(t, filepath.Join(dist, ".amp", "plugins", "loaf.ts"), hooks)
		writeInstallFile(t, filepath.Join(dist, ".amp", "plugins", "loaf-modes.ts"), modes)
		writeTestTargetAdapterManifest(t, dist, "amp", []map[string]string{
			{"id": hookID, "kind": "plugin", "source_path": ".amp/plugins/loaf.ts", "destination": "plugins/loaf.ts", "sha256": sha256Hex(hooks)},
			{"id": ampModesPluginArtifactID, "kind": "plugin", "source_path": ampModesPluginSourcePath, "destination": ampModesPluginDestination, "sha256": sha256Hex(modes)},
		})
	}
	writeDistribution("export const hooks = 1;\n", "export const modes = 1;\n")
	if err := syncTargetAdapterManifest(targetInstallOptions{Target: "amp", DistDir: dist, ConfigDir: config, HomeDir: home, Version: "0.5.0"}); err != nil {
		t.Fatal(err)
	}
	writeInstallFile(t, filepath.Join(config, loafInstallMarkerFile), "0.5.0\n")
	before, err := readTargetAdapterManifest(filepath.Join(config, targetInstallManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	ownedModes := targetAdapterArtifactsByID(before.Artifacts)[ampModesPluginArtifactID]
	writeInstallFile(t, filepath.Join(config, "plugins", "loaf-modes.ts"), "user-edited modes\n")
	writeDistribution("export const hooks = 2;\n", "export const modes = 2;\n")
	runInstallCapture(t, root, "upgrade", "--select", "amp/"+hookID)
	assertInstallFile(t, filepath.Join(config, "plugins", "loaf.ts"), "export const hooks = 2;\n")
	assertInstallFile(t, filepath.Join(config, "plugins", "loaf-modes.ts"), "user-edited modes\n")
	assertInstallFile(t, filepath.Join(config, loafInstallMarkerFile), "0.5.0\n")
	after, err := readTargetAdapterManifest(filepath.Join(config, targetInstallManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := targetAdapterArtifactsByID(after.Artifacts)[ampModesPluginArtifactID]; got.SHA256 != ownedModes.SHA256 {
		t.Fatalf("unselected Amp modes ownership changed: %#v -> %#v", ownedModes, got)
	}
}

func TestScopedUpgradeRejectsLegacyWholeTargetBeforeWrites(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "preview", false: "apply"}[dryRun], func(t *testing.T) {
			root, home := setupScopedUpgradeFixture(t)
			if err := os.Remove(filepath.Join(root, "dist", "opencode", targetBuildManifestFile)); err != nil {
				t.Fatal(err)
			}
			foreign := filepath.Join(home, ".config", "opencode", "foreign.txt")
			writeInstallFile(t, foreign, "user-owned\n")
			if err := os.Chmod(foreign, 0o640); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(foreign)
			if err != nil {
				t.Fatal(err)
			}
			surfaces := hashScopedSurfaces(t, home)
			args := []string{"upgrade", "--select", "opencode/hooks", "--select", "skills/skill:foundations"}
			if dryRun {
				args = append(args, "--dry-run", "--json")
			}
			var output bytes.Buffer
			err = (Runner{Stdout: &output, WorkingDir: root, Executable: distributionFixtureExecutable(root)}).Run(args)
			if err == nil || !strings.Contains(err.Error()+output.String(), "legacy whole-target") {
				t.Errorf("error = %v, output = %s, want legacy whole-target selection rejected", err, output.String())
			}
			after, err := os.Stat(foreign)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(before, after) || before.Mode() != after.Mode() || hashScopedSurfaces(t, home) != surfaces {
				t.Fatal("rejected legacy selection changed harness or skill surfaces")
			}
		})
	}
}

func TestUpgradeTargetPlanHonorsExplicitFilter(t *testing.T) {
	root, _ := setupScopedUpgradeFixture(t)
	writeClaudeMarketplaceManifest(t, root)
	r := Runner{ClaudePluginCLI: forbiddenScopedClaudeCLI{t}}
	options, err := parseUpgradeArgs([]string{"--to", "cursor", "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := r.buildInstallDryRunPlan(options.installPlanOptions(), root, root, "0.6.0", filepath.Join(root, "dist"), detectInstallTools(), true, false, &projectPartPlan{InScope: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) != 1 || plan.Targets[0].Target != "cursor" {
		t.Fatalf("filtered target plan = %#v, want only Cursor", plan.Targets)
	}
}

func TestUpgradeExplicitClaudeRequiresCLI(t *testing.T) {
	options, err := parseUpgradeArgs([]string{"--to", "claude-code"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = (Runner{}).selectUpgradeTargets(options, nil, false, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "not on PATH") {
		t.Fatalf("error = %v, want missing Claude CLI guidance", err)
	}
}

func TestUpgradePlanPreservesResolvedEmptyTargetSelection(t *testing.T) {
	root, _ := setupScopedUpgradeFixture(t)
	r := Runner{ClaudePluginCLI: forbiddenScopedClaudeCLI{t}}
	options := installOptions{upgrade: true, resolvedUpgradeTargets: []string{}}
	plan, err := r.buildInstallDryRunPlan(options, root, root, "0.6.0", filepath.Join(root, "dist"), detectInstallTools(), true, false, &projectPartPlan{InScope: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) != 0 {
		t.Fatalf("empty target selection expanded to %#v", plan.Targets)
	}
}

func TestUpgradeFollowUpPreservesResolvedTargetSelection(t *testing.T) {
	for _, test := range []struct {
		name    string
		targets []string
		want    string
	}{
		{name: "subset", targets: []string{"cursor"}, want: "loaf upgrade --to cursor"},
		{name: "including Claude", targets: []string{"cursor", "claude-code"}, want: "loaf upgrade --to cursor,claude-code"},
		{name: "none", targets: []string{}, want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := installOptions{upgrade: true, command: "upgrade", resolvedUpgradeTargets: test.targets}
			plan := installDryRunPlan{Skills: []artifactPlanDecision{{Kind: "skill", ID: "skill:foundations", Action: planActionUpdate}}}
			commands := installPlanFollowUpCommands(options, plan, false)
			if test.want == "" {
				if len(commands) != 0 {
					t.Fatalf("empty selection follow-up = %q, want no unfiltered apply suggestion", commands)
				}
			} else if len(commands) != 1 || commands[0] != test.want {
				t.Fatalf("follow-up = %q, want %q", commands, test.want)
			}
		})
	}
}

func TestUpgradeInteractiveDryRunAdvertisesOnlySelectedTargets(t *testing.T) {
	for _, selection := range []string{"cursor", "none"} {
		t.Run(selection, func(t *testing.T) {
			root, _ := setupScopedUpgradeFixture(t)
			var output bytes.Buffer
			err := (Runner{Stdout: &output, Stdin: strings.NewReader(selection + "\n"), WorkingDir: root, Executable: distributionFixtureExecutable(root)}).Run([]string{"upgrade", "-i", "--dry-run"})
			if err != nil {
				t.Fatalf("interactive preview: %v\n%s", err, output.String())
			}
			if selection == "none" {
				if strings.Contains(output.String(), "Apply with") {
					t.Fatalf("empty selection advertised an apply command:\n%s", output.String())
				}
			} else if !strings.Contains(output.String(), "loaf upgrade --to cursor") {
				t.Fatalf("preview lost selected target in follow-up:\n%s", output.String())
			}
		})
	}
}
