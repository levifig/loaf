# Knowledge Files

Conventions for knowledge files: structured domain knowledge in `docs/knowledge/` that persists across conversations. Architecture topics are part of the same knowledge base but follow the architecture topic convention in the architecture skill, not this schema.

## Contents
- Where Knowledge Belongs
- When to Create a Knowledge File
- Naming and Location
- The covers: Field
- Review Workflow
- Frontmatter Schema

## Where Knowledge Belongs

| Surface | Contains | Decision Test |
|---------|----------|---------------|
| **Code** (docstrings, types) | What the code does | Is it self-documenting? |
| **Knowledge files** | Domain rules and cross-cutting context | Requires context beyond the code? |
| **Architecture topics** | Current system model and why it has this shape | Does it describe architecture or its rationale? |
| **AGENTS.md** | Agent instructions, conventions | Is it about how agents should behave? |
| **MEMORY.md** | User preferences, session pointers | Is it personal or ephemeral? |

AGENTS.md may reference knowledge files but should never duplicate their content.

## When to Create a Knowledge File

Create a knowledge file when information:

1. **Requires context beyond the code** -- domain rules, business logic constraints, cross-cutting patterns that cannot be expressed in types or docstrings
2. **Spans multiple files or modules** -- conventions or contracts that apply across the codebase, not localized to one function
3. **Would be lost between conversations** -- insights that an agent needs repeatedly but cannot derive from code alone
4. **Supplies durable domain context** -- rules and operational knowledge that inform decisions across conversations

Do NOT create knowledge files for:
- Self-documenting code (types, docstrings, comments are sufficient)
- Architectural models or their rationale (use an architecture topic)
- Shared work scope, implementation sequencing, completion criteria, or status (use the canonical native tracker); strategic bets belong in strategy
- Agent behavior instructions (put those in AGENTS.md)
- User preferences or session pointers (those belong in MEMORY.md)

## Naming and Location

- Store knowledge files in `docs/knowledge/`, never elsewhere. Architecture homes and topic naming follow the architecture skill; do not apply the knowledge-file lifecycle or schema to them.
- Existing ADR locations and references remain valid until an explicitly authorized migration.

| Rule | Example | Rationale |
|------|---------|-----------|
| Kebab-case | `build-system.md` | Consistent, URL-safe |
| Descriptive | `thermal-physics.md` | Self-documenting |
| No date prefixes | `build-system.md` | Knowledge is living; dates imply snapshots |
| No abbreviations | `knowledge-management-design.md` | Clarity over brevity |

Point-in-time authored reports may use date prefixes, such as `.agents/reports/YYYYMMDDHHMMSS-module-depth.md`. The journal has no session entity, and current idea records are retrieved through `loaf idea`; preserve historical timestamped idea files as compatibility evidence rather than a new storage convention.

The CLI abbreviates to `kb` for typing (`loaf kb check`, `loaf kb validate`); the full word `knowledge` names storage paths such as directories and search collections. When the project indexes knowledge for search, a collection named `{repo-folder}-knowledge` is the convention: a repository in `~/Code/acme-app/` uses `acme-app-knowledge`. Preserve existing architecture or legacy ADR collection names unless their migration is requested; changing documentation conventions does not itself authorize reconfiguring search collections.

## The covers: Field

The `covers:` field links knowledge to code paths via glob patterns, enabling automatic staleness detection.

1. Author declares `covers:` globs in knowledge file frontmatter
2. Globs resolve from the git repository root
3. `loaf kb check` queries `git log --since={last_reviewed}` for matching files
4. If commits exist since last review, the knowledge file is flagged as potentially stale

```yaml
covers:
  - "src/pipeline/registry.py"          # Specific file
  - "src/models/engine_*.py"            # Wildcard within directory
  - "src/thermal/**/*.py"               # Recursive directory
```

Avoid broad globs such as `src/**`, `**/*.py`, or `*.md`: they turn ordinary churn into constant false-positive staleness alerts. Cover the code paths whose changes would actually invalidate the knowledge.

Omit `covers:` when the knowledge is purely conceptual, describes external domain rules not tied to implementation, or the mapping is not yet clear. Such files remain valid but cannot use automated staleness detection.

## Review Workflow

| Command | Does |
|---------|------|
| `loaf kb check` | Lists knowledge files whose covered code paths changed since `last_reviewed`. Excludes architecture documents. |
| `loaf kb validate` | Checks knowledge files for required metadata and dates, with warnings for unmatched coverage globs and missing references. Architecture topics, the overview, and retained decisions get only a nonempty-body and closed-frontmatter check, not architectural correctness or a universal ADR schema. Read failures appear in the results and fail the command; absent default document directories are allowed. |
| `loaf kb review <file>` | After substantively verifying a knowledge file, sets `last_reviewed` to today. Architecture documents are excluded, including aliases that resolve to an architecture path. |
| `loaf kb status` | Summarizes discovered knowledge and architecture documents. Coverage counts, staleness, and review age apply only to knowledge files. |

Do not bump review dates merely to silence stale warnings; a routine edit alone does not justify a new review date.

The `kb-staleness-nudge` hook runs `loaf check --hook kb-staleness-nudge` after edits, without Bash or Python. It accepts hook JSON on stdin or in `TOOL_INPUT`, stays quiet for irrelevant input and unavailable Git, and never marks a file reviewed. Disposable cache markers suppress repeated nudges per document, project, and harness conversation ID. When the harness supplies no conversation ID, the fallback is per project and branch per UTC day; neither marker is a session record or lifecycle.

## Frontmatter Schema

### Required Fields

#### topics

- **Type:** list of strings (min 1)
- **Purpose:** Semantic tags for routing and discovery. Agents and QMD use these
  to find relevant knowledge files.

```yaml
topics: [engine-registry, strategy-pattern]
```

Choose tags that are specific enough to be useful for routing but general enough
to remain stable as the codebase evolves. Prefer domain terms over implementation
terms (e.g., `thermal-physics` over `heat-calc-module`).

#### last_reviewed

- **Type:** string, ISO 8601 date (YYYY-MM-DD)
- **Purpose:** Records when a human last verified the file's accuracy. Used by
  `loaf kb check` to determine staleness.

```yaml
last_reviewed: 2026-03-14
```

Update this field only after genuinely reviewing the content -- do not bump it
during routine edits that do not involve verification. Use `loaf kb review <file>`
to update it properly.

### Recommended Fields

#### covers

- **Type:** list of glob strings
- **Purpose:** Links knowledge to code paths. Globs resolve from the git
  repository root. Enables automatic staleness detection.

```yaml
covers:
  - "src/pipeline/registry.py"
  - "src/models/engine_*.py"
```

**Glob resolution rules:**
- Paths are relative to the git root (not the knowledge file location)
- Standard glob syntax: `*` matches within a directory, `**` matches recursively
- Patterns that match no files are silently ignored (but may indicate drift)

**Precision guidelines:**
- Cover the specific files whose changes would invalidate this knowledge
- Avoid `src/**` or `**/*.py` -- too broad, causes alert fatigue
- When unsure, start narrow and widen as you learn which paths matter

### Optional Fields

#### consumers

- **Type:** list of strings
- **Purpose:** Agent routing hints. Indicates which agents or roles should be
  aware of this knowledge.

```yaml
consumers: [backend, power-systems]
```

Values are freeform but should correspond to agent names or plugin-group names
from `hooks.yaml`.

#### depends_on

- **Type:** list of strings (filenames)
- **Purpose:** Cross-references to other knowledge files. Indicates that this
  file's content assumes or builds on another.

```yaml
depends_on: [thermal-physics.md, conductor-limits.md]
```

Use bare filenames (not paths). All knowledge files live in `docs/knowledge/`,
so paths are implicit.

#### implementation_status

- **Type:** enum string
- **Valid values:** `in-progress` | `stable` | `deprecated`
- **Default:** omit if stable (absence implies stable)
- **Purpose:** Signals whether the knowledge reflects current, evolving, or
  obsolete state.

```yaml
implementation_status: in-progress
```

| Value | Meaning |
|-------|---------|
| `in-progress` | Knowledge is being actively developed or the code it covers is in flux |
| `stable` | Knowledge is current and verified |
| `deprecated` | Knowledge is outdated; the domain or code has moved on |

### Complete Example

```yaml
---
topics: [engine-registry, strategy-pattern, pipeline]
last_reviewed: 2026-03-14
covers:
  - "src/pipeline/registry.py"
  - "src/models/engine_*.py"
  - "src/pipeline/strategies/**/*.py"
consumers: [backend, power-systems]
depends_on: [thermal-physics.md]
implementation_status: stable
---

# Engine Registry

Domain rules for the engine registration and strategy selection system.

## Key Rules

- All engines must register via `EngineRegistry.register()`
- Strategy selection is deterministic given the same input parameters
- ...
```

### Common Mistakes

| Mistake | Why It Is Wrong | Fix |
|---------|-----------------|-----|
| Missing `topics` | Required for routing; file is invisible to discovery | Add at least one topic |
| `last_reviewed` in the future | Implies review that has not happened | Use today's date or earlier |
| `covers: ["src/**"]` | Too broad; every commit triggers staleness | Narrow to specific paths |
| `depends_on: ["docs/knowledge/foo.md"]` | Uses full path instead of filename | Use `depends_on: [foo.md]` |
| `implementation_status: done` | Not a valid enum value | Use `stable` |
| Topics that match code identifiers | Brittle; breaks on rename | Use domain-level terms |
