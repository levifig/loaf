package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWorkflowPrePRRejectsUnrelatedExistingChangelogEntry(t *testing.T) {
	repo := initCLIGitRepo(t)
	mkdirAll(t, filepath.Join(repo, ".github"))
	writeFile(t, filepath.Join(repo, ".github", "PULL_REQUEST_TEMPLATE.md"), coverageDeclaration)
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), "# Changelog\n\n## [Unreleased]\n\n### Fixed\n\n- Fix an older bug ([#279](https://github.com/levifig/loaf/issues/279)).\n")
	gitCLI(t, repo, "add", "CHANGELOG.md", ".github")
	gitCLI(t, repo, "-c", "user.name=Loaf Test", "-c", "user.email=loaf@example.test", "-c", "commit.gpgsign=false", "commit", "-m", "base notes")
	gitCLI(t, repo, "switch", "-c", "feature")
	body := filepath.Join(t.TempDir(), "pr.md")
	writeFile(t, body, "## Native work\n\n<!-- loaf:implemented-issues {\"issues\":[\"https://github.com/levifig/loaf/issues/284\"]} -->\n")
	var stderr bytes.Buffer
	err := (Runner{WorkingDir: repo, Stderr: &stderr, Stdin: bytes.NewBufferString(checkBashContext(`gh pr create --repo levifig/loaf --base main --title "fix: enforce changelog coverage" --body-file "` + body + `"`))}).Run([]string{"check", "--hook", "workflow-pre-pr"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 || !strings.Contains(stderr.String(), "284") {
		t.Fatalf("old unrelated note must not cover issue284: error=%v stderr=%q", err, stderr.String())
	}
}

func coverageCommit(t *testing.T, repo string) {
	t.Helper()
	gitCLI(t, repo, "add", ".")
	gitCLI(t, repo, "-c", "user.name=Loaf Test", "-c", "user.email=loaf@example.test", "-c", "commit.gpgsign=false", "commit", "-m", "fixture")
}

func coverageRepo(t *testing.T) string {
	t.Helper()
	repo := initCLIGitRepo(t)
	mkdirAll(t, filepath.Join(repo, ".github"))
	writeFile(t, filepath.Join(repo, ".github", "PULL_REQUEST_TEMPLATE.md"), coverageDeclaration)
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog("- Old unrelated note."))
	writeFile(t, filepath.Join(repo, "package.json"), `{"name":"fixture","version":"1.0.0"}`+"\n")
	coverageCommit(t, repo)
	gitCLI(t, repo, "switch", "-c", "feature")
	return repo
}

func TestWorkflowPrePRDoesNotDemandGitHubShadowIssues(t *testing.T) {
	repo := initCLIGitRepo(t)
	mkdirAll(t, filepath.Join(repo, ".agents"))
	writeFile(t, filepath.Join(repo, ".agents", "loaf.json"), `{"integrations":{"linear":{"tracker":true},"github":{"repository":"example/code","tracker":false}}}`)
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog("- Fix retries (https://linear.app/example/issue/EX-1)."))
	result := runNativeWorkflowPrePR(checkHookContext{ToolName: "Bash", ToolInput: checkHookInput{Command: `gh pr create --title "fix: preserve retry state" --body "Implements https://linear.app/example/issue/EX-1"`}}, repo)
	if !result.Passed || result.Blocked || len(result.Errors) != 0 {
		t.Fatalf("non-opted consumer blocked: %+v", result)
	}
	// Removing an opted-in template locally cannot bypass the committed policy.
	opted := coverageRepo(t)
	if err := os.Remove(filepath.Join(opted, ".github", "PULL_REQUEST_TEMPLATE.md")); err != nil {
		t.Fatal(err)
	}
	if !repositoryUsesIssueChangelog(opted) {
		t.Fatal("local deletion bypassed tracked policy")
	}
	gitCLI(t, opted, "add", ".github")
	gitCLI(t, opted, "-c", "user.name=Loaf Test", "-c", "user.email=loaf@example.test", "-c", "commit.gpgsign=false", "commit", "-m", "fixture deletion")
	if !repositoryUsesIssueChangelog(opted, "main") {
		t.Fatal("candidate template deletion bypassed review-base policy")
	}
}

func coverageGitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(output))
}

func TestIssueChangelogCLIAndEventParityPreserveTree(t *testing.T) {
	repo := coverageRepo(t)
	body := filepath.Join(t.TempDir(), "body.md")
	writeFile(t, body, coverageDeclaration)
	for _, covered := range []bool{false, true} {
		if covered {
			writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog("- Require issue coverage "+coverageLink+"."))
			coverageCommit(t, repo)
		}
		base, head := coverageGitOutput(t, repo, "rev-parse", "main"), coverageGitOutput(t, repo, "rev-parse", "HEAD")
		event := map[string]any{"repository": map[string]any{"full_name": coverageRepository}, "pull_request": map[string]any{"body": coverageDeclaration, "base": map[string]any{"sha": base, "repo": map[string]any{"full_name": coverageRepository}}, "head": map[string]any{"sha": head}}}
		encoded, _ := json.Marshal(event)
		eventPath := filepath.Join(t.TempDir(), "event.json")
		writeFile(t, eventPath, string(encoded))
		before := coverageGitOutput(t, repo, "status", "--porcelain=v1")
		var results []string
		for _, args := range [][]string{
			{"--base", base, "--head", head, "--issue", coverageIssue},
			{"--base", base, "--head", head, "--pr-body-file", body},
			{"--event-file", eventPath},
		} {
			var output bytes.Buffer
			err := (Runner{WorkingDir: repo, Stdout: &output}).Run(append([]string{"check", "issue-changelog", "--repository", coverageRepository, "--json"}, args...))
			if (err == nil) != covered {
				t.Fatalf("covered=%v error=%v output=%s", covered, err, output.String())
			}
			results = append(results, output.String())
		}
		if results[0] != results[1] || results[1] != results[2] {
			t.Fatalf("local/body/event differ: %v", results)
		}
		if after := coverageGitOutput(t, repo, "status", "--porcelain=v1"); before != after {
			t.Fatal("validator changed Git state")
		}
	}
	// Read local edits without staging them; committed coverage remains unchanged.
	writeFile(t, filepath.Join(repo, "README.md"), "staged user work\n")
	gitCLI(t, repo, "add", "README.md")
	writeFile(t, filepath.Join(repo, "README.md"), "unstaged user work\n")
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog("- Unrelated local note."))
	before := coverageGitOutput(t, repo, "status", "--porcelain=v1")
	err := checkIssueChangelog(repo, issueChangelogRequest{coverageRepository, "main", "WORKTREE", implementedIssues{Issues: []string{coverageIssue}}})
	if err == nil {
		t.Fatal("WORKTREE accepted missing local coverage")
	}
	if err := checkIssueChangelog(repo, issueChangelogRequest{coverageRepository, "main", "HEAD", implementedIssues{Issues: []string{coverageIssue}}}); err != nil {
		t.Fatal(err)
	}
	if after := coverageGitOutput(t, repo, "status", "--porcelain=v1"); after != before {
		t.Fatal("staged/unstaged edits changed")
	}
}

func TestIssueChangelogMissingContext(t *testing.T) {
	repo := coverageRepo(t)
	for _, tc := range []struct{ name, base, head, repository string }{
		{"missing-base", "", "HEAD", coverageRepository},
		{"unavailable-base", "missing", "HEAD", coverageRepository},
		{"missing-head", "main", "", coverageRepository},
		{"unavailable-head", "main", "missing", coverageRepository},
		{"missing-repository", "main", "HEAD", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkIssueChangelog(repo, issueChangelogRequest{tc.repository, tc.base, tc.head, implementedIssues{Issues: []string{coverageIssue}}}); err == nil {
				t.Fatal("missing context accepted")
			}
		})
	}
	for _, args := range [][]string{
		{}, {"--issue", coverageIssue, "--pr-body-file", "body.md"},
		{"--release-only", "--issue", coverageIssue},
		{"--event-file", "event.json", "--base", "main"},
		{"--base", "main", "--base", "other", "--head", "HEAD", "--issue", coverageIssue},
		{"--base", "main", "-base", "other", "--head", "HEAD", "--issue", coverageIssue},
		{"-base", "main", "-base", "other", "--head", "HEAD", "--issue", coverageIssue},
	} {
		var output bytes.Buffer
		if err := (Runner{WorkingDir: repo, Stdout: &output}).Run(append([]string{"check", "issue-changelog", "--repository", coverageRepository, "--json"}, args...)); err == nil {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}

func TestWorkflowChangelogCreationInputs(t *testing.T) {
	repo := coverageRepo(t)
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog("- Require coverage "+coverageLink+"."))
	coverageCommit(t, repo)
	body := filepath.Join(t.TempDir(), "pr body.md")
	writeFile(t, body, coverageDeclaration)
	prefix := `gh pr create --repo levifig/loaf --base main --title "fix: require issue coverage" `
	for _, tc := range []struct {
		name, command string
		pass          bool
	}{
		{"body-file", prefix + `--body-file "` + body + `"`, true},
		{"literal-body", prefix + `--body '` + coverageDeclaration + `'`, true},
		{"filled-with-body", prefix + `--fill --body-file "` + body + `"`, true},
		{"filled-without-declaration", prefix + `--fill`, false},
		{"missing-base", `gh pr create --repo levifig/loaf --body '` + coverageDeclaration + `'`, false},
		{"missing-repository", `gh pr create --base main --body '` + coverageDeclaration + `'`, false},
		{"dynamic-file", prefix + `--body-file "$BODY"`, false},
		{"stdin", prefix + `--body-file -`, false},
		{"duplicate-body", prefix + `--body x --body y`, false},
		{"alternate-head", prefix + `--head other --body-file "` + body + `"`, false},
		{"shell-setup", `cd other && ` + prefix + `--body-file "` + body + `"`, false},
		{"wrapper", `env X=1 ` + prefix + `--body-file "` + body + `"`, false},
		{"invalid-title", `gh pr create --repo levifig/loaf --base main --title "fix" --body '` + coverageDeclaration + `'`, false},
		{"nonconventional-title", `gh pr create --repo levifig/loaf --base main --title "A descriptive title for this PR" --body '` + coverageDeclaration + `'`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := runNativeWorkflowPrePR(checkHookContext{ToolName: "Bash", ToolInput: checkHookInput{Command: tc.command}}, repo)
			if result.Passed != tc.pass {
				t.Fatalf("pass=%v result=%+v", tc.pass, result)
			}
		})
	}
	result := runNativeWorkflowPrePR(checkHookContext{ToolName: "Bash", ToolInput: checkHookInput{Command: `printf '%s' 'gh pr create'`}}, repo)
	if !result.Passed {
		t.Fatalf("quoted example checked: %+v", result)
	}
	// An uncommitted note cannot make the actual PR candidate pass.
	gitCLI(t, repo, "reset", "--soft", "main") // fixture-owned history/index only
	result = runNativeWorkflowPrePR(checkHookContext{ToolName: "Bash", ToolInput: checkHookInput{Command: prefix + `--body-file "` + body + `"`}}, repo)
	if result.Passed {
		t.Fatal("hook accepted an uncommitted note")
	}
}

func TestIssueChangelogReleaseOnlyMovement(t *testing.T) {
	for _, tc := range []struct {
		name, extra, notes, packageJSON string
		pass                            bool
	}{
		{"exact", "", "", `{"name":"fixture","version":"1.2.3"}`, true},
		{"changelog-only", "", "", "", true},
		{"implementation", "source.go", "", `{"name":"fixture","version":"1.2.3"}`, false},
		{"new-note", "", "- Newly implemented behavior.", `{"name":"fixture","version":"1.2.3"}`, false},
		{"other-package-change", "", "", `{"name":"changed","version":"1.2.3"}`, false},
		{"version-mismatch", "", "", `{"name":"fixture","version":"1.2.4"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := coverageRepo(t)
			before := coverageChangelog("- Old unrelated note.")
			parts, _ := splitIssueChangelog(before)
			after := parts.prefix + "\n- _No unreleased changes yet._\n\n## [1.2.3] - 2026-10-14\n\n" + strings.TrimSpace(parts.unreleased) + "\n"
			if tc.notes != "" {
				after += "\n" + tc.notes + "\n"
			}
			after += "\n" + parts.released
			writeFile(t, filepath.Join(repo, "CHANGELOG.md"), after)
			if tc.packageJSON != "" {
				writeFile(t, filepath.Join(repo, "package.json"), tc.packageJSON+"\n")
			}
			if tc.extra != "" {
				writeFile(t, filepath.Join(repo, tc.extra), "package source\n")
			}
			coverageCommit(t, repo)
			request := issueChangelogRequest{coverageRepository, "main", "HEAD", implementedIssues{Issues: []string{}, ReleaseOnly: true}}
			err := checkIssueChangelog(repo, request)
			if (err == nil) != tc.pass {
				t.Fatalf("pass=%v error=%v", tc.pass, err)
			}
			if tc.pass {
				request.declaration.ReleaseOnly = false
				if err := checkIssueChangelog(repo, request); err == nil {
					t.Fatal("empty ordinary declaration bypassed")
				}
			}
		})
	}
	// Release-looking subject and tag cannot excuse ordinary implementation.
	repo := coverageRepo(t)
	writeFile(t, filepath.Join(repo, "source.go"), "package source\n")
	coverageCommit(t, repo)
	gitCLI(t, repo, "-c", "tag.gpgSign=false", "tag", "v1.2.3")
	if err := checkIssueChangelog(repo, issueChangelogRequest{coverageRepository, "main", "HEAD", implementedIssues{ReleaseOnly: true}}); err == nil {
		t.Fatal("tag bypassed coverage")
	}
}

func TestIssueChangelogWorkflowBoundary(t *testing.T) {
	root := testRepositoryRoot(t)
	body, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "issue-changelog.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{"name: Issue changelog coverage", "pull_request:", "synchronize", "edited", "contents: read", "persist-credentials: false", `--event-file "$GITHUB_EVENT_PATH"`, `--repository "$GITHUB_REPOSITORY"`} {
		if !strings.Contains(text, required) {
			t.Fatalf("workflow missing %q", required)
		}
	}
	for _, forbidden := range []string{"paths-ignore:", "pull_request_target:", "secrets.", "write-all", "contents: write", "github.event.pull_request.body", "git commit", "git push"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("unsafe/filtered workflow %q", forbidden)
		}
	}
}

func TestIssueChangelogEventRequiresExactContext(t *testing.T) {
	repo := coverageRepo(t)
	base, head := coverageGitOutput(t, repo, "rev-parse", "main"), coverageGitOutput(t, repo, "rev-parse", "HEAD")
	valid := `{"repository":{"full_name":"levifig/loaf"},"pull_request":{"body":` + strconv.Quote(coverageDeclaration) + `,"base":{"sha":"` + base + `","repo":{"full_name":"levifig/loaf"}},"head":{"sha":"` + head + `"}}}`
	for _, tc := range []struct {
		name, body string
		pass       bool
	}{
		{"valid", valid, true},
		{"missing-PR", `{"repository":{"full_name":"levifig/loaf"}}`, false},
		{"wrong-repository", strings.Replace(valid, `"full_name":"levifig/loaf"`, `"full_name":"other/repo"`, 1), false},
		{"wrong-base-repository", strings.ReplaceAll(valid, `"repo":{"full_name":"levifig/loaf"}`, `"repo":{"full_name":"other/repo"}`), false},
		{"missing-body", strings.Replace(valid, strconv.Quote(coverageDeclaration), "null", 1), false},
		{"nonimmutable-revision", strings.Replace(valid, head, "HEAD", 1), false},
		{"malformed", `{"repository":`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "event.json")
			writeFile(t, path, tc.body)
			_, err := readIssueChangelogEvent(path, coverageRepository)
			if (err == nil) != tc.pass {
				t.Fatalf("pass=%v err=%v", tc.pass, err)
			}
		})
	}
}
