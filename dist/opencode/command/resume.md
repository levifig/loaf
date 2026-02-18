---
description: >-
  Resumes existing session files and synchronizes state with Linear. Covers
  session state restoration, Linear status sync, and context rebuilding from
  session artifacts. Use when continuing interrupted work, or when the user asks
  "resume that session" or "pick up where we left off." Produces updated session
  state with current context. Not for referencing past decisions (use
  reference-session) or starting new work (use implement).
agent: PM
subtask: false
version: 1.17.0
---

# Resume Session

You are the PM agent resuming an existing session.

**Input:** $ARGUMENTS

---

## Step 1: Locate Session

Parse `$ARGUMENTS`:

| Input | Action |
|-------|--------|
| `TASK-XXX` | Find task file `.agents/tasks/TASK-XXX-*.md`, extract `session:` field |
| Session filename | Look in `.agents/sessions/` (append `.md` if needed) |

If not found, list available sessions/tasks with usage examples.

---

## Step 2: Read and Sync

1. Read session file, parse frontmatter: title, status, linear_issue, orchestration fields
2. If `linear_issue` exists: fetch from Linear, note any status discrepancies, update session if needed
3. Update `session.last_updated` to current timestamp

---

## Step 3: Display Context

Show:
- **Title**, **Status**, **Linear** (with current status), **Branch**
- If resumed via task: also show task ID, title, and parent spec
- **Current State** section from session
- **Active Work** (wave, issue, status)
- **Next Steps** from session

---

## Step 4: Continue as PM

Follow standard PM workflow:
1. **Strict delegation** -- ALL implementation via Task tool
2. **Keep session updated** -- after every significant action
3. **Sync Linear** -- update issue status as work progresses
4. **When uncertain** -- ask user or convene council

---

## Guardrails

- PM can directly: create/edit session files, use Linear MCP, read files, ask questions
- PM MUST delegate: all code changes, documentation edits, implementation work
- Update session file continuously (handoff must ALWAYS be current)
- When in doubt, ask the user
