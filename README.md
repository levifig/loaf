# Universal Agent Skills

Universal skills for AI coding assistants. One source, multiple targets: Claude Code, OpenCode, Cursor, Copilot.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/levifig/agent-skills/main/install.sh | bash
```

The installer will:
1. Detect which AI tools you have installed
2. Let you select which targets to install
3. Build and install to each selected target

### Update

```bash
~/.local/share/agent-skills/install.sh --update
```

### Manual Installation

If you prefer manual installation:

**Claude Code:**
```bash
/plugin marketplace add ~/.local/share/agent-skills/dist/claude-code
/plugin install orchestration@levifig   # PM coordination
/plugin install foundations@levifig     # Quality gates
/plugin install python@levifig          # Python projects
```

**OpenCode:** Copy `dist/opencode/` contents to `~/.config/opencode/`

**Cursor:** Copy `dist/cursor/.cursor/rules/` to your project

**Copilot:** Copy `dist/copilot/.github/` to your repository

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

## Development

For contributors working on the skills themselves:

```bash
git clone https://github.com/levifig/agent-skills.git
cd agent-skills
npm install
npm run build        # Build all targets
npm run build:claude-code  # Build specific target
```

See [AGENTS.md](AGENTS.md) for the full maintenance guide.

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

## License

MIT
