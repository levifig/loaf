---
id: TASK-166
title: 'Foundation: worktree probe + dual resolvers + ID allocator scans both views'
status: todo
priority: P1
created: '2026-05-18T23:58:39.458Z'
updated: '2026-05-18T23:58:39.458Z'
spec: SPEC-036
---

# TASK-166: Foundation: worktree probe + dual resolvers + ID allocator scans both views

## Description

Introduce a worktree-aware storage layer so project/process artifacts (sessions, kb, etc.) can live in a single shared location while branch-scoped artifacts (specs, tasks, plans) stay local. This task lands the **mechanism** with no behavior change at call sites outside the ID allocator — subsequent tasks migrate the actual call sites.

Concretely:
1. Add a worktree probe that returns the main worktree's `.agents/` directory via `dirname(git rev-parse --git-common-dir)`. Fall back to the local `.agents/` when outside a git context.
2. Add `findSharedAgentsDir()` (uses the probe) and either rename or alias the existing `findAgentsDir()` as `findLocalAgentsDir()` for semantic clarity at call sites.
3. Update the ID allocator (spec + task minting paths) to compute `max(existing IDs across shared + local views)` before assigning a new ID, eliminating per-worktree counter drift.

## Acceptance Criteria

- [ ] `findSharedAgentsDir()` returns the main worktree's `.agents/` for both the main checkout and any linked worktree
- [ ] `findSharedAgentsDir()` falls back to local `.agents/` when not in a git repo or when `git rev-parse --git-common-dir` fails
- [ ] `findLocalAgentsDir()` preserves the current `findAgentsDir()` behavior verbatim (parent-walk for `.agents/`)
- [ ] ID allocator computes max across both shared and local views before minting a new ID
- [ ] Unit test: simulated worktree — both resolvers return the expected paths
- [ ] Unit test: parallel ID allocation across two distinct `.agents/` views returns distinct next IDs
- [ ] Existing test suite remains green
- [ ] No call sites outside the ID allocator have switched to `findSharedAgentsDir` yet (that's TASK-167+)

## Files

- `cli/lib/tasks/resolve.ts` (extend or co-locate with new module)
- New: `cli/lib/agents-dir/` for the worktree probe and dual resolvers
- ID allocator call sites (spec creation, task creation, "next-id" computation)
- Tests alongside the new module

## Verification

```bash
npm run test
npm run typecheck
npm run test -- agents-dir
npm run test -- id-allocator
```

## Context

See SPEC-036 for full context. Foundation gate for Tracks B/C/D.
