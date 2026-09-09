package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestBuildNativeCodexCopiesJournalRuleArtifact(t *testing.T) {
	root := setupBuildCommandLoafRoot(t)
	seedNativeCodexBuildFixture(t, root)
	rule := "# generated Codex authority\n{{LOAF_BASIC_RULES}}\n"
	writeInstallFile(t, filepath.Join(root, "content", "codex", "rules", "loaf.rules.tmpl"), rule)

	if err := (Runner{WorkingDir: root}).Run([]string{"build", "--target", "codex"}); err != nil {
		t.Fatalf("build --target codex error = %v", err)
	}
	template := readBuildFileString(t, filepath.Join(root, "dist", "codex", ".codex", "rules", "loaf.rules.tmpl"))
	if template != rule || !strings.Contains(template, codexBasicRulesPlaceholder) {
		t.Fatalf("Codex rule template = %q, want exact unresolved source template", template)
	}
}

func TestBuildNativeCodexFailsWhenJournalRuleTemplateMissing(t *testing.T) {
	root := setupBuildCommandLoafRoot(t)
	seedNativeCodexBuildFixture(t, root)
	if err := os.Remove(filepath.Join(root, "content", "codex", "rules", "loaf.rules.tmpl")); err != nil {
		t.Fatalf("remove rule template: %v", err)
	}
	if err := (Runner{WorkingDir: root}).Run([]string{"build", "--target", "codex"}); err == nil || !strings.Contains(err.Error(), "journal rule template missing") {
		t.Fatalf("build error = %v, want missing-template failure", err)
	}
}

func TestRenderCodexJournalRuleRendersOnePathLoafPrefixPerBasicLeaf(t *testing.T) {
	template := "# policy\n" + codexBasicRulesPlaceholder + "\n"
	rendered, err := renderCodexJournalRule(template)
	if err != nil {
		t.Fatalf("render Codex policy: %v", err)
	}
	if strings.Contains(rendered, codexBasicRulesPlaceholder) || strings.Contains(rendered, codexJournalExecutablePlaceholder) {
		t.Fatalf("rendered policy = %q, want no unresolved placeholders", rendered)
	}
	if strings.Contains(rendered, "/opt/") || strings.Contains(rendered, "'/") {
		t.Fatalf("rendered policy = %q, want PATH loaf not an absolute pin", rendered)
	}
	for _, prefix := range BasicCommandAuthorityPrefixes() {
		needle := "pattern = [\"loaf\""
		for _, token := range prefix {
			needle += ", " + fmt.Sprintf("%q", token)
		}
		if !strings.Contains(rendered, needle+"]") {
			t.Fatalf("rendered policy missing prefix %v", prefix)
		}
	}
	if got := strings.Count(rendered, "prefix_rule("); got != len(BasicCommandAuthorityPrefixes()) {
		t.Fatalf("rendered prefix_rule count = %d, want %d", got, len(BasicCommandAuthorityPrefixes()))
	}
}

func TestParseInstallArgsCodexBasicCommandsRequiresCodexTarget(t *testing.T) {
	options, err := parseInstallArgs([]string{"--to", "codex", "--codex-basic-commands"})
	if err != nil {
		t.Fatalf("parse Codex opt-in = %v", err)
	}
	if !options.codexBasicCommands || options.target != "codex" {
		t.Fatalf("options = %#v, want Codex opt-in", options)
	}
	if _, err := parseInstallArgs([]string{"--to", "cursor", "--codex-basic-commands"}); err == nil || !strings.Contains(err.Error(), "requires --to codex or --to all") {
		t.Fatalf("invalid target error = %v, want explicit target refusal", err)
	}
}

func TestValidateCodexJournalExecutableRejectsMissingAndDisposablePaths(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		if err := validateCodexJournalExecutable("/project"); err == nil || !strings.Contains(err.Error(), "not on PATH") {
			t.Fatalf("trust validation error = %v, want missing executable refusal", err)
		}
	})
	t.Run("temporary", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "loaf")
		writeInstallFile(t, path, "#!/bin/sh\nexit 0\n")
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatalf("chmod fake loaf: %v", err)
		}
		t.Setenv("PATH", dir)
		if err := validateCodexJournalExecutable("/project"); err == nil || !strings.Contains(err.Error(), "forbidden path") {
			t.Fatalf("trust validation error = %v, want disposable-path refusal", err)
		}
	})
}

func TestTrustedCodexJournalExecutableRendersEntrypointNotCanonicalTarget(t *testing.T) {
	root := realpath(t, t.TempDir())
	target := filepath.Join(root, "cellar-1.0", "loaf")
	writeCodexExecutableFixture(t, target)
	entrypoint := filepath.Join(root, "bin", "loaf")
	symlinkCodexEntrypoint(t, target, entrypoint)
	got, err := trustedCodexJournalExecutable(filepath.Join(root, "project"), codexEntrypointOperations(root, entrypoint))
	if err != nil {
		t.Fatalf("trusted executable error = %v", err)
	}
	if got != entrypoint || !filepath.IsAbs(got) {
		t.Fatalf("trusted executable = %q, want absolute stable entrypoint %q", got, entrypoint)
	}
	if strings.Contains(got, "cellar-1.0") {
		t.Fatalf("trusted executable = %q, want no canonicalized segment", got)
	}
}

func TestTrustedCodexJournalExecutablePlainFileRenderEqualsCanonical(t *testing.T) {
	root := realpath(t, t.TempDir())
	path := filepath.Join(root, "trusted-bin", "loaf")
	writeCodexExecutableFixture(t, path)
	got, err := trustedCodexJournalExecutable(filepath.Join(root, "project"), codexEntrypointOperations(root, path))
	if err != nil {
		t.Fatalf("trusted executable error = %v", err)
	}
	if got != path {
		t.Fatalf("trusted executable = %q, want plain-file path %q where render equals canonical", got, path)
	}
}

func TestTrustedCodexJournalExecutableRejectsForbiddenTargetBehindSymlink(t *testing.T) {
	root := realpath(t, t.TempDir())
	project := filepath.Join(root, "project")
	target := filepath.Join(project, "bin", "loaf")
	writeCodexExecutableFixture(t, target)
	entrypoint := filepath.Join(root, "bin", "loaf")
	symlinkCodexEntrypoint(t, target, entrypoint)
	if _, err := trustedCodexJournalExecutable(project, codexEntrypointOperations(root, entrypoint)); err == nil || !strings.Contains(err.Error(), "forbidden path") {
		t.Fatalf("trust validation error = %v, want forbidden-target refusal", err)
	}
}

func TestTrustedCodexJournalExecutableRejectsForbiddenEntrypointWithLegitimateTarget(t *testing.T) {
	// The mirror of the forbidden-target case: the PATH entrypoint itself sits
	// inside a forbidden root while its canonical target is outside. Only the
	// render path is disposable here, and the render path is what rendered
	// policy ultimately trusts, so canonical-only validation would pass it.
	root := realpath(t, t.TempDir())
	project := filepath.Join(root, "project")
	target := filepath.Join(root, "cellar-1.0", "loaf")
	writeCodexExecutableFixture(t, target)
	entrypoint := filepath.Join(project, "bin", "loaf")
	symlinkCodexEntrypoint(t, target, entrypoint)
	_, err := trustedCodexJournalExecutable(project, codexEntrypointOperations(root, entrypoint))
	if err == nil || !strings.Contains(err.Error(), "forbidden path") {
		t.Fatalf("trust validation error = %v, want forbidden-entrypoint refusal", err)
	}
	if !strings.Contains(err.Error(), entrypoint) {
		t.Fatalf("trust validation error = %v, want the rejected entrypoint %q named", err, entrypoint)
	}
	if strings.Contains(err.Error(), "cellar-1.0") {
		t.Fatalf("trust validation error = %v, want no legitimate canonical target blamed", err)
	}
}

func TestTrustedCodexJournalExecutableRejectsGuidanceCharactersInEitherPath(t *testing.T) {
	t.Run("entrypoint", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		target := filepath.Join(root, "cellar-1.0", "loaf")
		writeCodexExecutableFixture(t, target)
		entrypoint := filepath.Join(root, "bin`tick", "loaf")
		symlinkCodexEntrypoint(t, target, entrypoint)
		if _, err := trustedCodexJournalExecutable(filepath.Join(root, "project"), codexEntrypointOperations(root, entrypoint)); err == nil || !strings.Contains(err.Error(), "unsupported guidance characters") {
			t.Fatalf("trust validation error = %v, want guidance-character refusal for entrypoint", err)
		}
	})
	t.Run("canonical target", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		target := filepath.Join(root, "cellar`1.0", "loaf")
		writeCodexExecutableFixture(t, target)
		entrypoint := filepath.Join(root, "bin", "loaf")
		symlinkCodexEntrypoint(t, target, entrypoint)
		if _, err := trustedCodexJournalExecutable(filepath.Join(root, "project"), codexEntrypointOperations(root, entrypoint)); err == nil || !strings.Contains(err.Error(), "unsupported guidance characters") {
			t.Fatalf("trust validation error = %v, want guidance-character refusal for canonical target", err)
		}
	})
}

func TestInstallCodexJournalRuleRendersSymlinkedEntrypointAcrossSurfaces(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	target := filepath.Join(fixture.root, "cellar-1.0", "loaf")
	writeCodexExecutableFixture(t, target)
	entrypoint := filepath.Join(fixture.root, "bin", "loaf")
	symlinkCodexEntrypoint(t, target, entrypoint)
	operations := codexEntrypointOperations(fixture.root, entrypoint)
	if err := installCodexJournalRuleWithOperations(fixture.options(true, false), fixture.codexHome, operations); err != nil {
		t.Fatalf("install with symlinked entrypoint: %v", err)
	}
	rule, err := os.ReadFile(fixture.dest())
	if err != nil {
		t.Fatalf("read rendered rule: %v", err)
	}
	if !strings.Contains(string(rule), strconv.Quote(codexPathLoafCommandName)) || strings.Contains(string(rule), entrypoint) || strings.Contains(string(rule), "cellar-1.0") {
		t.Fatalf("rendered rule = %q, want PATH loaf prefixes without an absolute pin", rule)
	}
	assertInstallPathMissing(t, filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath))
	if _, err := trustedCodexJournalExecutable(filepath.Join(fixture.root, "project"), operations); err != nil {
		t.Fatalf("trusted executable error = %v", err)
	}
	template := json.RawMessage(`{"matcher":"startup|resume|clear|compact","hooks":[{"type":"command","command":"{{LOAF_EXECUTABLE}} journal context --from-hook --codex-hook"}]}`)
	entry, err := renderCodexHookExecutableForOS(template, "darwin")
	if err != nil {
		t.Fatalf("render hook entry with leftover placeholder: %v", err)
	}
	if !strings.Contains(string(entry), codexJournalHookCommandTemplate) || strings.Contains(string(entry), entrypoint) || strings.Contains(string(entry), "cellar-1.0") {
		t.Fatalf("rendered hook entry = %q, want PATH loaf command without an absolute pin", entry)
	}
}

func TestTrustedCodexJournalExecutableSurvivesEntrypointRetarget(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	oldTarget := filepath.Join(fixture.root, "cellar-1.0", "loaf")
	writeCodexExecutableFixture(t, oldTarget)
	entrypoint := filepath.Join(fixture.root, "bin", "loaf")
	symlinkCodexEntrypoint(t, oldTarget, entrypoint)
	operations := codexEntrypointOperations(fixture.root, entrypoint)
	rendered, err := trustedCodexJournalExecutable(filepath.Join(fixture.root, "project"), operations)
	if err != nil {
		t.Fatalf("trusted executable error = %v", err)
	}
	if err := installCodexJournalRuleWithOperations(fixture.options(true, false), fixture.codexHome, operations); err != nil {
		t.Fatalf("install with symlinked entrypoint: %v", err)
	}
	installedRule, err := os.ReadFile(fixture.dest())
	if err != nil {
		t.Fatalf("read installed rule: %v", err)
	}

	// Simulate the Homebrew upgrade: the versioned target moves, the stable
	// entrypoint is repointed, and the old Cellar directory disappears.
	newTarget := filepath.Join(fixture.root, "cellar-2.0", "loaf")
	writeCodexExecutableFixture(t, newTarget)
	if err := os.Remove(entrypoint); err != nil {
		t.Fatalf("remove entrypoint symlink: %v", err)
	}
	symlinkCodexEntrypoint(t, newTarget, entrypoint)
	if err := os.RemoveAll(filepath.Join(fixture.root, "cellar-1.0")); err != nil {
		t.Fatalf("remove stale versioned target: %v", err)
	}

	if _, err := os.Stat(rendered); err != nil {
		t.Fatalf("previously rendered executable path stat = %v, want upgrade survival", err)
	}
	if _, err := os.Stat(oldTarget); !os.IsNotExist(err) {
		t.Fatalf("stale canonical target stat = %v, want removal proving a canonical pin would strand", err)
	}
	// The owned upgrade converges to byte-identical content: nothing to rewrite.
	if err := installCodexJournalRuleWithOperations(fixture.options(false, true), fixture.codexHome, operations); err != nil {
		t.Fatalf("upgrade after retarget: %v", err)
	}
	upgradedRule, err := os.ReadFile(fixture.dest())
	if err != nil {
		t.Fatalf("read upgraded rule: %v", err)
	}
	if string(upgradedRule) != string(installedRule) {
		t.Fatalf("upgraded rule = %q, want previously rendered content untouched %q", upgradedRule, installedRule)
	}
}

func TestCodexJournalRuleExecpolicyClassification(t *testing.T) {
	codex, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("codex command is not installed; execpolicy classification is not verified")
	}
	rulePath := filepath.Join(realpath(t, t.TempDir()), "loaf.rules")
	body, err := os.ReadFile(filepath.Join("..", "..", "content", "codex", "rules", "loaf.rules.tmpl"))
	if err != nil {
		t.Fatalf("read source rule: %v", err)
	}
	rendered, err := renderCodexJournalRule(string(body))
	if err != nil {
		t.Fatalf("render Codex rule: %v", err)
	}
	writeFile(t, rulePath, rendered)
	trusted := "loaf"
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "manual", args: []string{trusted, "journal", "log", "--execpolicy-safe", "decision(scope): message"}, want: "allow"},
		{name: "ordinary-log", args: []string{trusted, "journal", "log", "decision(scope): message"}, want: ""},
		{name: "from-hook", args: []string{trusted, "journal", "log", "--execpolicy-safe", "--from-hook"}, want: "allow"},
		{name: "detect-linear", args: []string{trusted, "journal", "log", "--execpolicy-safe", "--detect-linear"}, want: "allow"},
		{name: "namespace-only", args: []string{"loaf"}, want: ""},
		{name: "absolute-pin", args: []string{"/usr/local/bin/loaf", "journal", "log", "--execpolicy-safe", "decision(scope): message"}, want: ""},
		{name: "alternate-absolute", args: []string{"/usr/bin/loaf", "journal", "log", "--execpolicy-safe", "decision(scope): message"}, want: ""},
		{name: "unsafed-log", args: []string{"loaf", "journal", "log", "decision(scope): message"}, want: ""},
		{name: "journal", args: []string{trusted, "journal"}, want: ""},
		{name: "defer", args: []string{trusted, "journal", "defer", "later"}, want: "allow"},
		{name: "state", args: []string{trusted, "state", "status"}, want: "allow"},
		{name: "task", args: []string{trusted, "task", "list"}, want: "allow"},
		{name: "idea", args: []string{trusted, "idea", "list"}, want: "allow"},
		{name: "state-export-all", args: []string{trusted, "state", "export", "all", "--format", "json"}, want: "allow"},
		{name: "state-export-unknown", args: []string{trusted, "state", "export", "unknown", "--format", "json"}, want: ""},
		{name: "report-generate-triage", args: []string{trusted, "report", "generate", "triage", "--format", "markdown"}, want: "allow"},
		{name: "report-generate-unknown", args: []string{trusted, "report", "generate", "unknown", "--format", "markdown"}, want: ""},
		{name: "unsafe-report-body-file", args: []string{trusted, "report", "create", "report", "--body-file", "/etc/passwd"}, want: ""},
		{name: "unsafe-finding-import", args: []string{trusted, "finding", "import-json", "--report", "report", "/etc/passwd"}, want: ""},
		{name: "unsafe-spec-body-file", args: []string{trusted, "spec", "new", "spec", "--body-file", "/etc/passwd"}, want: ""},
		{name: "unsafe-change-path", args: []string{trusted, "change", "check", "/outside/change.md"}, want: ""},
		{name: "unsafe-state-doctor", args: []string{trusted, "state", "doctor", "--fix"}, want: ""},
		{name: "unsafe-release", args: []string{trusted, "release"}, want: ""},
		{name: "unsafe-spec-finalize", args: []string{trusted, "spec", "finalize", "SPEC-001"}, want: ""},
		{name: "env-wrapper", args: []string{"env", "LOAF_DB=/outside/target.sqlite", trusted, "journal", "log", "--execpolicy-safe", "decision(scope): message"}, want: ""},
		{name: "shell-wrapper", args: []string{"sh", "-c", trusted + " journal log --execpolicy-safe 'decision(scope): message'"}, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmdArgs := append([]string{"execpolicy", "check", "--rules", rulePath}, tc.args...)
			output, runErr := exec.Command(codex, cmdArgs...).CombinedOutput()
			if runErr != nil {
				t.Fatalf("codex execpolicy check error = %v\n%s", runErr, output)
			}
			var result struct {
				Decision     string `json:"decision"`
				MatchedRules []any  `json:"matchedRules"`
			}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("decode execpolicy output: %v\n%s", err, output)
			}
			if tc.want == "allow" {
				if result.Decision != "allow" || len(result.MatchedRules) == 0 {
					t.Fatalf("result = %s, want an allowed match", output)
				}
			} else if result.Decision == "allow" || len(result.MatchedRules) != 0 {
				t.Fatalf("result = %s, want no matching allow rule", output)
			}
		})
	}
}

func TestInstallCodexJournalRuleFirstInstallRequiresOptIn(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	if err := installCodexJournalRule(fixture.options(false, false), fixture.codexHome); err != nil {
		t.Fatalf("install without opt-in error = %v", err)
	}
	assertInstallPathMissing(t, fixture.dest())
	assertInstallPathMissing(t, fixture.manifest())
}

func TestInstallCodexJournalRuleExplicitOptInOwnsAndUpgrades(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	if err := fixture.install(t, true, false); err != nil {
		t.Fatalf("explicit install error = %v", err)
	}
	assertInstallFile(t, fixture.dest(), fixture.renderedBody())
	manifest := readCodexRuleManifestTest(t, fixture.manifest())
	if digest, ok := manifest.ownedDigest(codexJournalRuleRelativePath); !ok || digest != sha256Bytes([]byte(fixture.renderedBody())) {
		t.Fatalf("manifest = %#v, want source digest ownership", manifest)
	}

	updated := fixture.sourceBody() + "# updated\n"
	writeFile(t, fixture.source(), updated)
	if err := fixture.install(t, false, true); err != nil {
		t.Fatalf("owned upgrade error = %v", err)
	}
	expected, err := renderCodexJournalRule(updated)
	if err != nil {
		t.Fatalf("render updated rule: %v", err)
	}
	assertInstallFile(t, fixture.dest(), expected)
}

func TestInstallCodexJournalRulePreservesUnrelatedRulesAndRejectsConflicts(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	other := filepath.Join(fixture.codexHome, "rules", "other.rules")
	writeInstallFile(t, other, "prefix_rule(pattern=[\"git\"], decision=\"allow\")\n")
	writeInstallFile(t, fixture.dest(), "user-owned\n")
	if err := fixture.install(t, true, false); err == nil || !strings.Contains(err.Error(), "unowned") {
		t.Fatalf("unowned conflict error = %v, want refusal", err)
	}
	assertInstallFile(t, other, "prefix_rule(pattern=[\"git\"], decision=\"allow\")\n")
	assertInstallFile(t, fixture.dest(), "user-owned\n")
}

func TestInstallCodexJournalRuleRejectsModifiedOwnedAndRemovesStaleSafely(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	if err := fixture.install(t, true, false); err != nil {
		t.Fatalf("explicit install error = %v", err)
	}
	writeInstallFile(t, fixture.dest(), "locally modified\n")
	if err := fixture.install(t, false, true); err == nil || !strings.Contains(err.Error(), "modified") {
		t.Fatalf("modified owned error = %v, want refusal", err)
	}
	writeInstallFile(t, fixture.dest(), fixture.renderedBody())
	if err := os.Remove(fixture.source()); err != nil {
		t.Fatalf("remove generated source: %v", err)
	}
	if err := fixture.install(t, false, true); err != nil {
		t.Fatalf("stale owned removal error = %v", err)
	}
	assertInstallPathMissing(t, fixture.dest())
	manifest := readCodexRuleManifestTest(t, fixture.manifest())
	if _, ok := manifest.ownedDigest(codexJournalRuleRelativePath); ok {
		t.Fatalf("manifest = %#v, want stale ownership removed", manifest)
	}
}

func TestInstallCodexJournalRuleDoesNotInferOwnershipFromMarker(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	writeInstallFile(t, filepath.Join(fixture.options(false, true).ConfigDir, loafInstallMarkerFile), "9.9.9\n")
	writeInstallFile(t, fixture.dest(), "user-owned\n")
	if err := installCodexJournalRule(fixture.options(false, true), fixture.codexHome); err != nil {
		t.Fatalf("upgrade with marker-only install error = %v", err)
	}
	assertInstallFile(t, fixture.dest(), "user-owned\n")
}

func TestInstallCodexJournalRuleRecoversInterruptedOwnershipWrites(t *testing.T) {
	t.Run("adopts exact generated body after first install interruption", func(t *testing.T) {
		fixture := newCodexRuleInstallFixture(t)
		writeInstallFile(t, fixture.dest(), fixture.renderedBody())
		if err := fixture.install(t, true, false); err != nil {
			t.Fatalf("adopt exact generated body: %v", err)
		}
		manifest := readCodexRuleManifestTest(t, fixture.manifest())
		if digest, ok := manifest.ownedDigest(codexJournalRuleRelativePath); !ok || digest != sha256Bytes([]byte(fixture.renderedBody())) {
			t.Fatalf("manifest = %#v, want adopted generated digest", manifest)
		}
	})

	t.Run("preserves unowned legacy guidance without adopting it", func(t *testing.T) {
		fixture := newCodexRuleInstallFixture(t)
		guidancePath := filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath)
		block := generateCodexJournalGuidance()
		writeInstallFile(t, guidancePath, block)
		if err := fixture.install(t, true, false); err != nil {
			t.Fatalf("adopt exact guidance: %v", err)
		}
		manifest := readCodexRuleManifestTest(t, fixture.manifest())
		if _, ok := manifest.ownedDigest(codexJournalGuidanceRelativePath); ok {
			t.Fatalf("manifest = %#v, must not adopt global guidance", manifest)
		}
		assertInstallFile(t, guidancePath, block)
	})

	t.Run("heals stale manifest after upgrade interruption", func(t *testing.T) {
		fixture := newCodexRuleInstallFixture(t)
		if err := fixture.install(t, true, false); err != nil {
			t.Fatalf("initial install: %v", err)
		}
		updated := fixture.sourceBody() + "# updated\n"
		writeFile(t, fixture.source(), updated)
		renderedUpdated, err := renderCodexJournalRule(updated)
		if err != nil {
			t.Fatalf("render updated rule: %v", err)
		}
		writeInstallFile(t, fixture.dest(), renderedUpdated)
		if err := fixture.install(t, false, true); err != nil {
			t.Fatalf("heal stale manifest: %v", err)
		}
		manifest := readCodexRuleManifestTest(t, fixture.manifest())
		if digest, ok := manifest.ownedDigest(codexJournalRuleRelativePath); !ok || digest != sha256Bytes([]byte(renderedUpdated)) {
			t.Fatalf("manifest = %#v, want healed updated digest", manifest)
		}
	})
}

func TestInstallCodexJournalRulePreservesUserGlobalInstructions(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	path := filepath.Join(f.codexHome, "AGENTS.md")
	body := "# My Codex instructions\n\nKeep this text.\n"
	writeInstallFile(t, path, body)
	if err := f.install(t, true, false); err != nil {
		t.Fatal(err)
	}
	if err := f.install(t, false, true); err != nil {
		t.Fatal(err)
	}
	assertInstallFile(t, path, body)
}

func TestInstallCodexJournalRulePreservesModifiedGlobalGuidance(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	seedLegacyCodexGuidance(t, f, "")
	path := filepath.Join(f.codexHome, "AGENTS.md")
	body := strings.Replace(readFileString(t, path), "exact command", "locally changed command", 1)
	writeInstallFile(t, path, body)
	if err := f.install(t, true, false); err != nil {
		t.Fatal(err)
	}
	assertInstallFile(t, path, body)
}

func TestInstallCodexJournalRuleRetiresRuleAndGuidanceWithoutResolvingExecutable(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	guidancePath := filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath)
	writeInstallFile(t, guidancePath, "# user guidance\n")
	if err := fixture.installWithExecutable(t, filepath.Join(fixture.root, "trusted-bin", "loaf-v1"), true, false); err != nil {
		t.Fatalf("explicit install: %v", err)
	}
	seedLegacyCodexGuidance(t, fixture, "# user guidance\n")
	if err := os.Remove(fixture.source()); err != nil {
		t.Fatalf("remove template: %v", err)
	}
	noPath := &codexRuleInstallOperations{lookPath: func(string) (string, error) { return "", fmt.Errorf("PATH intentionally unavailable") }}
	if err := installCodexJournalRuleWithOperations(fixture.options(false, true), fixture.codexHome, noPath); err != nil {
		t.Fatalf("stale removal: %v", err)
	}
	assertInstallPathMissing(t, fixture.dest())
	body, err := os.ReadFile(guidancePath)
	if err != nil {
		t.Fatalf("read retained user guidance: %v", err)
	}
	if strings.Contains(string(body), codexJournalGuidanceStart) || strings.Contains(string(body), "Codex Auto journal capture") || !strings.Contains(string(body), "# user guidance") {
		t.Fatalf("global guidance = %q, want managed block removed", body)
	}
	manifest := readCodexRuleManifestTest(t, fixture.manifest())
	if len(manifest.Files) != 0 {
		t.Fatalf("manifest = %#v, want no remaining ownership", manifest)
	}
}

func TestRetirementRemovesOwnedRuleButPreservesModifiedGuidance(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	if err := f.install(t, true, false); err != nil {
		t.Fatal(err)
	}
	seedLegacyCodexGuidance(t, f, "")
	path := filepath.Join(f.codexHome, "AGENTS.md")
	body := strings.Replace(readFileString(t, path), "exact command", "locally changed command", 1)
	writeInstallFile(t, path, body)
	if err := os.Remove(f.source()); err != nil {
		t.Fatal(err)
	}
	noPath := &codexRuleInstallOperations{lookPath: func(string) (string, error) { return "", fmt.Errorf("PATH unavailable") }}
	if err := installCodexJournalRuleWithOperations(f.options(false, true), f.codexHome, noPath); err != nil {
		t.Fatal(err)
	}
	assertInstallPathMissing(t, f.dest())
	assertInstallFile(t, path, body)
	manifest := readCodexRuleManifestTest(t, f.manifest())
	if _, owned := manifest.ownedDigest(codexJournalRuleRelativePath); owned {
		t.Fatal("rule ownership retained")
	}
	if _, owned := manifest.ownedDigest(codexJournalGuidanceRelativePath); !owned {
		t.Fatal("preserved guidance record lost")
	}
}

func TestRetirementWithGuidanceOnlyOwnershipPreservesUserRuleBytes(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	if err := fixture.installWithExecutable(t, filepath.Join(fixture.root, "trusted-bin", "loaf-v1"), true, false); err != nil {
		t.Fatalf("explicit install: %v", err)
	}
	userRule := "# user-owned rule\nprefix_rule(pattern=[\"git\"], decision=\"allow\")\n"
	writeInstallFile(t, fixture.dest(), userRule)
	seedLegacyCodexGuidance(t, fixture, "# user guidance\n")
	manifest := readCodexRuleManifestTest(t, fixture.manifest())
	manifest.remove(codexJournalRuleRelativePath)
	if err := writeCodexManagedRuleManifest(fixture.manifest(), manifest); err != nil {
		t.Fatalf("write guidance-only manifest: %v", err)
	}
	if err := os.Remove(fixture.source()); err != nil {
		t.Fatalf("remove template: %v", err)
	}
	noPath := &codexRuleInstallOperations{lookPath: func(string) (string, error) { return "", fmt.Errorf("PATH intentionally unavailable") }}
	if err := installCodexJournalRuleWithOperations(fixture.options(false, true), fixture.codexHome, noPath); err != nil {
		t.Fatalf("guidance-only retirement: %v", err)
	}
	assertInstallFile(t, fixture.dest(), userRule)
}

func TestCodexJournalGuidanceEditsPreserveUserBytesExactly(t *testing.T) {
	block := generateCodexJournalGuidance()
	for _, original := range []string{
		"",
		"prefix \t\n\n  \n",
		"prefix \t  ",
		"\n\nprefix\n\nsuffix \t\n",
	} {
		appended := appendCodexJournalGuidance(original, block)
		if !strings.HasPrefix(appended, original) {
			t.Fatalf("append changed user prefix: original=%q appended=%q", original, appended)
		}
		r, ok := findCodexJournalGuidance(appended)
		if !ok {
			t.Fatalf("appended guidance missing managed range: %q", appended)
		}
		replaced := replaceCodexJournalGuidance(appended, r, generateCodexJournalGuidance())
		if !strings.HasPrefix(replaced, original) {
			t.Fatalf("replace changed user bytes: original=%q replaced=%q", original, replaced)
		}
		r, ok = findCodexJournalGuidance(replaced)
		if !ok {
			t.Fatalf("replaced guidance missing managed range: %q", replaced)
		}
		removed := removeCodexJournalGuidance(replaced, r)
		if removed != original {
			t.Fatalf("remove changed user bytes: original=%q removed=%q", original, removed)
		}
	}
}

func TestRetirementPreservesPreexistingEmptyGuidanceFile(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	guidancePath := filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath)
	writeInstallFile(t, guidancePath, "")
	if err := fixture.installWithExecutable(t, filepath.Join(fixture.root, "trusted-bin", "loaf-v1"), true, false); err != nil {
		t.Fatalf("explicit install: %v", err)
	}
	seedLegacyCodexGuidance(t, fixture, "")
	if err := os.Remove(fixture.source()); err != nil {
		t.Fatalf("remove template: %v", err)
	}
	noPath := &codexRuleInstallOperations{lookPath: func(string) (string, error) { return "", fmt.Errorf("PATH intentionally unavailable") }}
	if err := installCodexJournalRuleWithOperations(fixture.options(false, true), fixture.codexHome, noPath); err != nil {
		t.Fatalf("retirement: %v", err)
	}
	body, err := os.ReadFile(guidancePath)
	if err != nil {
		t.Fatalf("read preserved empty guidance: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("preserved empty guidance = %q, want empty", body)
	}
}

func TestInstallCodexJournalRuleRetiresOrphanedOwnedGuidance(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	guidancePath := filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath)
	writeInstallFile(t, guidancePath, "# user guidance\n")
	if err := fixture.installWithExecutable(t, filepath.Join(fixture.root, "trusted-bin", "loaf-v1"), true, false); err != nil {
		t.Fatalf("explicit install: %v", err)
	}
	seedLegacyCodexGuidance(t, fixture, "# user guidance\n")
	manifest := readCodexRuleManifestTest(t, fixture.manifest())
	manifest.remove(codexJournalRuleRelativePath)
	if err := writeCodexManagedRuleManifest(fixture.manifest(), manifest); err != nil {
		t.Fatalf("write orphaned manifest: %v", err)
	}
	if err := os.Remove(fixture.dest()); err != nil {
		t.Fatalf("remove installed rule: %v", err)
	}
	if err := os.Remove(fixture.source()); err != nil {
		t.Fatalf("remove template: %v", err)
	}
	noPath := &codexRuleInstallOperations{lookPath: func(string) (string, error) { return "", fmt.Errorf("PATH intentionally unavailable") }}
	if err := installCodexJournalRuleWithOperations(fixture.options(false, true), fixture.codexHome, noPath); err != nil {
		t.Fatalf("retire orphaned guidance: %v", err)
	}
	body, err := os.ReadFile(guidancePath)
	if err != nil {
		t.Fatalf("read retained user guidance: %v", err)
	}
	if strings.Contains(string(body), codexJournalGuidanceStart) || !strings.Contains(string(body), "# user guidance") {
		t.Fatalf("global guidance = %q, want only user guidance", body)
	}
}

func TestValidateCodexJournalGuidanceStructureRejectsMalformedOrDuplicateBlocks(t *testing.T) {
	block := generateCodexJournalGuidance()
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "start only", body: codexJournalGuidanceStart},
		{name: "end only", body: codexJournalGuidanceEnd},
		{name: "duplicate", body: block + "\n" + block},
		{name: "reversed", body: codexJournalGuidanceEnd + "\n" + codexJournalGuidanceStart},
		{name: "legacy reversed", body: codexLegacyGuidanceEnd + "\n" + codexLegacyGuidanceStart},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateCodexJournalGuidanceStructure(test.body); err == nil {
				t.Fatalf("validate malformed guidance = nil for %q", test.body)
			}
		})
	}
}

func TestFindCodexJournalGuidanceRecognizesLegacyJournalOnlyMarker(t *testing.T) {
	legacy := strings.Join([]string{codexLegacyGuidanceStart, "legacy journal-only guidance", codexLegacyGuidanceEnd}, "\n") + "\n"
	if err := validateCodexJournalGuidanceStructure(legacy); err != nil {
		t.Fatalf("validate legacy guidance: %v", err)
	}
	rangeValue, ok := findCodexJournalGuidance(legacy)
	if !ok || legacy[rangeValue.start:rangeValue.end] != legacy {
		t.Fatalf("legacy guidance range = %#v, found=%t, want complete block", rangeValue, ok)
	}
}

func TestInstallCodexBasicCommandsUpgradeRetiresLegacyJournalOnlyCapabilityWithoutOptIn(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	legacyRule := "prefix_rule(pattern=[\"/opt/legacy/loaf\", \"journal\", \"log\", \"--execpolicy-safe\"], decision=\"allow\")\n"
	legacyGuidance := strings.Join([]string{codexLegacyGuidanceStart, "legacy journal-only guidance", codexLegacyGuidanceEnd}, "\n") + "\n"
	writeInstallFile(t, fixture.dest(), legacyRule)
	writeInstallFile(t, filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath), legacyGuidance)
	manifest := codexManagedRuleManifest{Version: 1}
	manifest.set(codexJournalRuleRelativePath, sha256Bytes([]byte(legacyRule)))
	manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte(legacyGuidance)))
	if err := writeCodexManagedRuleManifest(fixture.manifest(), manifest); err != nil {
		t.Fatalf("write legacy manifest: %v", err)
	}
	if err := fixture.install(t, false, true); err != nil {
		t.Fatalf("legacy upgrade retirement: %v", err)
	}
	assertInstallPathMissing(t, fixture.dest())
	body, err := os.ReadFile(filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath))
	if err != nil {
		t.Fatalf("read retired guidance: %v", err)
	}
	if strings.Contains(string(body), codexLegacyGuidanceStart) || strings.Contains(string(body), codexLegacyGuidanceEnd) {
		t.Fatalf("retired guidance = %q, want legacy block removed", body)
	}
}

func TestReadCodexManagedRuleManifestRejectsDuplicatePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), codexJournalRuleManifest)
	digest := strings.Repeat("a", sha256.Size*2)
	body := fmt.Sprintf(`{"version":1,"files":[{"path":"loaf.rules","sha256":"%s"},{"path":"loaf.rules","sha256":"%s"}]}`, digest, digest)
	writeInstallFile(t, path, body)
	if _, err := readCodexManagedRuleManifest(path); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate manifest error = %v, want refusal", err)
	}
}

func TestDesiredCodexPolicyContentRendersPathLoafAndPreservesUserGuidance(t *testing.T) {
	root := t.TempDir()
	template := "# policy\n" + codexBasicRulesPlaceholder + "\n"
	writeInstallFile(t, filepath.Join(root, ".codex", "rules", codexJournalRuleTemplateRelativePath), template)
	options := targetInstallOptions{DistDir: root}
	rule, err := desiredCodexPolicyContent(options, artifactPlanDecision{Kind: "codex-rule", ID: "codex-rule:loaf.rules"}, "")
	if err != nil {
		t.Fatalf("desired Codex rule: %v", err)
	}
	if !strings.Contains(string(rule), strconv.Quote(codexPathLoafCommandName)+`, "journal", "log", "--execpolicy-safe"`) || strings.Contains(string(rule), "/opt/") || strings.Contains(string(rule), "{{") {
		t.Fatalf("desired rule = %q, want PATH loaf prefixes", rule)
	}
	live := "# user Codex instructions\n"
	if _, err := desiredCodexPolicyContent(options, artifactPlanDecision{Kind: "codex-guidance", ID: "codex-rule:AGENTS.md"}, live); err == nil {
		t.Fatal("retired global guidance must not be generated")
	}
}

func TestInstallCodexJournalRuleUpgradesUnmodifiedPinnedRuleToPathLoaf(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	pinned := filepath.Join(fixture.root, "trusted-bin", "loaf")
	oldRule := "# Loaf Codex policy\n" + legacyPinnedCodexBasicRules(pinned)
	oldGuidance := legacyPinnedCodexJournalGuidance(pinned)
	writeInstallFile(t, fixture.dest(), oldRule)
	writeInstallFile(t, filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath), oldGuidance)
	manifest := codexManagedRuleManifest{Version: 1}
	manifest.set(codexJournalRuleRelativePath, sha256Bytes([]byte(oldRule)))
	manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte(oldGuidance)))
	if err := writeCodexManagedRuleManifest(fixture.manifest(), manifest); err != nil {
		t.Fatalf("write pinned ownership: %v", err)
	}
	if err := fixture.install(t, false, true); err != nil {
		t.Fatalf("owned path-pin upgrade: %v", err)
	}
	body := string(readFileBytes(t, fixture.dest()))
	if strings.Contains(body, pinned) || !strings.Contains(body, strconv.Quote(codexPathLoafCommandName)+`, "journal", "log", "--execpolicy-safe"`) {
		t.Fatalf("upgraded rule = %q, want PATH loaf prefixes without the old pin", body)
	}
	guidance := string(readFileBytes(t, filepath.Join(fixture.codexHome, codexJournalGuidanceRelativePath)))
	if guidance != "" {
		t.Fatalf("upgraded guidance = %q, want obsolete owned block retired", guidance)
	}
}

func TestInstallCodexJournalRuleIgnoresMissingOwnedGuidance(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	if err := f.install(t, true, false); err != nil {
		t.Fatal(err)
	}
	seedLegacyCodexGuidance(t, f, "")
	path := filepath.Join(f.codexHome, "AGENTS.md")
	writeInstallFile(t, path, "# User instructions\n")
	if err := f.install(t, false, true); err != nil {
		t.Fatal(err)
	}
	assertInstallFile(t, path, "# User instructions\n")
}

func TestPlanCodexGuidancePreservesMissingOwnedBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeInstallFile(t, path, "# User instructions\n")
	manifest := codexManagedRuleManifest{Version: 1}
	manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte("absent block")))
	if got := planRetiredCodexGuidance(path, manifest); got.Action != planActionPreserve {
		t.Fatalf("decision = %#v", got)
	}
}

func TestInstallCodexJournalRulePreservesModifiedPinnedRule(t *testing.T) {
	fixture := newCodexRuleInstallFixture(t)
	pinned := filepath.Join(fixture.root, "trusted-bin", "loaf")
	oldRule := "# Loaf Codex policy\n" + legacyPinnedCodexBasicRules(pinned)
	writeInstallFile(t, fixture.dest(), oldRule+"\n# local edit\n")
	manifest := codexManagedRuleManifest{Version: 1}
	manifest.set(codexJournalRuleRelativePath, sha256Bytes([]byte(oldRule)))
	if err := writeCodexManagedRuleManifest(fixture.manifest(), manifest); err != nil {
		t.Fatalf("write pinned ownership: %v", err)
	}
	if err := fixture.install(t, false, true); err == nil || !strings.Contains(err.Error(), "modified") {
		t.Fatalf("modified pinned upgrade error = %v, want refusal", err)
	}
	assertInstallFile(t, fixture.dest(), oldRule+"\n# local edit\n")
}

func legacyPinnedCodexBasicRules(executable string) string {
	var rendered strings.Builder
	for _, prefix := range BasicCommandAuthorityPrefixes() {
		rendered.WriteString("prefix_rule(\n    pattern = [")
		rendered.WriteString(strconv.Quote(executable))
		for _, token := range prefix {
			rendered.WriteString(", ")
			rendered.WriteString(strconv.Quote(token))
		}
		rendered.WriteString("],\n    decision = \"allow\",\n)\n")
	}
	return rendered.String()
}

func legacyPinnedCodexJournalGuidance(executable string) string {
	return "\n" + strings.Join([]string{
		codexJournalGuidanceStart,
		"<!-- Maintained by loaf install/upgrade - do not edit manually -->",
		"## Loaf Codex basic command policy",
		"",
		"When Codex Auto mode records a durable decision, use this exact command; do not substitute a bare `loaf` command or a shell/environment wrapper:",
		"",
		"`" + journalContextShellQuote(executable) + " journal log --execpolicy-safe \"type(scope): description\"`",
		"",
		"The rule permits only explicitly classified basic Loaf command leaves, including routine state-plane operations and approved readers. It does not authorize unclassified or operator commands, a bare Loaf namespace, or a general Loaf data-directory writable root. Other harness adapters are not implied by this Codex policy.",
		codexJournalGuidanceEnd,
	}, "\n") + "\n"
}

type codexRuleInstallFixture struct {
	root      string
	codexHome string
	dist      string
}

func newCodexRuleInstallFixture(t *testing.T) codexRuleInstallFixture {
	t.Helper()
	root := realpath(t, t.TempDir())
	dist := filepath.Join(root, "dist", "codex")
	source := filepath.Join(dist, ".codex", "rules", codexJournalRuleTemplateRelativePath)
	writeInstallFile(t, source, "# Loaf Codex policy\n{{LOAF_BASIC_RULES}}\n")
	if err := os.MkdirAll(filepath.Join(root, "project"), 0o755); err != nil {
		t.Fatalf("create project fixture: %v", err)
	}
	return codexRuleInstallFixture{root: root, codexHome: filepath.Join(root, "codex-home"), dist: dist}
}

func (f codexRuleInstallFixture) options(autoJournal bool, upgrade bool) targetInstallOptions {
	return targetInstallOptions{Target: "codex", DistDir: f.dist, ConfigDir: filepath.Join(f.root, "reported-config"), CodexHome: f.codexHome, HomeDir: filepath.Join(f.root, "home"), ProjectRoot: filepath.Join(f.root, "project"), CodexBasicCommands: autoJournal, Upgrade: upgrade, Version: "9.9.9"}
}

func (f codexRuleInstallFixture) source() string {
	return filepath.Join(f.dist, ".codex", "rules", codexJournalRuleTemplateRelativePath)
}

func (f codexRuleInstallFixture) sourceBody() string {
	body, err := os.ReadFile(f.source())
	if err != nil {
		return ""
	}
	return string(body)
}

func (f codexRuleInstallFixture) renderedBody() string {
	body, err := renderCodexJournalRule(f.sourceBody())
	if err != nil {
		return ""
	}
	return body
}

func (f codexRuleInstallFixture) dest() string {
	return filepath.Join(f.codexHome, "rules", codexJournalRuleRelativePath)
}

func (f codexRuleInstallFixture) manifest() string {
	return filepath.Join(f.codexHome, "rules", codexJournalRuleManifest)
}

func (f codexRuleInstallFixture) install(t *testing.T, autoJournal bool, upgrade bool) error {
	t.Helper()
	return f.installWithExecutable(t, filepath.Join(f.root, "trusted-bin", "loaf"), autoJournal, upgrade)
}

func (f codexRuleInstallFixture) installWithExecutable(t *testing.T, path string, autoJournal bool, upgrade bool) error {
	t.Helper()
	writeInstallFile(t, path, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod trusted fixture: %v", err)
	}
	operations := &codexRuleInstallOperations{
		lookPath: func(name string) (string, error) {
			if name != "loaf" {
				return "", fmt.Errorf("unexpected executable %q", name)
			}
			return path, nil
		},
		forbiddenRoots: []string{filepath.Join(f.root, "project")},
	}
	return installCodexJournalRuleWithOperations(f.options(autoJournal, upgrade), f.codexHome, operations)
}

func writeCodexExecutableFixture(t *testing.T, path string) {
	t.Helper()
	writeInstallFile(t, path, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod executable fixture: %v", err)
	}
}

func symlinkCodexEntrypoint(t *testing.T, target string, entrypoint string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(entrypoint), 0o755); err != nil {
		t.Fatalf("create entrypoint directory: %v", err)
	}
	if err := os.Symlink(target, entrypoint); err != nil {
		t.Fatalf("symlink entrypoint: %v", err)
	}
}

func codexEntrypointOperations(root string, entrypoint string) *codexRuleInstallOperations {
	return &codexRuleInstallOperations{
		lookPath: func(name string) (string, error) {
			if name != "loaf" {
				return "", fmt.Errorf("unexpected executable %q", name)
			}
			return entrypoint, nil
		},
		forbiddenRoots: []string{filepath.Join(root, "project")},
	}
}

func readCodexRuleManifestTest(t *testing.T, path string) codexManagedRuleManifest {
	t.Helper()
	manifest, err := readCodexManagedRuleManifest(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return manifest
}

// Legacy guidance construction is retained only for migration fixtures.
func generateCodexJournalGuidance() string {
	return "\n" + strings.Join([]string{
		codexJournalGuidanceStart,
		"<!-- Maintained by loaf install/upgrade - do not edit manually -->",
		"## Loaf Codex basic command policy",
		"",
		"When Codex Auto mode records a durable decision, use this exact PATH command; do not substitute an absolute executable pin or a shell/environment wrapper:",
		"",
		"`" + codexPathLoafCommandName + " journal log --execpolicy-safe \"type(scope): description\"`",
		"",
		"The rule permits only explicitly classified basic Loaf command leaves, including routine state-plane operations and approved readers. It does not authorize unclassified or operator commands, a bare Loaf namespace, or a general Loaf data-directory writable root. Other harness adapters are not implied by this Codex policy.",
		codexJournalGuidanceEnd,
	}, "\n") + "\n"
}

func appendCodexJournalGuidance(content string, block string) string {
	return content + block
}

func replaceCodexJournalGuidance(content string, r codexJournalGuidanceRange, block string) string {
	return content[:r.start] + block + content[r.end:]
}
