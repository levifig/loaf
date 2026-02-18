# Loaf

> "Why have just a slice when you can get the whole loaf?"

Loaf is an opinionated agentic framework (*Levi's Opinionated Agentic Framework*) that transforms how you work with AI coding assistants. Instead of ad-hoc prompting, Loaf provides a **complete pipeline from idea to implementation to learning**—with full traceability at every step.

## Philosophy

**Spec-first development**: Ideas are shaped into bounded specs before any code is written. No more scope creep or undefined requirements.

**Multi-agent delegation**: A PM agent orchestrates work, specialized agents implement. The orchestrator never touches code directly—it plans, coordinates, and verifies.

**Full traceability**: Every piece of work flows through: Idea → Spec → Tasks → Code → Learnings. Nothing gets lost.

**Session continuity**: Work survives context loss, compaction, and `/clear`. Pick up exactly where you left off.

## The Command Pipeline

Loaf's commands form a three-phase workflow that mirrors how good software gets built:

```
┌─────────────────────────────────────────────────────────────┐
│                    PHASE 1: SHAPE                           │
│                                                             │
│   /idea  →  /brainstorm  →  /shape  →  Bounded Spec        │
│   (capture)   (explore)      (define)                       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    PHASE 2: BUILD                           │
│                                                             │
│   /breakdown  →  /implement                                 │
│   (decompose)    (single task or multi-task orchestration)  │
│                                                             │
│   Optional: /council-session for complex decisions          │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    PHASE 3: LEARN                           │
│                                                             │
│   /review-sessions  →  /reflect  →  Updated Strategy        │
│   (outcomes)           (learnings)                          │
└─────────────────────────────────────────────────────────────┘
```

### Phase 1: Shape

Transform raw ideas into implementable specs with clear boundaries.

| Command | Purpose |
|---------|---------|
| `/idea` | Quick capture of rough ideas into `.agents/ideas/` |
| `/brainstorm` | Deep exploration of a problem space |
| `/shape` | Rigorous shaping into bounded spec (what's IN and OUT) |
| `/strategy` | Discover and document strategic direction |

### Phase 2: Build

Decompose specs into atomic tasks and execute with specialized agents.

| Command | Purpose |
|---------|---------|
| `/breakdown` | Split spec into agent-sized atomic tasks |
| `/implement` | Start single-task work or multi-task orchestration with dependency tracking |
| `/council-session` | Convene agents for multi-perspective decisions |
| `/resume` | Continue after context loss or new conversation |

### Phase 3: Learn

Integrate outcomes into strategic knowledge.

| Command | Purpose |
|---------|---------|
| `/review-sessions` | Review completed sessions, identify patterns |
| `/reflect` | Integrate learnings into STRATEGY.md |
| `/reference-session` | Reuse patterns from prior sessions |

## Key Concepts

**Shape Up Methodology**: Inspired by Basecamp's Shape Up. Specs define appetite (how much effort), boundaries (what's out), and rabbit holes (known risks). Work is shaped, not just specified.

**Strict Delegation**: The PM agent plans and coordinates but never writes code. All implementation is delegated to specialized agents: `backend-dev`, `frontend-dev`, `dba`, `qa`, `devops`, `design`.

**Sessions as State Machines**: Every `/implement` creates a session file in `.agents/sessions/`. Sessions track: spec → tasks → agent assignments → outcomes. They survive `/clear`, compaction, and context switches.

**Hooks as Quality Gates**: Pre-tool hooks can block dangerous actions (secrets in commits). Post-tool hooks run linters, type checkers, security scans. Language-aware and automatic.

## Agents

**Orchestrator:**

| Agent | Use For |
|-------|---------|
| `pm` | Orchestrating work, managing sessions, delegating to specialists |

**Implementation Agents** (write code):

| Agent | Use For |
|-------|---------|
| `backend-dev` | Python, Ruby/Rails, Go, or TypeScript services |
| `frontend-dev` | React, Next.js, UI components |
| `devops` | Docker, Kubernetes, CI/CD |

**Advisory Agents** (expertise in councils, delegate implementation):

| Agent | Use For |
|-------|---------|
| `dba` | Schema design, migrations, queries |
| `qa` | Testing, code review, security |
| `design` | UI/UX, accessibility |
| `power-systems` | Grid design, power electronics |

## Skills

Domain knowledge that agents draw from:

| Skill | Coverage |
|-------|----------|
| `orchestration` | Sessions, councils, Linear integration, Shape Up |
| `foundations` | Code style, docs, security, commits |
| `python-development` | FastAPI, Pydantic, pytest, async |
| `typescript-development` | TypeScript, React, Next.js, type safety |
| `ruby-development` | Rails 8, Hotwire, Minitest, DHH philosophy |
| `go-development` | Go services, concurrency, testing |
| `database-design` | Schema design, migrations, optimization |
| `infrastructure-management` | Docker, K8s, Terraform |
| `interface-design` | Accessibility, design systems |
| `power-systems-modeling` | Grid design, power electronics |

## Multi-Target Support

Loaf is built once and deployed to multiple AI coding tools:

| Target | Features | Status |
|--------|----------|--------|
| Claude Code | Full (agents, skills, hooks, MCP, LSP) | Primary |
| OpenCode | Full (agents, skills, commands, hooks) | Full support |
| Cursor | Agents, skills, hooks | Full support |
| Codex | Skills only | Partial |
| Gemini | Skills only | Partial |

## Integrations

*Claude Code only.*

**MCP Servers:**

- **Serena** - Code intelligence and semantic search
- **Sequential Thinking** - Structured reasoning
- **Linear** - Issue tracking and project management

**LSP Servers:**

- **gopls**, **pyright**, **typescript-language-server**, **solargraph**

## Installation

### Claude Code

```bash
/plugin marketplace add levifig/loaf
```

Updates happen automatically via plugin marketplace.

### OpenCode, Cursor, Codex, Gemini

```bash
curl -fsSL https://raw.githubusercontent.com/levifig/loaf/main/install.sh | bash
```

The installer detects installed tools, lets you select targets, and downloads pre-built distributions.

**Update:**

```bash
# Interactive
curl -fsSL https://raw.githubusercontent.com/levifig/loaf/main/install.sh | bash

# Unattended (CI, scripts)
curl -fsSL https://raw.githubusercontent.com/levifig/loaf/main/install.sh | bash -s -- --upgrade
```

**Install locations:**

| Target | Location |
|--------|----------|
| OpenCode | `~/.config/opencode/` or `~/.opencode/` |
| Cursor | `~/.cursor/` |
| Codex | `$CODEX_HOME/skills/` or `~/.codex/skills/` |
| Gemini | `~/.gemini/skills/` |

## Plugin Scoping

*Claude Code only.*

Commands and agents are scoped to avoid conflicts:

```bash
/loaf:implement          # Start implementation session
/loaf:council-session    # Run a council deliberation
@loaf:backend-dev        # Reference an agent
```

## Development

```bash
git clone https://github.com/levifig/loaf.git
cd loaf
npm install
npm run build
```

See [AGENTS.md](.agents/AGENTS.md) for development guidelines.

**Testing locally:**

- Claude Code: `/plugin marketplace add /path/to/loaf`
- Others: `./install.sh` (detects dev mode, builds first)

## License

MIT
