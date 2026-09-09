---
type: glossary
topics:
  - glossary
last_reviewed: 2026-09-05
---

# Canonical Terms

## Skill

A unit of method or domain knowledge following the Agent Skills standard. Shared skill bodies are portable; harness-specific facts belong in labeled sections, sidecars, builders, or adapters.

## Target

A harness build destination: Claude Code, OpenCode, Cursor, Codex, or Amp. A target adapts shared authored content to a product's native packaging and capabilities.

## Sidecar

A target-specific file that adds product-only metadata or configuration to a shared source without changing the common body.

## Shared Template

A template under `content/templates/` distributed to multiple skills through `config/targets.yaml`. Examples include the ADR and grilling templates.

## Loaf Flow

The methods pitch → shape → implement → ship → release. Loaf owns their methodology and templates. The native tracker owns shared work; Git owns implementation; ship is the quality gate; release describes already-landed work.

## Native Tracker

The configured external collaboration system that owns shared work identity, definition, definition of done, out-of-scope, status, hierarchy, dependencies, assignment, and collaboration.

## Provider Skill

A provider-specific implementation of the `project-management/v1` contract. It guides an agent using a connection already exposed and authenticated by the harness. It does not make Loaf a provider proxy or credential store.

## Work Contract

The bounded definition of a shared work item in the native tracker: problem, intended outcome, definition of done, out-of-scope, and supported relationships. It is not a parallel local spec.

## Rideable Increment

A narrow, useful, end-to-end operator journey that preserves security, determinism, recovery, and data safety. Completing a horizontal layer without a usable journey is not a rideable increment.

## Private Continuity

One operator's project-scoped journal entries and wraps, sparks, ideas, explorations, decisions, findings, and handoffs. It is separate from tracker work and Git artifacts. Context is derived at read time, and no session entity or lifecycle exists.

## Report

A temporary skill output persisted only when it must outlive the response. The producing skill owns its template. Housekeeping may recommend cleanup or promotion, but the user approves destructive or durable changes. A report is not a universal Loaf record type or synchronization domain.

## One-Time Migration

A verified transition from a supported historical representation to its current authority. It preserves required bytes and provenance, validates the destination, and provides rollback. It does not establish ongoing synchronization or dual authority.

## Promoted Artifact

An intentionally durable document or output committed to Git through an explicit user-approved promotion. It is no longer temporary report output.

## Compatibility Surface

Shipped commands, files, schema labels, archive identifiers, or protocol bytes retained so existing users and data remain safe while current architecture is integrated. Compatibility does not make a retired workflow authoritative for new work.
