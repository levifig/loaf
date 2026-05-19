---
id: TASK-167
title: Route session storage through findSharedAgentsDir
status: todo
priority: P1
created: '2026-05-18T23:58:56.003Z'
updated: '2026-05-18T23:58:56.003Z'
spec: SPEC-036
depends_on:
  - TASK-166
---

# TASK-167: Route session storage through findSharedAgentsDir

## Description

Migrate session-module call sites from `findLocalAgentsDir` to `findSharedAgentsDir`. After this task, sessions journaled from any worktree converge to the main worktree's `.agents/sessions/`, fixing the original "session log misrouting in worktrees" symptom that motivated SPEC-036.

Specifically:
- `loaf session start` writes session files to the shared dir
- `loaf session log`, `loaf session end`, hook handlers all read/write the shared dir
- `findActiveSessionForBranch` and `findSessionByClaudeId` scan the shared sessions dir
- Enrichment paths and JSONL lookups continue to work (those use the Claude Code project dir, not `.agents/`, and are out of scope here)

Branch-local artifacts (specs, tasks, plans) remain on the local resolver — do not touch those call sites in this task.

## Acceptance Criteria

- [ ] All session reads/writes resolve through `findSharedAgentsDir`
- [ ] Cross-worktree test: start a session in a simulated main worktree, journal an entry from a simulated linked worktree, assert both reach the same file
- [ ] Existing session tests pass (modulo resolver-injection plumbing)
- [ ] No call site outside the session module has changed
- [ ] Hook handlers (start hook, post-compact, etc.) all use the shared resolver

## Files

- `cli/commands/session.ts`
- `cli/lib/session/find.ts`
- `cli/lib/session/resolve.ts`
- `cli/lib/session/store.ts`
- New test: `cli/lib/session/cross-worktree.test.ts`

## Verification

```bash
npm run test -- session
npm run test -- cross-worktree
```

## Context

See SPEC-036. Depends on TASK-166 for resolver primitives.
