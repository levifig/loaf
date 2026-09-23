package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Claude Code reads root AGENTS.md natively unless a CLAUDE.md shadows it, so
// the doctor check accepts an absent path or a link that resolves to root
// AGENTS.md, and flags anything else that would take its place.
func TestCheckClaudeInstructionsDiagnosesAndRepairs(t *testing.T) {
	for _, tc := range []struct {
		name        string
		canonical   string // "" means no root AGENTS.md
		setup       func(t *testing.T, root string)
		wantStatus  doctorStatus
		wantAgents  []string
		wantBackup  string
		keepsTarget string
	}{
		{
			name:       "absent passes",
			canonical:  "# Root\n",
			setup:      func(*testing.T, string) {},
			wantStatus: doctorPass,
		},
		{
			name:      "symlink to root AGENTS.md passes",
			canonical: "# Root\n",
			setup: func(t *testing.T, root string) {
				symlinkFile(t, "../AGENTS.md", filepath.Join(root, ".claude", "CLAUDE.md"))
			},
			wantStatus: doctorPass,
		},
		{
			name:      "symlink elsewhere fails and is removed",
			canonical: "# Root\n",
			setup: func(t *testing.T, root string) {
				writeDoctorFile(t, filepath.Join(root, "docs", "claude.md"), "# Other\n")
				symlinkFile(t, "../docs/claude.md", filepath.Join(root, ".claude", "CLAUDE.md"))
			},
			wantStatus:  doctorFail,
			keepsTarget: "docs/claude.md",
		},
		{
			name:      "dangling symlink fails and is removed",
			canonical: "# Root\n",
			setup: func(t *testing.T, root string) {
				symlinkFile(t, "../.agents/AGENTS.md", filepath.Join(root, ".claude", "CLAUDE.md"))
			},
			wantStatus: doctorFail,
		},
		{
			name:      "real file merges into root and leaves path absent",
			canonical: "# Root\n",
			setup: func(t *testing.T, root string) {
				writeDoctorFile(t, filepath.Join(root, ".claude", "CLAUDE.md"), "# Claude Notes\n\nclaude context\n")
			},
			wantStatus: doctorFail,
			wantAgents: []string{"# Root", "## Migrated from .claude/CLAUDE.md", "claude context"},
			wantBackup: "# Claude Notes\n\nclaude context\n",
		},
		{
			name: "legacy real file without root migrates and leaves path absent",
			setup: func(t *testing.T, root string) {
				writeDoctorFile(t, filepath.Join(root, ".claude", "CLAUDE.md"), "# Claude Notes\n\nclaude context\n")
			},
			wantStatus: doctorFail,
			wantAgents: []string{"# Claude Notes", "claude context"},
			wantBackup: "# Claude Notes\n\nclaude context\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeDoctorFixture(t, "9.8.7-test.1")
			if tc.canonical != "" {
				writeDoctorAgents(t, root, tc.canonical)
			}
			tc.setup(t, root)
			claude := filepath.Join(root, ".claude", "CLAUDE.md")
			ctx := doctorContext{projectRoot: root}
			check := checkClaudeInstructions()

			result := check.Run(ctx)
			if result.Status != tc.wantStatus {
				t.Fatalf("result = %#v, want status %s", result, tc.wantStatus)
			}
			if tc.wantStatus != doctorFail {
				return
			}
			if !result.Fixable {
				t.Fatalf("result = %#v, want fixable", result)
			}
			fix := check.Fix(ctx, result)
			if !fix.Fixed {
				t.Fatalf("fix = %#v, want fixed", fix)
			}
			if _, err := os.Lstat(claude); !os.IsNotExist(err) {
				t.Fatalf(".claude/CLAUDE.md Lstat err = %v, want the path absent after repair", err)
			}
			if tc.keepsTarget != "" {
				assertInstallFile(t, filepath.Join(root, filepath.FromSlash(tc.keepsTarget)), "# Other\n")
			}
			if tc.wantBackup != "" {
				assertInstallFile(t, claude+".bak", tc.wantBackup)
			} else if pathExistsForDoctor(claude + ".bak") {
				t.Fatal("link removal wrote a backup; a link carries no content to preserve")
			}
			body := string(readFileBytes(t, filepath.Join(root, "AGENTS.md")))
			for _, want := range tc.wantAgents {
				if !strings.Contains(body, want) {
					t.Fatalf("AGENTS.md = %q, want %q", body, want)
				}
			}
			if recheck := check.Run(ctx); recheck.Status != doctorPass {
				t.Fatalf("recheck = %#v, want pass after repair", recheck)
			}
		})
	}
}

func TestCheckClaudeInstructionsKeepsExistingBackup(t *testing.T) {
	root := writeDoctorFixture(t, "9.8.7-test.1")
	writeDoctorAgents(t, root, "# Root\n")
	claude := filepath.Join(root, ".claude", "CLAUDE.md")
	writeDoctorFile(t, claude, "# Newer Claude Notes\n")
	writeDoctorFile(t, claude+".bak", "# Earlier Backup\n")
	ctx := doctorContext{projectRoot: root}

	fix := checkClaudeInstructions().Fix(ctx, checkClaudeInstructions().Run(ctx))
	if !fix.Fixed {
		t.Fatalf("fix = %#v, want fixed", fix)
	}
	assertInstallFile(t, claude+".bak", "# Earlier Backup\n")
	assertInstallFile(t, claude+".bak.1", "# Newer Claude Notes\n")
	if pathExistsForDoctor(claude) {
		t.Fatal(".claude/CLAUDE.md still present after repair")
	}
}

func TestRunnerDoctorFixLegacyClaudeLayoutMigratesContentOnce(t *testing.T) {
	root := writeDoctorFixture(t, "9.8.7-test.1")
	claude := filepath.Join(root, ".claude", "CLAUDE.md")
	writeDoctorFile(t, claude, "# Claude Notes\n\nclaude context\n")
	var stdout bytes.Buffer

	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"doctor", "--fix", "--force"}); err != nil {
		t.Fatalf("doctor --fix --force error = %v\n%s", err, stdout.String())
	}
	body := string(readFileBytes(t, filepath.Join(root, "AGENTS.md")))
	if count := strings.Count(body, "claude context"); count != 1 {
		t.Fatalf("AGENTS.md carries the migrated content %d times, want once\n%s", count, body)
	}
	if pathExistsForDoctor(claude) {
		t.Fatal(".claude/CLAUDE.md still present after legacy migration")
	}
	assertInstallFile(t, claude+".bak", "# Claude Notes\n\nclaude context\n")
}

// A failed AGENTS.md write must leave both instruction files exactly as they
// were: the merge writes through a temporary file, and the Claude file moves
// back from its backup.
func TestClaudeMigrationRollsBackWhenAgentsWriteFails(t *testing.T) {
	skipWithoutEnforcedPermissions(t)
	for _, tc := range []struct {
		name string
		run  func(root string) bool
	}{
		{name: "doctor", run: func(root string) bool {
			ctx := doctorContext{projectRoot: root}
			return checkClaudeInstructions().Fix(ctx, checkClaudeInstructions().Run(ctx)).Fixed
		}},
		{name: "install", run: func(root string) bool {
			result := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true})
			return result.Action != "error" || !anyInstallSymlinkRefusal(map[string]installSymlinkResult{"claude": result})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeDoctorFixture(t, "9.8.7-test.1")
			canonical := filepath.Join(root, "AGENTS.md")
			claude := filepath.Join(root, ".claude", "CLAUDE.md")
			writeDoctorAgents(t, root, "# Root\n\nroot context\n")
			writeDoctorFile(t, claude, "# Claude Notes\n")
			// The root directory refuses new entries, so the temporary AGENTS.md
			// cannot be created; .claude stays writable so the backup move works.
			chmodForTest(t, root, 0o555)

			if tc.run(root) {
				t.Fatal("migration reported success although root AGENTS.md could not be written")
			}
			chmodForTest(t, root, 0o755)
			assertInstallFile(t, canonical, "# Root\n\nroot context\n")
			assertInstallFile(t, claude, "# Claude Notes\n")
			assertInstallPathMissing(t, claude+".bak")
		})
	}
}

// symlinkedClaudeDirFixture points the project's .claude at a sibling directory
// outside the project that holds a CLAUDE.md.
func symlinkedClaudeDirFixture(t *testing.T, claudeBody string, claudeLink string) (root string, outside string) {
	t.Helper()
	root = writeDoctorFixture(t, "9.8.7-test.1")
	writeDoctorAgents(t, root, "# Root\n")
	outside = realpath(t, t.TempDir())
	if claudeLink != "" {
		symlinkFile(t, claudeLink, filepath.Join(outside, "CLAUDE.md"))
	} else {
		writeDoctorFile(t, filepath.Join(outside, "CLAUDE.md"), claudeBody)
	}
	symlinkFile(t, outside, filepath.Join(root, ".claude"))
	return root, outside
}

func TestSymlinkedClaudeDirIsRefusedNotRepaired(t *testing.T) {
	for _, tc := range []struct {
		name       string
		claudeBody string
		claudeLink string
	}{
		{name: "real file outside the project", claudeBody: "# Outside\n"},
		// Lexically .claude/CLAUDE.md -> ../AGENTS.md names root AGENTS.md, but
		// through the symlinked directory it resolves beside the outside dir.
		{name: "lexically correct link that resolves elsewhere", claudeLink: "../AGENTS.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, outside := symlinkedClaudeDirFixture(t, tc.claudeBody, tc.claudeLink)
			outsideClaude := filepath.Join(outside, "CLAUDE.md")
			ctx := doctorContext{projectRoot: root}

			result := checkClaudeInstructions().Run(ctx)
			if result.Status != doctorFail || result.Fixable || !strings.Contains(result.Detail, ".claude is a symlink") {
				t.Fatalf("doctor result = %#v, want a non-fixable symlinked-directory failure", result)
			}
			if dup := checkDuplicateFencedSections().Run(ctx); dup.Status == doctorFail {
				t.Fatalf("duplicate-fenced-sections = %#v, must not offer a repair through the link", dup)
			}
			var stdout bytes.Buffer
			if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"doctor", "--fix", "--force"}); err == nil {
				t.Fatalf("doctor --fix --force succeeded through a symlinked .claude\n%s", stdout.String())
			}

			install := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true})
			if install.Action != "error" || !install.Refused || !strings.Contains(install.Message, ".claude is a symlink") {
				t.Fatalf("install result = %#v, want a refusal that stops the project pass", install)
			}
			if action, detail := planInstallClaudeInstructions(root, true); action != "error" {
				t.Fatalf("plan = %q, %q, want error", action, detail)
			}

			if tc.claudeLink != "" {
				assertRawSymlink(t, outsideClaude, tc.claudeLink)
			} else {
				assertInstallFile(t, outsideClaude, tc.claudeBody)
			}
			assertInstallPathMissing(t, outsideClaude+".bak")
			assertInstallFile(t, filepath.Join(root, "AGENTS.md"), "# Root\n")
			assertRawSymlink(t, filepath.Join(root, ".claude"), outside)
		})
	}
}

// A symlinked .claude with nothing at CLAUDE.md shadows nothing, so it neither
// fails doctor nor blocks install.
func TestSymlinkedClaudeDirWithoutClaudeFilePasses(t *testing.T) {
	root := writeDoctorFixture(t, "9.8.7-test.1")
	writeDoctorAgents(t, root, "# Root\n")
	outside := realpath(t, t.TempDir())
	writeDoctorFile(t, filepath.Join(outside, "settings.json"), "{}\n")
	symlinkFile(t, outside, filepath.Join(root, ".claude"))

	if result := checkClaudeInstructions().Run(doctorContext{projectRoot: root}); result.Status != doctorPass {
		t.Fatalf("doctor result = %#v, want pass", result)
	}
	if result := ensureInstallClaudeInstructions(root, installSymlinkOptions{AssumeYes: true}); result.Action != "already-correct" {
		t.Fatalf("install result = %#v, want already-correct", result)
	}
}

func TestCheckClaudeRootImportWarnsWhenRootClaudeFileHidesAgents(t *testing.T) {
	for _, tc := range []struct {
		name       string
		files      map[string]string
		link       string // root file symlinked to AGENTS.md
		wantStatus doctorStatus
		wantNamed  string
	}{
		{name: "absent passes", wantStatus: doctorPass},
		{name: "CLAUDE.md with import passes", files: map[string]string{"CLAUDE.md": "# Claude\n\n@AGENTS.md\n"}, wantStatus: doctorPass},
		{name: "CLAUDE.md linked to AGENTS.md passes", link: "CLAUDE.md", wantStatus: doctorPass},
		{name: "CLAUDE.md without import warns", files: map[string]string{"CLAUDE.md": "# Claude\n\nSee AGENTS.md for details.\n"}, wantStatus: doctorWarn, wantNamed: "CLAUDE.md"},
		{name: "CLAUDE.local.md without import warns", files: map[string]string{"CLAUDE.local.md": "# Local\n"}, wantStatus: doctorWarn, wantNamed: "CLAUDE.local.md"},
		{name: "one importing file does not excuse the other", files: map[string]string{"CLAUDE.md": "@AGENTS.md\n", "CLAUDE.local.md": "# Local\n"}, wantStatus: doctorWarn, wantNamed: "CLAUDE.local.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeDoctorFixture(t, "9.8.7-test.1")
			writeDoctorAgents(t, root, doctorFence("9.8.7-test.1"))
			for name, body := range tc.files {
				writeDoctorFile(t, filepath.Join(root, name), body)
			}
			if tc.link != "" {
				symlinkFile(t, "AGENTS.md", filepath.Join(root, tc.link))
			}

			result := checkClaudeRootImport().Run(doctorContext{projectRoot: root})
			if result.Status != tc.wantStatus {
				t.Fatalf("result = %#v, want %s", result, tc.wantStatus)
			}
			if tc.wantStatus == doctorWarn {
				if !strings.Contains(result.Message, tc.wantNamed) || !strings.Contains(result.Detail, "will not read root AGENTS.md") || !strings.Contains(result.Detail, "@AGENTS.md") {
					t.Fatalf("result = %#v, want the file named and the @AGENTS.md remedy", result)
				}
				if tc.wantNamed == "CLAUDE.local.md" && strings.Contains(result.Message, "CLAUDE.md and") {
					t.Fatalf("result = %#v, want only the non-importing file named", result)
				}
			}
			for name, body := range tc.files {
				assertInstallFile(t, filepath.Join(root, name), body)
			}
		})
	}
}

// The warning never fails doctor, and doctor --fix leaves the file alone.
func TestRunnerDoctorRootClaudeFileWarningDoesNotFailOrRepair(t *testing.T) {
	root := writeDoctorFixture(t, "9.8.7-test.1")
	writeDoctorAgents(t, root, doctorFence("9.8.7-test.1"))
	writeDoctorFile(t, filepath.Join(root, "CLAUDE.md"), "# Claude\n")
	var stdout bytes.Buffer

	if err := (Runner{Stdout: &stdout, WorkingDir: root, Executable: distributionFixtureExecutable(root)}).Run([]string{"doctor", "--fix", "--force"}); err != nil {
		t.Fatalf("doctor --fix --force error = %v, want warning-only success\n%s", err, stdout.String())
	}
	output := stripANSI(stdout.String())
	if !strings.Contains(output, "claude-root-import") || !strings.Contains(output, "without an @AGENTS.md import") {
		t.Fatalf("doctor output = %q, want the root CLAUDE.md warning", output)
	}
	assertInstallFile(t, filepath.Join(root, "CLAUDE.md"), "# Claude\n")
}
