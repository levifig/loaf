package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestReviewScopedCodexPolicyActuallyUpdates(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	old := legacyPinnedCodexBasicRules("/opt/homebrew/bin/loaf")
	writeInstallFile(t, f.dest(), old)
	guidance := legacyPinnedCodexJournalGuidance("/opt/homebrew/bin/loaf")
	writeInstallFile(t, filepath.Join(f.codexHome, "AGENTS.md"), guidance)
	manifest := codexManagedRuleManifest{Version: 1}
	manifest.set(codexJournalRuleRelativePath, sha256Bytes([]byte(old)))
	manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte(guidance)))
	if err := writeCodexManagedRuleManifest(f.manifest(), manifest); err != nil {
		t.Fatal(err)
	}
	options := f.options(false, true)
	options.ConfigDir = f.codexHome
	decisions := []artifactPlanDecision{
		{ID: "codex-rule:loaf.rules", Kind: "codex-rule", Destination: f.dest(), Action: planActionUpdate},
		{ID: "codex-rule:AGENTS.md", Kind: "codex-guidance", Destination: filepath.Join(f.codexHome, "AGENTS.md"), Action: planActionRetire},
	}
	if err := publishSelectedCodexPolicyArtifacts(options, decisions); err != nil {
		t.Fatal(err)
	}
	got := readFileString(t, f.dest())
	if got == old {
		t.Fatal("scoped policy apply returned success but left the absolute-pin rule unchanged")
	}
	if strings.Contains(got, "/opt/homebrew/bin/loaf") || !strings.Contains(got, strconv.Quote(codexPathLoafCommandName)+`, "journal", "log", "--execpolicy-safe"`) {
		t.Fatalf("published rule = %q, want PATH loaf prefixes", got)
	}
	gotGuidance := readFileString(t, filepath.Join(f.codexHome, "AGENTS.md"))
	if gotGuidance != "" {
		t.Fatalf("guidance = %q, want old block removed", gotGuidance)
	}
	recorded := readCodexRuleManifestTest(t, f.manifest())
	if digest, ok := recorded.ownedDigest(codexJournalRuleRelativePath); !ok || digest != sha256Bytes([]byte(got)) {
		t.Fatalf("rule ownership digest = %s ok=%v, want published rule hash", digest, ok)
	}
	if _, ok := recorded.ownedDigest(codexJournalGuidanceRelativePath); ok {
		t.Fatal("retired guidance ownership retained")
	}
}

func TestReviewScopedCodexPolicyRunner(t *testing.T) {
	root, home, rulePath, old := setupScopedCodexPolicyFixture(t)
	var out strings.Builder
	runner := Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}
	args := []string{"upgrade", "--select", "codex/codex-rule:loaf.rules", "--select", "codex/codex-rule:AGENTS.md"}
	if err := runner.Run(args); err != nil {
		t.Fatal(err)
	}
	if readFileString(t, rulePath) == old {
		t.Fatalf("full scoped apply claimed update but retained pins: %s", out.String())
	}
	if !strings.Contains(out.String(), "update") {
		t.Fatalf("apply output = %s, want reported updates", out.String())
	}
	assertScopedCodexPolicyPublished(t, home)
}

func TestScopedCodexPolicyApplyConverges(t *testing.T) {
	root, home, rulePath, old := setupScopedCodexPolicyFixture(t)
	args := []string{"upgrade", "--select", "codex/codex-rule:loaf.rules", "--select", "codex/codex-rule:AGENTS.md"}
	runInstallCapture(t, root, args...)
	firstRule := readFileString(t, rulePath)
	if firstRule == old {
		t.Fatal("first scoped policy apply retained absolute pins")
	}
	assertScopedCodexPolicyPublished(t, home)
	firstGuidance := readFileString(t, filepath.Join(home, ".codex", "AGENTS.md"))
	firstManifest := readFileString(t, filepath.Join(home, ".codex", "rules", codexJournalRuleManifest))
	var out strings.Builder
	if err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(args); err != nil {
		t.Fatalf("convergence apply: %v\n%s", err, out.String())
	}
	if readFileString(t, rulePath) != firstRule || readFileString(t, filepath.Join(home, ".codex", "AGENTS.md")) != firstGuidance {
		t.Fatal("second scoped policy apply rewrote converged PATH loaf policy")
	}
	if readFileString(t, filepath.Join(home, ".codex", "rules", codexJournalRuleManifest)) != firstManifest {
		t.Fatal("second scoped policy apply rewrote a converged ownership manifest")
	}
}

func TestScopedCodexPolicyApplyRollsBackOnFault(t *testing.T) {
	root, home, rulePath, old := setupScopedCodexPolicyFixture(t)
	oldGuidance := readFileString(t, filepath.Join(home, ".codex", "AGENTS.md"))
	oldManifest := readFileString(t, filepath.Join(home, ".codex", "rules", codexJournalRuleManifest))
	t.Cleanup(func() { scopedApplyTestFault = nil })
	scopedApplyTestFault = func(phase string) error {
		if phase == "codex-policy" {
			return errors.New("injected scoped Codex policy failure")
		}
		return nil
	}
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{
		"upgrade", "--select", "codex/codex-rule:loaf.rules", "--select", "codex/codex-rule:AGENTS.md",
	})
	if err == nil || !strings.Contains(err.Error(), "injected scoped Codex policy failure") {
		t.Fatalf("error = %v\n%s, want injected policy failure", err, out.String())
	}
	if readFileString(t, rulePath) != old {
		t.Fatal("failed policy apply left loaf.rules mutated")
	}
	if readFileString(t, filepath.Join(home, ".codex", "AGENTS.md")) != oldGuidance {
		t.Fatal("failed policy apply left AGENTS.md mutated")
	}
	if readFileString(t, filepath.Join(home, ".codex", "rules", codexJournalRuleManifest)) != oldManifest {
		t.Fatal("failed policy apply left ownership manifest mutated")
	}
}

func TestReviewCodexHookApplyMissingRuntime(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	codexHome := filepath.Join(home, ".codex")
	t.Setenv("CODEX_HOME", codexHome)
	writeInstallFile(t, filepath.Join(codexHome, loafInstallMarkerFile), "0.5.0\n")
	installTestHookDistribution(t, root, "codex", testCodexHookCatalogSource())
	t.Setenv("PATH", t.TempDir())
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run([]string{"upgrade", "--select", "codex/hook:SessionStart/session-start-loaf"})
	if err == nil {
		body, readErr := os.ReadFile(filepath.Join(codexHome, "hooks.json"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		t.Fatalf("missing PATH runtime accepted; output=%s; installed hooks=%s", out.String(), body)
	}
	if !strings.Contains(err.Error(), "PATH loaf is missing") || !strings.Contains(err.Error(), "loaf journal context --from-hook --codex-hook") || !strings.Contains(err.Error(), "brew install loaf") {
		t.Fatalf("hook-only preflight error = %v, want journal leaf plus install guidance", err)
	}
	if _, statErr := os.Stat(filepath.Join(codexHome, "hooks.json")); !os.IsNotExist(statErr) {
		t.Fatalf("missing PATH loaf still wrote hooks.json: %v", statErr)
	}
}

func TestSelectedAdapterIDsForTargetOmitCodexPolicy(t *testing.T) {
	refs := []scopedArtifactRef{
		{Target: "codex", ID: "codex-rule:loaf.rules"},
		{Target: "codex", ID: "codex-rule:AGENTS.md"},
		{Target: "codex", ID: "plugin:plugins/hooks.ts"},
		{Target: "codex", ID: "hook:SessionStart/session-start-loaf"},
	}
	got := selectedAdapterIDsForTarget(refs, "codex")
	if len(got) != 1 || got[0] != "plugin:plugins/hooks.ts" {
		t.Fatalf("adapter IDs = %v, want only non-policy adapter rows", got)
	}
}

func TestScopedCodexPolicyFixtureWithoutInstalledLoaf(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root, home, _, _ := setupScopedCodexPolicyFixture(t)
	if _, err := trustedCodexJournalExecutable(root, nil); err != nil {
		t.Fatalf("fixture must supply its own trusted PATH runtime: %v", err)
	}
	runInstallCapture(t, root, "upgrade", "--select", "codex/codex-rule:loaf.rules", "--select", "codex/codex-rule:AGENTS.md")
	assertScopedCodexPolicyPublished(t, home)
}

func setupScopedCodexPolicyFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	root, home := setupScopedUpgradeFixture(t)
	// Policy validation rejects runtimes in the simulated project and OS temp
	// roots. Use a disposable workspace runtime so the real trust check runs
	// without depending on an installed loaf or weakening its forbidden roots.
	workspace, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, err := os.MkdirTemp(workspace, ".loaf-codex-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(runtimeRoot); err != nil {
			t.Errorf("remove fixture runtime: %v", err)
		}
	})
	writeCapablePathLoaf(t, runtimeRoot)
	t.Setenv("PATH", filepath.Join(runtimeRoot, "path-bin"))
	codexHome := filepath.Join(home, ".codex")
	t.Setenv("CODEX_HOME", codexHome)
	dist := filepath.Join(root, "dist", "codex")
	writeInstallFile(t, filepath.Join(codexHome, loafInstallMarkerFile), "0.5.0\n")
	writeInstallFile(t, filepath.Join(dist, ".codex", "rules", codexJournalRuleTemplateRelativePath), "# policy\n{{LOAF_BASIC_RULES}}\n")
	installTestHookDistribution(t, root, "codex", testCodexHookCatalogSource())
	writeInstallFile(t, filepath.Join(codexHome, targetInstallManifestFile), readFileString(t, filepath.Join(dist, targetBuildManifestFile)))
	old := legacyPinnedCodexBasicRules("/opt/homebrew/bin/loaf")
	oldGuidance := legacyPinnedCodexJournalGuidance("/opt/homebrew/bin/loaf")
	rulePath := filepath.Join(codexHome, "rules", codexJournalRuleRelativePath)
	writeInstallFile(t, rulePath, old)
	writeInstallFile(t, filepath.Join(codexHome, "AGENTS.md"), oldGuidance)
	manifest := codexManagedRuleManifest{Version: 1}
	manifest.set(codexJournalRuleRelativePath, sha256Bytes([]byte(old)))
	manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte(oldGuidance)))
	if err := writeCodexManagedRuleManifest(filepath.Join(codexHome, "rules", codexJournalRuleManifest), manifest); err != nil {
		t.Fatal(err)
	}
	return root, home, rulePath, old
}

func assertScopedCodexPolicyPublished(t *testing.T, home string) {
	t.Helper()
	rule := readFileString(t, filepath.Join(home, ".codex", "rules", codexJournalRuleRelativePath))
	if strings.Contains(rule, "/opt/homebrew/bin/loaf") || !strings.Contains(rule, strconv.Quote(codexPathLoafCommandName)+`, "journal", "log", "--execpolicy-safe"`) {
		t.Fatalf("published rule = %q, want PATH loaf prefixes", rule)
	}
	guidance := readFileString(t, filepath.Join(home, ".codex", "AGENTS.md"))
	if guidance != "" {
		t.Fatalf("guidance = %q, want old block removed", guidance)
	}
	recorded := readCodexRuleManifestTest(t, filepath.Join(home, ".codex", "rules", codexJournalRuleManifest))
	if digest, ok := recorded.ownedDigest(codexJournalRuleRelativePath); !ok || digest != sha256Bytes([]byte(rule)) {
		t.Fatalf("rule ownership digest = %s ok=%v, want published rule hash", digest, ok)
	}
	if _, ok := recorded.ownedDigest(codexJournalGuidanceRelativePath); ok {
		t.Fatal("retired guidance ownership retained")
	}
}
