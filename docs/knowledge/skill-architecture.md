---
topics:
  - skills
  - agent-skills-standard
  - sidecars
  - references
  - templates
  - profiles
covers:
  - content/skills/**/*.md
  - content/skills/**/*.yaml
  - config/hooks.yaml
consumers:
  - implementer
  - reviewer
last_reviewed: '2026-07-14'
---

# Skill Architecture

Skills are the primary knowledge delivery mechanism. [Skill Portability](../architecture/skill-portability.md) owns the package-format choice, shared-authoring model, and rationale; this guide covers authoring layout and conventions.

## Key Rules

- **SKILL.md contains standard fields only.** `name`, `description`, `license`, `compatibility`, `metadata`. No tool-specific fields.
- **Sidecar files carry extensions.** `SKILL.claude-code.yaml` carries authorized Claude Code fields such as `user-invocable`, `model`, and `context`; `SKILL.opencode.yaml` carries authorized OpenCode metadata. The target's exact field ownership is enforced by the [build checker](../../internal/cli/build_skill_invariance.go).
- **Descriptions drive routing.** The model uses the description to choose from 100+ skills. Must start with action verb (third-person), include user-intent phrases, negative routing for confusable skills.
- **References are one level deep.** All references link from SKILL.md, never from other references. No nested chains.
- **Templates are structural artifacts.** `templates/` hold format templates (frontmatter schema, section headings). `references/` hold knowledge docs (conventions, patterns).
- **Author harness differences explicitly.** Use neutral workflow names in common prose and labeled product sections or a harness table for distinct invocation forms. Builders do not substitute commands in skill prose.

## Skill Structure

```
content/skills/{name}/
├── SKILL.md                  # Standard frontmatter + content
├── SKILL.claude-code.yaml    # Claude Code extensions
├── SKILL.opencode.yaml       # OpenCode extensions (commands)
├── references/               # Knowledge docs (loaded on demand)
└── templates/                # Artifact templates (loaded on demand)
```

Skills compile to a shared intermediate at `dist/skills/`, with shared templates projected into consuming packages and tracker-native Flow/provider sources overlaid from `vnext/content/`. Each target then packages the same common content; see [Build System](build-system.md).

## Journal Self-Logging

User-invocable workflow skills must log their invocation to the project journal as their first action. This creates an audit trail of which skills ran; the current branch and an opaque `harness_session_id` are attached automatically:

```bash
loaf journal log "skill(shape): shaping auth token rotation idea into spec"
loaf journal log "skill(wrap): end-of-conversation checkpoint"
```

The `/wrap` skill reads recent entries to check whether housekeeping or other periodic skills were run.

## Shared Templates

`content/templates/` files are distributed to skills at build time via the `shared-templates` config in `targets.yaml`:

| Template | Distributed To |
|----------|---------------|
| `journal.md` | implement, orchestration, housekeeping, bootstrap |
| `grilling.md` | refactor-deepen |

Skill-specific templates live in `content/skills/{name}/templates/`. Architecture owns its [topic and decision-record templates](../../content/skills/architecture/templates/); Reflect uses Architecture's guidance rather than receiving a shared ADR template. A skill references a distributed template by its projected relative path; see the authored [journal template](../../content/templates/journal.md). Tracker-native Flow templates are separately projected by the common overlay before target packaging.

## Agent Profiles

Each profile is a tool-boundary contract. Functional profiles define what an agent may touch; the background runner supplies the asynchronous system role.

| Profile | Tool Access | Purpose |
|---------|-------------|---------|
| **implementer** | Full write | Code, tests, config, docs — specialty via skills |
| **reviewer** | Read-only | Audits, reviews — mechanical independence |
| **researcher** | Read + web | Research, comparison — structured reports |
| **librarian** | Read + Edit (.agents/) | Journal curation, durable artifacts, wrap, pre-compaction preservation |
| **background-runner** | Read + Edit | Async non-blocking tasks (system) |

Skills load into profiles at spawn time. What an agent *can do* is fixed by profile; what it *knows* comes from skills.

## Categories

| Category | `user-invocable` | Examples |
|----------|:-:|---------|
| Reference/Knowledge | `false` | python-development, database-design, foundations, git-workflow, orchestration |
| Workflow/Process | `true` (default) | implement, research, shape, breakdown, ship, release, housekeeping, wrap |

## Cross-References

- [build-system.md](build-system.md) — how skills get distributed to targets
- [hook-system.md](hook-system.md) — how skills own hooks via `skill:` field in hooks.yaml
