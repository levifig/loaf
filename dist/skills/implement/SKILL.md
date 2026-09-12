---
name: implement
description: Implements a shaped native tracker work contract in Git with evidence-led verification. Use when a canonical record is complete and ready to build. Produces code, atomic history when authorized, and verified implementation evidence for ship.
---

# Implement

Read the live canonical work contract through [`project-management/v1`](../project-management/SKILL.md), then build the smallest coherent change in Git. The tracker defines the work; repository instructions and code define the implementation surface.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Harness Integration
- Topics

## Critical Rules

- As the first action, run `loaf journal log "skill(implement): <concise intent>"` against the current private local journal. If the write fails, report the failure and continue only when the work can safely proceed; never put invocation bookkeeping in the tracker.
- Before substantive edits, including regression tests, load the loaf-reference skill's Flow Semantics and apply its native issue prerequisite and narrow exceptions. For an explicit concrete implementation request without an issue ID, use project-management to find or create the covering contract and verify its scope, criteria, and selected state without asking for the same selection again.
- Re-read the native work record, completion criteria, hierarchy, dependencies, status, and recent relevant comments before planning. If the record is Backlog intent or otherwise unselected and the human has not selected its implementation, refuse and do not start. A bare invocation without a concrete implementation request reports the unblocked Todo frontier (selected, unblocked native records) and does not start. Never infer the issue from the current branch, worktree name, or open editor. In Progress is not the frontier.
- Read Todo from the destination tracker's native lane (GitHub: board Status), not from Issue open/closed and not from a different provider's MCP. An authenticated `gh` that can see the destination repository and named board is a GitHub connection. Linear MCP in the same session does not mean GitHub is unavailable.
- Confirm the live contract makes Rider, complete Journey, Entry point, observable Outcome, real Dogfood, Safety/integrity proof, Learning sought, and explicit Deferrals concrete. If it describes layers or future-only machinery instead, return it to shape.
- Inspect repository instructions and affected code before editing. Treat existing changes as user-owned.
- For architectural changes, follow the architecture skill's topic-maintenance and affected-surface review rules within the authorized scope; do not automatically create a decision record.
- Distinguish a material gap from technical discretion. Routine in-scope implementation details stay implementer decisions. A clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence stays with implement; do not bounce ordinary fixes to shaping or absorb unauthorized scope here. If the live contract is missing substance needed to deliver or review, or implementation would change promised outcome, criteria, safety, dependencies, or the basis of selected or resourced work, stop and return it to shape instead of silently redefining work. Unchanged criteria with unexpectedly expanded work still change the commitment. Material means uncertainty or change in what was promised or authorized, not the severity of an in-scope coding bug. Material changes need applicable authorization, an updated canonical definition with readback, and a retained reason and impact; assignment, write access, or reviewer role alone is not approval. Pause affected work when required approval is missing. Inaccessible required proof or absent authority is blocked; do not waive the criterion.
- Start each testable behavior with a focused failing test when practical, implement the minimum passing change, and keep refactoring behavior-neutral.
- Preserve the rideable journey across commits and delegation. Include only machinery exercised by its end-to-end path; reduce breadth before integrity.
- Keep commits cohesive and atomic only when the user or governing workflow authorizes commits. Never infer permission to push, merge, or publish.
- Use orchestration only when delegation is available, authorized, and materially useful; a single agent remains a valid execution path.
- Post a [tracker update](templates/tracker-update.md) only when progress, a blocker, or verification evidence benefits collaborators. Comments never modify the work contract.
- Re-read native state after any status transition or comment append and report indeterminate effects honestly.

## Verification

- Every completion criterion is mapped to current evidence or an explicit remaining gap. A remaining material gap — unclear or changed contract substance, or a material scope or commitment shift — returns to shape. A clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence stays with implement. Inaccessible required proof or absent authority is blocked; do not waive the criterion.
- The named rider can complete the journey through its real entry point, and observed dogfood plus safety/integrity evidence match the live contract.
- No added foundation or abstraction waits on an unspecified future consumer.
- Focused tests, affected package tests, formatting, lint or static analysis, and the relevant build were actually run and their outputs read.
- The implementation diff contains no unrequested authority, dependency, schema, or public-interface expansion.
- Commit references and working-tree state are reported exactly; uncommitted or unverified work is labeled.
- The live tracker record was read again before handoff to ship.
- Before the final response or handoff, load the project-management skill and apply its Capture Deferred Work rule to concrete deferred or out-of-scope work.

## Quick Reference

| Condition | Action |
|-----------|--------|
| Unselected / Backlog intent | Refuse; do not start |
| Bare invocation without a concrete implementation request | Report the unblocked Todo frontier from the destination tracker; do not start; do not infer from the branch |
| Explicit concrete implementation request without an issue ID | Find or create its native contract and verify scope, criteria, and selection before substantive edits |
| Branch or worktree looks like an issue | Ignore it unless the human named that issue |
| Linear MCP present, GitHub is the tracker | Use `gh` / GitHub connection; do not claim GitHub is unavailable |
| Clear unmet criterion, ordinary defect, obtainable missing in-scope evidence, or other in-scope technical work | Stay in implement; do not bounce to shape |
| Unclear or changed contract substance, or material scope or commitment shift | Return to shape with the exact gap; do not invent intent or absorb unauthorized scope |
| Inaccessible required proof or absent authority | Blocked; do not waive the criterion |
| External blocker | Preserve code state and report observed blocker evidence |
| Independent bounded tasks | Coordinate through orchestration when authorized |
| Review finding | Add a regression test, fix, and re-run affected gates |
| Criteria satisfied | Hand live reference, diff, and evidence to ship |

## Harness Integration

### Amp

When using native Amp Low, Medium, High, or Ultra, keep investigation, planning, tracker work, tests, and acceptance in the main thread. Send all implementation, including fixes and test writing, through `loaf_delegate` to a fresh Grok 4.6 Fast child with only Read and apply_patch in the current local workspace. Use native Oracle for read-only review and advice; the Review button remains unchanged. An incompatible or missing delegate does not authorize local implementation or another model. Native mode, Oracle, and Subagents Auto settings stay untouched. Reload or restart loads changed plugin bytes, but a persisted custom-mode definition is not rewritten; select builtin Low, Medium, High, or Ultra, or start a new builtin thread. Read [Amp native delegation](../orchestration/references/amp-native-delegation.md).

## Topics

Load the repository's language, testing, and domain guidance for the implementation itself.
