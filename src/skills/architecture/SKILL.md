---
name: architecture
description: >-
  Creates Architecture Decision Records (ADRs) through structured evaluation. Use when making
  technical decisions or when the user asks "should we use X or Y?"
---

# Architecture

Interview about technical decisions, update ARCHITECTURE.md, create ADRs.

## Contents
- CRITICAL: Interview First
- Process
- ADR Format
- Guardrails
- Council Trigger Conditions
- Related Skills

**Input:** $ARGUMENTS

---

## CRITICAL: Interview First

Before any architectural decision, understand:
1. What decision needs to be made?
2. What constraints exist (technical, business, team)?
3. What options have already been considered?
4. What would "good enough" look like vs "ideal"?
5. What's the impact of getting this wrong?

Use `AskUserQuestion` to gather context.

---

## Process

### Step 1: Understand the Decision

Parse `$ARGUMENTS`. If unclear, ask about: the problem, technical constraints, business constraints, time/resource constraints.

### Step 2: Gather Context

1. Read: VISION.md, ARCHITECTURE.md, existing ADRs in `docs/decisions/`
2. Check: how similar problems were solved, established conventions, previous attempts

### Step 3: Consider Options

For each viable option, evaluate: alignment with VISION/ARCHITECTURE, complexity added, reversibility, team capability, maintenance cost.

### Step 4: Convene Council (If Needed)

For complex/contentious decisions: 5 or 7 agents, diverse expertise. See `orchestration/councils` skill. **Council advises, user decides.**

### Step 5: Present Options

Show each option with pros, cons, and "fits when" context. Include recommendation and ask for user decision.

**Do NOT proceed without explicit user choice.**

### Step 6: Document the Decision

After user decides:
1. Update ARCHITECTURE.md with new decision and ADR reference
2. Create ADR following [ADR template](templates/adr.md)

### ADR Numbering

```bash
ls docs/decisions/ADR-*.md 2>/dev/null | \
  grep -oE 'ADR-[0-9]+' | sort -t- -k2 -n | tail -1 | awk -F- '{print $2 + 1}'
```

If none exist, start with `ADR-001`.

---

## Guardrails

1. **Interview first** -- understand the full context
2. **Check existing decisions** -- don't contradict without superseding
3. **Present options** -- user decides, not you
4. **Document thoroughly** -- ADRs explain the "why"
5. **Keep ARCHITECTURE.md current** -- update, don't just append

---

## Council Trigger Conditions

Convene when: decision affects multiple domains, team has conflicting opinions, high cost of reversal, novel problem, or user requests deliberation.

---

## Related Skills

- **orchestration/councils** - Council deliberation workflow
- **orchestration/product-development** - Where architecture fits in hierarchy
- **foundations** - Documentation standards for ADRs
