---
name: breakdown
description: >-
  Decomposes specs into atomic tasks with dependencies and priorities. Use when the user asks
  "break this down" or "create tasks for this spec."
---

# Breakdown

Decompose specifications into atomic, implementable tasks.

## Contents
- Task Breakdown Philosophy
- Task Backend Detection
- Process
- Priority Levels
- Guardrails
- Related Skills

**Input:** $ARGUMENTS

---

## Task Breakdown Philosophy

**Primary principle: separation of concerns.**

### Right-Sizing Rules

| Rule | Guideline |
|------|-----------|
| **One agent type** | Completable by ONE agent (backend-dev, frontend-dev, dba, qa, devops) |
| **One concern** | Touches one layer, service, or component |
| **Context-appropriate** | Fits in model context with room for exploration |
| **Not over-fragmented** | Don't split what naturally belongs together |

### The Right Size Test

1. Can a single specialized agent complete this? If no, split by agent type
2. Does it touch multiple unrelated concerns? If yes, split by concern
3. Will the agent need too much context? If yes, split into phases
4. Am I splitting just to have more tasks? If yes, merge back

### Anti-Patterns

| Don't | Do Instead |
|-------|------------|
| Split backend + tests into separate tasks | Keep tests with the code they test |
| Create a task per file | Group files by concern |
| Separate "implement" and "verify" tasks | Every task includes its own verification |

---

## Task Backend Detection

Check `.agents/loaf.yaml`:
```yaml
task_management:
  backend: linear  # or "local"
```
If no config exists, ask user.

---

## Process

### Step 1: Parse Input

`$ARGUMENTS` should reference a spec (e.g., "SPEC-001"). If unclear, list available specs.

### Step 2: Read the Spec

Extract: test conditions, scope, implementation notes, appetite.

### Step 3: Identify Task Boundaries

Break down by concern (DBA, Backend, Frontend, QA, DevOps). One concern per task. Explicit dependencies for sequential tasks.

### Step 4: Interview for Priorities

Ask about highest priority parts, required ordering, preferred implementation sequence.

### Step 5: Draft Task List

Create tasks following [task template](templates/task.md). Each task needs: clear title, priority, file hints, verification command, observable done condition.

### Step 6: Present for Approval

Show tasks with priorities, dependencies, and dependency graph. **Do NOT create tasks without explicit approval.** User may adjust priorities, combine/split, add tasks, or change dependencies.

### Step 7: Create Tasks

**Linear backend:** Create issues with title, description, labels, priority.

**Local backend:** Create files in `.agents/tasks/TASK-{id}-{slug}.md`.

### Step 8: Update Spec and Announce

Set spec status to `implementing`. Announce created tasks with next steps.

---

## Priority Levels

| Priority | Meaning |
|----------|---------|
| P0 | Urgent/blocking -- drop everything |
| P1 | High -- work next |
| P2 | Normal -- scheduled work (default) |
| P3 | Low -- when time permits |

---

## Guardrails

1. **One concern per task** -- don't mix backend + frontend
2. **Clear verification** -- how to prove it works
3. **Observable done condition** -- not subjective
4. **File hints** -- help session know where to look
5. **Get approval** -- don't create without confirmation
6. **Update spec status** -- mark as implementing

---

## Related Skills

- **shape** -- Create specs that get broken down
- **implement** -- Start session for a task or coordinate multiple tasks
