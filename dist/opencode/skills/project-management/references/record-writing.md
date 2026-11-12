# Record Writing

Write tracker records for the people who read them. A product owner or teammate should see what an issue is for, why it matters and whether it's done without reading agent detail. These rules hold on every tracker; the provider skill gives that tracker's syntax for collapsed sections, links, checkboxes and line breaks. Whether a write is drafted for the operator first is the project-management skill's [Draft Tracker Writes](../SKILL.md#draft-tracker-writes) rule.

## Contents

- Issues
- Verification Boxes
- Check Results
- Agent Notes
- Keeping Notes Current
- Comments
- Formatting
- Links and Files
- Relationships and Magic Words
- Example

## Issues

Open with one to three sentences and no heading: where the issue came from, what's wrong or missing, and why it matters. Also record the source natively when the tracker can: an attached PR, a relation, a customer request or a type label.

Then use these sections in this order, and leave out any that would be empty:

| Section | Holds |
|---------|-------|
| `### Steps to reproduce` | Bugs only: numbered steps, then what happened and what should have happened |
| `### Definition of Done` | Plain bullets describing observable outcomes, not tasks, files or functions |
| `### Out of Scope` | What this work deliberately leaves alone |
| `### Verification` | Checkboxes for the checks that prove the outcomes, in plain words |
| `### Risks` | What could go wrong and how it would show |

The human sections hold no commands or agent steps; those go in the agent notes. Keep scheduling and workflow out of the body. Owners, status, priority, estimates, due dates and milestones live in the tracker's own fields.

## Verification Boxes

Write each box as one check a teammate could judge, described in plain words. A box never holds a command, a script or an agent step; how to run the check goes under Checks in the agent notes.

Tick a box only when its check has passed:

- Whoever ran the check ticks its box. An agent ticks one only after the check ran in its own work, directly or through a worker it delegated to, and the result is recorded, naming the commit it ran on, where reviewers can find it, as [Check Results](#check-results) describes.
- A check only a person can do, such as a customer sign-off, stays for that person.
- A box that stays open carries its reason at the end of its line, for example `- [ ] The customer confirms the new export (open: waiting on their test run)`. A reason records the decision of a person with authority over the work, such as whoever selected it or owns its acceptance, not anyone with write access: they wrote it, or approved the exact text an agent then wrote for them. A reason can defer only a check that cannot run before the merge or that only a person can do. An objective check that can run now must run, and a safety check, even one only a person can do, must pass before the merge unless it can only run after it. A reason that doesn't meet these conditions doesn't count, and its box stays open. Whoever writes an approved reason names the approver where results are recorded. An agent never adds one on its own to get past a gate.
- A check that can only run after the merge, such as a deploy, is written with its reason from the start, for example `(open: checked after the merge)`, and ticked by whoever runs it once it has run.
- Ticking edits the issue body, so an agent's tick is a drafted edit unless standing permission covers edits. Offer it in the same batch as the closing comment and status move. When writing it, re-read it, change only that box, then read it back.

Every box is ticked, or carries its reason, before the issue is marked done or a PR that closes it is merged. A passed check recorded in the PR for the reviewed commit is enough for Ship's verdict; the tick itself still has to be written before that close or merge. A reason lets the work land before that check passes; it never counts as the check passing, so the review reports the criterion it covers as deferred by that decision and lists the box as an open limitation. Ship checks this before it closes or merges.

## Check Results

Record check results and command output in the pull request, in its description or a review comment, naming the commit each ran on. Neither the human sections nor the agent notes hold them: the notes say how to run a check, not what it returned. For work with no pull request, put them in the collapsed agent notes of the drafted comment that reports the work, or keep them in the private journal when no comment is posted.

## Agent Notes

Detail only agents need to continue the work goes after the human sections: a blank line, a `---` divider, then a collapsed section titled "Agent notes". The provider skill shows the exact wrapper. Leave the notes out when the human sections already say everything an agent needs.

Write the notes as Markdown bullets of the form `- **Label:** text`, with these labels in this order, using only labels with something to say:

| Label | Holds |
|-------|-------|
| **Reflects** | The branch, pull request and commit the notes describe, so a reader can judge how fresh they are |
| **Start** | The first file, command or place to open |
| **Find** | How to locate the rest: searches, paths, related code |
| **Approach** | The intended way to do the work, once decided |
| **Open questions** | Undecided choices; use it instead of Approach while the design is open |
| **Checks** | For each Verification box that needs more than its own line: what it needs, how to run it and what counts as a pass |
| **Constraints** | Rules the work must keep |
| **Pitfalls** | Known traps and how to avoid them |
| **Tried** | Approaches tried and dropped, each with why, so nobody repeats a dead end |
| **Findings** | Facts already learned that the next agent shouldn't rediscover |

Under Checks, give each box its own sub-bullet led by a short italic name for the box.

The notes hold the current state, not a log: rewrite a bullet when it changes instead of appending to it, and keep the block short enough to read at a glance. They never hold status, dates, assignees, relationships, check results or anything about conversations with people; those belong in tracker fields, native relations, the pull request or nowhere. Everyone with tracker access can read the notes, and anyone can on a public repository, so keep candid or personal detail in the private journal. Keep the format plain: no `loaf:` keys, schema lines or Loaf naming.

## Keeping Notes Current

When work on an issue pauses, is handed off or ships, rewrite its notes so they describe the work as it now stands: point Reflects at the current branch, pull request and commit, move dropped approaches to Tried with why, add what was learned to Findings and Pitfalls, and remove what is no longer true. A different agent on a different machine should be able to continue from the issue and its branch alone. Refresh only the notes of issues being worked; the edit follows the agent-notes rule in [Draft Tracker Writes](../SKILL.md#draft-tracker-writes).

## Comments

Lead every progress, review or status comment with one to three plain lines: what happened, what it means, and what happens next or what a person needs to do. Agent detail such as commit identities, file lists and, for work with no pull request, check results goes below a divider in collapsed agent notes with the same labels as an issue's; results go under Checks and commit identities under Findings. A comment is evidence or collaboration; it never changes the definition, status or relationships.

## Formatting

- Put code words, commands, paths, field names and identifiers in backticks, and longer code or output in fenced blocks.
- Write issue and PR references in the tracker's native form so they render as links. Keep them out of backticks, which stop the link.
- When an exact moment matters, such as a log line, a failed batch, a data window or an incident, write it as `YYYY-MM-DD HH:MM:SS` in UTC. Write calendar dates in prose naturally ("6 November"), and keep schedule dates in tracker fields.
- Use short sentences and everyday words. Prefer bullets to long paragraphs.

## Links and Files

Point only to places everyone with tracker access can reach: repositories, the tracker itself (issues, attachments and documents) and shared drives. Never write a local path, a private journal reference or a file that lives on one machine.

When the work involves files with no shared home, ask what to do instead of writing a local path, and suggest a default:

- Attach a finished deliverable to the issue.
- Give a source that will be revised a home in a repository.
- Leave it out of the record.

## Relationships and Magic Words

Magic words (`Closes`, `Fixes`, `Part of`, `Related to`) belong only in PR descriptions and commit messages. In a PR description, put them in their own paragraph just above the divider, with a blank line above and below; without the blank line below, the line above `---` renders as a heading.

In issues and comments, mention other issues in the prose where they matter, and set relationships with the tracker's own relation tools.

## Example

A bug on GitHub. On Linear only the collapsed section's syntax differs.

````markdown
Support reported that billing exports drop the last row when a filter is on. Customers reconciling invoices end up missing charges.

### Steps to reproduce

1. Open the billing page and filter by any month.
2. Export to CSV.
3. The file is missing the table's last row.

### Definition of Done

- Filtered exports contain every row the table shows
- Unfiltered exports are unchanged

### Verification

- [ ] A filtered export matches the table row for row
- [ ] The test suite passes

---

<details>
<summary>Agent notes</summary>

- **Start:** `app/exports/csv_writer.rb`
- **Checks:**
  - *A filtered export matches:* filter to a month with more than one page of rows, export, and expect the file's row count to equal the table's
  - *The test suite passes:* run `make test`
- **Pitfalls:** the writer paginates, so the missing row sits at a page boundary

</details>
````
