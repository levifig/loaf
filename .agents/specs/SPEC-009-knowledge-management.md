---
id: SPEC-009
title: Knowledge Management
created: '2026-03-14T19:48:00.000Z'
status: drafting
appetite: TBD
requirement: Loaf should manage project knowledge with staleness detection
---

# SPEC-009: Knowledge Management

## Problem Statement

Knowledge accumulates across overlapping surfaces, drifts silently, and no tool detects stale instructions. Projects need living knowledge bases where decay is visible and maintenance is woven into work.

## Proposed Solution

`loaf kb` commands + knowledge-base skill + QMD integration + lifecycle hooks.

Core innovation: `covers:` field in knowledge file frontmatter links files to code paths. Combined with `git log --since={last_reviewed}`, this detects staleness automatically.

## Scope

### In Scope
- `loaf kb init` — scaffold + QMD collection registration
- `loaf kb check` — staleness detection (`covers:` + git)
- `loaf kb validate` — frontmatter consistency
- `loaf kb status` — summary view
- `loaf kb review <file>` — mark as reviewed
- `loaf kb import` — interactive fuzzy import from another project
- `knowledge-base` skill — agent guidance for creating/updating knowledge
- SessionStart, PostToolUse, SessionEnd hooks

### Out of Scope
- Personal knowledge base
- Knowledge repos
- CI/CD integration
- Overnight implementation loop

### No-Gos
- Don't build a search engine (QMD exists)
- Don't auto-commit knowledge changes (human reviews)

## Test Conditions
- `loaf kb init` creates docs/knowledge/, docs/decisions/, QMD collections
- `loaf kb check` correctly identifies stale files (modified covers: paths since last_reviewed)
- `loaf kb validate` catches missing frontmatter fields, broken covers: globs
- `loaf kb import` shows fuzzy-searchable list of registered KBs
- SessionStart hook surfaces stale knowledge count
- PostToolUse hook nudges when editing covered code

## Notes

See [ADR-003](../../docs/decisions/ADR-003-qmd-as-retrieval-backend.md) and [ADR-004](../../docs/decisions/ADR-004-knowledge-naming-convention.md).
See `.agents/drafts/brainstorm-loaf-cli-knowledge-harness.md` for full brainstorm context.
