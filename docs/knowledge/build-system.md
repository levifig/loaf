---
topics: [build-system, targets, distribution]
covers:
  - "build/**/*.js"
  - "src/config/targets.yaml"
  - "src/config/hooks.yaml"
consumers: [backend-dev, pm]
last_reviewed: 2026-03-14
---

# Build System

Loaf compiles skills, agents, and hooks from a single source tree into multiple target-specific formats.

## Key Rules

- **Single source, multiple outputs.** All content is authored in `src/`. The build system transforms it per-target.
- **Targets are additive.** Each target gets what it supports — Claude Code gets everything, Codex gets skills only.
- **Sidecars carry target-specific fields.** SKILL.md has standard fields only. `.claude-code.yaml`, `.cursor.yaml`, etc. carry extensions. Build merges them.
- **Shared templates distribute at build time.** `src/templates/` files are copied to specified skills via `shared-templates` in `targets.yaml`.

## Build Flow

`npm run build` → `build/build.js` loads `hooks.yaml` + `targets.yaml` → calls each target's transformer in `build/targets/{target}.js` → output to `plugins/` or `dist/{target}/`.

## Targets

| Target | Output | Agents | Skills | Hooks | MCP |
|--------|--------|:------:|:------:|:-----:|:---:|
| claude-code | `plugins/loaf/` | Yes | Yes | Yes | Yes |
| cursor | `dist/cursor/` | Yes | Yes | Yes | No |
| opencode | `dist/opencode/` | Yes | Yes | No | No |
| codex | `dist/codex/` | No | Yes | No | No |
| gemini | `dist/gemini/` | No | Yes | No | No |

## Cross-References

- [skill-architecture.md](skill-architecture.md) — how skills are structured
- [hook-system.md](hook-system.md) — how hooks are registered and distributed
