---
topics:
  - knowledge
  - staleness
  - qmd
  - covers
covers:
  - internal/cli/kb*.go
last_reviewed: '2026-11-06'
---

# Knowledge Base Operations

## Contents

- Naming Conventions
- Knowledge Metadata
- Discover and Validate
- Check and Review
- Advisory Edit Nudges
- Optional QMD Integration
- Related Guidance

Use `loaf kb` to discover maintained documents, validate knowledge metadata, and find coverage that may need review. The [Knowledge Base skill](../../content/skills/knowledge-base/SKILL.md) owns the authoring workflow. [Authority Boundaries](../architecture/authority-boundaries.md#knowledge-and-tool-boundaries) explains knowledge ownership and optional tool policy.

## Naming Conventions

Maintained domain guidance and operating references live in `docs/knowledge/`, with descriptive kebab-case filenames. Current architecture lives under `docs/architecture/`, with `docs/ARCHITECTURE.md` as its system map. Follow the repository's architecture convention for those documents; the knowledge-file lifecycle does not govern them.

The CLI abbreviation is `kb`: use `loaf kb check`, not `loaf knowledge check`. Storage paths and QMD collection names use the full word `knowledge`.

## Knowledge Metadata

Knowledge records require at least one topic and a truthful review date. Add precise coverage globs when changes to particular source paths could invalidate the document:

```yaml
---
topics: [build-system, distribution]
last_reviewed: 2026-11-06
covers:
  - internal/cli/build.go
  - config/targets.yaml
---
```

The [frontmatter schema](../../content/skills/knowledge-base/references/frontmatter-schema.md) documents optional `consumers`, `depends_on`, and `implementation_status` fields. Coverage resolves from the Git root. Avoid covering the entire source tree or unrelated documentation: that turns ordinary churn into review noise. Conceptual guidance without a code dependency may omit coverage.

## Discover and Validate

```bash
loaf kb status --json
loaf kb validate --json
```

Native discovery recursively scans Markdown under the configured `knowledge.local` directories. Defaults include `docs/knowledge`, `docs/architecture`, and `docs/decisions`; enabling the standard architecture directory also discovers the optional `docs/ARCHITECTURE.md` overview. The old generated list `["docs/knowledge", "docs/decisions"]` expands to the current defaults at read time without rewriting configuration. Custom directory lists remain unchanged.

Ordinary Markdown without parseable knowledge frontmatter is skipped outside the standard architecture homes. A quiet check therefore does not prove that every Markdown file was inspected. The generated [CLI reference](cli-reference.md) currently has skill-style frontmatter without the required knowledge fields; validation reports that defect. Regeneration owns its bytes, so hand-editing the output is not a durable repair.

`validate` checks knowledge topics, dates, coverage matches, and dependencies. For architecture documents it checks a nonempty body and closed frontmatter when present; it does not require knowledge metadata, ADR numbering, particular headings, or architectural correctness. Read failures fail validation; absent default document directories are allowed.

`status` includes both knowledge and architecture in document totals and reports `architecture_files` separately. Coverage, staleness, and review-age metrics apply only to knowledge records.

## Check and Review

```bash
loaf kb check --json
loaf kb check --file internal/cli/build.go
loaf kb review docs/knowledge/build-system.md
```

`check` queries Git history since each knowledge file's `last_reviewed` date using its coverage globs. A matching commit means potentially stale, not proven wrong. Uncommitted edits do not themselves create a matching commit. Files without coverage cannot use this automated signal, and architecture documents are excluded.

Read a flagged document, compare its substantive claims with current code, tests, and other authoritative evidence, and correct it where necessary. Only then use `review` to set today's date. A routine edit or desire to silence a warning does not earn a new review date.

`review` refuses architecture documents, including aliases that resolve to architecture paths. It supports aliases to knowledge inside the repository and refuses resolved destinations outside it. Do not add lifecycle metadata to architecture merely to make it appear in knowledge review metrics.

## Advisory Edit Nudges

The registered PostToolUse check runs `loaf check --hook kb-staleness-nudge`. It reads hook JSON from stdin or `TOOL_INPUT`, identifies the edited file, and looks for stale local knowledge covering that path. It stays quiet for irrelevant input, unavailable Git, and knowledge without matching covered commits; failures do not block an edit.

Disposable cache markers suppress repeated warnings per document, project, and harness conversation ID. Without a conversation ID, the fallback is project and branch per UTC day. These markers are notification cache, not session records. The hook never updates review dates, and `staleness_threshold_days` does not gate its warnings.

SessionStart emits the journal continuity digest. It does not run a general knowledge-health or unprocessed-exploration-spark scan. Use the explicit KB commands for knowledge health. Imported `local-covers` mappings are not consumed by the current native nudge; cross-project staleness is not an implemented extension of this check.

## Optional QMD Integration

```bash
loaf kb init
loaf kb import shared-project --path /absolute/path/to/shared-project/docs/knowledge
```

`init` creates the default directories and missing knowledge configuration. When QMD is available, it registers `{repo-folder}-knowledge`, `{repo-folder}-architecture`, and `{repo-folder}-decisions` collections without replacing existing collections. The standalone architecture overview is discovered by native KB commands but sits outside those directory-scoped QMD collections.

`import` requires QMD, registers an external knowledge collection, and records its name in `.agents/loaf.json`. It does not copy or synchronize the source documents. Native local coverage checks remain local; use the source repository's own checks to review imported guidance against its implementation.

QMD is optional retrieval support, not Loaf's knowledge authority or a mandatory backend. These commands can change local directories, project configuration, or QMD collection registration; a documentation cleanup does not implicitly authorize running them.

## Related Guidance

- [Knowledge Base index](README.md) — contributor and operator references
- [Skill Architecture](skill-architecture.md) — authoring and distributing skill packages
- [Loaf Flow](loaf-flow.md) — work selection and workflow operation
- [Command Routing](cli-reference/command-routing.md) — shared work, private continuity, and legacy record access
- [Glossary](glossary.md) — canonical terminology, managed through `loaf kb glossary`
