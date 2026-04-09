---
id: SPEC-030
title: Keeper agent — session lifecycle management via task-driven journaling
source: 20260409-140251-session-lifecycle-states.md
created: 2026-04-09T15:30:00.000Z
status: implementing
branch: feat/keeper-agent-session-lifecycle
session: 20260409-195649-session.md
---

# SPEC-030: Keeper Agent — Session Lifecycle Management via Task-Driven Journaling

## Problem Statement

Session lifecycle management has two problems:

1. **Prompt hooks misused for advisory work.** The Stop hook (`session-state-update`), `implement-routing`, and `journal-nudge` are prompt-type hooks used for side effects (writing summaries, nudging behavior). Prompt hooks are binary gates, but we use them for advisory work — the result is conversation bleed, blocking legitimate tool calls, and noisy feedback injected into the user's flow.

2. **No dedicated agent for session lifecycle.** Session housekeeping (wrap-ups, state updates, archive readiness) falls on the main conversation agent, fragmenting its attention between the user's work and bookkeeping.

**Constraints discovered during implementation:** Claude Code agent hooks (`type: "agent"`) are read-only — they can Read/Grep/Glob but cannot Edit or Write. This rules out using agent hooks for session state writes. The Keeper operates as a spawned agent (via the Agent tool), not a hook agent.

## Strategic Alignment

- **Vision:** Loaf agents have clear, lore-grounded responsibilities. The fellowship gains a dedicated steward for the living record of work.
- **Personas:** All agent users benefit — the Warden stops doing bookkeeping, Smiths stop being interrupted by stop hooks, sessions get consistent lifecycle management.
- **Architecture:** Uses Claude Code's native `agent` hook type (haiku subagent with tool access, up to 50 turns). Replaces prompt hooks with the correct hook primitive.

## The Keeper

**Keepers** — Ents who tend the living record. Patient, thorough, and long-memoried, they shepherd session files through their lifecycle as Treebeard shepherded the forests. Read + Edit access to session files and knowledge artifacts. They do not forge code or scout the web — they tend what already exists.

Instance names follow the Entish tradition — slow, deliberate, tree-rooted:
`{TreeName} — {concise purpose description}`
Example: `Bregalad — session state update on stop`

### Fellowship Table (Updated)

| Profile | Concept | Race | Tool Access | Use For |
|---------|---------|------|-------------|---------|
| **implementer** | Smith | Dwarf | Full write | Code, tests, config, docs |
| **reviewer** | Sentinel | Elf | Read-only | Audits, reviews |
| **researcher** | Ranger | Human | Read + web | Research, comparison |
| **keeper** | Keeper | Ent | Read + Edit (.agents/) | Session lifecycle, state, wrap |
| **background-runner** | System | — | Read + Edit | Async non-blocking tasks |
| **context-archiver** | System | — | Read + Edit + Serena | Session preservation (absorbed by Keeper) |

### Behavioral Contract

- Tend session files: Current State, journal quality, wrap summaries, lifecycle transitions.
- Never modify code, tests, or config — only `.agents/` artifacts.
- Never research, review, or orchestrate — those are other profiles' work.
- Work quickly and silently. The user should not notice the Keeper unless something is wrong.
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

### Track 2: Keeper Agent Profile

Create a dedicated agent profile for session lifecycle work, spawned via the Agent tool (not hooks):

- `content/agents/keeper.md` — Ent lore, behavioral contract
- `content/agents/keeper.claude-code.yaml` — sonnet model, orchestration skill, Read+Edit+Glob+Grep tools
- SOUL.md updated with Keeper entry in fellowship table
- Used by `/wrap`, `/housekeeping`, and manual agent spawning

### Track 3: Absorb Context Archiver (Future)

The context-archiver agent handles PreCompact session preservation. The Keeper's responsibilities are a superset — absorb context-archiver into the Keeper profile. Deferred to a follow-up: the PreCompact prompt hook + compact.sh script work adequately today.

## Scope

### In Scope
- Keeper agent profile (`content/agents/keeper.md`) with lore and behavioral contract
- Remove `session-state-update`, `implement-routing`, and `journal-nudge` hooks
- Add PostToolUse hooks for TaskCreate/TaskUpdate → session journal entries
- Extend `loaf session log --from-hook` to parse task event payloads
- SOUL.md update with Keeper entry in fellowship table

### Out of Scope
- JSONL enrichment pipeline (SPEC-029)
- Housekeeping skill rewrite (Keeper participates but housekeeping orchestration is separate)
- Journal quality analysis (future Keeper capability, not initial scope)
- Context-archiver absorption (deferred — current PreCompact mechanism works)
- Session lifecycle state machine expansion (done/processed — deferred)

### Rabbit Holes
- Making the Keeper a full orchestrator — it tends, it doesn't coordinate
- Giving the Keeper write access to code — it touches only `.agents/` artifacts
- Agent hooks for write operations — Claude Code agent hooks are read-only

### No-Gos
- The Keeper must not modify code files. Tool access is scoped to `.agents/` paths only.
- Do not re-introduce prompt hooks for advisory side effects on Stop or PostToolUse.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Task hooks fire too frequently, cluttering journal | Low | Low | Only log TaskCreate and TaskUpdate(completed); skip other status changes |
| Agent forgets to use Task* tools consistently | Medium | Medium | Bake task discipline into orchestration skill defaults; hooks are catch-all, not sole source |
| Keeper model quality on sonnet insufficient for state summaries | Low | Medium | Start with sonnet; validate during /wrap testing |

## Resolved Questions

- [x] Can agent hooks scope tool access? **No — read-only (Read, Grep, Glob, WebFetch, WebSearch). No Edit/Write/Bash.** Keeper operates as spawned agent instead.
- [x] Should Keeper run on every Stop? **No — removed Stop hook entirely. Keeper is invoked explicitly via /wrap and housekeeping.**
- [x] How do journal entries get created without prompt hooks? **Task events (PostToolUse on TaskCreate/TaskUpdate), git hooks (PostToolUse on Bash), and skill self-logging.**

## Test Conditions

- [x] `implement-routing` no longer blocks non-implementation edits
- [x] `session-state-update` Stop hook removed — no conversation bleed
- [x] `journal-nudge` removed — no prompt-based advisory noise
- [x] SOUL.md and fellowship table updated with Keeper profile
- [ ] TaskCreate fires PostToolUse hook → journal entry logged
- [ ] TaskUpdate(completed) fires PostToolUse hook → journal entry logged
- [ ] TaskUpdate(in_progress) does NOT create journal entry
- [ ] Keeper agent profile builds to all targets
- [ ] `loaf build` succeeds, typecheck passes, tests pass

## Priority Order

1. **Track 1: Hook Cleanup + Task-Driven Journaling** — Remove problematic hooks, add task event hooks, extend CLI. Go/no-go: no conversation bleed, task events produce journal entries.
2. **Track 2: Keeper Agent Profile** — Agent profile with lore, behavioral contract, sidecar. Go/no-go: Keeper appears in built plugins, spawnable by Agent tool.
3. **Track 3: Context Archiver Absorption** — Deferred. Current PreCompact mechanism works.
