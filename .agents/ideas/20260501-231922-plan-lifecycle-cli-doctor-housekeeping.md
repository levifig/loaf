---
title: "Plan artifact lifecycle — loaf plan list/archive, doctor recognition, housekeeping awareness"
captured: 2026-05-01T23:19:23Z
status: raw
tags: [cli, plans, lifecycle, doctor, housekeeping, artifacts]
related:
  - SPEC-034-refactor-deepen-grilling-glossary
  - 20260501-225251-spec-plan-tasks-artifact-taxonomy
blocked_by: SPEC-034
---

# Plan artifact lifecycle

## Nugget

SPEC-034 introduces `.agents/plans/PLAN-NNN-*.md` as the output artifact of `/refactor-deepen`. Specs ship with `loaf spec list` / `archive` / `loaf doctor` checks / housekeeping skill awareness — plans need the same lifecycle infrastructure or they become an orphan artifact class. Codex flagged this during SPEC-034 review: "new artifact-class without lifecycle support is technical debt waiting." The simplest path is parallel CLI surface to specs.

Three pieces:

1. `loaf plan list` and `loaf plan archive` — top-level commands, parallel to `loaf spec`. List active plans in `.agents/plans/`, archive completed plans to `.agents/plans/archive/`.
2. `loaf doctor` recognizes `.agents/plans/`. Detects orphaned plans (no related spec referenced *and* no recent activity, threshold configurable, default 30d). Distinguishes "orphaned" (broken graph) from "simply stale" (just old, may still be relevant).
3. `housekeeping` skill awareness — stale plan detection, archive policy, surfaced during `/housekeeping` invocations alongside stale specs and tasks.

## Problem/Opportunity

Without this work, plans created by SPEC-034 will:

- Have no clean way to enumerate (manual `ls .agents/plans/`)
- Have no archive convention (different developers / sessions will improvise)
- Be invisible to `loaf doctor` — broken plan/spec graphs go undetected
- Pile up unflagged through housekeeping cycles — they don't appear in stale-file reports

The risk is graceful degradation: SPEC-034 ships, plans get created, lifecycle gradually rots. The cost-of-fix grows with the plan inventory.

## Initial Context

- **Originally Track C of SPEC-034.** Removed during shape session — Codex/me/Levi agreed the lifecycle work is its own product surface (listing semantics, archive semantics, doctor checks, housekeeping integration), not refactoring-skill scope.
- **Sequencing:** blocked-on SPEC-034 — needs at least one real plan in the wild before lifecycle commands can be designed against actual usage. Avoid designing lifecycle for hypothetical artifacts.
- **Implementation:** mostly parallel to existing spec lifecycle (`cli/commands/spec.ts`, `cli/lib/housekeeping/`). Should reuse rather than duplicate — possible refactor to `cli/lib/lifecycle/` with shared list/archive primitives parameterized by artifact type.
- **Open questions for shaping:**
  - Top-level `loaf plan` vs. nested under existing command (`loaf spec plan list`)? Working assumption: top-level, mirrors `loaf spec`.
  - How does orphan detection differentiate "no related spec" from "spec was archived"? Plans related to archived specs may still be relevant (the spec shipped, the plan tracked the implementation strategy).
  - Default staleness threshold for plans — same as specs (30d) or different? Refactor plans may have different rhythm than feature specs.
  - Should `loaf plan` carry the same `linear_parent` frontmatter pattern as specs in Linear-native mode, or is plan-as-Linear-issue not a thing in the deferred taxonomy spec?
- **Dependency on artifact taxonomy spec (`20260501-225251-spec-plan-tasks-artifact-taxonomy`):** if that taxonomy spec lands first, it may redefine what a plan is — potentially making this lifecycle work obsolete or restructured. If SPEC-034 lands first (likely), this lifecycle spec ships against the SPEC-034 plan shape, then the taxonomy spec adapts the lifecycle to the broader model.
