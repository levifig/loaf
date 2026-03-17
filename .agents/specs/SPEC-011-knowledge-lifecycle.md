---
id: SPEC-011
title: "Knowledge Lifecycle — doc relocation + /crystallize"
source: direct
created: 2026-03-16T23:50:30Z
status: drafting
appetite: "Medium (3-5 days)"
---

# SPEC-011: Knowledge Lifecycle

## Problem Statement

Loaf sessions generate architectural decisions, strategic insights, and vision refinements — but nothing systematically pushes those learnings back into the durable project documents. VISION.md, STRATEGY.md, and ARCHITECTURE.md sit in `.agents/` (gitignored, agent-internal) when they're actually core project documentation that belongs alongside `docs/knowledge/` and `docs/decisions/`. And when a session completes, the learnings stay trapped in ephemeral session files unless someone manually extracts them.

## Strategic Alignment

- **Vision:** Directly supports the "living knowledge base" pillar. Sessions are where knowledge is created; crystallization is how it flows into durable docs.
- **Personas:** Framework user gets discoverable project docs in `docs/`. Agents get a systematic way to update strategic context.
- **Architecture:** Follows ADR-006 (agents create, humans curate) — `/crystallize` proposes changes, humans approve.

## Solution Direction

### Part 1: Doc Relocation

Move strategic documents from `.agents/` to `docs/` root:
- `.agents/VISION.md` → `docs/VISION.md`
- `.agents/STRATEGY.md` → `docs/STRATEGY.md`
- `.agents/ARCHITECTURE.md` → `docs/ARCHITECTURE.md`

Update all references across skills, hooks, session templates, and AGENTS.md. These are project docs, not agent artifacts — they should be version-controlled and visible.

### Part 2: `/crystallize` Skill

A new skill that reads the current session's learnings and proposes targeted updates to strategic docs.

**Trigger model:** Manual invocation (`/crystallize`) + suggested by `/cleanup` and session-end hooks when unextracted learnings are detected. Never auto-runs.

**Flow:**
1. Read current session file (or specified session)
2. Identify extractable insights: key decisions, architectural changes, vision shifts, strategy updates, new patterns
3. Read current VISION.md, ARCHITECTURE.md, STRATEGY.md
4. Propose diffs to each doc — show what would change and why
5. Human reviews and approves/rejects each proposed change
6. Apply approved changes

**What gets crystallized:**
- Architectural decisions not captured in ADRs → suggest ADR or update ARCHITECTURE.md
- Vision shifts (new capabilities, changed direction) → propose VISION.md updates
- Strategy changes (phase transitions, priority shifts) → propose STRATEGY.md updates
- New patterns or conventions → suggest docs/knowledge/ additions

**What does NOT get crystallized:**
- Implementation details (those live in git commits)
- Task tracking (that's TASKS.json / Linear)
- Session-specific context (stays in session file)

## Scope

### In Scope
- Move VISION.md, STRATEGY.md, ARCHITECTURE.md to `docs/`
- Update all references in skills, hooks, templates, AGENTS.md
- Create `/crystallize` skill with diff-proposal flow
- Add "suggest crystallize" logic to cleanup skill and session-end hook
- Skill sidecar for Claude Code configuration

### Out of Scope
- Auto-running crystallize (always manual + suggested)
- Creating a CLI command for crystallize (skill-only for now)
- Modifying the docs/knowledge/ or docs/decisions/ systems
- Knowledge staleness detection (that's SPEC-009)

### Rabbit Holes
- Building a smart diff engine — just show proposed text changes, don't build a merge tool
- Trying to auto-detect ALL learnings — focus on explicit signals (decisions, key outcomes)
- Over-engineering the suggestion trigger — a simple "session has unextracted decisions" check is enough

### No-Gos
- Don't auto-write to strategic docs without human approval
- Don't move docs/knowledge/ or docs/decisions/ (they're already in the right place)
- Don't duplicate information that belongs in ADRs

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Crystallize proposals are too noisy/low-quality | Medium | Low | Start conservative — only propose for explicit decisions and outcomes, not every session detail |
| Doc relocation breaks skill references | Low | Medium | Grep for all references, update systematically, verify with full build |
| Users skip crystallize | Medium | Low | Suggestion prompts in cleanup and session-end; the skill is optional by design |

## Open Questions

- [x] Doc location → `docs/` root
- [x] Crystallize output → proposed diffs for human review
- [x] Trigger model → manual + suggested (never auto)

## Test Conditions

- [ ] `docs/VISION.md`, `docs/STRATEGY.md`, `docs/ARCHITECTURE.md` exist and are tracked in git
- [ ] No remaining references to `.agents/VISION.md` etc. in skills or hooks
- [ ] `loaf build` succeeds with relocated docs
- [ ] `/crystallize` reads a session and proposes specific changes to strategic docs
- [ ] Proposed changes are presented as diffs (old → new) for human review
- [ ] Approved changes are written to the correct doc files
- [ ] `/cleanup` suggests running `/crystallize` when sessions have extractable learnings
- [ ] Session-end hook suggests `/crystallize` when session contains key decisions

## Circuit Breaker

At 50%: Drop the session-end hook suggestion. Ship doc relocation + crystallize skill. Cleanup suggestion can be a fast-follow.

At 75%: Drop the cleanup integration. Ship doc relocation + standalone `/crystallize` skill.
