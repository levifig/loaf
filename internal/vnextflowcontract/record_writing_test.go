package vnextflowcontract

import (
	"io/fs"
	"strings"
	"testing"
)

const recordWritingPath = "skills/project-management/references/record-writing.md"

// People read tracker records first: the problem opens the record without a
// heading and every other field is a third-level section. The field markers
// keep their contract order, which the template validator checks separately.
func TestWorkContractTemplateOpensWithUnheadedProblem(t *testing.T) {
	t.Parallel()

	body := readFlowContentString(t, repositoryFlowContent(t), "templates/work-contract.md")
	_, afterMarker, ok := strings.Cut(body, "<!-- loaf:field problem -->\n")
	if !ok {
		t.Fatal("work contract is missing the problem field marker")
	}
	if !strings.HasPrefix(afterMarker, "{{problem}}") {
		t.Fatalf("problem must open the record without a heading, got %q", firstLine(afterMarker))
	}
	for _, heading := range []string{"### Definition of Done", "### Out of Scope", "### Verification", "### Risks"} {
		if !strings.Contains(body, "\n"+heading+"\n") {
			t.Errorf("work contract missing section heading %q", heading)
		}
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			t.Errorf("work contract renders second-level heading %q; record sections use ###", line)
		}
	}
}

func TestTrackerUpdateTemplateKeepsRoutingFieldsOutOfTheComment(t *testing.T) {
	t.Parallel()

	body := readFlowContentString(t, repositoryFlowContent(t), "templates/tracker-update.md")
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			t.Errorf("tracker update renders heading %q; comments lead with plain lines", line)
		}
	}
	for _, field := range []string{"native_ref", "status_intent"} {
		_, after, ok := strings.Cut(body, "<!-- loaf:field "+field+" -->\n")
		if !ok || !strings.Contains(firstLine(after), "routing only") {
			t.Errorf("tracker update field %q must be marked routing only, got %q", field, firstLine(after))
		}
	}
}

func TestRecordWritingReferenceAndProviderSyntaxShip(t *testing.T) {
	t.Parallel()

	content := repositoryFlowContent(t)
	if skill := readFlowContentString(t, content, "skills/project-management/SKILL.md"); !strings.Contains(skill, "(references/record-writing.md)") {
		t.Fatal("project-management skill does not link the record-writing reference")
	}
	reference := readFlowContentString(t, content, recordWritingPath)
	for _, want := range []string{
		"### Steps to reproduce",
		"### Definition of Done",
		"### Out of Scope",
		"### Verification",
		"### Risks",
		"**Start**",
		"**Find**",
		"**Approach**",
		"**Open questions**",
		"**Checks**",
		"**Constraints**",
		"**Pitfalls**",
		"**Findings**",
		"never hold status, dates, assignees, relationships",
		"`YYYY-MM-DD HH:MM:SS` in UTC",
		"only in PR descriptions and commit messages",
		"everyone with tracker access can reach",
	} {
		if !strings.Contains(reference, want) {
			t.Errorf("record-writing reference missing %q", want)
		}
	}

	providers := map[string][]string{
		"skills/github/SKILL.md": {
			"(../project-management/references/record-writing.md)",
			"<summary>Agent notes</summary>",
			"a blank line, the Markdown bullets",
			"`owner/repo#123`",
			"`- [x]`",
			"single newline renders as a line break",
		},
		"skills/linear/SKILL.md": {
			"(../project-management/references/record-writing.md)",
			"`>>> Agent notes`",
			"`+++`",
			"`- [X]`",
			"collapses runs of blank lines",
		},
	}
	for name, wants := range providers {
		body := readFlowContentString(t, content, name)
		for _, want := range wants {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing record syntax %q", name, want)
			}
		}
	}
}

func readFlowContentString(t *testing.T, content fs.FS, name string) string {
	t.Helper()
	body, err := fs.ReadFile(content, name)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}
