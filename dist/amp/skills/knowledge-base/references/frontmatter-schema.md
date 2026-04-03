# Frontmatter Schema

Detailed reference for knowledge file YAML frontmatter.

## Contents
- Required Fields
- Recommended Fields
- Optional Fields
- Complete Example
- Common Mistakes

## Required Fields

### topics

- **Type:** list of strings (min 1)
- **Purpose:** Semantic tags for routing and discovery. Agents and QMD use these
  to find relevant knowledge files.

```yaml
topics: [engine-registry, strategy-pattern]
```

Choose tags that are specific enough to be useful for routing but general enough
to remain stable as the codebase evolves. Prefer domain terms over implementation
terms (e.g., `thermal-physics` over `heat-calc-module`).

### last_reviewed

- **Type:** string, ISO 8601 date (YYYY-MM-DD)
- **Purpose:** Records when a human last verified the file's accuracy. Used by
  `loaf kb check` to determine staleness.

```yaml
last_reviewed: 2026-03-14
```

Update this field only after genuinely reviewing the content -- do not bump it
during routine edits that do not involve verification. Use `loaf kb review <file>`
to update it properly.

## Recommended Fields

### covers

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

## Optional Fields

### consumers

- **Type:** list of strings
- **Purpose:** Agent routing hints. Indicates which agents or roles should be
  aware of this knowledge.

```yaml
consumers: [backend, power-systems]
```

Values are freeform but should correspond to agent names or plugin-group names
from `hooks.yaml`.

### depends_on

- **Type:** list of strings (filenames)
- **Purpose:** Cross-references to other knowledge files. Indicates that this
  file's content assumes or builds on another.

```yaml
depends_on: [thermal-physics.md, conductor-limits.md]
```

Use bare filenames (not paths). All knowledge files live in `docs/knowledge/`,
so paths are implicit.

### implementation_status

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

## Complete Example

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

## Common Mistakes

| Mistake | Why It Is Wrong | Fix |
|---------|-----------------|-----|
| Missing `topics` | Required for routing; file is invisible to discovery | Add at least one topic |
| `last_reviewed` in the future | Implies review that has not happened | Use today's date or earlier |
| `covers: ["src/**"]` | Too broad; every commit triggers staleness | Narrow to specific paths |
| `depends_on: ["docs/knowledge/foo.md"]` | Uses full path instead of filename | Use `depends_on: [foo.md]` |
| `implementation_status: done` | Not a valid enum value | Use `stable` |
| Topics that match code identifiers | Brittle; breaks on rename | Use domain-level terms |
