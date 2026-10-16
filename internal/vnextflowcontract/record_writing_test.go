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

// A merge can close linked issues automatically, so ship confirms every
// Verification box before it closes or merges, and implement ticks only the
// boxes whose checks it ran.
func TestVerificationBoxesGateClosingAndMerging(t *testing.T) {
	t.Parallel()

	content := repositoryFlowContent(t)
	files := map[string][]string{
		recordWritingPath: {
			"Whoever ran the check ticks its box",
			"only after the check ran in its own work",
			"the result is recorded, naming the commit it ran on, where reviewers can find it",
			"stays for that person",
			"carries its reason at the end of its line",
			"A reason records a person's decision",
			"An agent never adds one on its own to get past a gate",
			"can only run after the merge, such as a deploy, is written with its reason from the start",
			"before the issue is marked done or a PR that closes it is merged",
			"it never counts as the check passing, so the review lists each such box as an open limitation",
			"re-read it, change only that box, then read it back",
		},
		"skills/ship/SKILL.md": {
			"confirm every Verification box is ticked",
			"unless its reason is written on the box",
			"name each open box",
			"stays for that person",
			"Never write a reason yourself to clear the gate",
			"before the merge, not after",
			"A tick counts only with a recorded result for the reviewed candidate",
			"changing a box's tick is the only definition write ship makes",
			"re-read the PR and the Verification boxes on each issue it closes",
			"if one is open without a reason, name it and report the issue as not complete instead of transitioning",
			"| Approve | Criteria, boundaries, and quality gates are proven, and every Verification box is ticked or carries its reason on the box.",
			"a Verification box is open with no reason on it and ship cannot run its check",
			"list it as an open limitation in the disposition, never as proof, and do not withhold Approve for it alone",
			"clear an agent's tick on a person-only box and leave it for that person",
			"every Verification box on each issue the merge closes or ship marks Done was ticked with a recorded result for the reviewed candidate",
		},
		"skills/implement/SKILL.md": {
			"tick each Verification box whose check you ran and passed",
			"Ticking is the only definition write implement makes",
			"open for their owner, and never write a reason on a box yourself",
			"Each Verification box whose check this work ran and passed is ticked with its result recorded",
		},
	}
	for name, wants := range files {
		body := readFlowContentString(t, content, name)
		for _, want := range wants {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing Verification-box rule %q", name, want)
			}
		}
	}
	// "Explained" is weaker than a reason written on the box and would also
	// loosen criteria that must stay proven.
	if ship := readFlowContentString(t, content, "skills/ship/SKILL.md"); strings.Contains(ship, "proven or explained") {
		t.Error("ship lets an explanation stand in for proof or a reason on the box")
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
