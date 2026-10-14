package cli

import (
	"fmt"
	"strings"
	"testing"
)

const coverageRepository = "levifig/loaf"
const coverageIssue = "https://github.com/levifig/loaf/issues/284"
const coverageLink = "([#284](https://github.com/levifig/loaf/issues/284))"
const coverageDeclaration = `<!-- loaf:implemented-issues {"issues":["https://github.com/levifig/loaf/issues/284"]} -->`

func coverageChangelog(notes string) string {
	return "# Changelog\n\n## [Unreleased]\n\n### Fixed\n\n" + notes + "\n\n## [1.0.0] - 2026-01-01\n\n- Initial release.\n"
}

func TestImplementedIssuesDeclaration(t *testing.T) {
	tests := []struct {
		name, body string
		valid      bool
	}{
		{"ordinary", coverageDeclaration, true},
		{"release", `<!-- loaf:implemented-issues {"issues":[],"release_only":true} -->`, true},
		{"missing", "Closes #284", false},
		{"empty", `<!-- loaf:implemented-issues {"issues":[]} -->`, false},
		{"missing-array", `<!-- loaf:implemented-issues {"release_only":true} -->`, false},
		{"null", `<!-- loaf:implemented-issues {"issues":null} -->`, false},
		{"malformed", `<!-- loaf:implemented-issues {"issues":[} -->`, false},
		{"unknown-field", `<!-- loaf:implemented-issues {"issues":[],"skip":true} -->`, false},
		{"duplicate-key", `<!-- loaf:implemented-issues {"issues":[],"issues":["` + coverageIssue + `"]} -->`, false},
		{"case-fold-key", `<!-- loaf:implemented-issues {"issues":["` + coverageIssue + `"],"ISSUES":["https://github.com/levifig/loaf/issues/285"]} -->`, false},
		{"null-release-flag", `<!-- loaf:implemented-issues {"issues":[],"release_only":null} -->`, false},
		{"trailing", strings.Replace(coverageDeclaration, "} -->", "} {} -->", 1), false},
		{"multiple", coverageDeclaration + coverageDeclaration, false},
		{"duplicate-url", `<!-- loaf:implemented-issues {"issues":["` + coverageIssue + `","` + coverageIssue + `"]} -->`, false},
		{"wrong-repository", strings.ReplaceAll(coverageDeclaration, "levifig/loaf", "other/loaf"), false},
		{"pull-request", strings.ReplaceAll(coverageDeclaration, "/issues/", "/pull/"), false},
		{"noncanonical", strings.ReplaceAll(coverageDeclaration, "/284", "/0284"), false},
		{"release-with-work", strings.Replace(coverageDeclaration, "} -->", `,"release_only":true} -->`, 1), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseImplementedIssues(tc.body, coverageRepository)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
	if _, err := parseImplementedIssues(coverageDeclaration, ""); err == nil {
		t.Fatal("missing repository accepted")
	}
}

func TestIssueChangelogCoverage(t *testing.T) {
	old := "- Fix an earlier bug " + coverageLink + "."
	base := coverageChangelog(old)
	declaration := implementedIssues{Issues: []string{coverageIssue}}
	tests := []struct {
		name, head string
		valid      bool
	}{
		{"new-linked", coverageChangelog(old + "\n- Enforce issue-linked coverage " + coverageLink + "."), true},
		{"refined", coverageChangelog("- Fix the earlier bug and preserve retry state " + coverageLink + "."), true},
		{"unchanged", base, false},
		{"format-only", coverageChangelog("* **Fix** an earlier\n  bug " + coverageLink + "!"), false},
		{"reference-only", coverageChangelog("- " + coverageLink), false},
		{"unrelated", coverageChangelog("- Enforce coverage ([#28](https://github.com/levifig/loaf/issues/28))."), false},
		{"wrong-repository", coverageChangelog("- Enforce coverage ([#284](https://github.com/other/loaf/issues/284))."), false},
		{"PR-url", coverageChangelog("- Enforce coverage ([#284](https://github.com/levifig/loaf/pull/284))."), false},
		{"label-mismatch", coverageChangelog("- Enforce coverage ([#28](https://github.com/levifig/loaf/issues/284))."), false},
		{"heading", coverageChangelog("### Fix coverage " + coverageLink), false},
		{"comment", coverageChangelog("<!--\n- Fix coverage " + coverageLink + "\n-->"), false},
		{"unterminated-comment", coverageChangelog("<!--\n- Fix coverage " + coverageLink), false},
		{"fence", coverageChangelog("```md\n- Fix coverage " + coverageLink + "\n```"), false},
		{"long-fence-with-short-example", coverageChangelog("````md\n```\n- Fix coverage " + coverageLink + "\n```\n````"), false},
		{"long-tilde-fence", coverageChangelog("~~~~md\n~~~\n- Fix coverage " + coverageLink + "\n~~~\n~~~~"), false},
		{"inline-code-with-short-delimiter", coverageChangelog("- Example ``one ` " + coverageLink + " ` two``."), false},
		{"inline-example", coverageChangelog("- Example `" + coverageLink + "`."), false},
		{"quote", coverageChangelog("> - Fix coverage " + coverageLink), false},
		{"released-only", coverageChangelog("") + "\n- Fix coverage " + coverageLink + ".\n", false},
		{"duplicate-heading", base + "\n## [Unreleased]\n- New coverage " + coverageLink, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIssueChangelogCoverage(base, tc.head, declaration)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
	// Adding a link to unchanged old prose is not a substantive update.
	if err := validateIssueChangelogCoverage(coverageChangelog("- Enforce coverage."), coverageChangelog("- Enforce coverage "+coverageLink+"."), declaration); err == nil {
		t.Fatal("reference-only edit accepted")
	}
	if err := validateIssueChangelogCoverage(coverageChangelog("- Fix `before` "+coverageLink+"."), coverageChangelog("- Fix `after` "+coverageLink+"."), declaration); err != nil {
		t.Fatalf("code-token refinement: %v", err)
	}
	if err := validateIssueChangelogCoverage(coverageChangelog("")+"\n- Copied released note "+coverageLink+".\n", coverageChangelog("- Copied released note "+coverageLink+"."), declaration); err == nil {
		t.Fatal("copied released prose accepted")
	}
	second := "https://github.com/levifig/loaf/issues/285"
	declaration.Issues = append(declaration.Issues, second)
	if err := validateIssueChangelogCoverage(base, coverageChangelog("- Enforce coverage "+coverageLink+"."), declaration); err == nil || !strings.Contains(err.Error(), second) {
		t.Fatalf("missing second issue: %v", err)
	}
	if err := validateIssueChangelogCoverage(base, coverageChangelog(fmt.Sprintf("- Enforce coverage %s ([#285](%s)).", coverageLink, second)), declaration); err != nil {
		t.Fatal(err)
	}
}
