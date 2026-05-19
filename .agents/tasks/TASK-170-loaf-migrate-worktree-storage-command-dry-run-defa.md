---
id: TASK-170
title: >-
  loaf migrate worktree-storage command (dry-run default, back-pointer file,
  round-trip test)
status: todo
priority: P2
created: '2026-05-18T23:58:56.163Z'
updated: '2026-05-18T23:58:56.163Z'
spec: SPEC-036
depends_on:
  - TASK-166
---

# TASK-170: loaf migrate worktree-storage command (dry-run default, back-pointer file, round-trip test)

## Description

Implement `loaf migrate worktree-storage` — a one-shot command that moves project/process artifacts from the worktree's local `.agents/` to the main worktree's `.agents/`. Dry-run by default; mutation requires explicit `--apply` (or equivalent confirmation flag).

Migration scope:
- Move `sessions/`, `kb/`, `ideas/`, `drafts/`, `reports/`, `councils/`, `AGENTS.md`, `loaf.json`, `SOUL.md` from local to shared
- Leave `specs/`, `tasks/`, `plans/` untouched
- Write a back-pointer file (e.g., `.agents/.moved-to-shared`) in the local `.agents/` indicating where artifacts went, with the main worktree's absolute path

Idempotency: re-running migrate on an already-migrated repo is a no-op with "nothing to do" output.

This task ships the migration tool itself. TASK-171 wires detection + refusal into shared-artifact commands so users get nudged to run this.

## Acceptance Criteria

- [ ] `loaf migrate worktree-storage` exists as a subcommand
- [ ] Default behavior is dry-run; output lists what would move without mutating
- [ ] `--apply` (or equivalent) performs the actual move with confirmation
- [ ] Back-pointer file `.moved-to-shared` written to local `.agents/` after successful migration, containing the main worktree's absolute path
- [ ] Idempotent: re-run on migrated repo exits with "nothing to do"
- [ ] Round-trip test: prepare fake pre-A2 layout → dry-run (no changes) → apply (changes correct) → re-apply (no-op)
- [ ] Refuses to migrate if not in a git repo (clear error)
- [ ] Handles cross-worktree invocation: decide whether migration must run from main checkout only, or works from any worktree; document the chosen behavior

## Files

- New: `cli/commands/migrate.ts`
- New: `cli/lib/migrate/worktree-storage.ts`
- New test: `cli/lib/migrate/worktree-storage.test.ts`

## Verification

```bash
npm run test -- migrate
loaf migrate worktree-storage --help
```

## Context

See SPEC-036. Depends on TASK-166 for resolver primitives. TASK-171 builds on this.
