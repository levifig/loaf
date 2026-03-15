---
id: ADR-003
title: QMD as Knowledge Retrieval Backend
status: Proposed
date: 2026-03-14
---

# ADR-003: QMD as Knowledge Retrieval Backend

## Decision

Use [QMD](https://github.com/tobi/qmd) as the retrieval backend for knowledge management. Loaf wraps QMD for collection setup and adds lifecycle features (staleness, growth loops).

## Context

Knowledge management needs both lifecycle (freshness, growth) and retrieval (search, discovery). Building retrieval from scratch would duplicate QMD's capabilities.

## Rationale

- QMD already has collections, context, BM25 + semantic search, MCP server
- Collection naming (`{repo-folder}-knowledge`, `{repo-folder}-decisions`) maps naturally
- BM25-only mode needs no models (lightweight dependency)
- Semantic search opt-in when models are installed
- Node.js runtime (same as Loaf's build system)
- Loaf focuses on what QMD doesn't do: staleness detection, growth loops, agent guidance

## Consequences

- QMD becomes a dependency (at least for full KB features)
- `loaf kb` commands are thin wrappers around QMD operations
- Cross-project knowledge import works via QMD collections pointing to other repos
- If QMD's API changes, only the wrappers need updating (skills/hooks are isolated)

## Alternatives Considered

- Build native search (SQLite FTS5) — would be a worse QMD
- No search, just frontmatter filtering — adequate for small KBs, insufficient for large ones
- Full MCP server — overkill when QMD's MCP server already exists
