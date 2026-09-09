---
id: ADR-014
title: Use Go for the Loaf Runtime
status: Accepted
date: 2026-05-28
revised: 2026-09-05
---

# ADR-014: Use Go for the Loaf Runtime

## Context

Loaf's deterministic operations include persistent state, migrations, diagnostics, filesystem safety, builds, and recovery. They need a stable native runtime with a bounded dependency surface and cross-platform packaging.

## Decision

Implement the public Loaf runtime in Go. The runtime owns deterministic Loaf operations; it does not own shared tracker work or provider credentials. JavaScript may remain development and packaging tooling where useful, but it is not a peer product runtime.

This decision absorbed the useful unified-command direction from the former ADR-005 without inheriting that proposal's speculative scope. [Runtime and Delivery](../architecture/runtime-and-delivery.md) owns the current runtime boundary, delivery model, and implementation gaps.

## Consequences

### Positive

- Hooks and users share one native command implementation.
- Persistent and recovery-sensitive behavior can use explicit typed packages and tests.

### Negative

- Native artifacts require a supported cross-platform build, signing, and distribution pipeline.
- SQLite and any other third-party runtime dependency require deliberate review.

### Neutral

- The development build may continue to use Node scripts until separately replaced.

## Alternatives Considered

### TypeScript runtime with built-in or third-party SQLite

This retained one language but kept stateful product execution tied to Node and its packaging surface.

### Rust runtime

Rust offered strong low-level guarantees but more implementation complexity than Loaf's CLI and state model required.

## Implementation Status

See [Runtime and Delivery](../architecture/runtime-and-delivery.md#current-model) for current implementation and delivery evidence. This record preserves the runtime-language commitment and alternatives, not a second evolving delivery specification.

## Revisions

- 2026-09-05 — Removed the completed TypeScript bridge plan, bounded runtime authority, and recorded ADR-005 absorption.
- 2026-09-05 — Retained the narrow language decision; moved current delivery context to its topic and removed the supersession link to the retired proposal, whose text remains in Git.
