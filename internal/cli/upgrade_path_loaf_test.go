package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRequiredPathLoafCheckHooksFromSelectedHookIDsAndSkillDiffs(t *testing.T) {
	ids := requiredPathLoafCheckHooks(installDryRunPlan{
		Skills: []artifactPlanDecision{{
			ID:     "skill:infrastructure-management",
			Action: planActionUpdate,
			Diff:   "+| Infra safety hook | `loaf check --hook validate-infra-safety` |\n",
		}},
		Targets: []targetDistributionPlan{{
			Target: "cursor",
			Artifacts: []artifactPlanDecision{
				{ID: "hook:preToolUse/validate-sql-safety", Action: hookActionAdd},
				{ID: "hook:postToolUse/kb-staleness-nudge", Action: hookActionUpdate},
				{ID: "hook-file:hooks/post-tool/kb-staleness-nudge.sh", Action: planActionRetire},
				{ID: "plugin:plugins/hooks.ts", Action: planActionUpdate, Diff: "+command: 'loaf check --hook kb-staleness-nudge'\n"},
			},
		}},
	})
	joined := strings.Join(ids, ",")
	for _, want := range []string{"kb-staleness-nudge", "validate-infra-safety", "validate-sql-safety"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("required = %v, missing %s", ids, want)
		}
	}
}

func TestRequiredPathLoafCheckHooksSkipsPreserveAndRetireOnly(t *testing.T) {
	ids := requiredPathLoafCheckHooks(installDryRunPlan{
		Skills: []artifactPlanDecision{{
			ID:     "skill:foundations",
			Action: planActionPreserve,
			Diff:   "loaf check --hook validate-commit\n",
		}},
		Targets: []targetDistributionPlan{{
			Target: "cursor",
			Artifacts: []artifactPlanDecision{
				{ID: "hook-file:hooks/post-tool/kb-staleness-nudge.sh", Action: planActionRetire},
			},
		}},
	})
	if len(ids) != 0 {
		t.Fatalf("required = %v, want none for preserve/retire-only", ids)
	}
}

func TestReviewPathRequirementsIgnoreDeletedCommands(t *testing.T) {
	plan := installDryRunPlan{Skills: []artifactPlanDecision{{
		ID:     "skill:foundations",
		Action: planActionUpdate,
		Diff:   "--- old\n+++ new\n@@ -1,1 +1,1 @@\n-Run `loaf check --hook retired-hook`\n+Run the new command\n",
	}}}
	if got := requiredPathLoafCheckHooks(plan); len(got) != 0 {
		t.Fatalf("deleted text became runtime requirement: %v", got)
	}
}

func TestRequiredPathLoafCapabilitiesPreferDesiredOverDeletedDiff(t *testing.T) {
	req := requiredPathLoafCapabilities(installDryRunPlan{Skills: []artifactPlanDecision{{
		ID:      "skill:foundations",
		Action:  planActionUpdate,
		Diff:    "--- old\n+++ new\n-`loaf check --hook retired-hook`\n-`loaf check units 25 C K`\n+kept\n",
		Desired: "Use `loaf check commit-msg -` and `loaf check --hook validate-commit`.\n",
	}}})
	if strings.Join(req.hooks, ",") != "validate-commit" {
		t.Fatalf("hooks = %v, want validate-commit from desired content", req.hooks)
	}
	if strings.Join(req.operators, ",") != "commit-msg" {
		t.Fatalf("operators = %v, want commit-msg from desired content", req.operators)
	}
}

func TestParsePathLoafVersionReadsIdentityLine(t *testing.T) {
	got := parsePathLoafVersion("\n\x1b[1mloaf\x1b[0m 0.5.0+gtest (dev build)\n")
	if got != "0.5.0+gtest" {
		t.Fatalf("version = %q", got)
	}
}

func TestReviewCodexJournalSelectionRequiresPathLoaf(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	plan := installDryRunPlan{
		Skills: []artifactPlanDecision{{ID: "skill:journal", Action: planActionUpdate, Desired: codexJournalHookCommandTemplate}},
	}
	err := scopedPathLoafPreflightError(plan)
	if err == nil {
		t.Fatal("Codex journal context desired command accepted with no loaf on PATH")
	}
	if !strings.Contains(err.Error(), "PATH loaf is missing") || !strings.Contains(err.Error(), "loaf journal context --from-hook --codex-hook") || !strings.Contains(err.Error(), "brew install loaf") {
		t.Fatalf("journal preflight error = %v, want missing PATH loaf plus journal leaf guidance", err)
	}
}

func TestRequiredPathLoafCapabilitiesRecognizeJournalContext(t *testing.T) {
	req := requiredPathLoafCapabilities(installDryRunPlan{
		Targets: []targetDistributionPlan{{
			Target: "codex",
			Artifacts: []artifactPlanDecision{{
				ID:      "hook:SessionStart/session-start-loaf",
				Action:  hookActionAdd,
				Desired: `{"command":"` + codexJournalHookCommandTemplate + `"}`,
			}},
		}},
	})
	if strings.Join(req.journals, ",") != "journal context --from-hook --codex-hook" {
		t.Fatalf("journals = %v, want the generated Codex journal invocation", req.journals)
	}
}

func TestReviewPathPreflightJournalUsesBoundedHelp(t *testing.T) {
	originalLook, originalRun := lookPathLoaf, runPathLoaf
	t.Cleanup(func() {
		lookPathLoaf = originalLook
		runPathLoaf = originalRun
	})
	lookPathLoaf = func(string) (string, error) { return "/fixture/loaf", nil }
	runPathLoaf = func(_ string, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "version":
			return []byte("loaf 0.5.0+gtest"), nil
		case "journal context --help":
			return []byte("Usage: loaf journal context\n--from-hook\n--codex-hook\n"), nil
		default:
			t.Fatalf("unexpected non-journal probe: %v", args)
			return nil, nil
		}
	}
	plan := installDryRunPlan{Skills: []artifactPlanDecision{{ID: "skill:journal", Action: planActionUpdate, Desired: codexJournalHookCommandTemplate}}}
	if err := scopedPathLoafPreflightError(plan); err != nil {
		t.Fatalf("capable journal runtime rejected: %v", err)
	}
}

func TestReviewPathPreflightRejectsJournalHelpMissingCodexHook(t *testing.T) {
	originalLook, originalRun := lookPathLoaf, runPathLoaf
	t.Cleanup(func() {
		lookPathLoaf = originalLook
		runPathLoaf = originalRun
	})
	lookPathLoaf = func(string) (string, error) { return "/fixture/loaf", nil }
	runPathLoaf = func(_ string, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "version":
			return []byte("loaf 0.5.0+gtest"), nil
		case "journal context --help":
			return []byte("Usage: loaf journal context\n--from-hook\n"), nil
		default:
			t.Fatalf("unexpected probe: %v", args)
			return nil, nil
		}
	}
	plan := installDryRunPlan{Skills: []artifactPlanDecision{{ID: "skill:journal", Action: planActionUpdate, Desired: codexJournalHookCommandTemplate}}}
	err := scopedPathLoafPreflightError(plan)
	if err == nil || !strings.Contains(err.Error(), "loaf journal context --from-hook --codex-hook") || !strings.Contains(err.Error(), "brew upgrade loaf") {
		t.Fatalf("error = %v, want missing --codex-hook capability plus upgrade guidance", err)
	}
}

func TestPathLoafMissingAndUnsupportedErrorsNameInstallUpgrade(t *testing.T) {
	missing := pathLoafMissingError(pathLoafRequirements{hooks: []string{"validate-infra-safety"}}).Error()
	if !strings.Contains(missing, "PATH loaf is missing") || !strings.Contains(missing, "loaf check --hook validate-infra-safety") || !strings.Contains(missing, "brew install loaf") || !strings.Contains(missing, "will not copy a private binary") {
		t.Fatalf("missing error = %s", missing)
	}
	unsupported := pathLoafUnsupportedError(pathLoafProbe{Path: "/tmp/old/loaf", Version: "0.5.0+g40baf53.dirty"}, []string{"loaf check --hook validate-sql-safety"}).Error()
	if !strings.Contains(unsupported, "too old or lacks") || !strings.Contains(unsupported, "/tmp/old/loaf (0.5.0+g40baf53.dirty)") || !strings.Contains(unsupported, "brew upgrade loaf") {
		t.Fatalf("unsupported error = %s", unsupported)
	}
}

func TestProbePathLoafUsesLookPath(t *testing.T) {
	original := lookPathLoaf
	t.Cleanup(func() { lookPathLoaf = original })
	lookPathLoaf = func(string) (string, error) { return "", errors.New("not on path") }
	if _, err := probePathLoaf(); err == nil {
		t.Fatal("expected look-path failure")
	}
	if err := scopedPathLoafPreflightError(installDryRunPlan{
		Targets: []targetDistributionPlan{{
			Artifacts: []artifactPlanDecision{{ID: "hook:preToolUse/validate-infra-safety", Action: hookActionAdd}},
		}},
	}); err == nil || !strings.Contains(err.Error(), "PATH loaf is missing") {
		t.Fatalf("preflight error = %v", err)
	}
}

func TestReviewPathProbeRejectsBrokenRuntime(t *testing.T) {
	for _, message := range []string{"", "dyld: Library not loaded", "This worktree has unmigrated agentic state", "unsupported command"} {
		t.Run(message, func(t *testing.T) {
			originalLook, originalRun := lookPathLoaf, runPathLoaf
			t.Cleanup(func() {
				lookPathLoaf = originalLook
				runPathLoaf = originalRun
			})
			lookPathLoaf = func(string) (string, error) { return "/fixture/loaf", nil }
			runPathLoaf = func(string, ...string) ([]byte, error) { return []byte(message), errors.New("exit status 127") }
			plan := installDryRunPlan{Targets: []targetDistributionPlan{{Target: "cursor", Artifacts: []artifactPlanDecision{{ID: "hook:preToolUse/validate-sql-safety", Action: hookActionAdd}}}}}
			if err := scopedPathLoafPreflightError(plan); err == nil {
				t.Fatal("broken runtime accepted as compatible")
			}
		})
	}
}

func TestReviewPathPreflightChecksNativeOperatorRequirements(t *testing.T) {
	originalLook := lookPathLoaf
	t.Cleanup(func() { lookPathLoaf = originalLook })
	lookPathLoaf = func(string) (string, error) { return "", errors.New("missing PATH loaf") }
	plan := installDryRunPlan{Skills: []artifactPlanDecision{{ID: "skill:power-systems-modeling", Action: planActionUpdate, Diff: "+| Units | `loaf check units 25 C K` | Convert temperature |\n"}}}
	if err := scopedPathLoafPreflightError(plan); err == nil {
		t.Fatal("native operator requirement allowed with no loaf on PATH")
	}
}

func TestReviewPathBrokenRuntimeDoesNotApply(t *testing.T) {
	root, home := setupScopedUpgradeFixture(t)
	executable := filepath.Join(root, "path-bin", "loaf")
	writeInstallFile(t, executable, "#!/bin/sh\nexit 127\n")
	if err := os.Chmod(executable, 0755); err != nil {
		t.Fatal(err)
	}
	before := hashScopedSurfaces(t, home)
	var out strings.Builder
	err := (Runner{WorkingDir: root, Executable: distributionFixtureExecutable(root), Stdout: &out}).Run(append([]string{"upgrade"}, scopedUpgradeArgs()...))
	if err == nil {
		t.Fatalf("apply succeeded with unusable PATH runtime; surfaces changed=%v", before != hashScopedSurfaces(t, home))
	}
	if before != hashScopedSurfaces(t, home) {
		t.Fatal("failed preflight mutated surfaces")
	}
}

func TestPathLoafVersionIdentityRequiresSemver(t *testing.T) {
	if identity, ok := pathLoafVersionIdentity("\n\x1b[1mloaf\x1b[0m 0.5.0+gtest (dev build)\n"); !ok || identity != "0.5.0+gtest" {
		t.Fatalf("valid identity = %q ok=%v", identity, ok)
	}
	for _, output := range []string{"Usage: loaf <command>", "loaf unavailable", "loaf not-a-version", ""} {
		if identity, ok := pathLoafVersionIdentity(output); ok {
			t.Fatalf("identity %q accepted from %q", identity, output)
		}
	}
}

func TestIndependentProbeRejectsUnprovenVersion(t *testing.T) {
	for _, version := range []string{"Usage: loaf <command>", "loaf unavailable"} {
		t.Run(version, func(t *testing.T) {
			originalLook, originalRun := lookPathLoaf, runPathLoaf
			t.Cleanup(func() {
				lookPathLoaf = originalLook
				runPathLoaf = originalRun
			})
			lookPathLoaf = func(string) (string, error) { return "/fixture/loaf", nil }
			runPathLoaf = func(_ string, args ...string) ([]byte, error) {
				joined := strings.Join(args, " ")
				if joined == "version" || joined == "--version" {
					return []byte(version), nil
				}
				return []byte("Usage: loaf check --hook <id>\nHook id: validate-sql-safety\n"), nil
			}
			plan := installDryRunPlan{Targets: []targetDistributionPlan{{Target: "cursor", Artifacts: []artifactPlanDecision{{ID: "hook:preToolUse/validate-sql-safety", Action: hookActionAdd}}}}}
			if err := scopedPathLoafPreflightError(plan); err == nil {
				t.Fatalf("preflight accepted non-version response %q", version)
			}
		})
	}
}

func TestIndependentOlderCapableRuntimeUsesOnlyHelp(t *testing.T) {
	originalLook, originalRun := lookPathLoaf, runPathLoaf
	t.Cleanup(func() {
		lookPathLoaf = originalLook
		runPathLoaf = originalRun
	})
	lookPathLoaf = func(string) (string, error) { return "/fixture/loaf", nil }
	runPathLoaf = func(_ string, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "version":
			return []byte("loaf 0.1.0"), nil
		case "check --help":
			return []byte("Usage: loaf check --hook <id>\nHook id: validate-sql-safety\n"), nil
		case "check units --help":
			return []byte("Usage: loaf check units <value> <from> <to>"), nil
		default:
			t.Fatalf("unexpected non-help probe: %v", args)
			return nil, nil
		}
	}
	plan := installDryRunPlan{Skills: []artifactPlanDecision{{ID: "skill:power-systems-modeling", Action: planActionUpdate, Desired: "Use `loaf check units 25 C K`."}}}
	if err := scopedPathLoafPreflightError(plan); err != nil {
		t.Fatalf("older capable runtime rejected: %v", err)
	}
}

func TestPathLoafProbeFallsBackToVersionFlag(t *testing.T) {
	originalLook, originalRun := lookPathLoaf, runPathLoaf
	t.Cleanup(func() {
		lookPathLoaf = originalLook
		runPathLoaf = originalRun
	})
	lookPathLoaf = func(string) (string, error) { return "/fixture/loaf", nil }
	runPathLoaf = func(_ string, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "version":
			return []byte("Usage: loaf <command>"), nil
		case "--version":
			return []byte("loaf 0.5.0+gtest"), nil
		case "check --help":
			return []byte("Usage: loaf check --hook <id>\nHook id: validate-sql-safety\n"), nil
		default:
			t.Fatalf("unexpected probe: %v", args)
			return nil, nil
		}
	}
	plan := installDryRunPlan{Targets: []targetDistributionPlan{{Target: "cursor", Artifacts: []artifactPlanDecision{{ID: "hook:preToolUse/validate-sql-safety", Action: hookActionAdd}}}}}
	if err := scopedPathLoafPreflightError(plan); err != nil {
		t.Fatalf("valid --version identity rejected: %v", err)
	}
}

func TestPathLoafProbeBudgetIsTwoSeconds(t *testing.T) {
	if pathLoafProbeBudget != 2*time.Second {
		t.Fatalf("pathLoafProbeBudget = %s, want the two-second per-probe wall clock", pathLoafProbeBudget)
	}
}

func TestPathLoafProbeTimesOutWithinBudget(t *testing.T) {
	executable := writeHangingPathLoaf(t)
	started := time.Now()
	_, err := runPathLoafWithin(context.Background(), 40*time.Millisecond, executable)
	if !isPathLoafProbeTimeout(err) {
		t.Fatalf("error = %v, want a deadline timeout", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("probe took %s, want it abandoned at its 40ms budget", elapsed)
	}
}

func TestReviewPathProbePropagatesTimeout(t *testing.T) {
	executable := writeHangingPathLoaf(t)
	originalLook, originalRun := lookPathLoaf, runPathLoaf
	t.Cleanup(func() {
		lookPathLoaf = originalLook
		runPathLoaf = originalRun
	})
	lookPathLoaf = func(string) (string, error) { return executable, nil }
	runPathLoaf = func(path string, args ...string) ([]byte, error) {
		return runPathLoafWithin(context.Background(), 40*time.Millisecond, path, args...)
	}
	started := time.Now()
	plan := installDryRunPlan{Targets: []targetDistributionPlan{{Target: "cursor", Artifacts: []artifactPlanDecision{{ID: "hook:preToolUse/validate-sql-safety", Action: hookActionAdd}}}}}
	err := scopedPathLoafPreflightError(plan)
	if err == nil || !isPathLoafProbeTimeout(err) || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("preflight error = %v, want a propagated timeout", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("preflight took %s, want it abandoned at the probe budget", elapsed)
	}
}

func writeHangingPathLoaf(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "loaf")
	writeInstallFile(t, path, "#!/bin/sh\nexec sleep 30\n")
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIndependentTimeoutBoundsInheritedPipes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loaf")
	writeInstallFile(t, path, "#!/bin/sh\nsleep 4 &\nwait\n")
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err := runPathLoafWithin(context.Background(), pathLoafProbeBudget, path, "version")
	elapsed := time.Since(started)
	if !isPathLoafProbeTimeout(err) {
		t.Fatalf("expected timeout, got %v", err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("2s probe waited %s for descendant stdout/stderr pipes after shell cancellation", elapsed)
	}
}

func TestPathLoafProbeWaitDelayBoundsPostCancelIO(t *testing.T) {
	if pathLoafProbeWaitDelay <= 0 {
		t.Fatal("pathLoafProbeWaitDelay must be positive so CombinedOutput abandons descendant pipes")
	}
	if pathLoafProbeBudget+pathLoafProbeWaitDelay >= 3*time.Second {
		t.Fatalf("budget %s + WaitDelay %s must stay under the 3s inherited-pipe ceiling", pathLoafProbeBudget, pathLoafProbeWaitDelay)
	}
}
