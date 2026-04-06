---
name: idea
description: >-
  Captures ideas into structured nuggets for later evaluation. Use when the user
  says "I have an idea" or "note this down." Also activate when a specific
  actionable concept crystallizes during conversation. Without args, scans
  brainstorm documents and session journals for unprocessed sparks to promote or
  discard. Not for deep exploration (use brainstorm) or shaping (use shape).
version: 2.0.0-dev.12
---

# Idea

Capture ideas quickly with minimal friction.

**Input:** $ARGUMENTS

---

## Contents
- Critical Rules
- Verification
- Quick Reference
- Purpose
- Process
- Idea Lifecycle
- Guardrails
- Related Skills

## Critical Rules

- Speed over completeness -- capture quickly, shape later
- 2-3 questions maximum -- don't turn capture into an interview
- Infer metadata automatically -- don't ask for tags, title, or links
- One idea per file -- keep them atomic
- No shaping here -- that's what `/shape` is for

## Verification

- Idea file created in `.agents/ideas/` with correct `YYYYMMDD-HHMMSS-slug.md` naming
- Frontmatter contains required fields (title, status: raw, created timestamp)
- If promoted from a spark, source document is marked `*(promoted)*`

## Quick Reference

| Status | Meaning |
|--------|---------|
| `raw` | Just captured, unprocessed |
| `shaping` | Being developed via /shape or /brainstorm |
| `shaped` | Converted to SPEC, idea file archived |
| `archived` | Decided not to pursue, kept for reference |

---

## Purpose

Ideas are raw nuggets -- unprocessed, unshaped, but worth remembering. The goal is **speed of capture**, not thoroughness. Shape later via `/shape`.

---

## Process

### Step 1: Parse Input

If `$ARGUMENTS` contains the idea, capture directly.

If `$ARGUMENTS` is empty, **scan for sparks** from two sources:

**Brainstorm documents:**
1. Search `.agents/drafts/*brainstorm*.md` for `## Sparks` sections
2. List unprocessed sparks (not marked as `*(promoted)*` or `*(discarded)*`)

**Session journals:**
1. Search `.agents/sessions/*.md` for `spark()` journal entries
2. Exclude sparks that have a matching `resolve(spark)` entry in the same session
3. Also scan `.agents/sessions/archive/*.md` for unresolved sparks from past sessions

Present all unprocessed sparks from both sources. For each, the user decides:
- **Promote** → create idea file, log resolution in source
- **Discard** → log resolution with reason, no idea file
- **Defer** → skip for now, resurface next time

**When promoting from a brainstorm:** create idea file with `origin:` field, mark spark as `*(promoted)*` in source document.

**When promoting from a session journal:** create idea file with `origin:` field pointing to the session file, append a `resolve(spark)` entry:
```
- YYYY-MM-DD HH:MM resolve(spark): slug → promoted to .agents/ideas/YYYYMMDD-HHMMSS-slug.md [YYYY-MM-DD HH:MM]
```

**When discarding from a session journal:** append a `resolve(spark)` entry:
```
- YYYY-MM-DD HH:MM resolve(spark): slug → discarded, reason [YYYY-MM-DD HH:MM]
```

If no sparks found and no arguments, ask **at most 2-3 questions**: core idea, problem/opportunity, immediate constraints.

### Step 2: Generate Idea File

Create file in `.agents/ideas/` following [idea template](templates/idea.md).

**Filename:** `{YYYYMMDD}-{HHMMSS}-{slug}.md`

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
- **brainstorm** -- Deep thinking on an idea or problem space (sparks from brainstorms can be promoted to ideas)
- **research** -- Investigate before capturing
