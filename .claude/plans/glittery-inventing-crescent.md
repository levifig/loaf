# Plan: Conversational Awareness in Knowledge-Base Skill + Hooks

## Context

The knowledge management pipeline (brainstorm → sparks → ideas → shape → spec) currently relies on explicit `/commands`. We want the agent to naturally recognize when exploration, capture, and decision-making are happening and respond accordingly — without waiting for commands.

**Approach:** Skills provide conversational guidance (what to look for). Hooks provide active reminders at transition points (session start/end). Together: passive awareness + active nudges.

## Changes

### 1. Add conversational awareness section to knowledge-base skill

**File:** `src/skills/knowledge-base/SKILL.md` — this skill doesn't exist yet, so we're defining what goes in it. For now, add the guidance to `docs/knowledge/agent-harness-vision.md` (the knowledge file that describes agent behavior). The actual skill creation is SPEC-009 work.

**File:** `docs/knowledge/agent-harness-vision.md`

Add a "Conversational Awareness" section:

```markdown
## Conversational Awareness

During normal conversation, watch for these patterns and respond naturally:

**Exploration happening** (user is open-ended, comparing, "what if we...")
→ Adopt divergent posture. Don't converge prematurely. Note emerging sparks.

**Domain insight discovered** (edge case, constraint, non-obvious rule)
→ Offer to create or update a knowledge file.
  "That's an important constraint. I'll capture it in docs/knowledge/."

**Architectural decision made** (choice between approaches, with reasoning)
→ Propose an ADR.
  "We just decided X because Y. I'll propose an ADR."

**Speculative idea emerges** (interesting but not actionable yet)
→ Note it as a spark in the current brainstorm doc, or mention it for later.
  "That's an interesting possibility — I'll note it as a spark."

**Idea crystallizes** (specific, actionable concept takes shape)
→ Offer to capture via /idea.
  "This is concrete enough to capture. Want me to /idea it?"

**Scope is bounding** (constraints clear, in/out defined)
→ Suggest shaping.
  "This has enough shape for a spec. Want me to /shape it?"

Don't interrupt flow to capture. Weave it into the conversation naturally.
Don't announce mode switches. Just adjust posture.
```

### 2. Extend SessionStart hook scope

The SessionStart hook (planned for SPEC-009) already surfaces stale knowledge. Extend it to also surface unprocessed sparks.

**Add to:** `docs/knowledge/knowledge-management-design.md` (in the hooks section)

```markdown
### SessionStart Hook (Extended)

Surfaces both knowledge health AND spark status:
- "3 knowledge files relevant. 1 stale."
- "5 unprocessed sparks across 2 brainstorm documents."

This reminds the agent (and user) that exploration artifacts exist and may need processing.
```

### 3. Extend SessionEnd hook scope

The SessionEnd hook (planned for SPEC-009) already prompts for knowledge consolidation. Extend it to also prompt for sparks.

**Add to:** `docs/knowledge/knowledge-management-design.md` (in the hooks section)

```markdown
### SessionEnd Hook (Extended)

Prompts for both knowledge consolidation AND spark capture:
- "You modified 3 paths covered by knowledge files. Any updates needed?"
- "This session involved exploration. Any sparks worth noting?"

If the session produced a brainstorm document, remind about the ## Sparks section.
```

### 4. Update brainstorm/idea/shape skill descriptions for natural triggers

**Files:** `src/skills/brainstorm/SKILL.md`, `src/skills/idea/SKILL.md`, `src/skills/shape/SKILL.md`

Expand description fields to include more natural language triggers:

**brainstorm:** add "Also activate when the user is thinking out loud, exploring tradeoffs, or comparing approaches without committing."

**idea:** add "Also activate when a specific actionable concept crystallizes during conversation, not just when explicitly asked."

**shape:** add "Also activate when an explored idea has accumulated enough constraints and scope definition to bound."

## Files to Modify

- `docs/knowledge/agent-harness-vision.md` — add Conversational Awareness section
- `docs/knowledge/knowledge-management-design.md` — extend SessionStart and SessionEnd hook descriptions
- `src/skills/brainstorm/SKILL.md` — expand description for natural triggers
- `src/skills/idea/SKILL.md` — expand description for natural triggers
- `src/skills/shape/SKILL.md` — expand description for natural triggers

## Verification

- Read updated files and confirm conversational guidance is clear and non-intrusive
- Skill descriptions still fit within character budget (~1024 chars for description field)
- `npm run build` still succeeds
- No changes to hook scripts (those are SPEC-009 implementation)
