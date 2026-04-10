---
id: SPEC-030
title: "Keeper agent — session lifecycle management via agent hooks"
source: "20260409-140251-session-lifecycle-states.md"
created: 2026-04-09T15:30:00Z
status: drafting
---

# SPEC-030: Keeper Agent — Session Lifecycle Management via Agent Hooks

## Problem Statement

Session lifecycle management currently relies on prompt-type hooks that misuse Claude Code's hook model. Prompt hooks are binary gates (`ok: true/false`) designed for pass/fail decisions, but we use them for advisory work — writing Current State summaries, nudging journal entries, routing implementation work. The result: hooks that correctly evaluate "this is fine, skip" still block tool calls because any response is treated as a decision.

Meanwhile, session housekeeping (wrap-ups, state updates, archive readiness) falls on the main conversation agent, fragmenting its attention between the user's work and bookkeeping. There's no dedicated agent profile for session lifecycle management.

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

### Track 1: Agent Hook Conversion

Replace the three prompt-type hooks with agent hooks that spawn a Keeper:

| Hook ID | Event | Current Type | New Type | Keeper Action |
|---------|-------|-------------|----------|---------------|
| `session-state-update` | Stop | `prompt` | `agent` | Read session file, assess if Current State needs updating, write contextual summary if meaningful work happened |
| `implement-routing` | PreToolUse | `prompt` | Remove or convert to `prompt` with clean binary question | N/A — this was misusing prompt hooks for advisory nudging |
| `session-pre-compact-nudge` | PreCompact | `prompt` | `agent` | Flush journal entries, write state summary before compaction |

Each agent hook includes a `statusMessage` for the spinner (so the user sees what's happening, not a generic wait). The agent hook configuration in `hooks.yaml`:

```yaml
session:
  - id: session-state-update
    skill: orchestration
    type: agent
    prompt: >-
      You are a Keeper. Read the active session file for the current branch.
      If the conversation involved meaningful work (edits, commits, decisions),
      update the ## Current State section with a 2-4 line contextual summary.
      Skip silently if this was just Q&A. Return {"ok": true} when done.
      Session file: $ARGUMENTS
    model: sonnet
    event: Stop
    timeout: 30
    statusMessage: "Keeper tending session state..."
    description: Keeper updates session state after meaningful turns
```

### Track 2: Session Lifecycle States

Add intermediate states to the session lifecycle:

```
active → stopped → done → processed → archived
```

- **stopped** — SessionEnd fired, conversation over
- **done** — Keeper confirmed complete (no loose ends, or loose ends acknowledged)
- **processed** — Wrap summary exists, JSONL enrichment complete (SPEC-029)
- **archived** — Moved to `archive/`

The Keeper manages transitions:
1. On **Stop**: update Current State (Track 1)
2. On **SessionEnd**: verify session has stop marker (existing behavior)
3. On **housekeeping trigger**: scan stopped sessions → mark done if age threshold met → run wrap if missing → mark processed → archive

### Track 3: Absorb Context Archiver

The context-archiver agent handles PreCompact session preservation. The Keeper's responsibilities are a superset — absorb context-archiver into the Keeper profile. The Keeper handles:
- PreCompact: journal flush + state summary (currently context-archiver + prompt hook)
- PostCompact: resumption context (currently shell script)

## Scope

### In Scope
- Keeper agent profile (`content/agents/keeper.md`) with lore and behavioral contract
- Agent hook conversion for `session-state-update` and `session-pre-compact-nudge`
- `implement-routing` hook reclassification (remove or convert to clean binary prompt)
- Session lifecycle states (`done`, `processed`) in `SessionFrontmatter`
- SOUL.md update with Keeper entry
- Absorb context-archiver responsibilities

### Out of Scope
- JSONL enrichment pipeline (SPEC-029 — the Keeper calls `loaf session sync` but doesn't implement parsing)
- Housekeeping skill rewrite (Keeper participates but housekeeping orchestration is separate)
- Journal quality analysis (future Keeper capability, not initial scope)

### Rabbit Holes
- Making the Keeper a full orchestrator — it tends, it doesn't coordinate
- Giving the Keeper write access to code — it touches only `.agents/` artifacts
- Over-engineering lifecycle state machines — simple linear progression, not a DAG

### No-Gos
- The Keeper must not block the user's workflow. Agent hooks that take >30s on Stop are unacceptable.
- The Keeper must not modify code files. Tool access is scoped to `.agents/` paths only.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Agent hook latency on Stop adds noticeable delay | Medium | Medium | Use haiku, cap at 30s timeout, fail-open (ok: true on timeout) |
| Keeper state updates conflict with user's manual edits | Low | Low | Keeper reads before writing, uses atomic file operations |
| Agent hook `$ARGUMENTS` doesn't include session file path | Medium | High | Verify agent hook input schema; fall back to `loaf session` CLI discovery |
| Model quality insufficient for contextual summaries | Low | Medium | Start with sonnet; downgrade to haiku after validating output quality with real sessions |

## Open Questions

- [ ] Does the agent hook `$ARGUMENTS` placeholder include enough context for the Keeper to find the session file? (Need to test the actual JSON input for Stop events)
- [ ] Can agent hooks scope tool access (e.g., only Edit files in `.agents/`)? Or is it full tool access?
- [ ] Should the Keeper run on every Stop, or only when `stop_hook_active` is false? (Avoid infinite loops)
- [ ] What's the interaction between the Keeper agent hook and the existing `session-end-loaf` command hook on SessionEnd? Order of execution?

## Test Conditions

- [ ] Keeper agent hook fires on Stop, reads session file, writes contextual Current State — no user intervention needed
- [ ] Keeper correctly skips Q&A-only turns (returns ok: true without writing)
- [ ] Keeper handles PreCompact: flushes journal, writes state summary before compaction
- [ ] Session lifecycle transitions work: stopped → done → processed → archived
- [ ] Context archiver functionality preserved — PreCompact session preservation still works
- [ ] Agent hook latency on Stop is <10s for typical sessions
- [ ] `implement-routing` no longer blocks non-implementation edits
- [ ] SOUL.md and fellowship table updated with Keeper profile

## Priority Order

Tracks ship in this order. If scope needs cutting, drop from the end.

1. **Track 1: Agent Hook Conversion** — Keeper profile + Stop/PreCompact hooks converted. Go/no-go: Current State updates happen automatically without blocking edits or showing spurious errors.
2. **Track 2: Session Lifecycle States** — done/processed states + housekeeping transitions. Go/no-go: `loaf session list` shows lifecycle states; housekeeping manages transitions.
3. **Track 3: Absorb Context Archiver** — Merge context-archiver into Keeper. Go/no-go: PreCompact preservation works via Keeper; context-archiver.md removed.
