---
id: TASK-169
title: >-
  Route config artifacts (AGENTS.md, loaf.json, SOUL.md) through
  findSharedAgentsDir
status: todo
priority: P2
created: '2026-05-18T23:58:56.109Z'
updated: '2026-05-18T23:58:56.109Z'
spec: SPEC-036
depends_on:
  - TASK-166
---

# TASK-169: Route config artifacts (AGENTS.md, loaf.json, SOUL.md) through findSharedAgentsDir

## Description

Migrate `AGENTS.md`, `loaf.json`, and `SOUL.md` read/write paths from `findLocalAgentsDir` to `findSharedAgentsDir`. These are singleton config files used by install, build, and config-read flows — a different surface from the user-content artifacts (which are collections), so it's split out.

Special care: the build process generates `.agents/AGENTS.md` and friends from templates; ensure the generated path still works in single-worktree setups (where shared == local).

## Acceptance Criteria

- [ ] `loafConfigPath()`, `agentsConfigPath()` and analogues resolve through `findSharedAgentsDir`
- [ ] `SOUL.md` reads resolve through `findSharedAgentsDir`
- [ ] Install/build flows still produce correct outputs in single-worktree setups
- [ ] In a simulated worktree, all three files resolve to the main worktree's copies
- [ ] Existing install/build tests pass

## Files

- `cli/lib/config/agents-config.ts`
- `cli/lib/install/*` (anywhere AGENTS.md / SOUL.md are read or written)
- `cli/lib/build/lib/*` (template emission paths)

## Verification

```bash
npm run test -- loaf-config
npm run test -- install
npm run test -- build
```

## Context

See SPEC-036. Depends on TASK-166.
