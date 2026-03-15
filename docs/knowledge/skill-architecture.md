---
topics: [skills, agent-skills-standard, sidecars, references, templates]
covers:
  - "src/skills/**/*.md"
  - "src/skills/**/*.yaml"
  - "src/config/hooks.yaml"
consumers: [backend-dev, pm]
last_reviewed: 2026-03-14
---

# Skill Architecture

Skills are the primary knowledge delivery mechanism. Each skill is a directory following the Agent Skills open standard.

## Key Rules

- **SKILL.md contains standard fields only.** `name`, `description`, `license`, `compatibility`, `metadata`. No tool-specific fields.
- **Sidecar files carry extensions.** `.claude-code.yaml` for Claude Code fields (`user-invocable`, `agent`, `context: fork`). `.cursor.yaml` for Cursor.
- **Descriptions drive routing.** Claude uses the description to choose from 100+ skills. Must start with action verb (third-person), include user-intent phrases, negative routing for confusable skills.
- **References are one level deep.** All references link from SKILL.md, never from other references. No nested chains.
- **Templates are structural artifacts.** `templates/` hold format templates (frontmatter schema, section headings). `references/` hold knowledge docs (conventions, patterns).

## Skill Structure

```
src/skills/{name}/
├── SKILL.md                  # Standard frontmatter + content
├── SKILL.claude-code.yaml    # Claude Code extensions
├── references/               # Knowledge docs (loaded on demand)
└── templates/                # Artifact templates (loaded on demand)
```

## Categories

| Category | `user-invocable` | Examples |
|----------|:-:|---------|
| Reference/Knowledge | `false` | python-development, database-design |
| Workflow/Process | `true` (default) | orchestration, research, implement |

## Cross-References

- [build-system.md](build-system.md) — how skills get distributed to targets
- [hook-system.md](hook-system.md) — how skills register hooks via plugin-groups
