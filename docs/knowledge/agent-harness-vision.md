---
topics: [harness, cross-platform, agents, agentic]
last_reviewed: 2026-03-14
---

# Agent Harness Vision

Loaf as an agentic harness — extended memory for agents that works across harnesses (Claude Code, Codex, OpenCode, Cursor).

## Core Principle: Hooks Automate What Skills Teach

On Claude Code: hooks fire automatically (SessionStart checks staleness, PostToolUse nudges on covered code). On other harnesses: the skill tells the agent to run the same commands manually via CLI. Same outcome, different trigger.

## Agent Experience

| Phase | Claude Code | Other Harnesses |
|-------|:-----------:|:---------------:|
| Session start | Hook surfaces stale knowledge + task status | Skill teaches: run `loaf kb check` and `loaf task status` |
| Working | PostToolUse nudges on covered code changes | Skill teaches: run `loaf kb check --file <path>` after edits |
| Search knowledge | QMD MCP tools | `qmd search --json` via Bash |
| Create/update knowledge | Write/Edit tools (same everywhere) | Same |
| Session end | Hook prompts consolidation | Skill teaches: check and consolidate before ending |

## CLI as Cross-Harness Equalizer

Core operations (Read, Write, Edit, Bash) exist on every harness. The `loaf` CLI provides intelligence. Hooks are a Claude Code/Cursor bonus, not a requirement.

## Agent Guidance Model

Agents are the primary authors of knowledge. The skill tells them:
- **When to create** — discovered domain rule, non-obvious edge case, architectural constraint
- **When to update** — covered code changed, new information from session
- **When to propose ADRs** — architectural decision made during brainstorm or implementation
- **Where to write** — local KB for project-specific, imported scope for domain knowledge

Humans review and curate. Agents maintain.

## Conversational Awareness

During normal conversation, watch for these patterns and respond naturally:

**Exploration happening** (user is open-ended, comparing, "what if we...")
→ Adopt divergent posture. Don't converge prematurely. Note emerging sparks.

**Domain insight discovered** (edge case, constraint, non-obvious rule)
→ Offer to create or update a knowledge file.

**Architectural decision made** (choice between approaches, with reasoning)
→ Propose an ADR.

**Speculative idea emerges** (interesting but not actionable yet)
→ Note it as a spark in the current brainstorm doc, or mention it for later.

**Idea crystallizes** (specific, actionable concept takes shape)
→ Offer to capture via `/idea`.

**Scope is bounding** (constraints clear, in/out defined)
→ Suggest shaping via `/shape`.

Don't interrupt flow to capture. Weave it into the conversation naturally. Don't announce mode switches — just adjust posture.

## Cross-References

- [knowledge-management-design.md](knowledge-management-design.md) — the knowledge system design
- [cli-design.md](cli-design.md) — the CLI that enables cross-harness parity
