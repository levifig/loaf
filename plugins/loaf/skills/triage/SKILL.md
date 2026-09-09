---
name: triage
description: >-
  Triages private sparks and ideas plus existing native tracker candidates
  without creating a local intake queue. Use when moving private captures to
  Backlog, or when deciding which existing records should advance, defer, close,
  or return to discovery. Produces verified native dispositions and explicit
  gaps.
user-invocable: true
version: 0.5.0
---

# Triage

Operate through [`project-management/v1`](../project-management/SKILL.md). Triage owns two jobs: dispose existing native candidates, and move private sparks or ideas onto tracker Backlog when the human chooses. It does not maintain a parallel Loaf queue or invent universal workflow states.

Bare invocation is the ceremony. Inventory first, show candidates, wait for a pick, then mutate. Do not ask the human to write a destination brief.

## Contents

- Critical Rules
- Operator Presentation
- Verification
- Quick Reference
- Topics

## Critical Rules

- As the first action, run `loaf journal log "skill(triage): <concise intent>"` against the current private local journal. If the write fails, report the failure and continue only when the work can safely proceed; never put invocation bookkeeping in the tracker.
- Confirm the exact native destination and candidate references before reading or mutating.
- Show candidates to the human before any recommendation or mutation. The human never has to fish an idea out of the journal or a CLI dump.
- Read each candidate, its current workflow state, and relevant recent comments before deciding.
- An unrecorded raw direction stays a spark or idea, goes to pitch, or moves to tracker Backlog when the human chooses. Do not capture it in a local issue or intake ledger. Moving to Backlog creates or adopts a native record in the uncommitted lane (GitHub: destination Project Status Backlog; Linear: the team's uncommitted/backlog state, or Triage inbox when that inbox is configured), then resolves the private capture against that native id.
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

On a bare `/triage` or "let's triage," do this before recommending:

1. Inventory open sparks, open ideas, and native tracker candidates that need a disposition.
2. Render a numbered table. Stable indexes for this turn. One row per candidate.

| # | Kind | Title | Age / status | Already on tracker? |
|---|------|-------|--------------|---------------------|
| 1 | idea | short title | captured date, open | no / `#N` Backlog / `#N` other lane |

3. After the table, expand at most the recommended next candidate in this shape:

**Candidate** `#n` (`idea:…` or spark alias)
**Title** …
**Kind** idea | spark | native record
**Nugget** one short paragraph
**Already on tracker?** none, or `#N` plus observed lane
**Suggested** move to Backlog | keep | archive | pitch | shape

4. Stop. Ask which indexes to take, and whether the suggested disposition stands. Do not mutate.

When the human replies `3`, `1, 4`, `1-3`, or `all ideas to Backlog`, only then create or adopt native records, set Backlog, and resolve the private captures. Search for duplicates first.

If destination config is incomplete, still show the table. Name the gap after the candidate, not instead of it. A missing recorded Project is a configuration gap to prompt; it is not a reason to hide the idea. A working harness GitHub connection (`gh`, MCP, or equivalent) that can see the destination repo and Project is enough.

## Verification

- The human saw a numbered candidate table before any recommendation or mutation.
- Every candidate disposition names its native reference and observed starting state.
- Every mutation used a runtime-supported provider operation and was verified by readback.
- No candidate disappeared because one independent mutation failed.
- Ambiguous destination, permissions, or workflow mapping is reported without mutation, after the table.
- Created or adopted tracker bodies contain no Loaf-internal idea, spark, or SQLite identifiers.
- The final result distinguishes moved to Backlog, kept, archived, pitched, shaped, deferred, closed, unchanged, failed, and indeterminate records in provider-native terms.
- Comparisons distinguish complete operator outcomes from enabling layers without treating foundation depth as product progress.

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

## Topics

No supporting references. Use the [tracker update](templates/tracker-update.md) when a self-contained disposition comment helps collaborators.
