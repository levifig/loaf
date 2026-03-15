---
id: ADR-005
title: Loaf CLI Evolution
status: Proposed
date: 2026-03-14
---

# ADR-005: Loaf CLI Evolution

## Decision

Evolve Loaf from a build-time framework (npm scripts) into a unified CLI tool (`loaf` command) that manages skills, knowledge, tasks, build, and multi-harness distribution.

## Context

Loaf has operational concepts (build, skill management, knowledge management, task management) that need CLI commands. Currently these are either npm scripts or manual operations.

## Rationale

- Knowledge management needs `loaf kb check`, `loaf kb init`, etc.
- Task system needs a programmatic interface beyond markdown files
- Multi-harness distribution needs `loaf install --to <target>` (EveryIn-inspired pattern)
- Makes Loaf a developer tool, not just a build artifact
- CLI is the cross-harness equalizer — agents on any harness can call `loaf` via Bash

## Consequences

- Need to choose CLI language (likely Node.js/TypeScript, consistent with build system)
- `npm run build` becomes `loaf build`
- CLI needs to be installable (`npm install -g loaf` or similar)
- Agents need skills that teach them how to use the CLI
- Future: TUI and potentially macOS GUI built on top of the CLI

## Alternatives Considered

- Keep npm scripts + separate Python scripts — fragmented, hard to discover
- Shell wrapper — insufficient for complex UX (fuzzy search, interactive prompts)
- Go binary — new runtime, though appealing for single-binary distribution
