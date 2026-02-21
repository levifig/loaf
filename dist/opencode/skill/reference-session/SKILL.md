---
name: reference-session
description: >-
  Imports decisions and context from past sessions. Use when the user asks
  "reference that earlier session" or "what did we decide before?"
---

# Reference Session

Import context from past sessions into current work without duplicating full context.

**Input:** $ARGUMENTS

---

## Process

### Step 1: Parse Arguments

Parse for: search term (session name fragment, Linear issue ID, or topic keyword), optional content type (`decisions`, `context`, or `all`; default: `decisions`).

```
/reference-session auth              # Search "auth", import decisions
/reference-session auth context      # Import full context summary
/reference-session PLT-123           # Search by Linear issue
/reference-session                   # List available session decisions
```

### Step 2: Search for Sessions

**Option A:** Use Serena MCP `list_memories()`, filter for `session-*-decisions.md`

**Option B:** Search `.agents/sessions/archive/*<search-term>*.md`

### Step 3: Display Matches

Show matching sessions with date, Linear issue, and decision count. If single match, proceed directly.

### Step 4: Read and Import

Read decision memory via Serena or extract from archived session file.

If active session exists: update session frontmatter `referenced_sessions` and add `## Referenced Decisions` section with decision name, what was decided, rationale, and context.

If no active session: display decisions without updating any file.

### Step 5: Report

Confirm what was imported: source session, content type, decision count, summary list, and updated session path.

---

## Content Types

| Type | Imports |
|------|---------|
| `decisions` (default) | Decision name, what was decided, rationale, council outcome |
| `context` | Problem statement, technical approach, key files, final outcome |
| `all` | Both decisions and context |

---

## Anti-Patterns

| Don't | Do Instead |
|-------|------------|
| Import full session files | Import only decisions/summaries |
| Import without tracking | Always update `referenced_sessions` |
| Import from unrelated sessions | Search by topic or Linear issue |

---

## Related Skills

- **resume-session** -- Resume active sessions (not reference past ones)
- **review-sessions** -- Session hygiene and cleanup
- **implement** -- Start new work that may reference past sessions
