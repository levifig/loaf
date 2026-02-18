# Loaf Development Guidelines

Guidelines for maintaining and extending Loaf - Levi's Opinionated Agentic Framework.

See [README.md](../README.md) for what Loaf is and how to install it.

## Quick Start

```bash
npm install && npm run build
```

## Project Structure

```
src/
├── skills/{name}/SKILL.md      # Domain knowledge + references/ + templates/
├── templates/                  # Shared templates (distributed at build time)
├── agents/{name}.md            # Thin routers (frontmatter: model, skills, tools)
├── hooks/{pre,post}-tool/      # Hook scripts
└── config/
    ├── hooks.yaml              # Hook definitions, plugin-groups
    └── targets.yaml            # Target defaults, sidecars, shared-templates
build/targets/{target}.js       # Target transformers
```

**Output:** `plugins/` (Claude Code), `dist/{target}/` (others)

## Quick Reference

| Component | Location | Key File |
|-----------|----------|----------|
| Skills | `src/skills/{name}/` | `SKILL.md` |
| Agents | `src/agents/{name}.md` | - |
| Hooks | `src/hooks/{pre,post}-tool/` | - |
| Config | `src/config/` | `hooks.yaml`, `targets.yaml` |

## Common Tasks

**Add skill:** Create `src/skills/{name}/SKILL.md`, add to `plugin-groups` in `hooks.yaml`

**Add agent:** Create `src/agents/{name}.md`, add to `plugin-groups` in `hooks.yaml`

**Add hook:** Create script in `src/hooks/{pre,post}-tool/`, register in `hooks.yaml`

**Add template:** Create in `src/skills/{name}/templates/` (skill-specific) or `src/templates/` + register in `shared-templates` in `targets.yaml`

**Add target:** Create `build/targets/{target}.js`, add to `targets.yaml` and `build.js`

## Skill Development

### Naming Conventions

Use domain-focused names in gerund or noun-phrase form:

| Pattern | Examples | Use For |
|---------|----------|---------|
| `{lang}-development` | `python-development`, `typescript-development` | Language skills |
| `{domain}-{activity}` | `database-design`, `infrastructure-management` | Domain skills |
| Single word | `foundations`, `orchestration`, `research` | Workflow/process skills |

**Constraints:**
- Lowercase letters, numbers, hyphens only
- Max 64 characters
- No reserved words: `anthropic`, `claude`
- Directory name must match `name` field

### SKILL.md Structure

Follow the [Agent Skills](https://agentskills.io) open standard:

```yaml
---
name: skill-name
description: >-
  Third-person description starting with action verb. Covers X, Y, Z.
  Use when [context triggers] or when the user asks "[natural language examples]".
---

# Skill Title

Brief intro paragraph.

## Contents
- Section One
- Section Two
- Section Three

## Section One
...
```

#### Frontmatter Fields (Standard)

Only these fields belong in `SKILL.md`:

| Field | Required | Notes |
|-------|----------|-------|
| `name` | Yes | Must match directory name |
| `description` | Yes | Max 1024 chars, third-person, action verbs |
| `license` | No | License name or file reference |
| `compatibility` | No | Environment requirements |
| `metadata` | No | Arbitrary key-value pairs |
| `allowed-tools` | No | Space-delimited tool list (experimental) |

#### Description Best Practices

1. **Start with action verb** (third-person):
   - Good: "Covers...", "Establishes...", "Coordinates..."
   - Bad: "Use for...", "I can help...", "You can use this..."

2. **Include user-intent phrases**:
   ```yaml
   description: >-
     Covers Python 3.12+ development... Use when building APIs,
     or when the user asks "how do I validate data?" or "what's
     the best way to structure a Python project?"
   ```

3. **Be specific** - Claude uses this to choose from 100+ skills

4. **Include negative routing** for disambiguation between confusable skills:
   ```yaml
   description: >-
     Covers Python 3.12+ development... Not for schema design
     decisions (use database-design) or deployment infrastructure
     (use infrastructure-management).
   ```

5. **Add success criteria** for workflow skills (what the skill produces):
   ```yaml
   description: >-
     Conducts project assessment... Produces state assessments,
     research findings with ranked options, or vision change
     proposals. Not for multi-agent coordination (use orchestration).
   ```

### Templates

Artifact format templates (session files, specs, ADRs, task files) live in `templates/` directories. SKILL.md references them with links instead of embedding inline.

**Skill-specific templates:** `src/skills/{name}/templates/` — templates unique to one skill.

**Shared templates:** `src/templates/` — distributed to multiple skills at build time via `shared-templates` config in `targets.yaml`.

```yaml
# targets.yaml
shared-templates:
  session.md: [implement, resume, orchestration, reference-session, review-sessions]
  plan.md: [implement, council-session]
  adr.md: [architecture, reflect]
```

**Reference pattern in SKILL.md:**
```markdown
Create session file following [templates/session.md](templates/session.md).
```

**Templates vs references:**
- `templates/` — structural artifacts (frontmatter schema, section headings, required fields)
- `references/` — knowledge documents (conventions, patterns, decision records)

### Sidecar Files

Claude Code-specific fields go in `SKILL.claude-code.yaml`:

```yaml
# Claude Code extensions
user-invocable: false
agent: backend-dev
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
```

| Field | Purpose |
|-------|---------|
| `user-invocable` | `false` for pure reference skills (hide from `/` menu) |
| `disable-model-invocation` | `true` for manual-only workflows |
| `argument-hint` | Autocomplete hint: `"[topic]"`, `"[file]"` |
| `context` | `fork` to run in subagent |
| `agent` | Subagent type when `context: fork` |
| `model` | Override model for this skill |
| `hooks` | Skill-scoped lifecycle hooks |

### Reference Files

#### Organization

```
skill-name/
├── SKILL.md              # Overview + navigation (< 500 lines)
├── SKILL.claude-code.yaml # Claude-specific overrides
├── references/           # Knowledge docs (loaded on demand)
│   ├── topic-a.md
│   └── topic-b.md
└── templates/            # Artifact format templates (loaded on demand)
    ├── artifact-a.md
    └── artifact-b.md
```

#### Reference Table Pattern

Use "Use When" (action-oriented), not "Coverage" (content-oriented):

```markdown
## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Core | [core.md](references/core.md) | Setting up projects, naming conventions |
| Testing | [testing.md](references/testing.md) | Writing tests, debugging failures |
```

#### Table of Contents

Files over 100 lines must have a TOC after the title:

```markdown
# Reference Title

## Contents
- Section One
- Section Two
- Section Three

Brief intro paragraph.

## Section One
...
```

#### Reference Rules

- **One level deep** - All references link from SKILL.md, not from other references
- **No nested chains** - Avoid `SKILL.md → advanced.md → details.md`
- **Forward slashes only** - `references/guide.md`, never `references\guide.md`

### Skill Categories

| Category | `user-invocable` | Example |
|----------|------------------|---------|
| Reference/Knowledge | `false` | `python-development`, `database-design` |
| Workflow/Process | `true` (default) | `orchestration`, `research` |

Reference skills provide background knowledge Claude loads automatically.
Users shouldn't invoke `/python-development` directly.

## Build System

### Commands

```bash
npm run build          # Build all targets
npm run build:claude-code  # Claude Code only
```

### Targets

| Target | Output | Notes |
|--------|--------|-------|
| claude-code | `plugins/loaf/` | Merges sidecars into output |
| opencode | `dist/opencode/` | Skills, agents, and commands (from skills) |
| cursor | `dist/cursor/` | Skills, agents, and hooks |
| codex | `dist/codex/` | Skills and agents only |
| gemini | `dist/gemini/` | Skills and agents only |

### Before Committing

- [ ] `npm run build` succeeds
- [ ] Frontmatter has required fields
- [ ] New skills registered in `hooks.yaml`
- [ ] Sidecar file for Claude-specific fields
- [ ] Reference files >100 lines have TOC
- [ ] Template links resolve (no broken `templates/` paths)
- [ ] No Windows-style paths

## Configuration

### hooks.yaml

Register new skills/agents in plugin-groups:

```yaml
plugin-groups:
  my-group:
    description: Group description
    agents: [agent-name]
    skills: [skill-name, foundations]
```

### targets.yaml

Configure target-specific behavior and sidecars.

## Anti-Patterns

| Don't | Do Instead |
|-------|------------|
| Put Claude fields in SKILL.md | Use `.claude-code.yaml` sidecar |
| Use "Coverage" in reference tables | Use "Use When" (action-oriented) |
| Skip TOC for long files | Add `## Contents` after title |
| Use backslash paths | Use forward slashes: `references/file.md` |
| Nest references deeply | Link all references from SKILL.md |
| Start descriptions with "Use for..." | Start with action verb: "Covers...", "Establishes..." |
| Make reference skills user-invocable | Set `user-invocable: false` in sidecar |
| Skip negative routing for confusable skills | Add "Not for..." in description |
| Leave success criteria undefined for workflow skills | Add "Produces..." in description |
| Embed artifact templates inline in SKILL.md | Extract to `templates/` and link |
| Restate general knowledge models already have | Document only decisions and constraints |

## Version Management

- Version in `package.json`
- Build injects version into output files
- Bump version before release commits
- Use semantic versioning

## Related Documentation

- [Agent Skills Specification](https://agentskills.io/specification)
- [Claude Code Skills Best Practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
