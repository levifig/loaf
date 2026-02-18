---
name: idea
description: >-
  Captures ideas quickly into atomic, well-structured nuggets for later
  evaluation. Covers rapid idea documentation with context, potential value, and
  next steps. Use when capturing a new idea without deep analysis, or when the
  user asks "I have an idea" or "note this down." Produces structured idea
  files. Not for deep exploration (use brainstorm) or turning ideas into specs
  (use shape).
version: 1.17.0
---

# Idea

Capture ideas quickly with minimal friction.

**Input:** $ARGUMENTS

---

## Purpose

Ideas are raw nuggets -- unprocessed, unshaped, but worth remembering. The goal is **speed of capture**, not thoroughness. Shape later via `/shape`.

---

## Process

### Step 1: Parse Input

If `$ARGUMENTS` contains the idea, capture directly. If empty/unclear, ask **at most 2-3 questions**: core idea, problem/opportunity, immediate constraints.

### Step 2: Generate Idea File

Create file in `.agents/ideas/` following [idea template](templates/idea.md).

**Filename:** `{YYYYMMDD}-{slug}.md`

### Step 3: Create and Announce

1. Generate timestamp: `date -u +"%Y-%m-%dT%H:%M:%SZ"`
2. Create the file (infer title, tags, and related links without asking)
3. Announce: `Captured: .agents/ideas/{filename}.md` with next steps

---

## Idea Lifecycle

```
raw -> shaping -> shaped (becomes SPEC) -> archived
```

| Status | Meaning |
|--------|---------|
| `raw` | Just captured, unprocessed |
| `shaping` | Being developed via /shape or /brainstorm |
| `shaped` | Converted to SPEC, idea file archived |
| `archived` | Decided not to pursue, kept for reference |

---

## Guardrails

1. **Speed over completeness** -- capture quickly, shape later
2. **2-3 questions max** -- don't turn this into an interview
3. **Infer, don't ask** -- metadata should be automatic
4. **One idea per file** -- keep them atomic
5. **No shaping here** -- that's what `/shape` is for

---

## Related Skills

- **shape** -- Develop an idea into a SPEC
- **brainstorm** -- Deep thinking on an idea or problem space
- **research** -- Investigate before capturing
