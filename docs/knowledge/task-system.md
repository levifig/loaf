---
topics:
  - trackers
  - compatibility
  - journal
covers:
  - internal/cli/cli.go
  - internal/state/task_*.go
  - docs/changes/**/*.md
  - .agents/specs/**/*.md
  - vnext/content/skills/project-management/**/*
consumers:
  - implementer
  - reviewer
last_reviewed: 2026-09-05
---

# Work Records and Compatibility

## Contents

- Current Shared Work
- Shipped Compatibility Surfaces
- Migration Rules
- Project Journal
- Cross-References

## Current Shared Work

New shared work lives only in the selected native tracker. The Loaf Flow is pitch → shape → implement → ship → release. Shape creates or updates the canonical tracker work contract through the selected `project-management/v1` provider skill and a connection already exposed by the harness.

The tracker owns identity, definition, definition of done, out-of-scope, status, hierarchy, dependencies, assignment, and collaboration. Git owns code, tests, deliberately authored documents, promoted artifacts, and implementation history. Loaf owns Flow methods and private continuity.

No current workflow falls back to a local issue when a tracker is unavailable. A mutating Flow step stops with the missing capability or connection rather than creating a second authority.

## Shipped Compatibility Surfaces

The public CLI still contains local `change`, `issue`, `task`, `spec`, `plan`, `intent`, and related commands. Historical repositories may also contain:

- `docs/changes/YYYYMMDD-slug/` Change folders;
- `.agents/specs/SPEC-*.md` records;
- SQLite issue, task, plan, intent, and release rows;
- task packets, cohorts, verification receipts, and rendered projections.

These surfaces preserve access to existing data and support deliberate migration. They are not the work authority for new shared work and must not be synchronized bidirectionally with a tracker.

Compatibility commands should be read according to their shipped `--help` and tests. Their presence does not override tracker-native guidance in the current Flow skills.

## Migration Rules

A migration from historical local work to the native tracker is one-time, agentic, and verified:

1. Resolve the exact source records and destination tracker project.
2. Preview the proposed mapping without mutation.
3. Preserve source bytes, stable identifiers, provenance, relationships, and status semantics needed for recovery.
4. Create or update tracker records through the selected provider skill and harness-owned connection.
5. Read back every material write and reconcile ambiguous or unsupported records explicitly.
6. Record the completed cutover and stop writing the local work representation.

Migration never introduces background push/pull, reconciliation queues, credential storage, provider mappings, or a writable local twin. The shipped Linear compatibility integration is historical behavior, not the provider-neutral current architecture.

Legacy Markdown import and database cutover mechanics retain exact-byte backups, hashes, transactional database writes, rerunnable decisions, and explicit rollback. Restoring files is a distinct operator-controlled action; it is not described as database-transactional.

## Project Journal

The journal is private continuity, not shared work. Entries are project-scoped facts tagged with an opaque harness conversation correlation value. There is no session entity, lifecycle, status, open or close operation, or rotation.

A wrap is an optional synthesis entry. Context is derived at read time. Linked work references may help an operator resume, but live tracker state is read from the tracker rather than mirrored into continuity.

Private continuity also includes sparks, ideas, explorations, decisions, findings, and handoffs. Scratchpad remains deferred. Temporary reports are not continuity records by default.

## Cross-References

- [Shared Work Model](work-model.md)
- [Loaf Flow](loaf-flow.md)
- [Architecture](../ARCHITECTURE.md)
- [Authority Boundaries](../architecture/authority-boundaries.md)
- [Private Continuity: Journal and Resumption](../architecture/private-continuity.md#journal-and-resumption)
