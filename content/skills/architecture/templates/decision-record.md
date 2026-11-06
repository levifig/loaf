# Decision Record Template

Default to no ADR. Use this minimal template only after a separate record is explicitly selected under Architecture's rules, normally after implementation is confirmed. An earlier record must distinguish agreement from implementation. Prefer the repository's own template. Omit prompts that add no useful context; these are not a mandatory frontmatter schema, numbering scheme, or revision ledger. Never fabricate decision dates, approval, or alternatives.

```markdown
# [One consequential choice]

Decision status: [Proposed, accepted, or no longer applicable; not implementation progress]
Recorded: [Actual recording date; distinguish an earlier decision date only when known]
Current architecture: [Link to owning topic]

## Context and Alternatives

[The original forces and credible alternatives actually weighed. Why does this choice need a separate record?]

## Decision

[The single choice, its applicability, and the tradeoff deliberately accepted.]

## Consequences

[Important benefits, costs, risks, and constraints. Link confirmed implementation evidence or identify pending implementation honestly; do not copy specifications or track delivery here.]
```
