---
id: ADR-004
title: Knowledge Naming Convention
status: Proposed
date: 2026-03-14
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
