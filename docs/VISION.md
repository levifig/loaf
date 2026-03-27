# Loaf Vision

Every agent session starts with perfect context — the right domain knowledge, the right conventions, the right personal preferences. Knowledge stays fresh because maintenance is woven into the work. What you learn in one project improves how agents help you in every project.

## What Loaf Becomes

Loaf evolves from a build-time skill framework into an **agent-agnostic CLI** for managing skills, knowledge, tasks, and distribution across agent harnesses.

### The Three Pillars

**1. Skills & Distribution** (exists today)
Agent Skills standard, multi-target build system, hooks, agents. Distribute knowledge and automation to Claude Code, Cursor, Codex, OpenCode, Gemini, and more.

**2. Knowledge Management** (building next)
Living knowledge bases that detect their own decay. Project knowledge + shared domain knowledge + personal knowledge — all searchable, all fresh. Powered by QMD for retrieval, Loaf for lifecycle.

**3. Autonomous Execution** (future)
Plan and spec during the day. Implement overnight. Claude Code builds, Codex reviews, feedback loops iterate until done. Commit, PR, morning review. Powered by Claude Code SDK + Codex SDK.

## The Agent Experience

Agents working with Loaf-equipped projects have:
- **Domain knowledge** loaded at session start (knowledge files, ADRs)
- **Staleness awareness** — stale knowledge is flagged, not silently wrong
- **Growth prompts** — sessions produce knowledge, not just code
- **Task context** — what's in progress, what's next, what's done
- **Cross-harness consistency** — same knowledge system works on Claude Code, Codex, Cursor

## The Human Experience

Humans working with Loaf have:
- **A CLI** that manages everything (`loaf kb`, `loaf task`, `loaf build`, `loaf install`)
- **Agent-authored knowledge** that humans review, not write from scratch
- **Visibility** into project health (`loaf kb status`, `loaf task status`)
- **Overnight implementation** — set up work, let agents execute, review results

## Principles

1. **Agent-creates, human-curates** — agents do the maintenance work, humans make judgment calls
2. **Hooks automate what skills teach** — Claude Code gets automation, other harnesses get instructions, same outcome
3. **CLI as cross-harness equalizer** — Bash works everywhere
4. **Maintenance as side effect** — knowledge stays fresh because it's woven into the work, not a separate chore
5. **Progressive complexity** — start with conventions, add automation, add intelligence
