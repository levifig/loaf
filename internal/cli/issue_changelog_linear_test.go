package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const linearCoverageIssue = "https://linear.app/example/issue/ENG-123"
const linearCoverageLink = "[ENG-123](" + linearCoverageIssue + ")"
const linearCoverageDeclaration = `<!-- loaf:implemented-issues {"issues":["https://linear.app/example/issue/ENG-123"]} -->`

func TestLinearImplementedIssuesDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		valid     bool
	}{
		{"canonical", linearCoverageIssue, true},
		{"hyphenated-workspace", "https://linear.app/example-team/issue/ENG2-123", true},
		{"http", "http://linear.app/example/issue/ENG-123", false},
		{"other-host", "https://linear.app.evil.test/example/issue/ENG-123", false},
		{"userinfo", "https://user@linear.app/example/issue/ENG-123", false},
		{"port", "https://linear.app:443/example/issue/ENG-123", false},
		{"missing-workspace", "https://linear.app/issue/ENG-123", false},
		{"other-path", "https://linear.app/example/project/ENG-123", false},
		{"lowercase-key", "https://linear.app/example/issue/eng-123", false},
		{"leading-zero", "https://linear.app/example/issue/ENG-0123", false},
		{"zero", "https://linear.app/example/issue/ENG-0", false},
		{"slug", linearCoverageIssue + "/example-title", false},
		{"query", linearCoverageIssue + "?x=1", false},
		{"fragment", linearCoverageIssue + "#example", false},
		{"trailing-slash", linearCoverageIssue + "/", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := implementedIssuesMarker + `{"issues":["` + tc.url + `"]} -->`
			_, err := parseImplementedIssues(body, coverageRepository)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
			if err != nil && !strings.Contains(err.Error(), "Linear") {
				t.Fatalf("unsupported input lacks provider guidance: %v", err)
			}
		})
	}
	for _, issues := range [][]string{{linearCoverageIssue, linearCoverageIssue}, {linearCoverageIssue, "https://github.com/other/repo/issues/123"}} {
		if err := validateImplementedIssues(implementedIssues{Issues: issues}, coverageRepository); err == nil {
			t.Fatalf("invalid declaration accepted: %v", issues)
		}
	}
	if err := validateImplementedIssues(implementedIssues{Issues: []string{linearCoverageIssue, coverageIssue}}, coverageRepository); err != nil {
		t.Fatal(err)
	}
	if err := validateImplementedIssues(implementedIssues{Issues: []string{linearCoverageIssue}}, ""); err == nil {
		t.Fatal("missing hosting repository accepted")
	}
}

func TestLinearIssueChangelogCoverage(t *testing.T) {
	declaration := implementedIssues{Issues: []string{linearCoverageIssue}}
	old := "- Preserve an older behavior " + linearCoverageLink + "."
	for _, tc := range []struct {
		name, base, head string
		valid            bool
	}{
		{"new", "- An older note.", "- Validate declared coverage " + linearCoverageLink + ".", true},
		{"parenthesized", "", "- Validate declared coverage (" + linearCoverageLink + ").", true},
		{"refined", old, "- Preserve an older behavior and validate retries " + linearCoverageLink + ".", true},
		{"unchanged", old, old, false},
		{"format-only", old, "* **Preserve** an older\n  behavior (" + linearCoverageLink + ")!", false},
		{"reference-only", "", "- (" + linearCoverageLink + ").", false},
		{"attached-link", "- Validate declared coverage.", "- Validate declared coverage (" + linearCoverageLink + ").", false},
		{"swapped-reference", "- Validate declared coverage " + coverageLink + ".", "- Validate declared coverage " + linearCoverageLink + ".", false},
		{"other-issue", "", "- Validate coverage [ENG-124](https://linear.app/example/issue/ENG-124).", false},
		{"other-workspace", "", "- Validate coverage [ENG-123](https://linear.app/another/issue/ENG-123).", false},
		{"label-mismatch", "", "- Validate coverage [ENG-124](" + linearCoverageIssue + ").", false},
		{"bare-url", "", "- Validate coverage " + linearCoverageIssue + ".", false},
		{"raw-html-block", "", "<div>\n- Correct behavior " + linearCoverageLink + "\n</div>", false},
		{"comment-in-url", "", "- Correct behavior [ENG-123](https://linear.app/example/issue/ENG-<!-- comment -->123).", false},
		{"comment-in-label", "", "- Correct behavior [ENG-<!-- comment -->123](" + linearCoverageIssue + ").", false},
		{"comment-in-prose", "", "- Correct <!-- harmless --> behavior " + linearCoverageLink + ".", true},
		{"unresolved-reference-image", "", "- Correct behavior ![See " + linearCoverageLink + "][diagram].", true},
		{"less-than-prose", "", "- Fix values < 0 " + linearCoverageLink + ".", true},
		{"close-paren-in-image-title", "", `- Correct behavior ![diagram](image.png ") ` + linearCoverageLink + `").`, false},
		{"open-paren-in-image-title", "", `- Correct behavior ![diagram](image.png "(") ` + linearCoverageLink + ".", true},
		{"other-link-title", "", `- Correct behavior [guide](guide.md "` + linearCoverageLink + `").`, false},
		{"noncanonical-title", "", `- Correct behavior [ENG-123](` + linearCoverageIssue + ` "title").`, false},
		{"noncanonical-angle-destination", "", "- Correct behavior [ENG-123](<" + linearCoverageIssue + ">).", false},
		{"noncanonical-space", "", "- Correct behavior [ENG-123]( " + linearCoverageIssue + ").", false},
		{"noncanonical-entity", "", "- Correct behavior [ENG-123](https://linear.app/example/issue/ENG&#45;123).", false},
		{"reference-link", "", "- Correct behavior [ENG-123][issue].\n\n[issue]: " + linearCoverageIssue, false},
		{"nested-image", "", "- Correct behavior ![See " + linearCoverageLink + "](diagram.png).", false},
		{"escaped-image-marker", "", `- Correct behavior \!` + linearCoverageLink + ".", true},
		{"even-backslashes", "", `- Correct behavior \\` + linearCoverageLink + ".", true},
		{"html-attribute", "", `- Correct behavior <span title="` + linearCoverageLink + `">details</span>.`, false},
		{"html-attribute-quoted-close", "", `- Correct behavior <span title="> ` + linearCoverageLink + `">details</span>.`, false},
		{"link-after-image", "", "- Correct behavior ![diagram](diagram.png) " + linearCoverageLink + ".", true},
		{"image-title", "", `- Correct behavior ![diagram](image.png "` + linearCoverageLink + `").`, false},
		{"reference-image", "", "- Correct behavior ![See " + linearCoverageLink + "][diagram].\n\n[diagram]: diagram.png", false},
		{"image", "", "- Illustration !" + linearCoverageLink, false},
		{"escaped-link", "", "- Example \\" + linearCoverageLink + ".", false},
		{"inline-code", "", "- Example `" + linearCoverageLink + "`.", false},
		{"fence", "", "```md\n- Example " + linearCoverageLink + "\n```", false},
		{"comment", "", "<!--\n- Example " + linearCoverageLink + "\n-->", false},
		{"heading", "", "### Example " + linearCoverageLink, false},
		{"quote", "", "> - Example " + linearCoverageLink, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIssueChangelogCoverage(coverageChangelog(tc.base), coverageChangelog(tc.head), declaration)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
	copied := "- Preserve released behavior " + linearCoverageLink + "."
	if err := validateIssueChangelogCoverage(coverageChangelog("")+"\n"+copied, coverageChangelog(copied), declaration); err == nil {
		t.Fatal("copied released note accepted")
	}
	declaration.Issues = append(declaration.Issues, coverageIssue)
	if err := validateIssueChangelogCoverage(coverageChangelog(""), coverageChangelog("- Validate coverage "+linearCoverageLink+"."), declaration); err == nil {
		t.Fatal("missing GitHub coverage accepted")
	}
	if err := validateIssueChangelogCoverage(coverageChangelog(""), coverageChangelog("- Validate coverage "+linearCoverageLink+" "+coverageLink+"."), declaration); err != nil {
		t.Fatal(err)
	}
}

func TestLinearIssueChangelogRealGitEntryPointParity(t *testing.T) {
	for _, tc := range []struct {
		name, notes string
		valid       bool
	}{
		{"covered", "- Validate real coverage (" + linearCoverageLink + ").", true},
		{"missing", "", false},
		{"unchanged", "- Old unrelated note.", false},
		{"unrelated", "- Validate other coverage [ENG-124](https://linear.app/example/issue/ENG-124).", false},
		{"mismatched", "- Validate coverage [ENG-124](" + linearCoverageIssue + ").", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := coverageRepo(t)
			writeFile(t, filepath.Join(repo, "CHANGELOG.md"), coverageChangelog(tc.notes))
			writeFile(t, filepath.Join(repo, "candidate.txt"), tc.name)
			coverageCommit(t, repo)
			base, head := coverageGitOutput(t, repo, "rev-parse", "main"), coverageGitOutput(t, repo, "rev-parse", "HEAD")
			body := filepath.Join(t.TempDir(), "body.md")
			writeFile(t, body, linearCoverageDeclaration)
			event := map[string]any{"repository": map[string]any{"full_name": coverageRepository}, "pull_request": map[string]any{"body": linearCoverageDeclaration, "base": map[string]any{"sha": base, "repo": map[string]any{"full_name": coverageRepository}}, "head": map[string]any{"sha": head}}}
			encoded, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			eventPath := filepath.Join(t.TempDir(), "event.json")
			writeFile(t, eventPath, string(encoded))
			writeFile(t, filepath.Join(repo, "user.txt"), "staged owner work")
			gitCLI(t, repo, "add", "user.txt")
			writeFile(t, filepath.Join(repo, "user.txt"), "unstaged owner work")
			before := coverageGitOutput(t, repo, "status", "--porcelain=v1")
			var outputs []string
			for _, args := range [][]string{
				{"--base", base, "--head", head, "--issue", linearCoverageIssue},
				{"--base", base, "--head", head, "--pr-body-file", body},
				{"--event-file", eventPath},
			} {
				var output bytes.Buffer
				err := (Runner{WorkingDir: repo, Stdout: &output}).Run(append([]string{"check", "issue-changelog", "--repository", coverageRepository, "--json"}, args...))
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%v error=%v output=%s", tc.valid, err, output.String())
				}
				outputs = append(outputs, output.String())
			}
			if outputs[0] != outputs[1] || outputs[1] != outputs[2] {
				t.Fatalf("entry points differ: %v", outputs)
			}
			result := runNativeWorkflowPrePR(checkHookContext{ToolName: "Bash", ToolInput: checkHookInput{Command: `gh pr create --repo levifig/loaf --base main --title "fix: validate coverage" --body-file "` + body + `"`}}, repo)
			if result.Passed != tc.valid || result.Blocked == tc.valid {
				t.Fatalf("pre-PR result=%+v valid=%v", result, tc.valid)
			}
			if after := coverageGitOutput(t, repo, "status", "--porcelain=v1"); before != after {
				t.Fatal("owner Git state changed")
			}
		})
	}
}

func TestGitHubCoverageRequiresCanonicalRealLink(t *testing.T) {
	declaration := implementedIssues{Issues: []string{coverageIssue}}
	for _, note := range []string{
		"- Correct behavior [#284](" + coverageIssue + ").",
		"- Example `" + coverageLink + "` and real [#284](" + coverageIssue + ").",
		"- Correct behavior ![See " + coverageLink + "](diagram.png).",
		"- Correct behavior <span title=\"" + coverageLink + "\">details</span>.",
	} {
		if err := validateIssueChangelogCoverage(coverageChangelog(""), coverageChangelog(note), declaration); err == nil {
			t.Fatalf("noncanonical or non-link coverage accepted: %s", note)
		}
	}
}

func TestLinearCoverageKeepsEntryAndSourceBoundaries(t *testing.T) {
	declaration := implementedIssues{Issues: []string{linearCoverageIssue}}
	old := "- Parent behavior.\n  - Existing behavior " + linearCoverageLink + "."
	if err := validateIssueChangelogCoverage(coverageChangelog(old), coverageChangelog(old+"\n\n- Existing behavior."), declaration); err == nil {
		t.Fatal("unchanged nested link credited a new unlinked entry")
	}
	head := coverageChangelog("- Correct behavior `"+linearCoverageLink+"`.") + "\n- Correct behavior " + linearCoverageLink + ".\n"
	if err := validateIssueChangelogCoverage(coverageChangelog(""), head, declaration); err == nil {
		t.Fatal("released link credited an Unreleased code example")
	}
	base := coverageChangelog("- Old unrelated note.")
	head = coverageChangelog("- Correct new behavior " + linearCoverageLink + ".")
	if err := validateIssueChangelogCoverage(strings.ReplaceAll(base, "\n", "\r\n"), strings.ReplaceAll(head, "\n", "\r\n"), declaration); err != nil {
		t.Fatalf("CRLF source: %v", err)
	}
}
