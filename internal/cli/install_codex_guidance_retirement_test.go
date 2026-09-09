package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyGuidanceOwnershipDoesNotElectPermissionRules(t *testing.T) {
	f := newCodexRuleInstallFixture(t)
	seedLegacyCodexGuidance(t, f, "# User instructions\n")
	noPath := &codexRuleInstallOperations{lookPath: func(string) (string, error) {
		t.Error("guidance cleanup must not require a PATH runtime")
		return "", fmt.Errorf("PATH unavailable")
	}}
	if err := installCodexJournalRuleWithOperations(f.options(false, true), f.codexHome, noPath); err != nil {
		t.Fatal(err)
	}
	assertInstallPathMissing(t, f.dest())
	assertInstallFile(t, filepath.Join(f.codexHome, "AGENTS.md"), "# User instructions\n")
	manifest := readCodexRuleManifestTest(t, f.manifest())
	if len(manifest.Files) != 0 {
		t.Fatalf("unexpected ownership after cleanup: %#v", manifest.Files)
	}
}

func seedLegacyCodexGuidance(t *testing.T, f codexRuleInstallFixture, user string) {
	t.Helper()
	block := legacyPinnedCodexJournalGuidance("/old/loaf")
	writeInstallFile(t, filepath.Join(f.codexHome, "AGENTS.md"), user+block)
	manifest, err := readCodexManagedRuleManifest(f.manifest())
	if err != nil {
		t.Fatal(err)
	}
	manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte(block)))
	if err := writeCodexManagedRuleManifest(f.manifest(), manifest); err != nil {
		t.Fatal(err)
	}
}

func TestCodexRulesDoNotRequireGlobalGuidance(t *testing.T) {
	for _, mode := range []string{"absent", "user-file", "symlink", "broken-symlink", "modified-owned", "unchanged-owned"} {
		t.Run(mode, func(t *testing.T) {
			f := newCodexRuleInstallFixture(t)
			path := filepath.Join(f.codexHome, "AGENTS.md")
			user := "# Shared user instructions\n"
			block := legacyPinnedCodexJournalGuidance("/old/loaf")
			manifest := codexManagedRuleManifest{Version: 1}
			switch mode {
			case "user-file":
				writeInstallFile(t, path, user)
			case "symlink", "broken-symlink":
				target := filepath.Join(f.root, "shared.md")
				if mode == "symlink" {
					writeInstallFile(t, target, user)
				}
				if err := os.MkdirAll(f.codexHome, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
				manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte(block)))
			case "modified-owned":
				writeInstallFile(t, path, user+block+"user tail\n")
				manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte("old block")))
			case "unchanged-owned":
				writeInstallFile(t, path, user+block)
				manifest.set(codexJournalGuidanceRelativePath, sha256Bytes([]byte(block)))
			}
			if len(manifest.Files) > 0 {
				if err := writeCodexManagedRuleManifest(f.manifest(), manifest); err != nil {
					t.Fatal(err)
				}
			}
			for _, upgrade := range []bool{false, true, true} {
				if err := f.install(t, !upgrade, upgrade); err != nil {
					t.Fatal(err)
				}
			}
			assertInstallFile(t, f.dest(), f.renderedBody())
			switch mode {
			case "absent":
				assertInstallPathMissing(t, path)
			case "user-file", "unchanged-owned":
				assertInstallFile(t, path, user)
			case "modified-owned":
				assertInstallFile(t, path, user+block+"user tail\n")
			case "symlink", "broken-symlink":
				if _, err := os.Readlink(path); err != nil {
					t.Fatal(err)
				}
				if mode == "symlink" {
					assertInstallFile(t, path, user)
				}
			}
		})
	}
}

func TestScopedCodexRuleUpgradeIgnoresSymlinkedGlobalGuidance(t *testing.T) {
	originalPath := os.Getenv("PATH")
	root, home, rulePath, old := setupScopedCodexPolicyFixture(t)
	t.Setenv("PATH", originalPath)
	path := filepath.Join(home, ".codex", "AGENTS.md")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, "shared.md")
	writeInstallFile(t, target, "# User instructions\n")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	runInstallCapture(t, root, "upgrade", "--select", "codex/codex-rule:loaf.rules", "--select", "codex/codex-rule:AGENTS.md")
	if readFileString(t, rulePath) == old {
		t.Fatal("rule did not upgrade")
	}
	if _, err := os.Readlink(path); err != nil {
		t.Fatal(err)
	}
	assertInstallFile(t, target, "# User instructions\n")
}
