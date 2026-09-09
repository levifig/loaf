# Knowledge Base

Loaf's domain knowledge — what agents need to understand about this project.

| File | Topics | Covers |
|------|--------|--------|
| [build-system.md](build-system.md) | build, targets, distribution | `internal/cli/build*.go`, `config/targets.yaml`, `config/hooks.yaml` |
| [glossary.md](glossary.md) | glossary | — |
| [hook-system.md](hook-system.md) | hooks, lifecycle, validation | `config/hooks.yaml`, `internal/cli/check.go`, `content/hooks/**/*` |
| [knowledge-management-design.md](knowledge-management-design.md) | knowledge, staleness, qmd | `internal/cli/kb.go`, `docs/knowledge/*.md` |
| [loaf-flow.md](loaf-flow.md) | workflow, tracker-native Flow, rideable increments | tracker-native Flow skill sources and provider-neutral handoffs |
| [skill-architecture.md](skill-architecture.md) | skills, agent-skills-standard, sidecars | `content/skills/**/*.md`, `content/skills/**/*.yaml`, `config/hooks.yaml` |
| [task-system.md](task-system.md) | tracker work, local-record compatibility, journal | shipped local record commands and one-time migration guidance |
| [work-model.md](work-model.md) | shared-work authority, Flow, implementation, release | tracker-native Flow and project-management skills |
| [cloud-attach-walkthrough.md](cloud-attach-walkthrough.md) | shipped cloud attach compatibility, CI secrets | shipped cloud bootstrap and attach surfaces |

Start with the [architecture overview](../ARCHITECTURE.md) for current topics and their selected decision records.
