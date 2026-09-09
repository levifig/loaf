# Naming Conventions

Where knowledge files live, how to name them, and how they relate to other
documentation surfaces. Based on the project's naming convention (see
`docs/knowledge/knowledge-management-design.md#naming-conventions`).

## Directory Structure

```
docs/
├── knowledge/          # Domain knowledge (cross-cutting context, rules)
│   ├── build-system.md
│   ├── thermal-physics.md
│   └── knowledge-management-design.md
└── architecture/       # Current architecture and rationale
    ├── authority-boundaries.md
    └── persistence.md
```

- `docs/knowledge/` for domain knowledge files
- Architecture homes and topic naming follow the architecture skill; do not apply the knowledge-file lifecycle or schema to them
- Existing ADR locations and references remain valid until an explicitly authorized migration

## Filename Rules

| Rule | Example | Rationale |
|------|---------|-----------|
| Kebab-case | `build-system.md` | Consistent, URL-safe |
| Descriptive | `thermal-physics.md` | Self-documenting |
| No date prefixes | `build-system.md` | Knowledge is living; dates imply snapshots |
| No abbreviations | `knowledge-management-design.md` | Clarity over brevity |

**Contrast with other artifacts that DO use date prefixes:**
- Sessions: `YYYYMMDD-HHMMSS-session-slug.md`
- Ideas: `YYYYMMDD-HHMMSS-idea-slug.md`

Knowledge files are living documents, not point-in-time snapshots.

## CLI Abbreviation

Per the project's naming convention (see `docs/knowledge/knowledge-management-design.md#naming-conventions`), the CLI uses `kb` for ergonomics:

- `loaf kb check` (not `loaf knowledge check`)
- `loaf kb validate` (not `loaf knowledge validate`)

The full word `knowledge` is for storage paths (directories, QMD collections).
The abbreviation `kb` is for typing (CLI commands).

## QMD Collection Naming

When indexed by QMD:

- `{repo-folder}-knowledge` for knowledge files
- Preserve existing architecture or legacy ADR collection names unless their migration is requested

Example: a repo in `~/Code/gridsight-core-gds/` uses `gridsight-core-gds-knowledge` for its knowledge files. Changing documentation conventions does not itself authorize reconfiguring search collections.
