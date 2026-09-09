<!--
Use this only when the user or repository explicitly retains Loaf's historical living-ADR format and no repository template exists. Architecture provides a separate template for new exceptional decision records. Do not use an ADR as a work record, implementation plan, incident report, or container for exact schema/protocol mechanics that belong beside code and tests.

Route shared implementation scope and status through the provider-neutral shaping workflow into the selected native tracker. A technical design may be linked from that record; it is never a second work identity.

Five statuses: Proposed, Accepted, Rejected, Deprecated, Superseded. Implemented is evidence, not a status. Deprecated means no longer binding (retired or rehomed). Superseded requires named replacement ADRs and reciprocal links. Never invent historical dates or alternatives.
-->

# Legacy ADR Template

**Location:** use the repository's existing decision-record home and numbering convention.

```yaml
---
id: ADR-001
title: "PostgreSQL as the Primary Database"
status: Accepted # Proposed | Accepted | Rejected | Deprecated | Superseded
date: 2026-01-23 # creation or proposal date
# revised: 2026-02-14 # latest material revision; mirror in Revisions
# accepted_date: 2026-01-30 # only when evidence establishes a distinct date
# rejected_date: 2026-01-30 # required iff Rejected
# deprecated_date: 2026-02-14 # required iff Deprecated
# supersedes: ADR-000 # scalar or list; optional
# superseded_by: # scalar or list; required iff Superseded
#   - ADR-002
#   - ADR-003
---
```

```markdown
# ADR-001: PostgreSQL as the Primary Database

## Context

[Why a durable choice was needed. Keep current framing concise.]

## Decision

[The current choice and its applicability. State accepted-but-unimplemented gaps explicitly.]

## Consequences

### Positive

- [Benefit]

### Negative

- [Tradeoff]

### Neutral

- [Operational or compatibility implication]

## Alternatives Considered

### [Credible alternative actually considered]

[Why it was not selected. Do not fabricate alternatives.]

## Implementation Status

[Optional. Link current code/tests or name precise pending implementation work.]

## Rejected

[Required only for Rejected: what was declined, when, and why.]

## Deprecated

[Required only for Deprecated: whether retired or rehomed, why, and the current home if any.]

## Superseded

[Required only for Superseded: replacement ADR link(s) and responsibility transferred.]

## Revisions

- 2026-01-23 — Initial record.
- 2026-02-14 — [Material change; prior text remains in git history.]
```
