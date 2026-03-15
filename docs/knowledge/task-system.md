---
topics: [tasks, specs, shape-up, sessions, orchestration]
covers:
  - ".agents/specs/**/*.md"
  - ".agents/tasks/**/*.md"
  - ".agents/sessions/**/*.md"
  - "src/skills/breakdown/**/*"
  - "src/skills/implement/**/*"
  - "src/skills/orchestration/**/*"
consumers: [pm, backend-dev]
last_reviewed: 2026-03-14
---

# Task System

Loaf implements a Shape Up-inspired task management system: specs define bounded work, tasks break it down, sessions track execution.

## Key Rules

- **Appetite over estimates.** Decide how much time work is worth, not how long it'll take.
- **Fixed time, variable scope.** Time is fixed; scope flexes when needed.
- **Circuit breakers.** Checkpoint at 50% appetite to re-evaluate.
- **One concern per task.** One agent type, one concern, fits in context with room for exploration.
- **Tasks are agent-authored, human-reviewed.** Agents create tasks via `/breakdown`, humans approve.

## Pipeline

```
/shape → SPEC file → /breakdown → TASK files → /implement → Sessions → Done
```

## Structure

| Artifact | Location | Purpose |
|----------|----------|---------|
| Specs | `.agents/specs/SPEC-XXX-slug.md` | Bounded work definitions (appetite, scope, test conditions) |
| Tasks | `.agents/tasks/TASK-XXX-slug.md` | Individual work items (acceptance criteria, verification) |
| Task index | `.agents/tasks/TASKS.json` | Programmatic index (CLI/TUI readable) |
| Sessions | `.agents/sessions/YYYYMMDD-HHMMSS-slug.md` | Execution context (linked to tasks) |

## TASKS.json

Programmatic index alongside individual task .md files. CLI reads/writes it. Future: TUI kanban, local web kanban.

```json
{
  "tasks": [
    { "id": "TASK-019", "title": "...", "spec": "SPEC-001", "status": "todo", "priority": "P1" }
  ],
  "completed": [
    { "id": "TASK-001", "title": "...", "completed_at": "2026-01-24T10:00:00Z" }
  ]
}
```

JSON is the machine index. Markdown files hold the detail. Same pattern as package-lock.json.

## Linear Integration

Optional sync layer. Local tasks are the primary interface. Linear pushes for team visibility when configured. No Linear = no problem.

## Cross-References

- [cli-design.md](cli-design.md) — `loaf task` and `loaf spec` CLI commands
- [knowledge-management-design.md](knowledge-management-design.md) — knowledge system uses similar Shape Up patterns
