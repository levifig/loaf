---
name: brainstorm
description: >-
  Conducts deep, structured brainstorming with divergent thinking and trade-off analysis.
  Use when the user asks "help me think through this" or "what are the options?"
---

# Brainstorm

Think deeply about an existing idea or explore a new problem space.

**Input:** $ARGUMENTS

---

## Purpose

Brainstorming is **generative thinking** -- expanding possibilities before narrowing.

Unlike `/idea` (quick capture) or `/shape` (rigorous bounding), brainstorming is exploratory: process an existing idea, explore a problem space, or generate options before committing.

---

## Mode Detection

| Input Pattern | Mode |
|---------------|------|
| Idea file reference | Idea Processing (deep dive on captured idea) |
| Problem/question | Problem Exploration |
| Empty | Open Brainstorm ("What should we be thinking about?") |

---

## Process

### Idea Processing

1. Read idea from `.agents/ideas/{filename}.md`
2. Gather context: VISION.md, STRATEGY.md, ARCHITECTURE.md, related ideas
3. **Deep exploration**: ask about user value, problem depth, alternatives, risks, dependencies, scope (minimal vs maximal)
4. Generate options: conventional, minimal, ambitious, contrarian approaches
5. Document with core insight, explored directions (approach/pros/cons), open questions, recommendation, next steps
6. Update idea file status if proceeding

### Problem Exploration

1. **Interview**: problem definition, who's affected, impact, prior attempts, constraints
2. Gather strategic context
3. **Diverge**: first principles, inversion, analogy, extreme constraints, persona lens
4. **Converge**: filter by strategy alignment, feasibility, value
5. Document with problem statement, options, analysis, recommendation, next steps

### Open Brainstorm

1. Assess: list ideas in `.agents/ideas/`, check recent sessions, review VISION for gaps
2. Surface opportunities: what's blocking? What's not being pursued? Untested assumptions?
3. Present options for exploration

---

## Guardrails

1. **Diverge before converging** -- generate options before judging
2. **Stay exploratory** -- don't prematurely commit
3. **Document the thinking** -- even discarded options are valuable
4. **Connect to strategy** -- ground exploration in context
5. **Know when to stop** -- set boundaries on exploration

---

## Related Skills

- **idea** -- Quick capture (may precede brainstorming)
- **shape** -- Rigorous bounding (often follows brainstorming)
- **research** -- Fact-finding (complements brainstorming)
- **strategy** -- Strategic context (grounds brainstorming)
