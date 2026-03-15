# Spec Template

**Location:** `docs/specs/SPEC-{id}-{slug}.md`

```yaml
---
id: SPEC-XXX
title: "[Clear, descriptive title]"
source: "[idea file or 'direct']"
created: YYYY-MM-DDTHH:MM:SSZ
status: drafting
appetite: "[time budget]"
---

# SPEC-XXX: [Title]

## Problem Statement

[What problem are we solving? Why does it matter? Who does it affect?]

## Strategic Alignment

- **Vision:** [How this advances our north star]
- **Personas:** [Which personas benefit, how]
- **Architecture:** [Relevant constraints or patterns]

## Solution Direction

[High-level approach -- direction, not blueprint. Enough for an implementer to make good decisions, not so much that it's prescriptive.]

## Scope

### In Scope
- [Core functionality 1]
- [Core functionality 2]

### Out of Scope
- [Explicitly excluded 1]
- [Explicitly excluded 2]

### Rabbit Holes
- [Tempting complexity to avoid 1]
- [Tempting complexity to avoid 2]

### No-Gos
- [Forbidden approach 1]
- [Forbidden approach 2]

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| [Risk 1] | Low/Med/High | Low/Med/High | [How to handle] |

## Open Questions

- [ ] [Unresolved question 1]
- [ ] [Unresolved question 2]

## Test Conditions

- [ ] [Observable outcome 1]
- [ ] [Observable outcome 2]
- [ ] [Observable outcome 3]

## Circuit Breaker

At 50% appetite: [What to do if not on track]

At 75% appetite: [Hard decision point]
```
