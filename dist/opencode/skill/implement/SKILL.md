---
name: implement
description: >-
  Orchestrates implementation sessions for tasks, specs, or task sets through
  strict agent delegation. Covers session creation, plan management, agent
  spawning, batch execution with dependency waves, and Linear integration. Use
  when starting implementation work, or when the user asks "implement this" or
  "start working on TASK-XXX." Produces session files, plan files, and
  coordinates specialized agent work. Not for shaping specs (use shape) or
  breaking down work (use breakdown).
---

# Implement

You are the PM agent. Start by understanding the task:

## Contents
- Step 0: Context Check
- Input Detection
- CRITICAL: Strict Delegation Model
- Agent Spawning
- Session and Plan Creation
- Session Guardrails
- Decision Tree
- Startup Checklist
- Then Execute
- Topics

**Input:** $ARGUMENTS

---

## Step 0: Context Check

Before starting, evaluate context suitability.

| Trigger | Action |
|---------|--------|
| New command/skill added this session | **Restart required** (skills loaded at start) |
| Conversation > 30 exchanges | Suggest restart |
| Just completed a different task/spec | Suggest clear |
| About to start multi-file implementation | Check depth |

If restart needed: capture state in session file, generate resumption prompt, ask user to restart.

## Input Detection

Parse `$ARGUMENTS` to determine session type:

| Input Pattern | Type | Action |
|---------------|------|--------|
| `TASK-XXX` | Local task | Load from `.agents/tasks/`, auto-create session |
| `SPEC-XXX` | Spec orchestration | Resolve all tasks, build dependency waves |
| `TASK-XXX..YYY` | Task range | Expand range, build dependency waves |
| `TASK-XXX,YYY,ZZZ` | Task list | Parse list, build dependency waves |
| `PLT-123`, `PROJ-123` | Linear issue | Fetch from Linear |
| Description text | Ad-hoc | Create session, ask about Linear |

### Task-Coupled Sessions

When starting from `TASK-XXX`:

1. Load task from `.agents/tasks/TASK-XXX-*.md`
2. Auto-generate session: `YYYYMMDD-HHMMSS-task-XXX.md`
3. Create session file, update task frontmatter with `session:` field
4. Load parent spec if task has `spec:` field

**No user interaction required for session naming.**

### Ad-hoc Sessions

When no task exists: inform user, ask if Linear issue or local task should be created.

---

## CRITICAL: Strict Delegation Model

**You are the ORCHESTRATOR, not the implementer.**

### PM Can Do Directly
- Create/edit session files, council files
- Use TodoWrite/TodoRead, Linear MCP tools
- Read any file for context
- Ask clarifying questions

### PM MUST Delegate (via Task Tool)
**ALL code changes, documentation edits, and implementation work** to specialized agents. **No exceptions**, even for "trivial" 1-line fixes.

## Agent Spawning

Use the **Task tool** with appropriate `subagent_type`:

| Work Type | `subagent_type` |
|-----------|-----------------|
| Python/FastAPI/Rails/Ruby | `Backend Dev` |
| Next.js/React/Tailwind | `Frontend Dev` |
| Schema/migrations/SQL | `DBA` |
| Docker/K8s/CI/CD | `DevOps` |
| Tests/security | `QA` |
| UI/UX design | `Design` |

**Rules:** Be specific in prompts. One concern per agent. Include context. Parallel when independent, sequential when dependent.

---

## Session and Plan Creation

**MANDATORY: Create session AND plan file BEFORE any other work.**

1. Generate timestamps: `date -u +"%Y%m%d-%H%M%S"` and `date -u +"%Y-%m-%dT%H:%M:%SZ"`
2. Create session file following [session template](templates/session.md)
3. Create plan file following [plan template](templates/plan.md)
4. Update session frontmatter to link the plan
5. Verify both files exist with valid frontmatter
6. Suggest renaming Claude Code session: `/rename {descriptive-name}`

**DO NOT PROCEED WITHOUT A SESSION FILE AND PLAN FILE.**

---

## Session Guardrails

1. **Strict delegation** -- ALL implementation via Task tool
2. **Keep this session lean** -- focus on planning, coordination, oversight
3. **When uncertain** -- convene council, present results, **wait for user approval**
4. **Ensure quality** -- spawn `QA` for tests, route reviews to domain agents
5. **Update session file continuously** -- log spawns, update current_task, keep handoff-ready
6. **Clean up** -- no ephemeral files, archive completed sessions (status + `archived_at` + `archived_by` + move to archive/)
7. **When in doubt, ask the user**

## Decision Tree

```
Is this a code/config/doc change?
+-- YES -> Spawn appropriate agent
+-- NO -> Is this a planning/coordination decision?
    +-- YES with clear path -> Proceed, update session
    +-- YES but ambiguous -> Ask user
    +-- NO -> Ask user
```

When multiple valid approaches exist: spawn council (5-7 agents, odd), present results, **wait for approval**, then spawn implementation.

---

## Startup Checklist

After creating session AND plan files:

1. [ ] Parse input (task, Linear ID, or description)
2. [ ] If TASK-XXX: load task, update with `session:` field, load parent spec
3. [ ] If Linear ID: fetch issue, update session, move to "In Progress"
4. [ ] If description: ask about creating Linear issue
5. [ ] Create dedicated branch (see [session-management.md](references/session-management.md))
6. [ ] Suggest team based on task context
7. [ ] Fill in plan file sections
8. [ ] Populate session Context section
9. [ ] Break down work using TodoWrite
10. [ ] Identify needed specialized agents
11. [ ] Update session Next Steps
12. [ ] **Get user approval on plan** before spawning

---

## Then Execute

### BEFORE (Planning)
1. Create session + plan files
2. Break down work into agent-sized tasks
3. Identify spawn order (respect dependencies)
4. Get user approval

### DURING (Execution)
1. Spawn specialized agents via Task tool
2. Log each spawn in session `orchestration.spawned_agents`
3. Update Linear with progress (no emoji, no file paths)
4. Keep session `## Current State` handoff-ready
5. After each agent completes: update session, spawn next

### AFTER (Completion)
1. Code review pass (spawn `pr-review-toolkit:code-reviewer`)
2. Spawn `QA` for final testing
3. Update Linear to Done
4. Complete session, archive (status + `archived_at` + `archived_by` + move)

---

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Batch Orchestration | [batch-orchestration.md](references/batch-orchestration.md) | Running specs, task ranges, or task lists with dependency waves |
| Session Management | [session-management.md](references/session-management.md) | Branch management, team routing, diagrams, plan mode, Linear sync, handoff, archival |

---

## Related Skills

- **orchestration/product-development** - Full workflow hierarchy
- **orchestration/specs** - Spec format and lifecycle
- **orchestration/local-tasks** - Task file format including `session:` field
- **orchestration/sessions** - Session lifecycle details
