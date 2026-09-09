package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerReleaseHelpIsNative(t *testing.T) {
	workingDir := realpath(t, t.TempDir())
	var stdout bytes.Buffer

	err := Runner{
		Stdout:     &stdout,
		WorkingDir: workingDir,
	}.Run([]string{"release", "--help"})
	if err != nil {
		t.Fatalf("release --help error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Usage: loaf release <subcommand>") || !strings.Contains(output, "suggest") || !strings.Contains(output, "cut") {
		t.Fatalf("output = %q, want retroactive suggest/cut help", output)
	}
	if strings.Contains(output, "Create a new release with changelog") {
		t.Fatalf("output = %q, want legacy apply-path help removed", output)
	}
}

func TestReleaseLegacyFlagsFailWithGuidance(t *testing.T) {
	workingDir := realpath(t, t.TempDir())
	for _, args := range [][]string{
		{"release", "--bump", "patch"},
		{"release", "--pre-merge"},
		{"release", "--post-merge"},
		{"release", "--yes"},
		{"release"},
	} {
		var stdout bytes.Buffer
		err := Runner{
			Stdout:     &stdout,
			WorkingDir: workingDir,
		}.Run(args)
		if err == nil {
			t.Fatalf("%v error = nil, want guidance", args)
		}
		msg := err.Error() + stdout.String()
		if !strings.Contains(msg, "suggest") || !strings.Contains(msg, "cut") {
			t.Fatalf("%v error = %v\n%s, want suggest/cut guidance", args, err, stdout.String())
		}
	}
}

// commitBootstrapAgentsFiles records genesis .agents/ files (e.g. loaf.conf) so
// release cut's clean-tree gate passes after state init.
func commitBootstrapAgentsFiles(t *testing.T, repo string) {
	t.Helper()
	status := gitOutputReleaseTest(t, repo, "status", "--porcelain=v1", ".agents")
	if strings.TrimSpace(status) == "" {
		return
	}
	gitCLI(t, repo, "add", ".agents")
	gitCLI(t, repo, "commit", "-m", "chore: bootstrap loaf project files")
}

func seedReleaseTaggedRepo(t *testing.T) string {
	t.Helper()
	repo := realpath(t, t.TempDir())
	gitCLI(t, repo, "init", "-b", "main")
	gitCLI(t, repo, "config", "user.name", "Loaf Test")
	gitCLI(t, repo, "config", "user.email", "loaf@example.test")
	gitCLI(t, repo, "config", "commit.gpgsign", "false")
	gitCLI(t, repo, "config", "tag.gpgsign", "false")
	writeFile(t, filepath.Join(repo, "package.json"), "{\n  \"name\": \"release-fixture\",\n  \"version\": \"1.0.0\",\n  \"scripts\": {\n    \"build\": \"echo build\"\n  }\n}\n")
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), strings.Join([]string{
		"# Changelog",
		"",
		"## [Unreleased]",
		"",
		"- _No unreleased changes yet._",
		"",
	}, "\n"))
	gitCLI(t, repo, "add", ".")
	gitCLI(t, repo, "commit", "-m", "chore: initial release")
	gitCLI(t, repo, "tag", "v1.0.0")
	return repo
}

func gitOutputReleaseTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestInsertReleaseChangelogPreservesCuratedUnreleased(t *testing.T) {
	existing := strings.Join([]string{
		"# Changelog",
		"",
		"## [Unreleased]",
		"",
		"### Added",
		"- Curated operator note for the release",
		"",
		"## [1.0.0] - 2025-01-01",
		"",
		"- Initial release",
		"",
	}, "\n")
	drafted := "## [1.1.0] - 2026-08-27\n\n### LOAF-1 — Ship auth\n- feat: add auth (abc1234)"
	got := insertReleaseChangelog(existing, drafted)
	if !strings.Contains(got, "Curated operator note for the release") {
		t.Fatalf("curated unreleased prose missing:\n%s", got)
	}
	if strings.Contains(got, "### LOAF-1") {
		t.Fatalf("drafted issue section should not replace curated prose:\n%s", got)
	}
	if !strings.Contains(got, "## [1.1.0] - 2026-08-27") {
		t.Fatalf("release header missing:\n%s", got)
	}
	if !strings.Contains(got, "- _No unreleased changes yet._") {
		t.Fatalf("unreleased stub not reset:\n%s", got)
	}
}

func TestInsertReleaseChangelogUsesDraftWhenUnreleasedEmpty(t *testing.T) {
	existing := strings.Join([]string{
		"# Changelog",
		"",
		"## [Unreleased]",
		"",
		"- _No unreleased changes yet._",
		"",
		"## [1.0.0] - 2025-01-01",
		"",
		"- Initial release",
		"",
	}, "\n")
	drafted := "## [1.1.0] - 2026-08-27\n\n### LOAF-1 — Ship auth\n- feat: add auth (abc1234)"
	got := insertReleaseChangelog(existing, drafted)
	if !strings.Contains(got, "### LOAF-1 — Ship auth") {
		t.Fatalf("drafted notes missing:\n%s", got)
	}
}

func TestInsertReleaseChangelogCreatesUnreleasedBeforeFirstExistingRelease(t *testing.T) {
	existing := strings.Join([]string{
		"# Changelog",
		"",
		"Project-specific introduction.",
		"",
		"## [1.0.0] - 2025-01-01",
		"",
		"- Initial release",
		"",
		"## [0.9.0] - 2024-12-01",
		"",
		"- Preview release",
		"",
	}, "\n")
	drafted := "## [1.1.0] - 2026-09-04\n\n### Fixed\n- Preserve newest-first changelog order"

	got := insertReleaseChangelog(existing, drafted)

	for _, want := range []string{"Project-specific introduction.", "- Initial release", "- Preview release", "## [Unreleased]", "- _No unreleased changes yet._", drafted} {
		if !strings.Contains(got, want) {
			t.Fatalf("updated changelog missing %q:\n%s", want, got)
		}
	}
	unreleased := strings.Index(got, "## [Unreleased]")
	newRelease := strings.Index(got, "## [1.1.0]")
	firstHistorical := strings.Index(got, "## [1.0.0]")
	secondHistorical := strings.Index(got, "## [0.9.0]")
	if !(unreleased < newRelease && newRelease < firstHistorical && firstHistorical < secondHistorical) {
		t.Fatalf("release order is not Unreleased, new, then historical:\n%s", got)
	}
	for _, heading := range []string{"## [Unreleased]", "## [1.1.0]", "## [1.0.0]", "## [0.9.0]"} {
		if count := strings.Count(got, heading); count != 1 {
			t.Fatalf("heading %q occurs %d times, want once:\n%s", heading, count, got)
		}
	}
}

func TestWriteReleaseChangelogPreservesExistingBytes(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		for _, trailing := range []string{"", newline, newline + newline} {
			t.Run(fmt.Sprintf("newline=%q/trailing=%q", newline, trailing), func(t *testing.T) {
				root := t.TempDir()
				header := "# Changelog" + newline + newline + "Introduction.  " + newline + newline
				history := "## [1.0.0] - 2025-01-01" + newline + newline + "- Original note.  " + trailing
				path := filepath.Join(root, "CHANGELOG.md")
				writeFile(t, path, header+history)
				drafted := "## [1.1.0] - 2026-09-04\n\n- Fix placement"
				if err := writeReleaseChangelog(root, drafted); err != nil {
					t.Fatal(err)
				}
				got := string(readFileBytes(t, path))
				insertion := "## [Unreleased]\n\n- _No unreleased changes yet._\n\n" + drafted + "\n\n"
				if want := header + insertion + history; got != want {
					t.Fatalf("changelog changed outside insertion site:\ngot  %q\nwant %q", got, want)
				}
			})
		}
	}
}

func TestWriteReleaseChangelogWithoutHistory(t *testing.T) {
	drafted := "## [1.0.0] - 2026-09-04\n\n- Initial release"
	for _, existing := range []string{"", "# Changelog", "# Changelog\n\nIntroduction.\n"} {
		t.Run(fmt.Sprintf("existing=%q", existing), func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "CHANGELOG.md")
			writeFile(t, path, existing)
			if err := writeReleaseChangelog(root, drafted); err != nil {
				t.Fatal(err)
			}
			got := string(readFileBytes(t, path))
			if !strings.HasPrefix(got, existing) || !strings.Contains(got, "## [Unreleased]\n\n- _No unreleased changes yet._\n\n"+drafted) {
				t.Fatalf("missing preserved introduction or new release: %q", got)
			}
		})
	}
	root := t.TempDir()
	if err := writeReleaseChangelog(root, drafted); err != nil {
		t.Fatal(err)
	}
	if got := string(readFileBytes(t, filepath.Join(root, "CHANGELOG.md"))); got != createReleaseChangelog(drafted) {
		t.Fatalf("new changelog differs from existing document generator: %q", got)
	}
}

func TestValidateExplicitReleaseVersionAllowsHigherMinor(t *testing.T) {
	if err := validateExplicitReleaseVersion("0.3.1", "minor", "0.5.0"); err != nil {
		t.Fatalf("0.5.0 should be allowed for minor bump from 0.3.1: %v", err)
	}
}

func TestValidateExplicitReleaseVersionRejectsBelowMinimum(t *testing.T) {
	err := validateExplicitReleaseVersion("0.3.1", "minor", "0.3.2")
	if err == nil || !strings.Contains(err.Error(), "below minimum") {
		t.Fatalf("expected below minimum error, got %v", err)
	}
}

func TestValidateExplicitReleaseVersionRejectsInvalidSemver(t *testing.T) {
	err := validateExplicitReleaseVersion("0.3.1", "minor", "not-a-version")
	if err == nil || !strings.Contains(err.Error(), "Invalid version") {
		t.Fatalf("expected invalid version error, got %v", err)
	}
}
