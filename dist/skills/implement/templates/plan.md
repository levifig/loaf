# Plan File Template

**Location:** `.agents/plans/YYYYMMDD-HHMMSS-<description>.md`

```yaml
---
session: YYYYMMDD-HHMMSS-<description>.md
created: "YYYY-MM-DDTHH:MM:SSZ"
status: drafting
appetite: ""
---

# Plan: <description>

## Appetite

*Time budget for this work (e.g., "2 hours", "1 day"). Fixed time, variable scope.*

## Problem

*What problem are we solving? Why does it matter?*

## Solution Shape

*High-level approach, not a detailed spec. What's the general shape of the solution?*

## Rabbit Holes

*What NOT to do. Scope boundaries. Things that seem related but should be avoided.*

## No-Gos

*Explicit exclusions. Features or approaches we're deliberately not doing.*

## Circuit Breaker

At 50% of appetite spent: Re-evaluate if we're on track. If not, consider:
- Simplifying scope
- Taking a different approach
- Stopping early and documenting learnings
```
