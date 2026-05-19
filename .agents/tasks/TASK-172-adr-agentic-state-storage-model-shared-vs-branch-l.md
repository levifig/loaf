---
id: TASK-172
title: 'ADR: agentic state storage model (shared vs branch-local)'
status: todo
priority: P3
created: '2026-05-18T23:59:01.537Z'
updated: '2026-05-18T23:59:01.537Z'
spec: SPEC-036
depends_on:
  - TASK-167
  - TASK-168
  - TASK-169
  - TASK-171
---

# TASK-172: ADR: agentic state storage model (shared vs branch-local)

## Description

Author the ADR documenting the agentic state storage model decision. Written **after** implementation completes so the narrative reflects what actually shipped.

Required sections (per the project ADR convention in `docs/decisions/`):

- **Title** and **decision date** (ISO 8601)
- **Status:** Accepted
- **Context:** the worktree fragmentation problem — session misrouting, ID clashes, fragmented project knowledge; the underlying category error of mixing project/process state with branch-scoped artifacts in one tree
- **Decision:** split agentic state into shared (project/process) and branch-local (specs/tasks/plans); shared store lives in the main worktree's `.agents/`; ID allocation scans both views
- **Alternatives Considered:**
  - A1 (sessions-only routing) — rejected: doesn't address ID clashes or fragmented knowledge
  - A3 (full `.agents/` centralization) — rejected: removes specs/tasks from PR diffs
  - Symlinks — rejected: fragile across `git worktree add`
  - Storage under `.git/loaf/` — rejected: not inspectable as a normal directory
- **Consequences:** positive (cross-worktree session continuity, collision-free IDs, shared project knowledge); negative (sessions no longer appear in PR diffs, new migration step for existing repos)
- **Compliance:** ADR + migration command represent a hard cut; pre-A2 layouts are refused, not silently fallback'd

## Acceptance Criteria

- [ ] ADR file exists under `docs/decisions/` with the project's standard naming
- [ ] All required sections present
- [ ] ADR status set to `Accepted`
- [ ] Decision date in ISO 8601
- [ ] SPEC-036 frontmatter or body updated with a link to the ADR

## Files

- New: `docs/decisions/ADR-NNN-agentic-state-storage-model.md`
- Edit: `.agents/specs/SPEC-036-worktree-aware-agents-storage-routing.md` (add ADR link)

## Verification

```bash
ls docs/decisions/*agentic-state-storage*
# Manual review of section completeness
```

## Context

See SPEC-036. Depends on TASK-167, TASK-168, TASK-169, TASK-171 so the ADR reflects shipped reality.
