---
name: brainstorm
description: >-
  Conducts structured brainstorming with divergent thinking and trade-off
  analysis. Use when the user asks "help me think through this," "what are
  the options," or is exploring tradeoffs. Produces docs with sparks. Not
  for quick ideas or shaping.
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
5. Create brainstorm document following [brainstorm template](templates/brainstorm.md)
6. Update idea file status if proceeding

**Output file:** `.agents/drafts/{YYYYMMDD}-{HHMMSS}-brainstorm-{slug}.md`

### Problem Exploration

1. **Interview**: problem definition, who's affected, impact, prior attempts, constraints
2. Gather strategic context
3. **Diverge**: first principles, inversion, analogy, extreme constraints, persona lens
4. **Converge**: filter by strategy alignment, feasibility, value
5. Create brainstorm document following [brainstorm template](templates/brainstorm.md)

**Output file:** `.agents/drafts/{YYYYMMDD}-{HHMMSS}-brainstorm-{slug}.md`

### Open Brainstorm

1. Assess: list ideas in `.agents/ideas/`, check recent sessions, review VISION for gaps
2. Surface opportunities: what's blocking? What's not being pursued? Untested assumptions?
3. Present options for exploration

### Capture Sparks (All Modes)

After the main brainstorm concludes, identify **sparks** -- speculative ideas that emerged but aren't part of the main direction. These are byproducts of exploration, not the conclusion.

Add a `## Sparks` section at the end of the brainstorm document:

```markdown
## Sparks

- **Title** -- one-line description
- **Title** -- one-line description
```

Sparks are:
- **Lightweight** -- one bullet, one line each. Don't expand or analyze.
- **Byproducts** -- they emerged during brainstorming, not the main output
- **Worth remembering** -- interesting enough to not lose, not ready for `/shape`

Spark lifecycle:
- Unprocessed (default) -- sitting in the brainstorm document
- `*(promoted)*` -- processed into an idea via `/idea`
- `~~Strikethrough~~ *(abandoned)*` -- decided not to pursue

Brainstorm documents are **archived, never deleted** -- they hold exploration context, reasoning, and unprocessed sparks. A brainstorm doc remains active while it has unprocessed sparks.

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
