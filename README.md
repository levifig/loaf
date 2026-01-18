# Universal Agent Skills

Universal skills for AI coding assistants. One source, multiple targets: Claude Code, OpenCode, Cursor, Copilot.

## Quick Start

```bash
npm install
npm run build
```

## Installation

### Claude Code

```bash
/plugin marketplace add levifig/agent-skills
/plugin install orchestration@levifig   # PM coordination
/plugin install foundations@levifig     # Quality gates
/plugin install python@levifig          # Python projects
```

### Other Tools

Copy from `dist/`:
- **OpenCode**: `dist/opencode/` → your config directory
- **Cursor**: `dist/cursor/.cursor/` → project root
- **Copilot**: `dist/copilot/.github/` → repository

## Agents

| Agent | Use For |
|-------|---------|
| `pm` | Orchestrating work, managing sessions, delegating |
| `backend-dev` | Python, Rails, or TypeScript services |
| `frontend-dev` | React, Next.js, UI components |
| `dba` | Schema design, migrations, queries |
| `devops` | Docker, Kubernetes, CI/CD |
| `qa` | Testing, code review, security |
| `design` | UI/UX, accessibility |

## Skills

| Skill | Coverage |
|-------|----------|
| `orchestration` | Sessions, councils, Linear integration |
| `foundations` | Code style, docs, security, commits |
| `python` | FastAPI, Pydantic, pytest, async |
| `typescript` | React, Next.js, type safety |
| `rails` | Rails 8, Hotwire, Minitest |
| `infrastructure` | Docker, K8s, Terraform |
| `design` | Accessibility, design systems |

## Structure

```
skills/          # Domain knowledge
agents/          # Thin routing agents (7 total)
commands/        # Portable commands
hooks/           # Pre/post tool hooks
config/          # Build configuration
build/           # Build system
dist/            # Generated (gitignored)
```

## Adding Content

**New skill:**
1. Create `skills/{name}/SKILL.md`
2. Add references in `skills/{name}/reference/`
3. Register in `config/hooks.yaml`

**New agent:**
1. Create `agents/{name}.md` with frontmatter
2. Add to plugin group in `config/hooks.yaml`

## License

MIT
