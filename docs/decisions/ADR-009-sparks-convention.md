---
id: ADR-009
title: Sparks Convention in Brainstorm Documents
status: Accepted
date: 2026-03-14
---

# ADR-009: Sparks Convention in Brainstorm Documents

## Decision

Speculative ideas from brainstorming sessions ("sparks") live as a `## Sparks` section at the end of brainstorm documents. They are promoted to ideas via `/idea` or abandoned. Brainstorm documents are archived, never deleted.

## Context

Brainstorming produces speculative byproducts — ideas that aren't ready for `/shape` but are worth remembering. These need a home that doesn't create file sprawl or a separate tracking system.

## Rationale

- Sparks belong with their origin context (the brainstorm that produced them)
- No new artifact type needed — just a section convention
- `/idea` (no args) scans brainstorm docs for unprocessed sparks, bridging brainstorm → idea pipeline
- Brainstorm documents are historical records of exploration — archiving preserves the reasoning
- Lifecycle is simple: unprocessed → promoted (`*(promoted)*`) or abandoned (`~~strikethrough~~`)

## Alternatives Considered

- Standalone `SPARKS.md` file — tried and abandoned; disconnects sparks from their context
- Individual spark files in `.agents/sparks/` — too heavy for lightweight captures
- Backlog items — too formal; sparks are pre-backlog

## Consequences

- Brainstorm skill includes "Capture Sparks" step
- Idea skill scans brainstorm docs when invoked without arguments
- Idea files get `origin:` field for traceability
- Brainstorm docs are retained while they hold unprocessed sparks
