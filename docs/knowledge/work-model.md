---
topics:
  - workflow
  - trackers
  - implementation
  - releases
covers:
  - vnext/content/skills/pitch/**/*
  - vnext/content/skills/shape/**/*
  - vnext/content/skills/implement/**/*
  - vnext/content/skills/ship/**/*
  - vnext/content/skills/release/**/*
  - vnext/content/skills/project-management/**/*
consumers:
  - implementer
  - reviewer
  - researcher
last_reviewed: 2026-09-05
---

# Shared Work Model

## Contents

- Authorities
- Loaf Flow
- Work Contract
- Implementation and Verification
- Release
- Migration and Compatibility
- Private Outputs

Shared work has one canonical home: the selected native tracker. Loaf supplies portable methods and templates; it does not maintain a local work database in parallel.

## Authorities

[Authority Boundaries](../architecture/authority-boundaries.md) owns the responsibility and artifact-placement model. This guide explains how the Flow operates within it: use the selected `project-management/v1` provider skill through an already exposed harness connection, with no local work mirror.

## Loaf Flow

The shared-work flow is:

1. **Pitch** discovers the problem. It may be brief or omitted when the problem is already clear.
2. **Shape** creates or updates the canonical work definition and makes the outcome, boundaries, proof, and supported relationships executable. Writing the definition does not select the work; Todo requires human selection.
3. **Implement** reads the selected record and changes the repository through ordinary Git mechanics. Backlog intent is refused; a bare invocation reports the unblocked Todo frontier without starting or inferring an issue from the branch.
4. **Ship** verifies the implementation, reviews the result, and updates tracker status only after provider readback.
5. **Release** groups coherent work that has already landed into a published version.

Triage may move private captures to tracker Backlog before selection; pitch can fill an existing intent record. An explicit filing request during pitch uses that same triage path. These are methods, not duplicate lifecycle records. Provider-specific field names and state transitions stay in the provider skill; GitHub lanes come from the configured Project Status, not Issue open/closed.

## Work Contract

The tracker record carries the shared contract. At minimum, shaping establishes:

- the problem and intended outcome;
- definition of done and verification expectations;
- explicit out-of-scope boundaries;
- supported parent, dependency, and related-work links;
- the provider's workflow state and collaboration context.

A technical design, ADR, committed document, or temporary research report may support the work, but it never becomes a second work identity. ADRs retain durable architectural rationale. Exact implementation plans and executable criteria stay close to code or in the tracker according to their purpose.

## Implementation and Verification

Implementation uses repository-native branches, worktrees, commits, tests, reviews, and pull requests. Git is the implementation witness; an agent's completion claim is not evidence.

The ship method is the sole quality gate. It validates the tracker contract against the code and relevant checks, reviews the effective diff, and writes status only through the provider after reading back the result. CI verifies source and required generated artifacts; it does not repair or push branches.

Progress is planned and assessed as rideable increments. Each increment completes a useful operator journey and includes its diagnostics and recovery behavior.

## Release

Release is retroactive. It publishes a coherent set of already-landed work and records the relevant version and evidence. It does not advance local cohorts, receipts, or a second release planner.

Loaf uses major-zero Semantic Versioning while its public contracts are still evolving. Native artifact provenance and delivery follow the accepted distribution architecture; exact update and rollback mechanics remain pending.

## Migration and Compatibility

Historical local issues, task packets, Changes, cohorts, receipts, and related commands may remain as compatibility inputs. A migration into the native tracker is explicit, previewable, verified, and one-way. Once moved, the tracker is authoritative; Loaf does not synchronize changes back or retain a writable local twin.

Compatibility commands are not endorsed as the workflow for new shared work. Documentation and generated skills must say so plainly while those commands remain shipped.

## Private Outputs

Private continuity is distinct from shared work and includes journal entries and wraps, sparks, ideas, explorations, decisions, findings, and handoffs. Scratchpad is deferred.

Most skill results return through the harness. A report is persisted only when its result must outlive the response, using the producing skill's template. Housekeeping can recommend leaving, extracting then deleting, deleting, or deliberately promoting it; the user approves destructive cleanup and durable promotion.

See [Loaf Flow](loaf-flow.md), [Architecture](../ARCHITECTURE.md), and [Authority Boundaries](../architecture/authority-boundaries.md).
