---
name: strategy
description: >-
  Discovers and maintains strategic context in STRATEGY.md through structured
  investigation. Covers persona refinement, market landscape analysis, problem
  space mapping, and strategic positioning. Use when establishing or evolving
  project strategy, or when the user asks "what's our strategy?" or "update the
  strategic direction." Produces strategy document updates with evidence. Not
  for post-implementation reflection (use reflect) or quick idea capture (use
  idea).
version: 1.16.1
---

# Strategy

Deep discovery for personas, market landscape, and problem space.

**Input:** $ARGUMENTS

---

## Purpose

STRATEGY.md captures the **landscape** -- who we're building for, what we understand about the problem space, and how we're positioned.

Distinct from:
- **VISION.md** -- Where we're going (north star)
- **ARCHITECTURE.md** -- How we build (technical constraints)

---

## Mode Detection

| Input Pattern | Mode |
|---------------|------|
| Empty or "discover" | Full Discovery |
| "personas" | Persona Discovery |
| "market" or "landscape" | Market Analysis |
| "problems" or "problem-space" | Problem Space |
| "glossary" | Domain Glossary |
| Specific topic | Targeted Update |

---

## Process

### Step 1: Gather Context

1. Read VISION.md (strategy must align)
2. Read existing STRATEGY.md (extend, don't duplicate)
3. Check recent sessions for implementation learnings

### Step 2: Interview Deeply

Strategy discovery requires **extensive interviewing**. Use `AskUserQuestion` frequently. Ask non-obvious questions about users, competitors, problem space, and domain language.

**Full Discovery** covers: users/personas, problem space, market landscape, domain language.

**Focused modes** drill into the specific area (personas, market, problems, or glossary).

### Step 3: Draft Updates

Create additions/updates to STRATEGY.md following the [strategy template](templates/strategy.md).

### Step 4: Present for Approval

Present proposed sections. **Do NOT update STRATEGY.md without explicit approval.**

User may: approve, adjust, add context, or request more discovery.

### Step 5: Update STRATEGY.md

After approval: create (if new) or merge content. Announce updated sections.

---

## Guardrails

1. **Interview deeply** -- strategy is domain knowledge extraction
2. **Align with VISION** -- strategy serves the north star
3. **Define anti-personas** -- who we're NOT building for
4. **Adjacent problems** -- document what's out of scope
5. **Get approval** -- don't update without confirmation
6. **Keep it current** -- outdated strategy is worse than none

---

## Related Skills

- **shape** -- Uses STRATEGY.md for context during shaping
- **reflect** -- Updates STRATEGY.md based on shipping learnings
- **research** -- Investigation that may inform strategy
- **brainstorm** -- Deep thinking that may surface strategy insights
