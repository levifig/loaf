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
