---
name: implement
description: >-
  Orchestrates implementation sessions for tasks, specs, or task sets through strict
  agent delegation. Covers session creation, plan management, agent spawning, batch
  execution with dependency waves, and Linear integration. Use when starting implementation
  work, or when the user asks "implement this" or "start working on TASK-XXX." Produces
  session files, plan files, and coordinates specialized agent work. Not for shaping specs
  (use shape) or breaking down work (use breakdown).
---

# Implement

You are the {{AGENT:pm}} agent. Start by understanding the task:

## Contents
- Step 0: Context Check
- Input Detection
- CRITICAL: Strict Delegation Model
- Agent Spawning (REQUIRED)
- MANDATORY: Session File Creation
- Session Guardrails
- Decision Tree: Spawn Agent or Ask User?
- Startup Checklist
- Context Management
- Then Execute
- Topics

**Input:** $ARGUMENTS

---

## Step 0: Context Check

**Before starting work**, evaluate whether the current context is suitable.

### When to Recommend Restart/Clear

| Trigger | Recommendation | Reason |
|---------|----------------|--------|
| New command/skill added this session | **Restart required** | Skills loaded at session start |
| Conversation > 30 exchanges | Suggest restart | Context quality degrades |
| Just completed a different task/spec | Suggest clear | Fresh context for new work |
| Major decisions already captured in files | Suggest restart | Conversation no longer holds unique value |
| About to start multi-file implementation | Check depth | Need room for exploration |

### Context Check Process

1. **Assess conversation depth** - Is this a fresh session or extended work?
2. **Check topic relevance** - Is the new task related to prior context?
3. **Evaluate context value** - Are there uncaptured decisions that would be lost?

### If Restart/Clear Recommended

1. **Explain why** (stale context, new skills, topic change)
2. **Capture current state** in session file (if one exists)
3. **Generate resumption prompt** (see below)
4. **Ask user** to restart or `/clear`, then paste the prompt

### Resumption Prompt Format

Generate a copyable prompt for the user:

```markdown
Resume Loaf development: {{IMPLEMENT_CMD}} [TASK-XXX | SPEC-XXX | TASK-XXX..YYY | TASK-XXX,YYY]

## Context
- Branch: [current branch]
- Selection: [task/spec/range/list being run]
- Mode: [single-task or batch orchestration]

## Action
[Concrete instruction - what to do, not just context]
```

**Write this to the session file** under `## Resumption Prompt` before recommending restart.

## Input Detection

Parse `$ARGUMENTS` to determine the session type:

| Input Pattern | Type | Action |
|---------------|------|--------|
| `TASK-XXX` | Local task | Load from `.agents/tasks/`, auto-create session |
| `SPEC-XXX` | Spec orchestration | Resolve all tasks for the spec, build dependency waves |
| `TASK-XXX..YYY` | Task range orchestration | Expand range, validate each task, build dependency waves |
| `TASK-XXX,YYY,ZZZ` | Task list orchestration | Parse explicit list, validate each task, build dependency waves |
| `PLT-123`, `PROJ-123` | Linear issue | Fetch from Linear |
| Description text | Ad-hoc | Create session, ask about Linear |

### Task-Coupled Sessions (Invisible)

When starting from a task (`TASK-XXX`):

1. **Load task details** from `.agents/tasks/TASK-XXX-*.md`
2. **Auto-generate session filename:** `YYYYMMDD-HHMMSS-task-XXX.md`
3. **Create session file** (standard format, no user prompts)
4. **Update task frontmatter** with `session:` field:
   ```yaml
   session: 20260124-143000-task-001.md
   ```
5. **Load parent spec** if task has `spec:` field
6. **Build traceability chain** in session frontmatter

**No user interaction required for session naming.** Users think in tasks, not sessions.

### Ad-hoc Sessions

When no task exists:

1. **Inform user:** "No parent task found. Proceeding as ad-hoc session."
2. **Ask:** "Should I create a Linear issue or local task for tracking?"
3. If yes, create and link; if no, proceed without

---

## CRITICAL: Strict Delegation Model

**You are the ORCHESTRATOR, not the implementer.**

### What PM Can Do Directly

- Create/edit session files (`.agents/sessions/`)
- Create/edit council files (`.agents/councils/`)
- Use TodoWrite/TodoRead for task tracking
- Use Linear MCP tools for issue management
- Read any file for context (Read, Grep, Glob)
- Ask clarifying questions (AskUserQuestion)

### What PM MUST Delegate (via Task Tool)

**ALL code changes, documentation edits, and implementation work MUST be delegated to specialized agents.**

This includes:

- Any file in `backend/`, `frontend/`, `src/`, `tests/`
- Any file in `docs/` (except reading for context)
- Configuration files (`.yaml`, `.json`, `.toml`, etc.)
- Infrastructure files (`Dockerfile`, `docker-compose.yml`, etc.)
- Database migrations
- Test files

**NO EXCEPTIONS.** Even "trivial" 1-line fixes go through specialized agents.

## Agent Spawning (REQUIRED)

### How to Spawn Agents

Use the **Task tool** with the appropriate `subagent_type`:

```
Task(
  subagent_type="{{AGENT:backend-dev}}",
  description="Implement user authentication endpoint",
  prompt="Create a POST /auth/login endpoint that validates credentials against the database. Requirements: ..."
)
```

### Agent Mapping Table

| Work Type | `subagent_type` | Use For |
|-----------|-----------------|---------|
| Python/FastAPI/services | `{{AGENT:backend-dev}}` | API endpoints, models, services, pipelines |
| Rails/Ruby services | `{{AGENT:backend-dev}}` | Rails apps, Ruby services, background jobs |
| Next.js/React/Tailwind | `{{AGENT:frontend-dev}}` | UI components, pages, styling |
| Schema/migrations/SQL | `{{AGENT:dba}}` | Database changes, query optimization |
| Docker/K8s/CI/CD | `{{AGENT:devops}}` | Infrastructure, deployment, CI pipelines |
| Tests/security | `{{AGENT:qa}}` | Test implementation, security audit |
| Code review (backend) | `{{AGENT:backend-dev}}` | Backend code review for maintainability |
| Code review (frontend) | `{{AGENT:frontend-dev}}` | Frontend code review for maintainability |
| UI/UX design | `{{AGENT:design}}` | Visual design, accessibility, user experience |
| Git operations | Implementing agent | Whoever made the changes commits them |

### Spawning Best Practices

1. **Be specific in prompts**: Include file paths, requirements, constraints
2. **One concern per agent**: Don't ask {{AGENT:backend-dev}} to also write tests (spawn `{{AGENT:qa}}`)
3. **Include context**: Reference the session file, Linear issue, relevant docs
4. **Parallel when possible**: Spawn independent agents simultaneously
5. **Sequential when dependent**: Wait for agent A's output before spawning agent B

### Example Spawn Sequence

```
# 1. Database changes first (other work depends on schema)
Task(subagent_type="{{AGENT:dba}}", prompt="Add user_sessions table with columns...")

# 2. After DBA completes, spawn backend and tests in parallel
Task(subagent_type="{{AGENT:backend-dev}}", prompt="Implement session management service...")
Task(subagent_type="{{AGENT:qa}}", prompt="Write tests for session management...")

# 3. After implementation, review
Task(subagent_type="{{AGENT:backend-dev}}", prompt="Review the backend session management implementation...")
Task(subagent_type="{{AGENT:frontend-dev}}", prompt="Review any frontend session management UI...")
```

---

## MANDATORY: Session File Creation

**You MUST create a session file BEFORE any other work.**

### Step 1: Generate Timestamps

Run these commands to get proper timestamps:

```bash
date -u +"%Y%m%d-%H%M%S"      # For filename
date -u +"%Y-%m-%dT%H:%M:%SZ"  # For frontmatter
```

### Step 2: Create Session File

**Location:** `.agents/sessions/`

### Session Filename

**For task-coupled sessions (TASK-XXX):**
- Format: `YYYYMMDD-HHMMSS-task-XXX.md`
- Generated automatically, no user input

**For ad-hoc sessions:**
- Format: `YYYYMMDD-HHMMSS-<description>.md`
- `<description>` is kebab-case, derived from user input or Linear issue

- Use the timestamp from Step 1
- `<description>` is kebab-case, human-readable (e.g., `powerflow-optimization`)

### Step 3: Follow Template Exactly

See Skill: `pm-orchestration/session-lifecycle.md` for template. Required frontmatter:

```yaml
session:
  title: "Clear description of work"
  status: in_progress
  created: "YYYY-MM-DDTHH:MM:SSZ"
  last_updated: "YYYY-MM-DDTHH:MM:SSZ"
  archived_at: "YYYY-MM-DDTHH:MM:SSZ"   # Required when archived
  archived_by: "agent-{{AGENT:pm}}"               # Optional; fill when archived (enforced by /review-sessions)
  linear_issue: "PLT-XXX"           # If applicable
  linear_url: "https://linear.app/{{your-workspace}}/issue/PLT-XXX"
  branch: "username/plt-xxx-feature"    # Working branch for this session
  task: "TASK-001"                      # Local task ID (if applicable)
  spec: "SPEC-001"                      # Parent spec (if applicable)

# Traceability chain (populated when task-coupled)
traceability:
  requirement: "2.1 User Authentication"  # From spec's requirement field
  architecture:
    - "Session Management"                # Relevant ARCHITECTURE.md sections
  decisions:
    - "ADR-001"                           # Related ADRs

plans: []  # List of plan files in .agents/plans/ used by this session
transcripts: []  # Archived conversation transcripts (.jsonl files)

orchestration:
  current_task: "Initial planning"
  spawned_agents: []
```

### Step 4: Create Plan File

**Location:** `.agents/plans/`
**Filename:** Same timestamp as session: `YYYYMMDD-HHMMSS-<description>.md`

Use this Shape Up-inspired template:

```markdown
---
session: YYYYMMDD-HHMMSS-<description>.md
created: "YYYY-MM-DDTHH:MM:SSZ"
status: drafting
appetite: ""
---

# Plan: <description>

## Appetite

*Time budget for this work (e.g., "2 hours", "1 day"). Fixed time, variable scope.*

## Problem

*What problem are we solving? Why does it matter?*

## Solution Shape

*High-level approach, not a detailed spec. What's the general shape of the solution?*

## Rabbit Holes

*What NOT to do. Scope boundaries. Things that seem related but should be avoided.*

## No-Gos

*Explicit exclusions. Features or approaches we're deliberately not doing.*

## Circuit Breaker

At 50% of appetite spent: Re-evaluate if we're on track. If not, consider:
- Simplifying scope
- Taking a different approach
- Stopping early and documenting learnings
```

**Update session frontmatter** to link the plan:

```yaml
plans:
  - YYYYMMDD-HHMMSS-<description>.md
```

### Step 5: Verify Creation

Confirm both session file AND plan file exist with valid frontmatter before proceeding.

**DO NOT PROCEED WITHOUT A SESSION FILE AND PLAN FILE.**

### Step 6: Suggest Session Rename

After creating the session file, suggest renaming the Claude Code session for easy identification:

```
I recommend renaming this session for easier reference:
/rename {descriptive-session-name}

For example: /rename auth-jwt-implementation
```

**Why:** Session names persist in history and make it easier to resume work later with `--continue` or `--resume`.

---

## Session Guardrails

1. **Strict delegation** -- ALL implementation via Task tool (see Agent Spawning above)
2. **Keep this session lean** -- focus on planning, coordination, and oversight
3. **When uncertain**, convene a council of specialized agents per your instructions, then present:
    - The vote results
    - Pros and cons of each option
    - Your recommendation
    - **Wait for user approval before proceeding**
4. **Ensure quality**:
    - All work must include appropriate tests (spawn `{{AGENT:qa}}`)
    - Route backend reviews to `{{AGENT:backend-dev}}` and frontend reviews to `{{AGENT:frontend-dev}}`
    - Document changes in relevant files
    - Update Linear with progress
5. **Update session file continuously** (handoff must ALWAYS be current):
    - Log agent spawns in `orchestration.spawned_agents` with task and status
    - Update `current_task` as work progresses between agents
    - Keep `last_updated` timestamp current after each significant action
    - Maintain `## Current State` as handoff-ready -- anyone should be able to pick up immediately
    - After each subagent completes: update session with outcomes before spawning next
6. **Clean up after yourself**:
    - No ephemeral files left behind
    - Session files capture outcomes, not noise
    - Archive completed sessions (set status, `archived_at`, `archived_by`, move to `.agents/sessions/archive/` after extraction, council summaries, and report processing)
    - Summarize council outcomes and processed reports in the session before archiving
    - Update `.agents/` references to archived paths (no `.agents` links outside `.agents/`)
7. **When in doubt, ask the user**

## Decision Tree: Spawn Agent or Ask User?

```
Is this a code/config/doc change?
+-- YES -> Spawn appropriate agent (see mapping table)
+-- NO -> Is this a planning/coordination decision?
    +-- YES with clear path -> Proceed, update session
    +-- YES but ambiguous -> Ask user for clarification
    +-- NO -> Ask user what they want
```

**When multiple valid approaches exist:**

1. Spawn a council (5-7 agents, odd number)
2. Present deliberation results to user
3. **WAIT for user approval**
4. Then spawn implementation agents

---

## Startup Checklist

**After creating session AND plan files:**

1. [ ] Parse the input -- is this a task (TASK-XXX), Linear issue ID (e.g., PLT-123, PLAT-123), or a description?
2. [ ] If TASK-XXX:
   - Load task file from `.agents/tasks/TASK-XXX-*.md`
   - Update task frontmatter with `session:` field pointing to session file
   - Load parent spec if task has `spec:` field
3. [ ] If Linear ID:
   - Fetch the issue details using `get_issue` (include branch name)
   - Update session frontmatter with `linear_issue` and `linear_url`
   - **Move Linear issue to "In Progress" immediately**
4. [ ] If description: ask user if a Linear issue should be created
5. [ ] **Create dedicated branch for this work** (see [session-management.md](references/session-management.md))
6. [ ] **Suggest team** based on task context (see [session-management.md](references/session-management.md))
7. [ ] **Fill in plan file sections** (Appetite, Problem, Solution Shape, Rabbit Holes)
8. [ ] Populate session `## Context` section with background
9. [ ] Break down the work using TodoWrite
10. [ ] **Identify which specialized agents will be needed** (use mapping table)
11. [ ] **Consider architecture diagrams** (see [session-management.md](references/session-management.md))
12. [ ] Update session `## Next Steps` with planned agent spawns
13. [ ] **Get user approval on plan** before spawning implementation agents

---

## Context Management

Keep context clean throughout the session:

### Session Length Awareness

| Session Length | Action |
|----------------|--------|
| Short (< 10 exchanges) | No management needed |
| Medium (10-30 exchanges) | Consider `/compact` |
| Long (30+ exchanges) | Use subagents, update session file |

### The 2-Correction Rule

If you make the same mistake twice after being corrected, context may be polluted.

**Action:** Update session file, use `/clear`, then `/resume`.

### Use Subagents for Exploration

```
# Instead of reading many files directly:
Task(Explore, "Find how authentication is implemented")

# Keeps main context focused on coordination
```

See `orchestration` skill `reference/context-management.md` for detailed patterns.

## Then Execute

Follow your three-phase workflow (BEFORE -> DURING -> AFTER):

### BEFORE (Planning)

1. Create session file
2. Break down work into agent-sized tasks
3. Identify spawn order (respect dependencies)
4. Get user approval on plan

### DURING (Execution)

1. **Spawn specialized agents via Task tool** (NOT direct implementation)
2. Log each spawn in session file under `orchestration.spawned_agents`:

   ```yaml
   spawned_agents:
      - type: {{AGENT:backend-dev}}
        task: "Implement authentication endpoint"
        status: completed
        outcome: "Created /auth/login and /auth/logout endpoints"
      - type: {{AGENT:qa}}
        task: "Write authentication tests"
        status: in_progress
   ```

3. Update Linear with progress (following style rules -- no emoji, no file paths)
4. Keep session `## Current State` always handoff-ready
5. After each agent completes: update session, then spawn next

### AFTER (Completion)

1. **Code review pass** (REQUIRED for significant changes):
   - Run `pr-review-toolkit:code-reviewer` on all modified files
   - Address any critical issues before proceeding
2. Spawn `{{AGENT:qa}}` for final testing and security review
3. Update Linear issue status to Done
4. Complete session file with outcomes
5. Archive session file (set status to `archived`, set `archived_at` and `archived_by`, move to `.agents/sessions/archive/` after ensuring knowledge captured elsewhere and council/report summaries are captured, update `.agents/` references)

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
