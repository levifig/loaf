---
id: SPEC-030
title: Librarian agent — session lifecycle management via task-driven journaling
source: 20260409-140251-session-lifecycle-states.md
created: 2026-04-09T15:30:00.000Z
status: implementing
branch: feat/librarian-session-lifecycle
session: 20260409-200731-session.md
---

# SPEC-030: Librarian Agent — Session Lifecycle Management via Task-Driven Journaling

## Problem Statement

Session lifecycle management has two problems:

1. **Prompt hooks misused for advisory work.** The Stop hook (`session-state-update`), `implement-routing`, and `journal-nudge` are prompt-type hooks used for side effects (writing summaries, nudging behavior). Prompt hooks are binary gates, but we use them for advisory work — the result is conversation bleed, blocking legitimate tool calls, and noisy feedback injected into the user's flow.

2. **No dedicated agent for session lifecycle.** Session housekeeping (wrap-ups, state updates, archive readiness) falls on the main conversation agent, fragmenting its attention between the user's work and bookkeeping.

**Constraints discovered during implementation:** Claude Code agent hooks (`type: "agent"`) are read-only — they can Read/Grep/Glob but cannot Edit or Write. This rules out using agent hooks for session state writes. The Librarian operates as a spawned agent (via the Agent tool), not a hook agent.

## Strategic Alignment

- **Vision:** Loaf agents have clear, lore-grounded responsibilities. The fellowship gains a dedicated steward for the living record of work.
- **Personas:** All agent users benefit — the Warden stops doing bookkeeping, Smiths stop being interrupted by stop hooks, sessions get consistent lifecycle management.
- **Architecture:** Uses Claude Code's native `agent` hook type (haiku subagent with tool access, up to 50 turns). Replaces prompt hooks with the correct hook primitive.

## The Librarian

**Librarians** — Ents who tend the living record. Patient, thorough, and long-memoried, they shepherd session files through their lifecycle as Treebeard shepherded the forests. Read + Edit access to session files and knowledge artifacts. They do not forge code or scout the web — they tend what already exists.

Instance names follow the Entish tradition — slow, deliberate, tree-rooted:
`{TreeName} — {concise purpose description}`
Example: `Bregalad — session state update on stop`

### Fellowship Table (Updated)

| Profile | Concept | Race | Tool Access | Use For |
|---------|---------|------|-------------|---------|
| **implementer** | Smith | Dwarf | Full write | Code, tests, config, docs |
| **reviewer** | Sentinel | Elf | Read-only | Audits, reviews |
| **researcher** | Ranger | Human | Read + web | Research, comparison |
| **librarian** | Librarian | Ent | Read + Edit (.agents/) | Session lifecycle, state, wrap |
| **background-runner** | System | — | Read + Edit | Async non-blocking tasks |
| **context-archiver** | System | — | Read + Edit + Serena | Session preservation (absorbed by Librarian) |

### Behavioral Contract

- Tend session files: Current State, journal quality, wrap summaries, lifecycle transitions.
- Never modify code, tests, or config — only `.agents/` artifacts.
- Never research, review, or orchestrate — those are other profiles' work.
- Work quickly and silently. The user should not notice the Librarian unless something is wrong.
- Default to sonnet model for summary quality. Downgrade to haiku after validating output quality — the summaries need to be contextually useful, not just mechanical.

## Solution Direction

### Track 1: Hook Cleanup + Task-Driven Journaling

**Remove problematic hooks:**
- `session-state-update` (Stop prompt) — removed. Caused conversation bleed.
- `implement-routing` (PreToolUse prompt on Edit|Write) — removed. Blocked legitimate edits.
- `journal-nudge` (PostToolUse prompt on Agent|WebFetch|WebSearch) — removed. Replaced by task events.

**Add task-based journal hooks:**
- PostToolUse on `TaskCreate` → `loaf session log --from-hook` → logs `task(create): #N — subject`
- PostToolUse on `TaskUpdate` → `loaf session log --from-hook` → logs `task(complete): #N` (only on completion)

**Journal entry sources (layered):**

| Source | Mechanism | When |
|--------|-----------|------|
| Skills | `loaf session log` in Critical Rules | Self-logging on invocation |
| Git events | PostToolUse command hooks | Commits, PRs, merges |
| Task events | PostToolUse command hooks | Task created/completed |
| PreCompact | Prompt hook (stays) | Emergency journal flush |

### Track 2: Librarian Agent Profile

Create a dedicated agent profile for session lifecycle work, spawned via the Agent tool (not hooks):

- `content/agents/librarian.md` — Ent lore, behavioral contract
- `content/agents/librarian.claude-code.yaml` — sonnet model, orchestration skill, Read+Edit+Glob+Grep tools
- SOUL.md updated with Librarian entry in fellowship table
- Used by `/wrap`, `/housekeeping`, and manual agent spawning

### Track 3: Wrap Skill — Composable Session Close

Make `/wrap` work as both interactive command and composable step (callable from `/release`):

- Librarian writes final state summary (AI-quality, from conversation context)
- Generates wrap report: what was done, decisions, loose ends
- Cleans up dangling `## Current State` sections — wrap report replaces them
- Closes journal (stop marker), marks session `complete`
- No archive — session stays in `sessions/` for housekeeping to handle
- Non-interactive mode: when invoked as sub-skill (e.g., from `/release`), skips prompts

### Track 4: Housekeeping Skill — Scheduling-Ready Maintenance

Make `/housekeeping` autonomous and idempotent:

- Find orphaned sessions (no matching branch, stale timestamps)
- Consolidate split sessions (same `claude_session_id`)
- Archive completed sessions past age threshold (`complete` → `archived`)
- Verify session/spec linkage consistency
- Check stale knowledge files
- Structured report output, no `AskUserQuestion` calls
- Works when invoked by `loaf schedule` or cron — no conversation context needed

### Track 5: Absorb Context Archiver (Future)

The context-archiver agent handles PreCompact session preservation. The Librarian's responsibilities are a superset — absorb context-archiver into the Librarian profile. Deferred: the PreCompact prompt hook + compact.sh script work adequately today.

## Scope

### In Scope
- Librarian agent profile (`content/agents/librarian.md`) with lore and behavioral contract
- Remove `session-state-update`, `implement-routing`, and `journal-nudge` hooks
- TaskCompleted session hook for journal auto-entries
- UserPromptSubmit hook for orchestration context injection
- PostCompact CLI command for rich resumption context
- Session `claude_session_id`-first lookup with split consolidation
- SOUL.md update with Librarian entry in fellowship table
- Wrap skill: composable session close, callable from `/release`
- Housekeeping skill: scheduling-ready, autonomous, idempotent

### Out of Scope
- JSONL enrichment pipeline (SPEC-029)
- Journal quality analysis (future Librarian capability)
- Context-archiver absorption (deferred — current PreCompact mechanism works)

### Rabbit Holes
- Making the Librarian a full orchestrator — it tends, it doesn't coordinate
- Giving the Librarian write access to code — it touches only `.agents/` artifacts
- Agent hooks for write operations — Claude Code agent hooks are read-only

### No-Gos
- The Librarian must not modify code files. Tool access is scoped to `.agents/` paths only.
- Do not re-introduce prompt hooks for advisory side effects on Stop or PostToolUse.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Task hooks fire too frequently, cluttering journal | Low | Low | Only log TaskCompleted; skip in_progress and other status changes |
| Agent forgets to use Task* tools consistently | Medium | Medium | UserPromptSubmit hook injects orchestration conventions; hooks are catch-all, not sole source |
| Librarian model quality on sonnet insufficient for state summaries | Low | Medium | Start with sonnet; validate during /wrap testing |

## Resolved Questions

- [x] Can agent hooks scope tool access? **No — read-only (Read, Grep, Glob, WebFetch, WebSearch). No Edit/Write/Bash.** Librarian operates as spawned agent instead.
- [x] Should Librarian run on every Stop? **No — removed Stop hook entirely. Librarian is invoked explicitly via /wrap and housekeeping.**
- [x] How do journal entries get created without prompt hooks? **TaskCompleted hook, git event hooks (PostToolUse on Bash), UserPromptSubmit context injection, and skill self-logging.**

## Test Conditions

- [x] `implement-routing` no longer blocks non-implementation edits
- [x] `session-state-update` Stop hook removed — no conversation bleed
- [x] `journal-nudge` removed — no prompt-based advisory noise
- [x] SOUL.md and fellowship table updated with Librarian profile
- [x] Librarian agent profile builds to all targets
- [x] `loaf build` succeeds, typecheck passes, tests pass
- [ ] TaskCompleted session hook → journal entry logged (completed + cancelled)
- [ ] UserPromptSubmit hook injects session context on every prompt
- [ ] Live test: no Stop hook feedback bleeding into conversation
- [ ] PostCompact prints rich resumption context (state + spec + journal + git)
- [ ] Session consolidation: splits detected and merged on session start
- [ ] `/wrap` closes session (complete status), does not archive
- [ ] `/wrap` callable from `/release` without interactive prompts
- [ ] `/housekeeping` runs autonomously (no AskUserQuestion)
- [ ] `/housekeeping` archives completed sessions past age threshold

## Priority Order

1. **Track 1: Hook Cleanup + Task-Driven Journaling** — Remove problematic hooks, add task/session event hooks, extend CLI. ✅
2. **Track 2: Librarian Agent Profile** — Agent profile with lore, behavioral contract, sidecar. ✅
3. **Track 3: Wrap Skill** — Composable session close, callable from `/release`.
4. **Track 4: Housekeeping Skill** — Scheduling-ready autonomous maintenance.
5. **Track 5: Context Archiver Absorption** — Deferred.
