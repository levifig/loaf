# Architecture Topic Template

**Default location:** `docs/architecture/<topic>.md`, linked from `docs/ARCHITECTURE.md`. Respect the repository's explicitly established equivalent home. Revise an existing topic when it already owns the subject.

These are optional writing prompts, not required headings or a schema. Omit irrelevant sections and replace all prompts with supported content. Do not add decision IDs, statuses, frontmatter, or a revision ledger unless the repository explicitly requires them.

```markdown
# [Topic]

## Current Model

[Explain the agreed model, boundaries, and applicability. Distinguish shipped behaviour from agreed direction that is not implemented yet.]

## Rationale and Tradeoffs

[Why this shape fits the current constraints. Retain significant rejected alternatives and their reasons when they still inform future work; do not invent alternatives. If a choice has an exceptional ADR, summarize its current constraint and link to the detailed rationale there instead of duplicating it.]

## Constraints

[Invariants and compatibility obligations that changes must preserve or explicitly revise. State where they apply.]

## Implementation and Gaps

[Link supporting code and tests. Name precise gaps between implemented behaviour and agreed direction without duplicating tracker status, sequencing, or completion checklists. Unresolved proposals are not agreed architecture.]

## Related References

[Link owning topics and technical references instead of duplicating them. Keep exact schemas, protocols, algorithms, and vectors beside their implementation or in their existing specification home.]
```
