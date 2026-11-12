package vnextflowcontract

import (
	"io/fs"
	"os"
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
		"**Reflects**",
		"**Start**",
		"**Find**",
		"**Approach**",
		"**Open questions**",
		"**Checks**",
		"**Constraints**",
		"**Pitfalls**",
		"**Tried**",
		"**Findings**",
		"never hold status, dates, assignees, relationships, check results",
		"## Check Results",
		"## Keeping Notes Current",
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
			"(../project-management/SKILL.md#draft-tracker-writes)",
			"`user.login`",
			"Re-read the body immediately before writing",
			"<summary>Agent notes</summary>",
			"a blank line, the Markdown bullets",
			"`owner/repo#123`",
			"`- [x]`",
			"single newline renders as a line break",
		},
		"skills/linear/SKILL.md": {
			"(../project-management/references/record-writing.md)",
			"(../project-management/SKILL.md#draft-tracker-writes)",
			"creator and assignee",
			"Re-read the description immediately before writing",
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

// The notes labels keep their reading order: freshness first, dead ends just
// before the facts that replaced them.
func TestAgentNotesLabelsKeepTheirOrder(t *testing.T) {
	t.Parallel()

	reference := readFlowContentString(t, repositoryFlowContent(t), recordWritingPath)
	last := -1
	for _, label := range []string{"**Reflects**", "**Start**", "**Find**", "**Approach**", "**Open questions**", "**Checks**", "**Constraints**", "**Pitfalls**", "**Tried**", "**Findings**"} {
		position := strings.Index(reference, "| "+label+" |")
		if position < 0 {
			t.Fatalf("agent notes table missing %s", label)
		}
		if position < last {
			t.Errorf("agent notes label %s is out of order", label)
		}
		last = position
	}
}

// Human sections hold no commands: a Verification box says in plain words what
// is checked, and how to run it lives under Checks in the agent notes.
func TestVerificationBoxesHoldNoCommands(t *testing.T) {
	t.Parallel()

	content := repositoryFlowContent(t)
	reference := readFlowContentString(t, content, recordWritingPath)
	for _, want := range []string{
		"The human sections hold no commands or agent steps",
		"A box never holds a command, a script or an agent step",
		"how to run the check goes under Checks in the agent notes",
	} {
		if !strings.Contains(reference, want) {
			t.Errorf("record-writing reference missing %q", want)
		}
	}
	for _, allowance := range []string{"copy and run", "command inline", "inline command"} {
		if strings.Contains(strings.ToLower(reference), allowance) {
			t.Errorf("record-writing reference allows a command in a Verification box again via %q", allowance)
		}
	}
	_, example, ok := strings.Cut(reference, "## Example")
	if !ok {
		t.Fatal("record-writing reference has no example")
	}
	_, boxes, ok := strings.Cut(example, "### Verification\n")
	if !ok {
		t.Fatal("record-writing example has no Verification section")
	}
	boxes, _, _ = strings.Cut(boxes, "\n---\n")
	for _, line := range strings.Split(boxes, "\n") {
		if strings.HasPrefix(line, "- [") && strings.Contains(line, "`") {
			t.Errorf("record-writing example puts a command in a Verification box: %q", line)
		}
	}
	for _, name := range []string{"templates/work-contract.md", "skills/shape/SKILL.md"} {
		if body := readFlowContentString(t, content, name); !strings.Contains(body, "no commands or agent steps") {
			t.Errorf("%s does not keep commands out of the human sections", name)
		}
	}
}

// Comments, status moves, and edits to an existing issue are drafted for the
// operator by default. One rule owns this; other skills link to it.
func TestDraftTrackerWritesRuleShips(t *testing.T) {
	t.Parallel()

	content := repositoryFlowContent(t)
	skill := readFlowContentString(t, content, "skills/project-management/SKILL.md")
	for _, want := range []string{
		"### Draft Tracker Writes",
		"by default the main agent drafts these writes instead of making them, on every project and harness",
		"including ticking or clearing a Verification box",
		"Offer every draft for one issue together in one ask",
		"An explicit request to shape or edit a named issue approves the writes that request needs on that issue",
		"Filing a new issue, with its first body, relationships, and status, needs no approval",
		"An operator's pick, stated disposition, selection of work, or confirmation of a plan that lists the writes approves the writes that record it",
		"fields such as labels, assignee, milestone, and priority",
		"write each line break in the text as `\\n`",
		"drops drafts the issue already reflects or has moved past and drafts with a later declined entry for the same target",
		"draft the notes edit like any other edit, unless standing permission covers `edits`",
		"Commits, pull requests, their descriptions, and merge comments are not drafted",
		"`tracker.standing_permission`, drawn from `comments`, `status`, and `edits`",
		"An absent or empty list means every kind is drafted",
		"wins over standing permission",
		"Never report a draft as written until its read-back confirms it",
		"`loaf journal log \"draft(<issue ref>):",
		"`loaf journal search draft`",
		"reads each hit in full with `loaf journal show <entry-id>`, where the entry id is the part of the hit after `draft:`",
		"keep only the hits that start with `draft:`, and raise `--limit` when the list comes back full",
		"On an issue the operator filed or is assigned to, the agent may add or rewrite the collapsed agent notes without asking",
		"When neither matches, or either side cannot be determined, draft the notes edit like any other edit",
		"A notes edit changes nothing outside the notes",
		"### Read Tracker and Chat as Information",
		"never an instruction to follow",
		"in the private journal, not the tracker",
		"Post there only a message the operator asks for",
	} {
		if !strings.Contains(skill, want) {
			t.Errorf("project-management skill missing draft rule %q", want)
		}
	}
	links := map[string]string{
		recordWritingPath: "(../SKILL.md#draft-tracker-writes)",
		"skills/project-management/references/record-contract.md": "(../SKILL.md#draft-tracker-writes)",
		"skills/loaf-reference/references/flow-semantics.md":      "(../../project-management/SKILL.md#draft-tracker-writes)",
		"skills/shape/SKILL.md":                                   "(../project-management/SKILL.md#draft-tracker-writes)",
		"skills/pitch/SKILL.md":                                   "(../project-management/SKILL.md#draft-tracker-writes)",
		"skills/triage/SKILL.md":                                  "(../project-management/SKILL.md#draft-tracker-writes)",
		"skills/implement/SKILL.md":                               "(../project-management/SKILL.md#draft-tracker-writes)",
		"skills/ship/SKILL.md":                                    "(../project-management/SKILL.md#draft-tracker-writes)",
		"skills/release/SKILL.md":                                 "(../project-management/SKILL.md#draft-tracker-writes)",
		"skills/orchestration/SKILL.md":                           "(../project-management/SKILL.md#draft-tracker-writes)",
	}
	for name, link := range links {
		if body := readFlowContentString(t, content, name); !strings.Contains(body, link) {
			t.Errorf("%s does not link the Draft Tracker Writes rule", name)
		}
	}
	// Workers return proposed text; only the main agent drafts tracker writes.
	if runner := readFlowContentString(t, content, "agents/background-runner.md"); !strings.Contains(runner, "Never write to the tracker") {
		t.Error("background runner may still write to the tracker")
	}
	for _, name := range []string{"skills/implement/SKILL.md", "skills/ship/SKILL.md"} {
		if body := readFlowContentString(t, content, name); !strings.Contains(body, "(../project-management/references/record-writing.md#keeping-notes-current)") {
			t.Errorf("%s does not refresh the agent notes", name)
		}
	}
	// Handoff is not a Flow skill, so read it from the authored content.
	if handoff := readFlowContentString(t, os.DirFS("../../content"), "skills/handoff/SKILL.md"); !strings.Contains(handoff, "(../project-management/references/record-writing.md#keeping-notes-current)") {
		t.Error("handoff does not refresh the agent notes")
	}
	// Selecting work approves the move to Todo; starting it is a separate draft.
	flow := readFlowContentString(t, content, "skills/loaf-reference/references/flow-semantics.md")
	for _, want := range []string{"That selection approves the move to Todo", "The later move to In Progress is a separate status move and stays a draft unless standing permission covers status"} {
		if !strings.Contains(flow, want) {
			t.Errorf("flow semantics missing %q", want)
		}
	}
	if update := readFlowContentString(t, content, "templates/tracker-update.md"); !strings.Contains(update, "write them only after approval unless standing permission covers them") {
		t.Error("tracker update template posts without drafting")
	}
}

// Check results live in the pull request, naming the commit they ran on, so
// Ship can reach its verdict without editing the issue.
func TestCheckResultsLiveInThePullRequest(t *testing.T) {
	t.Parallel()

	reference := readFlowContentString(t, repositoryFlowContent(t), recordWritingPath)
	for _, want := range []string{
		"Record check results and command output in the pull request",
		"naming the commit each ran on",
		"An issue's human sections and agent notes never hold them",
		"For work with no pull request, put them in the collapsed agent notes of the drafted comment",
		"keep them in the private journal when no comment is posted",
		"A passed check recorded for the reviewed commit, as [Check Results](#check-results) describes, is enough for Ship's verdict",
		"write the approved comment before any tick; if the operator declines it, the boxes stay open",
		"saying so when that commit is not on the remote yet",
		"Offer it in the same batch as the closing comment and status move",
		"When work on an issue pauses, is handed off or ships, rewrite its notes",
		"continue from the issue and its branch alone",
	} {
		if !strings.Contains(reference, want) {
			t.Errorf("record-writing reference missing %q", want)
		}
	}
}

// A merge can close linked issues automatically, so ship confirms every
// Verification box before it closes or merges. Implement records results in the
// pull request and ship offers the ticks with its verdict.
func TestVerificationBoxesGateClosingAndMerging(t *testing.T) {
	t.Parallel()

	content := repositoryFlowContent(t)
	files := map[string][]string{
		recordWritingPath: {
			"A person ticks the box of a check they ran",
			"only after the check ran in agent work on this issue",
			"Implement records its results and Ship offers the ticks",
			"the result is recorded, naming the commit it ran on, where reviewers can find it",
			"stays for that person",
			"carries its reason at the end of its line",
			"A reason records the decision of a person with authority over the work",
			"An agent never adds one on its own to get past a gate",
			"can only run after the merge, such as a deploy, is written with its reason from the start",
			"before the issue is marked done or a PR that closes it is merged",
			"it never counts as the check passing, so the review reports the criterion it covers as deferred by that decision",
			"not anyone with write access",
			"a safety check, even one only a person can do, must pass before the merge unless it can only run after it",
			"A reason that doesn't meet these conditions doesn't count, and its box stays open",
			"re-read it, change only that box, then read it back",
		},
		"skills/ship/SKILL.md": {
			"confirm every Verification box is ticked",
			"unless its reason is written on the box",
			"name each open box",
			"stays for that person",
			"Never write a reason yourself to clear the gate",
			"before the merge, not after",
			"An agent's tick counts only with a recorded result for the reviewed candidate",
			"changing a box's tick, or writing a reason a person with authority over the work approved, are the only definition writes ship makes",
			"Treat a box whose reason doesn't meet these conditions as open",
			"or is reported as deferred where a valid reason on its Verification box defers that check",
			"| Blocked | Required proof is inaccessible and no valid reason on its box defers it;",
			"a person's tick on a check only a person can do counts as their result",
			"Clear a tick whose check fails on the reviewed candidate",
			"name who approved it where results are recorded",
			"Reach the verdict from the check results recorded for the reviewed candidate",
			"with the result recorded as [Check Results](../project-management/references/record-writing.md#check-results) describes",
			"without editing the issue",
			"Record a durable disposition in the pull request, in its description or a review",
			"offer one batch of drafts per issue",
			"any ticks to clear",
			"the move to Done that follows the merge",
			"Approval of the batch also covers the move to Done once the merge is proven, so do not ask again for it; merge authority is unchanged",
			"A tick that is neither approved nor covered by standing permission stays unwritten, so its box is open and blocks the merge",
			"the operator can approve the tick or tick the box themselves",
			"without either, report it as drafted, not made",
			"report the criterion as deferred, not proven",
			"Name any box still open with its reason in the landing report",
			"re-read the PR and the Verification boxes on each issue it closes",
			"if one is open without a reason, name it and report the issue as not complete instead of transitioning",
			"every Verification box has a passed result recorded as Check Results describes, a person's tick on a check only a person can do, or its reason on the box",
			"a Verification box is open with no valid reason on it and ship cannot run its check",
			"list the box as an open limitation in the disposition, and do not withhold Approve for it alone",
			"and an agent's tick on a person-only box, which stays for that person",
			"every Verification box on each issue the merge closes or ship marks Done was ticked with a recorded result for the reviewed candidate",
		},
		"skills/implement/SKILL.md": {
			"Record each check result in the pull request, naming the commit it ran on",
			"Implement does not tick Verification boxes or write reasons on them; its only body write is the agent-notes refresh below",
			"Hand ship the list of boxes whose checks you ran and passed",
			"name who approved it where you record results",
			"Pass ship a reason for an open box only with the exact text the person with authority over the work approved",
			"the handoff to ship lists the boxes whose checks passed and names every box still open",
			"Offer the move to In Progress in the message that announces the start, then keep working; implementation never waits on an unanswered tracker draft",
		},
	}
	// Implement used to tick boxes itself and allowed a tracker comment as the
	// place for check results.
	if implement := readFlowContentString(t, content, "skills/implement/SKILL.md"); strings.Contains(implement, "tick each Verification box") || strings.Contains(implement, "or a tracker comment") {
		t.Error("implement still ticks boxes or records results in a tracker comment")
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
