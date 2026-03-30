---
type: idea
created: '2026-03-30'
status: captured
---

# Idea: `--dry-run` for All Mutating CLI Commands

## Nugget

Every `loaf` command that creates, modifies, or deletes files should support `--dry-run` to preview planned changes without executing them. Currently only `release` and `cleanup` have it.

## What It Would Do

Add `--dry-run` to: `build`, `install`, `init`, `setup`, `task`, `spec`. Each would output what it plans to do (files to write, dirs to create, copies to make) without actually doing it.

## Why

Discovered during `/release` skill development — the principle that destructive commands should be previewable applies uniformly. `install` is the highest-value addition since it writes into external config directories (`~/.config/`), where surprises are most costly.

## Priority Order

1. `install` — touches external dirs, highest risk of surprise
2. `build` — regenerable output, but useful for debugging target issues
3. `task` / `spec` — lower risk (writes to `.agents/`), but consistency matters
4. `init` / `setup` — one-time commands, lowest urgency

## Open Questions

- Should `--dry-run` output be machine-readable (JSON) for piping, or just human-readable?
- Should `build --dry-run` show a diff of what would change in existing output files?
