---
id: ADR-006
title: Agent-Creates, Human-Curates Model
status: Deprecated
date: 2026-03-14
deprecated_date: 2026-05-02
deprecated_reason: 'Recategorized — guiding principle, not architecturally significant decision.'
migrated_to: docs/ARCHITECTURE.md#authorship-model--agents-create-humans-curate
---

# ADR-006: Agent-Creates, Human-Curates Model

## Decision

Agents are the primary authors of knowledge files, ADRs, tasks, and specs. Humans review, approve, and curate. The CLI is for human management and health checks, not for authoring.

## Context

Traditional documentation is human-authored and agent-consumed. This creates a maintenance burden — humans write docs as a separate task, docs drift. The research principle "maintenance as side effect of work" suggests inverting this.

## Rationale

- Agents are already doing the work — they're closest to what's being learned
- Knowledge creation should happen during brainstorming, development, debugging — not as a separate documentation sprint
- Humans are better at judgment (is this worth documenting?) than at the writing itself
- The growth loop: agent discovers insight → proposes knowledge file → human reviews → committed
- Same pattern for ADRs: agent proposes after architectural discussion → human reviews
- Same pattern for tasks: agent creates via `/breakdown` → human approves

## Consequences

- Knowledge-base skill must guide agents on WHEN to create knowledge (not just format)
- Review workflow needed: agents propose, humans accept/edit/reject
- CLI commands are management-oriented (`check`, `validate`, `status`, `review`), not authoring-oriented
- Quality control depends on human review — agents may write redundant or low-quality knowledge
- PostToolUse and SessionEnd hooks prompt agents at the right moments

## Deprecated

This ADR was recategorized on 2026-05-02 against the tightened architecture-skill bar. The "agents create, humans curate" model is a guiding principle / operating philosophy — it doesn't fit the architecturally-significant criteria (structure, quality attributes, dependencies, interfaces, construction techniques), and ADRs are immutable post-acceptance whereas principles evolve over time (see [content/skills/architecture/SKILL.md](../../content/skills/architecture/SKILL.md#the-bar)).

**The principle itself remains in force.** Only the *classification* changed: the principle now lives in [docs/ARCHITECTURE.md](../ARCHITECTURE.md#authorship-model--agents-create-humans-curate) as an operating principle, where it can evolve via `/reflect`.

This record is retained per the append-only-log discipline ("_was_ the decision, _no longer_ the decision" — Nygard) but is no longer the operative source.
