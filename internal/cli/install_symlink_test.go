package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureInstallClaudeInstructionsLeavesAbsentPathAndCorrectLinkAlone(t *testing.T) {
	t.Run("absent stays absent", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		writeInstallFile(t, filepath.Join(root, "AGENTS.md"), "# Root\n")

		result := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true})
		if result.Action != "already-correct" || result.Error != "" {
			t.Fatalf("result = %#v, want already-correct without error", result)
		}
		assertInstallPathMissing(t, filepath.Join(root, ".claude", "CLAUDE.md"))
		assertInstallPathMissing(t, filepath.Join(root, ".claude"))
		if action, detail := planInstallClaudeInstructions(root, true); action != "already-correct" {
			t.Fatalf("plan = %q, %q, want already-correct", action, detail)
		}
	})

	t.Run("correct link untouched", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		writeInstallFile(t, filepath.Join(root, "AGENTS.md"), "# Root\n")
		link := filepath.Join(root, ".claude", "CLAUDE.md")
		mkdirAll(t, filepath.Dir(link))
		if err := os.Symlink("../AGENTS.md", link); err != nil {
			t.Fatalf("Symlink error = %v", err)
		}

		result := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true})
		if result.Action != "already-correct" || result.Error != "" {
			t.Fatalf("result = %#v, want already-correct without error", result)
		}
		assertRawSymlink(t, link, "../AGENTS.md")
		if action, detail := planInstallClaudeInstructions(root, true); action != "already-correct" {
			t.Fatalf("plan = %q, %q, want already-correct", action, detail)
		}
	})
}

func TestEnsureInstallClaudeInstructionsRemovesWrongLinksWithConsentControls(t *testing.T) {
	setup := func(t *testing.T, target string) (string, string) {
		t.Helper()
		root := realpath(t, t.TempDir())
		writeInstallFile(t, filepath.Join(root, "AGENTS.md"), "# Root\n")
		writeInstallFile(t, filepath.Join(root, "docs", "claude.md"), "# Other\n")
		link := filepath.Join(root, ".claude", "CLAUDE.md")
		mkdirAll(t, filepath.Dir(link))
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("Symlink error = %v", err)
		}
		return root, link
	}

	for _, target := range []string{"../docs/claude.md", "../.agents/AGENTS.md"} {
		t.Run("assume yes removes "+target, func(t *testing.T) {
			root, link := setup(t, target)
			if action, detail := planInstallClaudeInstructions(root, true); action != "removed" {
				t.Fatalf("plan = %q, %q, want removed", action, detail)
			}
			result := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true})
			if result.Action != "removed" || result.Error != "" {
				t.Fatalf("result = %#v, want removed without error", result)
			}
			assertInstallPathMissing(t, link)
			assertInstallFile(t, filepath.Join(root, "docs", "claude.md"), "# Other\n")
			assertInstallFile(t, filepath.Join(root, "AGENTS.md"), "# Root\n")
		})
	}

	t.Run("prompt decline leaves link", func(t *testing.T) {
		root, link := setup(t, "../docs/claude.md")
		prompted := false
		result := ensureInstallClaudeInstructions(root, installSymlinkOptions{
			Prompt: func(question string) bool {
				prompted = strings.Contains(question, "Remove the link?")
				return false
			},
		})
		if result.Action != "declined-remove" || !prompted {
			t.Fatalf("result = %#v prompted=%v, want declined-remove with prompt", result, prompted)
		}
		assertRawSymlink(t, link, "../docs/claude.md")
	})

	t.Run("non interactive skips", func(t *testing.T) {
		root, link := setup(t, "../docs/claude.md")
		if action, detail := planInstallClaudeInstructions(root, false); action != "skipped-no-tty" {
			t.Fatalf("plan = %q, %q, want skipped-no-tty", action, detail)
		}
		result := ensureInstallClaudeInstructions(root, installSymlinkOptions{NonInteractive: true})
		if result.Action != "skipped-no-tty" || result.Error != "" {
			t.Fatalf("result = %#v, want skipped-no-tty without error", result)
		}
		assertRawSymlink(t, link, "../docs/claude.md")
	})
}

func TestEnsureInstallClaudeInstructionsMigratesRealFileAndLeavesPathAbsent(t *testing.T) {
	root := realpath(t, t.TempDir())
	canonical := filepath.Join(root, "AGENTS.md")
	claude := filepath.Join(root, ".claude", "CLAUDE.md")
	writeInstallFile(t, canonical, "# Canonical\n")
	original := "# User Notes\n\n<!-- loaf:managed:start v1.0.0 -->\nold managed\n<!-- loaf:managed:end -->\n\nKeep this.\n"
	writeInstallFile(t, claude, original)
	writeInstallFile(t, claude+".bak", "# Earlier Backup\n")

	if action, detail := planInstallClaudeInstructions(root, true); action != "migrated" {
		t.Fatalf("plan = %q, %q, want migrated", action, detail)
	}
	result := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true})
	if result.Action != "migrated" || !result.Merged || result.Error != "" || result.BackupPath != claude+".bak.1" {
		t.Fatalf("result = %#v, want migrated with merge into a collision-safe backup", result)
	}
	assertInstallPathMissing(t, claude)
	assertInstallFile(t, claude+".bak", "# Earlier Backup\n")
	assertInstallFile(t, claude+".bak.1", original)
	body := string(readFileBytes(t, canonical))
	if strings.Contains(body, "old managed") || strings.Contains(body, "<!-- loaf:managed:start") {
		t.Fatalf("canonical body = %q, want managed fence stripped", body)
	}
	for _, want := range []string{"# Canonical", "## Migrated from .claude/CLAUDE.md", "# User Notes", "Keep this."} {
		if !strings.Contains(body, want) {
			t.Fatalf("canonical body = %q, want %q", body, want)
		}
	}

	retry := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true})
	if retry.Action != "already-correct" {
		t.Fatalf("retry = %#v, want already-correct", retry)
	}
	if count := strings.Count(string(readFileBytes(t, canonical)), "## Migrated from .claude/CLAUDE.md"); count != 1 {
		t.Fatalf("migration headings = %d, want 1", count)
	}
}

func TestEnsureInstallClaudeInstructionsDeclinesOrSkipsRealFileMigration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options installSymlinkOptions
		want    string
	}{
		{name: "prompt decline", options: installSymlinkOptions{Prompt: func(string) bool { return false }}, want: "declined-replace"},
		{name: "non interactive", options: installSymlinkOptions{NonInteractive: true}, want: "skipped-no-tty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := realpath(t, t.TempDir())
			writeInstallFile(t, filepath.Join(root, "AGENTS.md"), "# Root\n")
			claude := filepath.Join(root, ".claude", "CLAUDE.md")
			writeInstallFile(t, claude, "# Claude Notes\n")

			result := ensureInstallClaudeInstructions(root, tc.options)
			if result.Action != tc.want || result.Error != "" {
				t.Fatalf("result = %#v, want %s without error", result, tc.want)
			}
			assertInstallFile(t, claude, "# Claude Notes\n")
			assertInstallFile(t, filepath.Join(root, "AGENTS.md"), "# Root\n")
			assertInstallPathMissing(t, claude+".bak")
		})
	}
}

func TestEnsureProjectInstallSymlinksRoutesSelectedTargets(t *testing.T) {
	t.Run("claude and agents targets create root canonical without a Claude file", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		results := ensureProjectInstallSymlinks(root, []string{"cursor"}, true, installSymlinkOptions{AssumeYes: true})
		if results[".claude/CLAUDE.md"].Action != "already-correct" || results["./AGENTS.md"].Action != "created" {
			t.Fatalf("results = %#v, want root file created and no Claude file", results)
		}
		canonical := filepath.Join(root, "AGENTS.md")
		assertInstallFile(t, canonical, "")
		assertInstallPathMissing(t, filepath.Join(root, ".claude", "CLAUDE.md"))
		if installIsSymlink(canonical) {
			t.Fatalf("%s is a symlink, want real canonical file", canonical)
		}
	})

	t.Run("no matching targets is noop", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		results := ensureProjectInstallSymlinks(root, nil, false, installSymlinkOptions{AssumeYes: true})
		if len(results) != 0 {
			t.Fatalf("results = %#v, want empty", results)
		}
		if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
			t.Fatalf("canonical stat err = %v, want not exist", err)
		}
	})

	t.Run("codex only creates root agents file", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		results := ensureProjectInstallSymlinks(root, []string{"codex"}, false, installSymlinkOptions{AssumeYes: true})
		if results["./AGENTS.md"].Action != "created" {
			t.Fatalf("results = %#v, want root AGENTS file", results)
		}
		if _, ok := results[".claude/CLAUDE.md"]; ok {
			t.Fatalf("results = %#v, want no Claude result", results)
		}
	})

	t.Run("claude only creates root canonical and no Claude file", func(t *testing.T) {
		root := realpath(t, t.TempDir())
		results := ensureProjectInstallSymlinks(root, nil, true, installSymlinkOptions{AssumeYes: true})
		if results[".claude/CLAUDE.md"].Action != "already-correct" {
			t.Fatalf("results = %#v, want no Claude file created", results)
		}
		if results["./AGENTS.md"].Action != "created" {
			t.Fatalf("results = %#v, want Claude install to create root canonical file", results)
		}
		assertInstallPathMissing(t, filepath.Join(root, ".claude"))
	})
}

func TestEnsureProjectInstallSymlinksMigratesLegacyCanonicalLayout(t *testing.T) {
	root := realpath(t, t.TempDir())
	legacy := filepath.Join(root, ".agents", "AGENTS.md")
	writeInstallFile(t, legacy, "# Legacy Canonical\n")
	if err := os.Symlink(".agents/AGENTS.md", filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatalf("Symlink legacy root AGENTS.md error = %v", err)
	}
	writeInstallFile(t, filepath.Join(root, ".claude", "CLAUDE.md"), "# Claude Notes\n")

	results := ensureProjectInstallSymlinks(root, []string{"cursor"}, true, installSymlinkOptions{AssumeYes: true})
	if results[".claude/CLAUDE.md"].Action != "migrated" || !results[".claude/CLAUDE.md"].Merged {
		t.Fatalf("Claude result = %#v, want migrated with merge", results[".claude/CLAUDE.md"])
	}
	if results["./AGENTS.md"].Action != "migrated" {
		t.Fatalf("AGENTS result = %#v, want migrated legacy canonical", results["./AGENTS.md"])
	}

	canonicalPath := filepath.Join(root, "AGENTS.md")
	canonical := string(readFileBytes(t, canonicalPath))
	if !strings.Contains(canonical, "## Migrated from .claude/CLAUDE.md") || !strings.Contains(canonical, "# Claude Notes") {
		t.Fatalf("canonical body = %q, want migrated Claude notes", canonical)
	}
	if !strings.Contains(canonical, "# Legacy Canonical") {
		t.Fatalf("canonical body = %q, want legacy canonical content", canonical)
	}
	assertInstallFile(t, filepath.Join(root, ".claude", "CLAUDE.md.bak"), "# Claude Notes\n")
	if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy canonical still exists: %v", err)
	}
	assertInstallPathMissing(t, filepath.Join(root, ".claude", "CLAUDE.md"))
	if installIsSymlink(canonicalPath) {
		t.Fatalf("root canonical remains a symlink")
	}
}

func TestEnsureRootInstallAgentsFileRequiresConsentForConflictingRealFiles(t *testing.T) {
	for _, tc := range []struct {
		name        string
		options     installSymlinkOptions
		wantAction  string
		wantRetired bool
	}{
		{name: "assume yes", options: installSymlinkOptions{AssumeYes: true}, wantAction: "migrated", wantRetired: true},
		{name: "prompt accept", options: installSymlinkOptions{Prompt: func(string) bool { return true }}, wantAction: "migrated", wantRetired: true},
		{name: "prompt decline", options: installSymlinkOptions{Prompt: func(string) bool { return false }}, wantAction: "declined-replace"},
		{name: "non interactive", options: installSymlinkOptions{NonInteractive: true}, wantAction: "skipped-no-tty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := realpath(t, t.TempDir())
			canonical := filepath.Join(root, "AGENTS.md")
			legacy := filepath.Join(root, ".agents", "AGENTS.md")
			writeInstallFile(t, canonical, "# Root\n")
			writeInstallFile(t, legacy, "# Legacy\n")

			result := ensureRootInstallAgentsFile(root, tc.options)
			if result.Action != tc.wantAction || result.Error != "" {
				t.Fatalf("result = %#v, want %s without error", result, tc.wantAction)
			}
			body := string(readFileBytes(t, canonical))
			if tc.wantRetired {
				if !strings.Contains(body, "## Migrated from .agents/AGENTS.md") || !strings.Contains(body, "# Legacy") {
					t.Fatalf("canonical = %q, want merged legacy content", body)
				}
				if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
					t.Fatalf("legacy stat = %v, want retired", err)
				}
			} else {
				assertInstallFile(t, canonical, "# Root\n")
				assertInstallFile(t, legacy, "# Legacy\n")
			}
		})
	}
}

func TestEnsureRootInstallAgentsFilePreservesExistingBackupAndIsRetryIdempotent(t *testing.T) {
	root := realpath(t, t.TempDir())
	canonical := filepath.Join(root, "AGENTS.md")
	legacy := filepath.Join(root, ".agents", "AGENTS.md")
	writeInstallFile(t, canonical, "# Root\n")
	writeInstallFile(t, legacy, "# Legacy\n")
	writeInstallFile(t, legacy+".bak", "# Earlier Backup\n")

	result := ensureRootInstallAgentsFile(root, installSymlinkOptions{AssumeYes: true})
	if result.Action != "migrated" || result.BackupPath == legacy+".bak" {
		t.Fatalf("result = %#v, want migration with collision-safe backup", result)
	}
	assertInstallFile(t, legacy+".bak", "# Earlier Backup\n")
	assertInstallFile(t, result.BackupPath, "# Legacy\n")

	result = ensureRootInstallAgentsFile(root, installSymlinkOptions{AssumeYes: true})
	if result.Action != "already-correct" {
		t.Fatalf("retry result = %#v, want already-correct", result)
	}
	body := string(readFileBytes(t, canonical))
	if count := strings.Count(body, "## Migrated from .agents/AGENTS.md"); count != 1 {
		t.Fatalf("canonical migration headings = %d, want 1\n%s", count, body)
	}
}

func TestEnsureRootInstallAgentsFileRollsBackLegacyRetirementWhenMergeFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write permissions are not portable on Windows")
	}
	root := realpath(t, t.TempDir())
	canonical := filepath.Join(root, "AGENTS.md")
	legacy := filepath.Join(root, ".agents", "AGENTS.md")
	writeInstallFile(t, canonical, "# Root\n")
	writeInstallFile(t, legacy, "# Legacy\n")
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

	result := ensureRootInstallAgentsFile(root, installSymlinkOptions{AssumeYes: true})
	if result.Action != "error" {
		t.Fatalf("result = %#v, want merge error", result)
	}
	assertInstallFile(t, canonical, "# Root\n")
	assertInstallFile(t, legacy, "# Legacy\n")
}

func TestEnsureRootInstallAgentsFileRejectsDirectoryCanonical(t *testing.T) {
	root := realpath(t, t.TempDir())
	mkdirAll(t, filepath.Join(root, "AGENTS.md"))

	result := ensureRootInstallAgentsFile(root, installSymlinkOptions{AssumeYes: true})
	if result.Action != "error" || !strings.Contains(result.Error, "directory") {
		t.Fatalf("result = %#v, want directory rejection", result)
	}
}

func assertInstallSymlinkTarget(t *testing.T, linkPath string, wantAbs string) {
	t.Helper()
	if !installIsSymlink(linkPath) {
		t.Fatalf("%s is not a symlink", linkPath)
	}
	if !installSymlinkPointsTo(linkPath, wantAbs) {
		t.Fatalf("%s resolves to %q, want %q", linkPath, resolveInstallSymlinkTarget(linkPath), filepath.Clean(wantAbs))
	}
}

func assertRawSymlink(t *testing.T, linkPath string, wantTarget string) {
	t.Helper()
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%s) error = %v", linkPath, err)
	}
	if target != wantTarget {
		t.Fatalf("Readlink(%s) = %q, want %q", linkPath, target, wantTarget)
	}
}
