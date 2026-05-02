---
id: ADR-004
title: Knowledge Naming Convention
status: Deprecated
date: 2026-03-14
deprecated_date: 2026-05-02
deprecated_reason: 'Recategorized — naming convention without architectural significance.'
migrated_to: docs/knowledge/knowledge-management-design.md#naming-conventions
---

# ADR-004: Knowledge Naming Convention

## Decision

Use `knowledge` and `decisions` as directory and collection names. Not `kb`, not `docs`, not abbreviated forms.

## Context

Needed consistent, readable naming for knowledge directories (`docs/knowledge/`, `docs/decisions/`) and QMD collection suffixes (`{repo}-knowledge`, `{repo}-decisions`).

## Rationale

- Visual consistency: `knowledge` and `decisions` are similar length, scan well together
- `kb` is too short versus `decisions` — creates visual asymmetry
- `knowledge` is unambiguous — everyone understands what it means
- `decisions` (not `adrs`) is more accessible to non-engineers

## Consequences

- Directory structure: `docs/knowledge/`, `docs/decisions/`
- QMD collections: `{repo-folder}-knowledge`, `{repo-folder}-decisions`
- ADR files still use `ADR-XXX` prefix (the record format, not the directory name)
- CLI subcommand uses `kb` for ergonomics: `loaf kb check`, not `loaf knowledge check`. The full word is for storage (directories, collections), the abbreviation is for typing (CLI commands).

## Deprecated

This ADR was recategorized on 2026-05-02 against the tightened architecture-skill bar. The naming convention concerns aesthetics (visual symmetry, accessibility) without measurable effect on the system's structure, quality attributes, dependencies, interfaces, or construction techniques — it does not clear the architectural-significance threshold (see [content/skills/architecture/SKILL.md](../../content/skills/architecture/SKILL.md#the-bar)).

**The convention itself remains in force.** Only the *classification* changed: the rationale and ongoing governance now live in [docs/knowledge/knowledge-management-design.md](../knowledge/knowledge-management-design.md#naming-conventions), where they can evolve via `/reflect` rather than via ADR supersession.

This record is retained per the append-only-log discipline ("_was_ the decision, _no longer_ the decision" — Nygard) but is no longer the operative source.
