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
last_reviewed: '2026-09-29'
---

# Skill Architecture

Skills are the primary knowledge delivery mechanism. [Skill Portability](../architecture/skill-portability.md) owns the package-format choice, shared-authoring model, and rationale; this guide covers authoring layout and conventions. Read it before adding or editing a skill, a skill reference or template, or an agent profile.

## Key Rules

- **SKILL.md contains standard fields only.** See [Frontmatter](#frontmatter). No tool-specific fields.
- **Sidecar files carry extensions.** `SKILL.claude-code.yaml` carries authorized Claude Code fields such as `user-invocable`, `model`, and `context`; `SKILL.opencode.yaml` carries authorized OpenCode metadata. The target's exact field ownership is enforced by the [build checker](../../internal/cli/build_skill_invariance.go).
- **Descriptions drive routing.** The model uses the description to choose from 100+ skills. Must start with action verb (third-person), include user-intent phrases, negative routing for confusable skills. See [Descriptions](#descriptions).
- **References are one level deep.** All references link from SKILL.md, never from other references. No nested chains.
- **Templates are structural artifacts.** `templates/` hold format templates (frontmatter schema, section headings). `references/` hold knowledge docs (conventions, patterns).
- **Author harness differences explicitly.** Use neutral workflow names in common prose and labeled product sections or a harness table for distinct invocation forms. Builders do not substitute commands in skill prose. See [Harness-Neutral Authoring](#harness-neutral-authoring).
- **Document decisions and constraints, not general knowledge.** Do not restate what models already know.

## Skill Structure

```
content/skills/{name}/
├── SKILL.md                  # Standard frontmatter + overview + navigation (< 500 lines)
├── SKILL.claude-code.yaml    # Claude Code extensions
├── SKILL.opencode.yaml       # OpenCode extensions (commands)
├── references/               # Knowledge docs (loaded on demand)
└── templates/                # Artifact templates (loaded on demand)
```

Skills are auto-discovered from `content/skills/` at build time; do not register a skill itself in `config/hooks.yaml`. Only hook instances are registered there, and a skill that ships no hooks needs no entry. Skills compile to a shared intermediate at `dist/skills/`, with shared templates projected into consuming packages and tracker-native Flow/provider sources overlaid from `vnext/content/`. Each target then packages the same common content; see [Build System](build-system.md).

### SKILL.md Layout

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
- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules
...
```

Standard section order:

1. **Critical Rules** — Must-follow constraints, guardrails, delegation patterns
2. **Verification** — How to confirm success, test conditions, validation steps
3. **Quick Reference** — Tables, decision trees, command cheatsheets
4. **Topics** — Detailed references linked from SKILL.md

## Naming

Use domain-focused names in gerund or noun-phrase form. Skills that DO things get verbs; skills that ARE things get nouns, so "use me to act" and "reference me to know" are distinguishable at a glance.

| Pattern | Examples | Use For |
|---------|----------|---------|
| Verb (workflow) | `implement`, `shape`, `research` | Workflow/process skills |
| `{lang}-development` | `python-development`, `typescript-development` | Language skills |
| `{domain}-{activity}` | `database-design`, `infrastructure-management` | Domain skills |
| Single word | `foundations`, `orchestration` | Knowledge/process skills |

Constraints: lowercase letters, numbers, and hyphens only; max 64 characters; no reserved words (`anthropic`, `claude`); the directory name must match the `name` field.

## Frontmatter

Only these Agent Skills standard fields belong in `SKILL.md`:

| Field | Required | Notes |
|-------|----------|-------|
| `name` | Yes | Must match directory name |
| `description` | Yes | Max 1024 chars, third-person, action verbs |
| `license` | No | License name or file reference |
| `compatibility` | No | Environment requirements |
| `metadata` | No | Arbitrary key-value pairs |

The standard also defines an experimental `allowed-tools` field, but Loaf's build treats it as a Claude Code sidecar-owned key; put it in `SKILL.claude-code.yaml` (see [Sidecars](#sidecars)).

## Descriptions

Use a two-tier structure for Claude's 250-character truncation versus the full description:

1. **First sentence (≤250 chars):** Action verb, what it covers, key trigger phrases.
2. **Rest:** User-intent examples, negative routing, success criteria.

```yaml
description: >-
  Covers Python 3.12+ with FastAPI, Pydantic, async patterns, pytest, SQLAlchemy.
  Use when building APIs, or when the user asks "how do I validate data?"
  or "what's the best way to structure a Python project?"
  Not for schema design decisions (use database-design).
```

- **Start with a third-person action verb:** "Covers...", "Establishes...", "Coordinates...". Never "Use for...", "I can help...", or "You can use this...".
- **Include user-intent phrases** (`or when the user asks "..."`).
- **Be specific:** the model uses the description to choose among 100+ skills.
- **Add negative routing** for confusable skills: "Not for schema design decisions (use database-design) or deployment infrastructure (use infrastructure-management)."
- **Add success criteria** for workflow skills: "Produces state assessments, research findings with ranked options, or vision change proposals."

## Sidecars

Claude Code-specific fields go in `SKILL.claude-code.yaml`:

```yaml
# Claude Code extensions
user-invocable: false
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
```

| Field | Purpose |
|-------|---------|
| `user-invocable` | `false` for pure reference skills (hide from `/` menu) |
| `disable-model-invocation` | `true` for manual-only workflows |
| `argument-hint` | Autocomplete hint: `"[topic]"`, `"[file]"` |
| `allowed-tools` | Tool allowlist for the skill |
| `context` | `fork` to run in subagent |
| `model` | Override model for this skill |

The authoritative per-target key sets are `nativeBuildSidecarOwnedFrontmatterKeysByTarget` in the [build checker](../../internal/cli/build_skill_invariance.go); a key in the wrong sidecar, or a built value that differs from its sidecar, fails the build. Claude Code's skill-scoped `hooks` frontmatter is not currently an authorized key; register hooks in `config/hooks.yaml` (see [Hook System](hook-system.md)).

## References

```
skill-name/
├── SKILL.md              # Overview + navigation (< 500 lines)
├── references/           # Knowledge docs (loaded on demand)
│   ├── topic-a.md
│   └── topic-b.md
└── templates/            # Artifact format templates (loaded on demand)
    └── artifact-a.md
```

- **One level deep.** All references link from SKILL.md, not from other references. Avoid chains such as `SKILL.md → advanced.md → details.md`.
- **Forward slashes only:** `references/guide.md`, never `references\guide.md`.
- **Files over 100 lines have a TOC** (`## Contents` with the section list) directly after the title.
- **Reference tables say "Use When"** (action-oriented), not "Coverage" (content-oriented):

```markdown
## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Core | [core.md](references/core.md) | Setting up projects, naming conventions |
| Testing | [testing.md](references/testing.md) | Writing tests, debugging failures |
```

## Templates

Artifact format templates (architecture topics, journal entries) live in `templates/` directories. SKILL.md links to them instead of embedding them inline:

```markdown
Use [templates/journal.md](templates/journal.md) for the project journal render and entry-format reference.
```

Skill-specific templates live in `content/skills/{name}/templates/`. Architecture owns its [topic and decision-record templates](../../content/skills/architecture/templates/), plus the legacy template for repositories that retain Loaf's historical ADR convention; Reflect uses Architecture's guidance rather than receiving a shared ADR template.

Shared templates live in `content/templates/` and are distributed to skills at build time via `shared-templates` in `config/targets.yaml`:

| Template | Distributed To |
|----------|---------------|
| `journal.md` | implement, orchestration, housekeeping, bootstrap |
| `grilling.md` | refactor-deepen |

To add a shared template, create it in `content/templates/` and register its consumers under `shared-templates`. A skill references a distributed template by its projected relative path; see the authored [journal template](../../content/templates/journal.md). Tracker-native Flow templates are separately projected by the common overlay before target packaging.

## Harness-Neutral Authoring

Skill bodies are harness-neutral by authoring, not by build-time substitution. Every target opens the same bytes: there is no target identity at read time and no per-harness rewrite on the skill-copy path. [Skill Portability](../architecture/skill-portability.md) owns the rationale.

**Default:** describe the behaviour and let the model select its own tools. Prefer "ask one question at a time, with a recommendation, using your harness's structured question tool if it has one" over naming `AskUserQuestion` as the only interview surface. Prefer naming a workflow in prose (`the implement workflow`) over hard-coding a slash form that is wrong on some harnesses.

**Labeled harness sections** carry facts that are genuinely product-specific — paste-ready allowlist tokens, product-unique spawn APIs, product-unique paths, or invocation forms that differ by channel. They are not a licence to relabel content that should simply be neutral prose.

Primary shape — topic parent, then one `### <Product Name>` subsection per product that has a known distinct fact:

```markdown
## Spawning Background Agents

Background-agent APIs differ by product. Read only the labeled section for the harness you are running.

### Claude Code

Use the Task tool with run_in_background: true and subagent_type background-runner.
(Paste-ready call site lives in a fenced block under this heading.)

### Cursor

Background agents are configured via the is_background: true YAML property.
```

See `content/skills/orchestration/references/background-agents.md` and `content/skills/foundations/references/permissions.md` for full worked examples with multi-line fences.

Rules for the primary shape:

1. Product headings use the product's display name (`Claude Code`, `Codex`, `Cursor`, `OpenCode`, `Amp`), not build target slugs.
2. Include only harnesses with a known distinct fact — omit speculative "same as X" sections.
3. Fenced configuration is paste-ready configuration for that product alone; never merge two products' tokens into one fence (the historical `TodoWrite, TodoRead` → `update_plan, update_plan` corruption).
4. Lead with a one-line instruction that the reader takes only their product's section.

Compact shape — a table with a Harness column — for one-line facts (slash invocation forms, a single allowlist token):

```markdown
| Harness | Invoke skill |
|---------|----------------|
| Claude Code (plugin) | `/loaf:name` |
| OpenCode, Cursor, Codex, Amp | `/name` |
```

Use the compact table when every row is a single token or short phrase; use `###` subsections when a variant needs multi-paragraph explanation or a multi-line fence. Do not use inline "if your harness is X" clauses for anything longer than a parenthetical — they bury the common case and do not scale past two products.

Worked examples of genuine product-specific facts (keep as labeled content; do not neutralize away):

- Fenced allowlist entries whose literal tokens differ by product (`foundations/references/permissions.md`)
- Distinct spawn/configuration mechanisms (`orchestration/references/background-agents.md`)
- The Claude Code `@AGENTS.md` import in a root `CLAUDE.md` for sessions that cannot read `AGENTS.md` directly, and the rule that a real `.claude/CLAUDE.md` suppresses direct `AGENTS.md` reading (the paths are the fact; root `AGENTS.md` stays the canonical project-instructions name in prose)
- Review policy that must inspect both `AGENTS.md` and `CLAUDE.md` when both exist
- Slash-command invocation forms that differ by channel (`/loaf:name` on Claude Code's plugin path vs bare `/name` elsewhere)

## Journal Self-Logging

User-invocable workflow skills must log their invocation to the project journal as their first action, with context: arguments, intent, or what triggered the invocation. This creates an audit trail of which skills ran; the current branch and an opaque `harness_session_id` are attached automatically, and there is no start step or active session to find:

```bash
loaf journal log "skill(shape): shaping auth token rotation in the selected native issue"
loaf journal log "skill(housekeeping): routine cleanup, no specific trigger"
loaf journal log "skill(wrap): end-of-conversation checkpoint"
```

The `/wrap` skill reads recent entries to check whether housekeeping or other periodic skills were run. The [journal template](../../content/templates/journal.md) owns the entry types and format rules.

## Agent Profiles

Profiles in `content/agents/{name}.md` describe intended responsibilities and tool boundaries. Actual access and enforcement come from the host's available tools and permission controls; do not treat a profile as mechanically restricted unless the host enforces that boundary.

| Profile | Intended tool boundary | Purpose |
|---------|------------------------|---------|
| **implementer** | Full write | Code, tests, config, docs — specialty via skills |
| **reviewer** | Read-only | Audits, reviews — intended independence from implementation |
| **researcher** | Read + web | Research, comparison — structured reports |
| **librarian** | Read + Edit (.agents/) | Journal curation, durable artifacts, wrap checkpoints, pre-compaction preservation |
| **background-runner** | Read + Edit | Async non-blocking tasks (system) |

Skills load into profiles at spawn time. What an agent is intended to touch comes from its profile; what it *knows* comes from skills.

## Categories

| Category | `user-invocable` | Examples |
|----------|:-:|---------|
| Reference/Knowledge | `false` | python-development, database-design, foundations, git-workflow, orchestration |
| Workflow/Process | `true` (default) | implement, research, shape, breakdown, ship, release, housekeeping, wrap |

Reference skills provide background knowledge the model loads automatically; set `user-invocable: false` in the sidecar so users are not offered `/python-development`.

## Authoring Checklist

Before committing skill changes:

- [ ] Frontmatter has the required fields, and only standard fields
- [ ] New skills live under `content/skills/` (auto-discovered at build); only hook instances are registered in `hooks.yaml`
- [ ] Claude-specific fields are in the sidecar
- [ ] Reference files over 100 lines have a TOC
- [ ] Template links resolve (no broken `templates/` paths)
- [ ] No Windows-style paths

## Anti-Patterns

| Don't | Do Instead |
|-------|------------|
| Put Claude fields in SKILL.md | Use the `.claude-code.yaml` sidecar |
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
| Name a harness-specific tool as the only option | Describe the behaviour; let the model pick its tool |
| Merge product tokens into one fence or rely on build-time rewrite | Use a labeled harness section (`### Product`) or compact harness table |
| Reintroduce per-target skill body rendering | Keep one authored body; labeled sections carry product-specific facts |

## Cross-References

- [build-system.md](build-system.md) — how skills get distributed to targets
- [hook-system.md](hook-system.md) — how skills own hooks via `skill:` field in hooks.yaml
- [Agent Skills Specification](https://agentskills.io/specification)
- [Claude Code Skills Best Practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
