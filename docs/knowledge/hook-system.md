---
topics: [hooks, lifecycle, validation, agents]
covers:
  - "src/hooks/**/*"
  - "src/config/hooks.yaml"
consumers: [backend-dev, pm]
last_reviewed: 2026-03-14
---

# Hook System

Hooks are shell or Python scripts that run at lifecycle events. They validate, nudge, and surface context.

## Key Rules

- **Pre-tool hooks can block** (exit 2). Post-tool and session hooks are informational only.
- **Hooks receive JSON on stdin.** Tool name, input, file paths — parsed with shell or Python.
- **Agent-aware via `AGENT_TYPE` env var.** Hooks can vary behavior for backend-dev vs pm vs main.
- **Timeout-bounded.** Each hook has a timeout in milliseconds. Respect it.
- **Library utilities in `src/hooks/lib/`.** Agent detection, config reading, JSON parsing, timeout management.

## Hook Events

| Event | Timing | Can Block | Use Case |
|-------|--------|:---------:|----------|
| PreToolUse | Before Edit/Write/Bash | Yes | Secrets check, format validation |
| PostToolUse | After Edit/Write/Bash | No | Linting, code analysis |
| SessionStart | Session begins | No | Context injection, status |
| SessionEnd | Session ends | No | Cleanup, completion checklist |
| PreCompact | Before context archival | No | State preservation |

## Registration

Hooks are defined in `src/config/hooks.yaml` with: `id`, `skill`, `script`, `matcher`, `blocking`, `timeout`, `description`. They're grouped under `pre-tool`, `post-tool`, or `session`.

## Cross-References

- [build-system.md](build-system.md) — how hooks get distributed to targets
- [skill-architecture.md](skill-architecture.md) — how skills own hooks via plugin-groups
