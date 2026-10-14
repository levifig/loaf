---
name: council
description: >-
  Convenes multi-agent council deliberations for decisions requiring diverse
  perspectives. Use when the user asks "call a council", "gather specialists",
  "get a council opinion", or "what do the experts think". Also activate when
  the user wants a structured debate between domain-specific viewpoints. Not for
  single-perspective research (use research) or architectural decisions that
  don't need multi-agent deliberation (use architecture).
user-invocable: true
argument-hint: '[topic]'
version: 0.6.0-rc.3
---

# Council

## Contents

- Critical Rules
- Verification
- Quick Reference
- Native Work References
- Journal Commands
- Topics

Convene multi-agent councils for complex decisions requiring diverse expert perspectives.

## Critical Rules

Before the first journal write, select the command in [Journal Commands](#journal-commands). Use that form for every entry in this skill and its references.

**Always**
- Select odd number of agents (5 or 7) — never even
- Spawn ALL agents in parallel in a single response
- Select and state the council composition, then proceed directly to spawning
- Document each perspective and create synthesis with consensus/disagreements
- Wait for explicit user decision — council advises, user decides
- Preserve the council file after the decision; cleanup requires separately authorized Housekeeping
- Log decision to the project journal: `loaf journal log "decision(scope): council outcome and user's choice"`
- Create the artifact with `loaf council new --title <title> --body-file <path>`

**Never**
- Use even number of agents (risks ties)
- Spawn agents sequentially — parallel only
- Pause for composition approval before spawning unless the user explicitly asks for it
- Make the decision yourself — present synthesis, let user choose
- Skip documenting minority perspectives

## Verification

After work completes, verify:
- Council created with `loaf council new --title <title> --body-file <path>` as `.agents/councils/COUNCIL-YYYYMMDD-slug.md`
- All 5-7 agents spawned and perspectives collected
- Synthesis includes consensus points, disagreements, trade-offs
- User decision recorded in council file
- Decision logged to the project journal
- Council retained; no unsupported archive command or automatic deletion was inferred

## Quick Reference

### Council Composition

Select 5-7 agents covering relevant domains:
- Technical domains (backend, frontend, infrastructure)
- Quality domains (security, performance, UX)
- Process domains (testing, DevOps, documentation)

### Spawn Pattern

All agents in ONE response:
```
Spawn agent 1 (domain X) + agent 2 (domain Y) + agent 3 (domain Z) + ...
```

Each agent receives:
- Decision question and options
- Domain-specific analysis factors
- Instruction to focus on THEIR expertise only

### Synthesis Structure

1. Consensus points (all agents agree)
2. Key disagreements (different perspectives)
3. Trade-off analysis per option
4. Overall recommendation with confidence level
5. Explicit deferral if genuinely ambiguous

## Native Work References

Council bodies are authored files under `.agents/councils/`, not SQLite bodies or shared work definitions. `loaf council new --title <title> --body-file <path>` writes the file metadata and registers its identity in initialized state. Discover with `loaf council list` and read with `loaf council show <ref>`; these CLI operations currently require available SQLite state. An explicit file can also be read directly without claiming database retrieval.

When a council resolves a native issue's open questions, retain private deliberation locally and record only appropriate decision evidence or an authorized reference through the selected provider. The native tracker continues to own scope, hierarchy, and state. Deliberately adopted architectural conclusions belong in maintained Git topics; saving a council does not automatically promote its whole body into Git.

Record the user's decision in the council file and the project journal. The current council command has no archive operation. Preserve existing files and identifiers; later cleanup belongs to Housekeeping with exact approval.

## Journal Commands

Use the ordinary command `loaf journal log "type(scope): description"` with your harness's permissions. Apply only the labeled section for the harness you are running.

### Codex

With Loaf's managed basic-command policy installed, use `loaf journal log --execpolicy-safe "type(scope): description"` for every entry, including invocation logs. Keep `--execpolicy-safe` immediately after `journal log` and invoke PATH `loaf` as a standalone command, without an absolute executable pin or shell/environment wrapper. The flag does not install the policy or grant permission by itself; without the policy, use the ordinary command and Codex's approval flow.

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Council Template | [templates/council.md](templates/council.md) | Drafting the body for `loaf council new --body-file` |
| Composition | `council/SKILL.md` | Selecting council agents |
| Delegation | `orchestration/references/delegation.md` | Spawning subagents |
