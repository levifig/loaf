# Loaf Architecture

## Current Architecture (v2.0)

```
cli/                            # CLI tool (TypeScript, bundled by tsup)
├── index.ts                    # Commander.js entry point
├── commands/
│   ├── build.ts                # loaf build
│   └── install.ts              # loaf install
└── lib/
    ├── build/
    │   ├── types.ts            # Shared types (BuildContext, HooksConfig, etc.)
    │   ├── targets/            # Target transformers (*.ts)
    │   └── lib/                # Utilities (version, sidecar, substitutions, etc.)
    ├── detect/tools.ts         # AI tool detection
    └── install/installer.ts    # Target-specific installers

content/                        # Distributable content (separated from tooling)
├── skills/{name}/SKILL.md      # Domain knowledge (Agent Skills standard)
├── agents/{name}.md            # Agent definitions (frontmatter: model, skills, tools)
├── hooks/{pre,post}-tool/      # Lifecycle hooks (shell/Python scripts)
└── templates/                  # Shared templates (distributed at build time)

config/
├── hooks.yaml                  # Hook definitions + plugin-groups
└── targets.yaml                # Target defaults + shared-templates mapping

Output:
├── dist-cli/                   # Bundled CLI (single JS file via tsup)
├── plugins/loaf/               # Claude Code plugin (with hooks, MCP servers)
└── dist/{target}/              # Other targets (cursor, opencode, codex, gemini)
```

### Build Flow

```
cli/index.ts → tsup → dist-cli/index.js (bundled CLI)
content/ + config/ → loaf build → dist/ + plugins/
```

Each target transformer reads content (skills/agents/hooks) and config, then produces target-specific output. Skills get sidecar files merged. Hooks get registered in plugin manifests. Shared templates get distributed to specified skills.

All TypeScript, bundled into a single file by tsup. No dynamic imports.

### Targets

| Target | Output | Agents | Skills | Hooks | MCP |
|--------|--------|:------:|:------:|:-----:|:---:|
| claude-code | plugins/loaf/ | Yes | Yes | Yes | Yes |
| cursor | dist/cursor/ | Yes | Yes | Yes | No |
| opencode | dist/opencode/ | Yes | Yes | No | No |
| codex | dist/codex/ | No | Yes | No | No |
| gemini | dist/gemini/ | No | Yes | No | No |

## Planned Architecture

### Loaf CLI

A unified `loaf` command that wraps build operations, knowledge management, task management, and multi-harness distribution. Language TBD (Node.js likely, same runtime as build system).

### Knowledge Management Layer

```
docs/knowledge/          # Knowledge files with frontmatter (covers:, topics:, etc.)
docs/decisions/          # ADRs (immutable decision records)
.agents/loaf.json        # Project config (local KB dirs, imports, settings)

QMD (retrieval backend):
  Collections → indexed directories
  Context → semantic metadata
  Search → BM25 + optional vector + reranking
  MCP server → agent access

Loaf (lifecycle layer):
  Skill → agent guidance (when to create, update, search)
  Hooks → staleness detection, growth prompts
  CLI → loaf kb check, validate, status, review
```

### Task System

```
.agents/specs/SPEC-XXX.md       # Shape Up specs (appetite, scope, test conditions)
.agents/tasks/TASK-XXX-slug.md  # Task details (criteria, context, verification)
.agents/tasks/TASKS.json        # Programmatic index (CLI/TUI readable)
```

### Config

```
.agents/loaf.json               # Project-level (knowledge, implementation settings)
~/.local/state/loaf/            # User-level (registered KBs, default settings)
~/.config/loaf/                 # User preferences
```
