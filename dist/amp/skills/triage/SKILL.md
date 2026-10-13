---
name: triage
description: >-
  Triages private sparks and ideas plus existing native tracker candidates
  without creating a local intake queue. Use when moving private captures to
  Backlog, when the user asks to triage a milestone, or when deciding which
  existing records should advance, defer, close, or return to discovery.
  Produces verified native dispositions and explicit readiness gaps. Not for
  changing release scope or implementing selected work.
version: 0.6.0-rc.2
---

# Triage

Operate through [`project-management/v1`](../project-management/SKILL.md). Triage owns two jobs: dispose existing native candidates, and move private sparks or ideas onto tracker Backlog when the human chooses. It does not maintain a parallel Loaf queue or invent universal workflow states.

Bare invocation is the ceremony. Inventory first, show candidates, wait for a pick, then mutate. Do not ask the human to write a destination brief.

## Contents

- Critical Rules
- Operator Presentation
- Milestone Readiness
- Verification
- Quick Reference
- Journal Commands
- Topics

## Critical Rules

Before the first journal write, select the command in [Journal Commands](#journal-commands). Use that form for every entry in this skill and its references.

- As the first action, run `loaf journal log "skill(triage): <concise intent>"` against the current private local journal. If the write fails, report the failure and continue only when the work can safely proceed; never put invocation bookkeeping in the tracker.
- Confirm the exact native destination and candidate references before reading or mutating.
- Show candidates to the human before any recommendation or mutation. The human never has to fish an idea out of the journal or a CLI dump.
- Read each candidate, its current workflow state, and relevant recent comments before deciding.
- For GitHub candidates, read the native version milestone and show its exact title, "unassigned," or unavailable evidence in the candidate table; private captures have no native milestone. A version bucket informs comparison; it neither selects implementation nor supplies a workflow state. Follow [version planning](../loaf-reference/references/flow-semantics.md#version-planning); do not create future buckets or assign the backlog during inventory. Other providers keep their existing path.
- When the human explicitly requests a milestone readiness pass, follow [Milestone Readiness](#milestone-readiness). It runs on demand when starting a version, never automatically or on a schedule. Do not create, delete, close, rename, or reassign milestones, change Issue scope, or start implementation during the pass.
- An unrecorded raw direction stays a spark or idea, goes to pitch, or moves to tracker Backlog when the human chooses. Do not capture it in a local issue or intake ledger. Moving to Backlog creates or adopts a native record in the uncommitted lane (GitHub: board Status Backlog; Linear: the team's uncommitted/backlog state, or Triage inbox when that inbox is configured), then resolves the private capture against that native id.
- Never say "publish." The disposition is move to Backlog, keep, archive, pitch, or shape.
- Do not move anything to Backlog until the human picks by table index, native id, or an explicit "all of these." Multiple picks in one turn are allowed.
- Adopt an existing native record that already represents the candidate. Do not mint a second Issue.
- Tracker bodies carry shared provenance only: a related native issue, the dogfood or session reason, a public URL. Never write Loaf-internal ids onto the tracker — idea aliases (`IDEA-…`), spark aliases (`SPARK-…`), `idea:hex`, SQLite row ids, or journal entry ids. Resolve the private capture against the native id in Loaf; that link stays private.
- Base dispositions on evidence, strategic fit supplied by the user or project context, duplication, readiness, and blockers.
- Prefer the next complete rideable operator journey over isolated component layers. If a candidate is foundation work, retain its explicit link to the immediate journey that consumes it; triage does not invent or shape that journey.
- Read valid native workflow states before any transition. Never assume names such as backlog, canceled, or done exist.
- Use comments only to explain a disposition when collaboration benefits; the native state and fields carry the disposition itself.
- Re-read every changed candidate and return partial or unsupported results independently.

## Operator Presentation

On a bare `/triage` or "let's triage," do this before recommending. An explicit milestone filter uses the bounded procedure below instead; bare triage keeps this inventory and private-capture flow.

1. Inventory open sparks, open ideas, and native tracker candidates that need a disposition.
2. Render a numbered table. Stable indexes for this turn. One row per candidate.

| # | Kind | Title | Age / status | Already on tracker? |
|---|------|-------|--------------|---------------------|
| 1 | idea | short title | captured date, open | no / `#N` Backlog / `#N` other lane |

When GitHub is selected, add a **Milestone** column: exact native title, "unassigned," or "unavailable" for native candidates; use — for private captures. Other providers retain the table above.

3. After the table, expand at most the recommended next candidate in this shape:

**Candidate** `#n` (`idea:…` or spark alias)
**Title** …
**Kind** idea | spark | native record
**Nugget** one short paragraph
**Already on tracker?** none, or `#N` plus observed lane
**Suggested** move to Backlog | keep | archive | pitch | shape

4. Stop. Ask which indexes to take, and whether the suggested disposition stands. Do not mutate.

When the human replies `3`, `1, 4`, `1-3`, or `all ideas to Backlog`, only then create or adopt native records, set Backlog, and resolve the private captures. Search for duplicates first.

If destination config is incomplete, still show the table. Name the gap after the candidate, not instead of it. A missing recorded board is a configuration gap to prompt; it is not a reason to hide the idea. A working harness GitHub connection (`gh`, MCP, or equivalent) that can see the destination repository and named board is enough. Refer to that board by title, not as `Project #N`.

## Milestone Readiness

### GitHub

For an explicit request such as "triage <milestone>," use the selected GitHub provider and existing native reads and Status transitions. Other providers retain their existing path; this procedure adds no provider operation, capability, CLI command, configuration, or local readiness record.

1. Resolve the requested native milestone on the exact repository. If the selector is missing or ambiguous, ask for the intended milestone without guessing or mutating. Inventory all of its native Issues whose observed project-board Status is Backlog, following pagination and excluding pull requests and other lanes. A milestone assignment or an open Issue alone does not prove Backlog. Report unavailable membership or Status evidence; do not claim a complete inventory from partial reads.
2. Before any recommendation or mutation, show the existing numbered candidate table with stable indexes, native Issue references, observed Backlog Status, and exact milestone title. An empty result is a valid outcome; do not broaden the pass to other milestones, private captures, or other lanes.
3. Read each candidate's current body, relevant comments, and native blocked-by dependencies. Check whether its problem, definition of done, and out-of-scope boundaries are substantively complete and independently reviewable, using the existing Shape contract. Headings alone are insufficient. Read the current state of every blocker: any open blocked-by edge makes the Issue unready, while a closed blocker does not. Unavailable definition or dependency evidence leaves readiness unverified, never presumed ready. Judge completeness and blockers only; shaping already decided milestone membership, so do not reconsider release scope or reassign work.
4. After the inventory table, report each candidate as ready for selection, needing Shape, blocked, or unverified, with the concrete missing details, native open blockers, or unavailable evidence. An Issue can have both missing details and blockers. Hand incomplete definitions to Shape by naming the native Issue and gaps; keep them in Backlog and do not rewrite or bulk-shape their bodies inside this pass. A complete but blocked Issue needs its blocker resolved, not invented reshaping.
5. Ask which ready candidates the human selects by displayed index, native Issue reference, or "all ready." Readiness alone does not select work. "All ready" covers only the ready rows in the displayed inventory. Do not move anything to Todo before that explicit human pick, and do not use comments, labels, or body prose to represent selection or dependencies.
6. Before each selected transition, re-read the Issue, milestone membership, board Status, definition, and native blockers. If it left the requested milestone or Backlog, gained a blocker, or is no longer provably complete, skip it and report the changed evidence; an earlier pick does not bypass readiness. For each still-ready selected Issue, use the native board Status transition to Todo, then independently read back that Issue's board item and confirm Todo. Report failures or indeterminate outcomes separately; do not silently retry ambiguous writes or let one failure hide other candidates.
7. Return per-Issue starting and observed final states, selected ready Issues moved to Todo, incomplete Issues handed to Shape, blocked or unverified Issues, and ready Issues left unselected in Backlog. Read back unchanged Backlog state for the unready candidates. Preserve milestone fields and assignments, definitions, dependencies, and all unselected work. Stop after the dispositions; selecting Todo does not invoke Implement.

## Verification

- The human saw a numbered candidate table before any recommendation or mutation.
- Every candidate disposition names its native reference and observed starting state.
- When GitHub is selected, candidate tables distinguish native milestone assignments, unassigned work, and unavailable milestone evidence without inferring Status or selection.
- Every mutation used a runtime-supported provider operation and was verified by readback.
- No candidate disappeared because one independent mutation failed.
- Ambiguous destination, permissions, or workflow mapping is reported without mutation, after the table.
- Created or adopted tracker bodies contain no Loaf-internal idea, spark, or SQLite identifiers.
- The final result distinguishes moved to Backlog, kept, archived, pitched, shaped, deferred, closed, unchanged, failed, and indeterminate records in provider-native terms.
- Comparisons distinguish complete operator outcomes from enabling layers without treating foundation depth as product progress.
- A requested milestone pass inventories only its observed Backlog Issues, shows the numbered table first, checks substantive Shape completeness and native open blockers, and requires a human pick before each verified Todo transition.
- Unready milestone candidates remain in Backlog with specific gaps or blockers; incomplete definitions return to Shape without changes to scope, milestone assignments, or bodies. Unavailable evidence is not a readiness pass, and bare triage retains its private-capture behavior.

## Quick Reference

| Evidence | Disposition |
|----------|-------------|
| Private spark or idea worth sharing | Move to Backlog; resolve the capture against the native id |
| Private spark or idea not worth sharing | Keep private or archive |
| Problem still unclear | Return to pitch |
| Narrative clear, not selected | Advance to shape; do not move to Todo unless the human selects it |
| Duplicate or superseded | Use supported native fields/state and explain with evidence |
| Blocked by missing decision | Preserve native state or use a supported deferred state |
| Ready, shaped, and selected | Leave canonical fields intact; hand to implement |

## Journal Commands

Use the ordinary command `loaf journal log "type(scope): description"` with your harness's permissions. Apply only the labeled section for the harness you are running.

### Codex

With Loaf's managed basic-command policy installed, use `loaf journal log --execpolicy-safe "type(scope): description"` for every entry, including invocation logs. Keep `--execpolicy-safe` immediately after `journal log` and invoke PATH `loaf` as a standalone command, without an absolute executable pin or shell/environment wrapper. The flag does not install the policy or grant permission by itself; without the policy, use the ordinary command and Codex's approval flow.

## Topics

No supporting references. Use the [tracker update](templates/tracker-update.md) when a self-contained disposition comment helps collaborators.
