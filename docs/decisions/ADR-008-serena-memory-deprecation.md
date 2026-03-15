---
id: ADR-008
title: Deprecate Serena Memory for Knowledge Persistence
status: Proposed
date: 2026-03-14
---

# ADR-008: Deprecate Serena Memory for Knowledge Persistence

## Decision

Keep Serena's code intelligence tools (find_symbol, get_symbols_overview, find_referencing_symbols, replace_symbol_body). Deprecate Serena's memory system (write_memory, read_memory, etc.) for knowledge persistence.

## Context

With knowledge files (`docs/knowledge/`) + Claude Code auto-memory (`MEMORY.md`) + decisions (`docs/decisions/`), Serena memories become a third redundant layer that drifts without anyone noticing.

## Rationale

- Two knowledge surfaces (knowledge files + MEMORY.md) are sufficient with clear ownership
- Serena memories have no staleness detection, no connection to code paths, no cross-reference validation
- Serena's code intelligence tools remain genuinely valuable — they provide semantic code understanding that complements Claude Code's text-based tools
- Reducing memory surfaces reduces drift

## Consequences

- Serena memories should store only transient code analysis state, not domain knowledge
- The knowledge-base skill documents this policy
- Existing Serena memories should be reviewed and migrated to knowledge files if valuable
