---
id: SPEC-036
title: Worktree-aware .agents/ storage routing
source: direct
created: '2026-05-19T00:30:00Z'
status: implementing
branch: feat/worktree-storage
---

# SPEC-036: Worktree-aware `.agents/` storage routing

## Problem Statement

`findAgentsDir()` walks up from `process.cwd()` to the nearest `.agents/`. In a git worktree, that resolves to the worktree's own checkout — a separate tree from the main checkout. Three concrete fallouts, observed on this branch:

1. **Sessions misroute across worktrees.** A session created in worktree A is invisible to worktree B; `claude_session_id` Tier 1/2 routing falls through to branch routing or spawns a duplicate session.
2. **ID allocation clashes.** Two worktrees can each mint `TASK-166` in parallel, producing collisions at merge.
3. **Shared knowledge fragments.** KB, ideas, drafts, reports — all branch-scoped today, even though they're project-scoped by intent.

Underlying category error: `.agents/` mixes **project/process state** (sessions, kb, ideas, drafts, reports, councils, config) with **branch-scoped artifacts** (specs, tasks, plans). Worktrees expose the cost.

## Strategic Alignment

- **Vision:** No `VISION.md` exists — flagging as a strategic gap (see Strategic Tensions). Working assumption: Loaf optimizes for AI-assisted development workflows where worktrees are a first-class human concurrency tool.
- **Personas:** Developers running parallel branches via `git worktree add`; AI agents that need stable session continuity regardless of which worktree the human invokes them from.
- **Architecture:** Introduces a worktree-aware storage layer atop the current single-tree assumption. Warrants an ADR alongside (see ADR Companion).

## Solution Direction

Two resolvers replace the single `findAgentsDir()`:

- **`findSharedAgentsDir()`** — resolves to the **main worktree's `.agents/`** via `dirname(git rev-parse --git-common-dir)`. Falls back to `findAgentsDir()` outside a git context.
- **`findLocalAgentsDir()`** — the current `findAgentsDir()` behavior, kept for branch-scoped artifacts.

Each call site picks the resolver matching the artifact kind:

| Storage | Artifacts |
|---------|-----------|
| **Shared** (main worktree's `.agents/`) | `sessions/`, `kb/`, `ideas/`, `drafts/`, `reports/`, `councils/`, `AGENTS.md`, `loaf.json`, `SOUL.md` |
| **Local** (worktree's own `.agents/`) | `specs/`, `tasks/`, `plans/` |

**ID allocator** scans `max(existing IDs across shared + local views)` at mint time. No counter file — derivable from frontmatter, matches SPEC-035's source-of-truth direction. If performance ever requires a counter, it can be added later as a pure cache without changing the model.

Storage location for shared artifacts is the **main worktree's existing `.agents/`** — not a new location like `.git/loaf/`. Greppable, inspectable, no behavior change for single-worktree users.

## Scope

### In Scope
- Worktree probe via `git rev-parse --git-common-dir`
- Two resolvers (`findSharedAgentsDir`, `findLocalAgentsDir`)
- Refactor call sites by artifact kind, per the storage table
- ID allocator scans both shared and local views
- One-shot migration command: `loaf migrate worktree-storage` (dry-run by default, explicit confirmation to mutate)
- "Automatically compelled" migration nudge: detect pre-A2 layout on every invocation; commands touching shared artifacts refuse until migration runs; commands touching only local artifacts keep working
- Tests for worktree creation, parallel ID allocation, cross-worktree session continuity, migration round-trip

### Out of Scope
- Changes to `TASKS.json`'s existence or shape (SPEC-035 owns that)
- Counter files (`max()` is sufficient)
- Cross-machine sync
- Linear-native mode interactions (Linear-stored artifacts are already cross-tree by definition)
- Backwards-compatibility or silent fallback to pre-A2 layout

### Rabbit Holes
- Building a cross-worktree lock manager. Resist: append-only journaling + atomic rename is already safe; add a contention test instead.
- Auto-migrating on first run without consent. Resist: explicit command; nudging is enough.
- Replacing `findAgentsDir()` globally. Resist: it's correct for branch-local artifacts.
- Re-litigating `plans/` and `sparks/` classification beyond the documented defaults.

### No-Gos
- **Symlinks.** Fragile across `git worktree add`, get accidentally committed, confuse newcomers.
- **Storing shared state under `.git/`.** Use the main worktree's `.agents/` so it remains a normal, inspectable directory.
- **Silent fallback** to pre-A2 layout. Migration is a hard cut; un-migrated repos refuse shared-artifact commands.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Concurrent journal writes corrupt a session file | Low | Med | Existing atomic-rename pattern; add a cross-worktree contention test |
| Migration moves files the user didn't expect | Med | High | Dry-run by default + explicit confirmation + back-pointer file in old location |
| Hooks invoked outside a git context fail to resolve shared dir | Low | Med | Resolver falls back to `findAgentsDir()` transparently |
| SPEC-035 lands and reshapes `TASKS.json` mid-flight | Med | Low | This spec only touches allocator logic, not the index file |
| Users expect to commit `.agents/sessions/` for PR review handoff | Med | Med | Document the new model; offer `loaf session export` for ad-hoc sharing |
| Migration nudge becomes noisy and gets ignored | Med | Med | Refuse shared-artifact commands until migration; nudge text points at exactly one command (`loaf migrate worktree-storage`) |

## Open Questions

- [ ] Confirm `sparks/` is journal-only (not a directory). If it exists as a dir somewhere, classify it.
- [ ] Migration back-pointer format: sentinel file in old shared subdirs (`.migrated → <new-path>`), or a single root-level marker in `.agents/.migration-state`?
- [ ] Should `loaf migrate worktree-storage` be idempotent (re-run safe), or refuse if already migrated?

## Test Conditions

- [ ] In a fresh worktree on a different branch, `loaf session start` finds and resumes the active session from the main worktree (not a new one)
- [ ] Journal entries appended from any worktree reach the same session file
- [ ] `loaf task new` invoked concurrently from two worktrees allocates distinct IDs
- [ ] `loaf spec new` from a worktree allocates a SPEC ID that doesn't collide with the main worktree's draft specs
- [ ] Un-migrated repos refuse shared-artifact commands with a prompt to run `loaf migrate worktree-storage`
- [ ] After migration, all shared subdirs live under the main worktree's `.agents/` and worktrees see them transparently
- [ ] `loaf session log` outside a git repo continues to work using the local `.agents/` resolution
- [ ] Branch-local artifacts (`specs/`, `tasks/`, `plans/`) remain in the worktree's tree and ship with that branch's PR diff
- [ ] Removing a worktree (`git worktree remove`) doesn't leave dangling references in the shared store
- [ ] Migration is dry-run by default and requires explicit confirmation to mutate

## Priority Order

Tracks ship in this order. Each go/no-go gate must pass before the next track starts.

1. **Track A — Foundation.** Worktree probe, dual resolvers, ID allocator scans both sides. Go/no-go: parallel-ID-allocation test passes; existing test suite green.
2. **Track B — Session migration.** Route session reads/writes/start hook through the shared resolver. Go/no-go: cross-worktree session continuity test passes; refusal nudge fires correctly in un-migrated state.
3. **Track C — Other shared artifacts.** KB, ideas, drafts, reports, councils, `AGENTS.md`, `loaf.json`, `SOUL.md` relocate. Go/no-go: each subdir's existing tests pass after relocation.
4. **Track D — Migration command + nudge.** `loaf migrate worktree-storage` with dry-run + back-pointer; pre-A2 detection wired into the relevant command surfaces. Go/no-go: migration round-trip test passes; nudge fires on every invocation in un-migrated repos and refuses shared-artifact commands until resolved. *Can be dropped from this spec if scope tightens — manual migration with documentation is acceptable for early adopters.*

## Strategic Tensions

1. **Missing strategic docs.** No `VISION.md`, `STRATEGY.md`, or `ARCHITECTURE.md` to evaluate against. This spec is a candidate to seed `ARCHITECTURE.md` with an explicit "agentic state storage model" section via `/loaf:reflect` after shipping.
2. **Overlap with SPEC-035.** SPEC-035 may eliminate `TASKS.json`; this spec keeps that question open by only touching allocator logic. Sequencing is independent in either direction.
3. **PR review surface change.** Today `.agents/sessions/` content can appear in PR diffs, giving reviewers context. Post-A2, sessions never appear in PRs. Documented as a trade — `loaf session export` covers the rare "I want to share this session in a review" case.

## ADR Companion

Implementation should land with an ADR: **"Agentic state separates project/process state from branch-scoped artifacts; the main worktree's `.agents/` is the shared store for project/process state."** Decision is structurally significant and difficult to reverse (changes the storage contract and what ships in PRs) — meets the ADR bar per the architecture skill.

## Provenance

Originated from a conversation on 2026-05-19 diagnosing session log misrouting in worktrees. Routed through `/loaf:shape` with explicit user approval; "A2" refers to the middle of three options surfaced during the diagnosis (A1 = sessions only; A2 = sessions + ID allocator centralization, files stay branch-local; A3 = all of `.agents/` shared). User selected A2, plus the refinement to scope `plans/` as local alongside `specs/` and `tasks/`, and to make migration "automatically compelled" via per-invocation detection rather than auto-run.
