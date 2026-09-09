---
id: ADR-012
title: CI Verifies Source Artifacts Instead of Repairing Branches
status: Accepted
date: 2026-04-30
revised: 2026-09-05
---

# ADR-012: CI Verifies Source Artifacts Instead of Repairing Branches

## Context

A workflow once rebuilt generated artifacts after merge and tried to push a repair commit to the protected default branch. That conflicted with branch protection and made failures ambiguous.

## Decision

CI verifies that reviewed source and required tracked generated content agree; it never repairs or pushes to the source branch. Contributors commit any tracked generated text with the source that produced it until a different delivery path is proven.

Release automation may build, sign, and publish native binaries from reviewed source without committing those binaries or bypassing branch protection. Publication evidence and artifact verification belong to the release workflow, not an auto-fix commit.

## Consequences

### Positive

- CI failures represent real drift instead of push-permission noise.
- Reviewed source remains the authority for published artifacts.

### Negative

- Contributors must run deterministic content generation when tracked outputs remain part of the repository.

### Neutral

- Native release binaries need not be tracked merely because generated harness content currently is.

## Alternatives Considered

### Let CI commit rebuilt artifacts

This was rejected because it changes the reviewed branch after approval and requires protection bypasses.

### Stop tracking all generated content immediately

This requires a proven replacement delivery path and is not implied by the verify-only rule.

## Implementation Status

See [Runtime and Delivery](../architecture/runtime-and-delivery.md#reviewed-source-and-release-evidence) for current verification and publication boundaries. This record preserves the branch-protection incident and rejected repair approaches, not a second evolving release specification.

## Revisions

- 2026-09-05 — Separated source-branch verification from future release artifact publication and removed npm-specific procedure from the binding decision.
- 2026-09-05 — Retained the narrow verify-only decision and linked current implementation context to its owning topic.
