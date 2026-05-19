---
id: TASK-171
title: Pre-A2 layout detection + refusal nudge in shared-artifact commands
status: todo
priority: P2
created: '2026-05-18T23:59:01.486Z'
updated: '2026-05-18T23:59:01.486Z'
spec: SPEC-036
depends_on:
  - TASK-170
---

# TASK-171: Pre-A2 layout detection + refusal nudge in shared-artifact commands

## Description

Detect pre-A2 layout on every Loaf invocation and refuse shared-artifact commands with a prompt to run `loaf migrate worktree-storage`. Local-artifact commands (`loaf task`, `loaf spec`, `loaf version`, `loaf build`, `loaf migrate` itself) keep working — the user isn't bricked, but the path of least resistance is migration.

Pre-A2 detection signal: in a worktree (git common dir ≠ git dir) where the main worktree's `.agents/` does not contain the expected shared subdirs (or no back-pointer file from TASK-170 exists in local `.agents/`). Tune the heuristic during implementation; the back-pointer file is one authoritative signal.

## Acceptance Criteria

- [ ] Pre-A2 detection helper exists and is unit-tested
- [ ] Commands touching shared artifacts (session, kb, idea, draft, report, council, etc.) refuse with a clear "run `loaf migrate worktree-storage`" message in pre-A2 state
- [ ] Local-artifact commands continue to work in pre-A2 state
- [ ] `loaf migrate worktree-storage` itself bypasses the refusal (so the user can fix the state)
- [ ] In single-worktree setups (no linked worktrees), detection always returns "n/a" so no nudge fires
- [ ] Refusal message is consistent across commands (single source of truth for the wording)

## Files

- New: `cli/lib/agents-dir/detect-pre-a2.ts`
- Wiring in: session, kb, idea, draft, report, council command modules

## Verification

```bash
npm run test -- pre-a2-detection
npm run test -- refusal-nudge
```

## Context

See SPEC-036. Depends on TASK-170 — migration must be available before refusal blocks shared commands.
