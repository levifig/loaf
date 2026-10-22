---
name: implement
description: >-
  Implements an already-verified selected native tracker contract. Use when live
  scope, criteria, and selection readback are complete. For concrete requests
  such as "Let's implement it", use project-management first, before loading
  this skill. A bare invocation reports the unblocked Todo frontier without
  starting work. Produces code, authorized atomic history, and verification
  evidence for ship. Not for initial discovery or review-only work.
subtask: false
version: 0.6.0-rc.3
---

# Implement

Load this skill for a concrete implementation request only after [Flow Semantics](../loaf-reference/references/flow-semantics.md#before-entering-implementation) establishes the live selected contract through [`project-management/v1`](../project-management/SKILL.md). The tracker defines the work; repository instructions and code define the implementation surface. A bare invocation may load this guidance to report the frontier, without starting execution.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Harness Integration
- Journal Commands
- Topics

## Critical Rules

Before the first journal write, select the command in [Journal Commands](#journal-commands). Use that form for every entry in this skill and its references.

- As the first action, run `loaf journal log "skill(implement): <concise intent>"` against the current private local journal. If the write fails, report the failure and continue only when the work can safely proceed; never put invocation bookkeeping in the tracker.
- Before announcing implementation, loading or invoking this workflow for a concrete request, dispatching an implementation worker, or making substantive edits, apply the common Flow entry rule and its narrow exceptions. Read the named issue or search for matching work; reuse a complete selected contract without repeating Shape, shape only missing details, and verify native scope, criteria and selected state. Honor an existing concrete implementation request without asking for selection again. Before starting, identify the verified native reference and completion criteria proportionately. An invocation journal entry records intent, not work authority.
- Re-read the native work record, completion criteria, hierarchy, dependencies, status, and recent relevant comments before planning. If the record is Backlog intent or otherwise unselected and the human has not selected its implementation, refuse and do not start. A bare invocation without a concrete implementation request reports the unblocked Todo frontier (selected, unblocked native records) and does not start. Never infer the issue from the current branch, worktree name, or open editor. In Progress is not the frontier.
- Read Todo from the destination tracker's native lane (GitHub: board Status), not from Issue open/closed and not from a different provider's MCP. An authenticated `gh` that can see the destination repository and named board is a GitHub connection. Linear MCP in the same session does not mean GitHub is unavailable.
- When resuming work, list the journal's unresolved blockers with `loaf journal context --layer unresolved-blockers`; the default digest only counts those older than 30 days. For each one about this work that no longer holds, log `loaf journal log "unblock(<block id>): <what cleared it>"`; an id resolves only that block and leaves others with the same scope open.
- Confirm the live contract makes Rider, complete Journey, Entry point, observable Outcome, real Dogfood, Safety/integrity proof, Learning sought, and explicit Deferrals concrete. If it describes layers or future-only machinery instead, return it to shape.
- Inspect repository instructions and affected code before editing. Treat existing changes as user-owned.
- For architectural changes, follow the architecture skill's topic-maintenance and affected-surface review rules within the authorized scope; do not automatically create a decision record.
- Distinguish a material gap from technical discretion. Routine in-scope implementation details stay implementer decisions. A clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence stays with implement; do not bounce ordinary fixes to shaping or absorb unauthorized scope here. If the live contract is missing substance needed to deliver or review, or implementation would change promised outcome, criteria, safety, dependencies, or the basis of selected or resourced work, stop and return it to shape instead of silently redefining work. Unchanged criteria with unexpectedly expanded work still change the commitment. Material means uncertainty or change in what was promised or authorized, not the severity of an in-scope coding bug. Material changes need applicable authorization, an updated canonical definition with readback, and a retained reason and impact; assignment, write access, or reviewer role alone is not approval. Pause affected work when required approval is missing. Inaccessible required proof or absent authority is blocked; do not waive the criterion.
- Start each testable behavior with a focused failing test when practical, implement the minimum passing change, and keep refactoring behavior-neutral.
- Preserve the rideable journey across commits and delegation. Include only machinery exercised by its end-to-end path; reduce breadth before integrity.
- Add or substantively update the repository changelog for every implemented canonical work record, including fixes, tests, docs, maintenance, and features. Link each record using the repository's reference convention; one accurate entry may cover several records, and refining an existing entry is preferable to duplication. Discussion, inspection, and unimplemented ideas do not get entries. Declare the complete implemented work set through the repository's explicit review input and run its coverage gate against the actual review base and candidate. An unchanged, released, unrelated, or formatting-only note does not cover new implementation. The main agent verifies canonical identity and completeness through the tracker; deterministic parsing cannot attest either fact.
- Keep commits cohesive and atomic only when the user or governing workflow authorizes commits. Never infer permission to push, merge, or publish; merges happen only through ship.
- Use orchestration only when delegation is available, authorized, and materially useful; a single agent remains a valid execution path.
- Post a [tracker update](templates/tracker-update.md) only when progress, a blocker, or verification evidence benefits collaborators, written as [Record Writing](../project-management/references/record-writing.md) describes: one to three plain lines first, agent detail collapsed. Comments never modify the work contract.
- Before handing to ship, tick each Verification box whose check you ran and passed, after recording the result, naming the commit it ran on, where reviewers can find it from the issue: the PR description or a tracker comment. Ticking a box, or writing a reason a person with authority over the work approved, are the only definition writes implement makes. Leave boxes you did not run, and checks only a person can do, open for their owner, and never write a reason on a box yourself unless the person with authority over the work approved its exact text; name who approved it where you record results. Follow [Record Writing](../project-management/references/record-writing.md#verification-boxes).
- Re-read native state after any status transition, comment append, or box tick and report indeterminate effects honestly.

## Verification

- Every completion criterion is mapped to current evidence or an explicit remaining gap. A remaining material gap — unclear or changed contract substance, or a material scope or commitment shift — returns to shape. A clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence stays with implement. Inaccessible required proof or absent authority is blocked; do not waive the criterion.
- The named rider can complete the journey through its real entry point, and observed dogfood plus safety/integrity evidence match the live contract.
- No added foundation or abstraction waits on an unspecified future consumer.
- Focused tests, affected package tests, formatting, lint or static analysis, and the relevant build were actually run and their outputs read.
- The implementation diff contains no unrequested authority, dependency, schema, or public-interface expansion.
- Commit references and working-tree state are reported exactly; uncommitted or unverified work is labeled.
- Each Verification box whose check this work ran and passed is ticked with its result recorded, and the handoff names every box still open.
- The live tracker record was read again before handoff to ship.
- Before the final response or handoff, load the project-management skill and apply its Capture Deferred Work rule to concrete deferred or out-of-scope work.

## Quick Reference

| Condition | Action |
|-----------|--------|
| Unselected / Backlog intent | Refuse; do not start |
| Bare invocation without a concrete implementation request | Report the unblocked Todo frontier from the destination tracker; do not start; do not infer from the branch |
| Concrete implementation request with or without an issue ID | Use project-management and the common Flow entry rule before loading Implement, announcing execution, delegating implementation, or editing |
| Branch or worktree looks like an issue | Ignore it unless the human named that issue |
| Linear MCP present, GitHub is the tracker | Use `gh` / GitHub connection; do not claim GitHub is unavailable |
| Clear unmet criterion, ordinary defect, obtainable missing in-scope evidence, or other in-scope technical work | Stay in implement; do not bounce to shape |
| Unclear or changed contract substance, or material scope or commitment shift | Return to shape with the exact gap; do not invent intent or absorb unauthorized scope |
| Inaccessible required proof or absent authority | Blocked; do not waive the criterion |
| External blocker | Preserve code state and report observed blocker evidence |
| Independent bounded tasks | Coordinate through orchestration when authorized |
| Review finding | Add a regression test, fix, and re-run affected gates |
| Criteria satisfied | Tick the Verification boxes you ran and passed, then hand live reference, diff, and evidence to ship |

## Harness Integration

### Amp

When using native Amp Low, Medium, High, or Ultra, the selected main mode may implement directly in the current thread, or use ordinary optional delegation when the harness exposes it, governing instructions allow it, and the task is independently bounded. Tracker prerequisite, tests, and acceptance remain with the main agent. Native Oracle remains the read-only review and advice path; the Review button remains unchanged.

## Journal Commands

Use the ordinary command `loaf journal log "type(scope): description"` with your harness's permissions. Apply only the labeled section for the harness you are running.

### Codex

With Loaf's managed basic-command policy installed, use `loaf journal log --execpolicy-safe "type(scope): description"` for every entry, including invocation logs. Keep `--execpolicy-safe` immediately after `journal log` and invoke PATH `loaf` as a standalone command, without an absolute executable pin or shell/environment wrapper. The flag does not install the policy or grant permission by itself; without the policy, use the ordinary command and Codex's approval flow.

## Topics

Load the repository's language, testing, and domain guidance for the implementation itself.
