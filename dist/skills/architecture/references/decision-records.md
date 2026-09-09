# Exceptional Architecture Decision Records

Topics explain the current architecture. An ADR preserves one consequential choice and the context that made it reasonable. ADRs are an intentional option, not obsolete artifacts or a required accompaniment to each topic. The Architecture skill owns the selection and authorization rules.

## Scope a single choice

Name the decision, not the subsystem: “Keep the relay outside the plaintext trust boundary” is a candidate; “Private synchronization” is a topic. Broad impact can justify a narrowly scoped record, but significance alone does not make a separate artifact useful. Explain what would be lost if the rationale remained only in the topic.

Preserve the actual forces, alternatives, chosen tradeoff, consequences, and applicability. Do not invent an option comparison after the fact to dignify implementation history. When reconstructing an earlier choice, distinguish the recording date from the known decision date and label inferred reasoning. Keep exact protocols, algorithms, schemas, test vectors, and implementation plans in their existing homes.

## Keep authority navigable

Use the repository's decision-record home and naming convention; without one, use a descriptive decision filename under `docs/decisions/`. Numbering is not required by this model. Use the decision-record template linked from Architecture when no repository template exists.

The topic states the currently applicable constraint, a brief rationale where useful, and a link to the ADR's detailed decision context. The ADR links back to the owning topic. The topic is the current-state entry point, not permission to override an accepted decision silently. If they disagree, surface the conflict and establish the authorized choice before reconciling them.

## Preserve decision context when direction changes

For new exceptional records, preserve the accepted choice and original context. Correct errors or add explicitly identified clarification without making it appear that a later choice was the original one. Ordinary implementation learning updates the topic, not an ADR revision ledger.

When an authorized decision materially replaces the choice, mark the old record no longer applicable and identify its replacement. If the replacement independently merits an ADR, link the records reciprocally; otherwise link to the current topic and explain the changed constraint there. A replacement is not automatically another ADR. Rejection, withdrawal, or retirement should likewise be explicit, with no fabricated dates. The record's decision status is separate from implementation progress.

Do not retroactively apply these history rules to repositories that explicitly retain a different ADR lifecycle. Architecture's legacy reference describes Loaf's former revise-in-place convention; migration from it is a separate authorized task.
