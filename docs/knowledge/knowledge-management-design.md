---
topics:
  - knowledge
  - staleness
  - qmd
  - covers
  - growth-loops
covers:
  - internal/cli/kb.go
  - docs/knowledge/*.md
last_reviewed: '2026-07-14'
---

# Knowledge Management Design

## Contents

- Naming Conventions
- Knowledge File Schema
- Staleness Detection
- Growth Loops
- Memory Surface Boundary
- GridSight as Reference Implementation
- Search Instead of Sync
- QMD Integration
- Knowledge Lifetimes
- Knowledge as Extended Agent Memory
- Memory Surface Policy
- Personal Knowledge Graduation
- The Boundary Test: What Goes Where
- Staleness Detection Mitigations
- Cross-Project: local-covers
- Cross-Project: Tightly-Coupled Repos
- Optional QMD Integration
- Cross-References

This guide covers `loaf kb` metadata and retrieval mechanics. [Authority Boundaries](../architecture/authority-boundaries.md#knowledge-and-tool-boundaries) owns knowledge authority and optional tool policy; the [architecture overview](../ARCHITECTURE.md) is the entry point for current architectural models and rationale.

## Naming Conventions

Current repository conventions use `docs/knowledge/` for maintained domain guidance, `docs/architecture/` for living architecture topics, and `docs/decisions/` for deliberately retained narrow decisions. Full-word paths were chosen for readability rather than abbreviated `kb` or ADR-specific `adrs`; these are naming conventions, not immutable architecture. QMD setup uses `{repo-folder}-knowledge`, `{repo-folder}-architecture`, and `{repo-folder}-decisions` collections.

The CLI shorthand is `kb`: `loaf kb check`, not `loaf knowledge check`. The full word is for storage (durable, read by humans and tools); the abbreviation is for typing (ergonomic, used dozens of times a day). ADR files keep the `ADR-XXX` prefix — that's the record format, not the directory name.

## Knowledge File Schema

```yaml
---
topics: [engine-registry, strategy-pattern]    # Required (min 1)
last_reviewed: 2026-03-14                       # Required
covers:                                         # Recommended (enables staleness)
  - "src/pipeline/registry.py"
  - "src/models/engine_*.py"
consumers: [backend, power-systems]             # Optional (agent routing)
depends_on: [thermal-physics.md]                # Optional (cross-references)
implementation_status: in-progress              # Optional
---
```

## Staleness Detection

The `covers:` field links knowledge to code paths:

1. Parse `covers:` globs → expand to matching files
2. Query `git log --since={last_reviewed}` for those files
3. If commits exist → knowledge is potentially stale
4. Surface to agent (advisory, not blocking)

`covers:` is recommended, not required. Files without it can't use automated staleness but are valid knowledge files.

## Growth Loops

**Staleness → Review → Update.** Code edited → `covers:` match → agent nudged → reviews or updates → resets `last_reviewed`.

**Conversation → Consolidation → Knowledge.** Before compaction or at an optional wrap → agent identifies durable learning → agent creates or updates a knowledge file → human reviews.

### Hook Scope (Extended Beyond Knowledge)

**SessionStart hook** surfaces both knowledge health AND spark status:
- Relevant knowledge files and any stale coverage
- Unprocessed sparks from exploration artifacts

This reminds the agent that exploration artifacts exist and may need processing.

**PreCompact hook and `/wrap`** prompt for both knowledge consolidation AND spark capture:
- "You modified paths covered by knowledge files. Any updates needed?"
- "This conversation involved exploration. Any sparks worth noting?"

There is no SessionEnd journal hook, so this nudge lives at the PreCompact journal-flush point and in the voluntary `/wrap` checkpoint rather than a conversation-close event. Current spark capture belongs to its owning skill and private continuity; historical brainstorm sections do not define a required lifecycle.

## Memory Surface Boundary

Knowledge can accumulate across surfaces that serve different owners:

- `docs/knowledge/` holds structured project knowledge.
- `docs/architecture/` holds evolving architectural models and rationale; `docs/decisions/` holds deliberately selected narrow records under Architecture's history convention.
- Personal agent memory holds user-level preferences and cross-project context.
- The project journal holds durable events and optional synthesis.
- Conversation context holds temporary working hypotheses.

Do not duplicate the same fact across these surfaces. The knowledge management system assigns clear ownership and uses coverage metadata to surface potential drift.

## GridSight as Reference Implementation

The knowledge system was first prototyped in GridSight (`gridsight-core-gds`). Key patterns that carried forward:
- Knowledge files with YAML frontmatter (topics, consumers, depends_on, implementation_status, last_reviewed)
- README.md as index with dependency graph
- CLAUDE.md routing table mapping agent tasks → knowledge files
- Cross-references section in each file
- Categories (product/, technical/) as optional organization

What GridSight lacked (and Loaf adds): `covers:` field for staleness automation, QMD for retrieval, growth loops via hooks, CLI for management.

## Search Instead of Sync

Cross-project knowledge sharing doesn't require copying or vendoring. Index knowledge where it lives and make it searchable. QMD collections are pointers, not copies. One source of truth per file, many consumers, zero drift. Like how Google solved "information spread across the web" — index it, don't centralize it.

## QMD Integration

Existing Loaf commands can use QMD for retrieval setup:

- `loaf kb init` → local directories and config, plus `qmd collection add` when available
- `loaf kb import` → `qmd collection add` for external repo + `.agents/loaf.json` update
- Collection naming: `{repo-folder}-knowledge`, `{repo-folder}-architecture`, `{repo-folder}-decisions`

The [initializer](../../internal/cli/kb.go) creates all three directories and registers their collections only when QMD is available; rerunning it adds missing collections without replacing existing ones. The import path requires QMD collection registration. The standalone `docs/ARCHITECTURE.md` overview is discovered by native KB commands, but is not part of these directory-scoped QMD collections.

Native discovery recursively scans Markdown under the configured `knowledge.local` directories. Defaults include all three directories; enabling `docs/architecture` also discovers the optional `docs/ARCHITECTURE.md` overview. The exact old generated list `["docs/knowledge", "docs/decisions"]` expands to the new defaults at read time without rewriting the config. Other custom lists remain unchanged. Plain Markdown under the standard architecture and decisions paths is included, including index documents; ordinary knowledge files still require discoverable frontmatter.

`loaf kb validate` applies knowledge metadata rules only to knowledge files. For architecture topics, the overview, and retained decisions, it checks only a nonempty document body and terminated frontmatter when present. It does not enforce knowledge fields, ADR numbering, particular headings, YAML semantics, or architectural correctness. File and subtree read failures appear in validation results and fail the command; absent default document directories remain optional. The Architecture skill and the repository's retained decision convention govern substantive review.

`loaf kb status` reports total documents and an `architecture_files` count. Coverage counts, staleness, and review age apply only to knowledge files; `loaf kb check` excludes architecture documents, and `loaf kb review` refuses them even if they carry historical knowledge metadata. Review checks the resolved destination as well as the supplied path, supports aliases to knowledge within the repository, and refuses destinations outside it. Do not add `topics` or `last_reviewed` to architecture documents for tooling compatibility. This scoped runtime correction does not certify this guide's older operational claims or reset its review date.

`loaf kb glossary` is the Markdown-backed terminology surface. It supports
canonical terms, aliases, candidate terms, and stabilization without moving
glossary prose into the operational task/session index.

## Knowledge Lifetimes

```
PERSISTENT (this person)     — crosses all projects, grows over career
  └→ DURABLE (this project)  — lives in git, shared with team
      └→ EPHEMERAL (this conversation) — lives in the context window
```

| Lifetime | What | Where |
|----------|------|-------|
| **Persistent** | User preferences, expertise, patterns | User-managed harness instructions or memory |
| **Durable** | Domain knowledge, conventions, architecture | `docs/knowledge/`, `docs/architecture/`, selected `docs/decisions/`, and root `AGENTS.md` |
| **Ephemeral** | Active work context, working hypotheses | Conversation context |

Flow: persistent informs durable, durable informs ephemeral. Upward: conversation insights consolidate into durable knowledge, and durable patterns may graduate to persistent memory.

## Knowledge as Extended Agent Memory

The knowledge base supplies agents with domain rules and cross-cutting context. Link to strategy, owning architecture topics, and canonical tracker work rather than duplicating future plans, scope, or status here.

## Memory Surface Policy

| Surface | Contains |
|---------|----------|
| `docs/knowledge/` | Domain guidance and technical references |
| `docs/architecture/` | Current architectural model and rationale |
| `docs/decisions/` | Rare, deliberately selected single-choice records |
| User-managed memory | Preferences and continuity under the user's chosen harness policy |
| Project journal | Durable project events and optional synthesis |
| Conversation context | Ephemeral work context |
| Additional harness memory tools | User-managed; not Loaf's canonical knowledge or private-continuity store |

Loaf does not prescribe a Serena-specific memory policy. The user may choose additional memory tools; neither this guide nor a documentation migration authorizes erasing or repurposing their data.

## Personal Knowledge Graduation

Personal knowledge grows through correction, not capture. When you say "don't use bare except clauses," that's a personal preference that applies everywhere — not just this project.

The graduation path:
```
Per-project correction → repeated across N projects → pattern recognized →
suggested promotion to user-managed global instructions → user approves → permanent
```

This is a convention and documentation boundary, not an automated cross-project pattern detector. The skill documents what belongs in personal agent instructions versus project knowledge; promotion still requires user approval.

## The Boundary Test: What Goes Where

| Surface | Contains | Decision Test |
|---------|----------|---------------|
| **Code** (docstrings, types) | What the code does | Is it self-documenting? → stays in code |
| **Knowledge files** | Domain rules, cross-cutting context | Requires context beyond the code? → knowledge file |
| **Architecture topics** | Current model and why it has this shape | Architectural model or rationale? → owning topic; Architecture evaluates exceptional ADRs |
| **AGENTS.md** | Project agent instructions, conventions | Is it about project agent behavior? → root AGENTS.md |
| **MEMORY.md** | User preferences, session pointers | Is it personal/ephemeral? → MEMORY.md |

Root AGENTS.md references knowledge files but never duplicates their content. Verified harness compatibility paths do not become a second real instruction body.

## Staleness Detection Mitigations

| Risk | Mitigation |
|------|-----------|
| Alert fatigue (too many nudges) | Cooldown: max 1 nudge per knowledge file per session |
| Broad globs = constant nudging | Threshold: only nudge if >N days since review OR >N commits |
| Agent updates incorrectly | Advisory: "Consider reviewing" not "I've updated it" |
| Configurable thresholds | `staleness_threshold_days` in `.agents/loaf.json` |

## Cross-Project: `local-covers`

When importing knowledge from another repo, the knowledge file's `covers:` globs are relative to its origin repo. For staleness detection in the importing repo, use `local-covers` in the import config:

```json
{
  "knowledge": {
    "imports": [
      {
        "name": "gridsight-core-gds",
        "local-covers": {
          "thermal-physics": ["src/thermal_adapter.py"],
          "engine-registry": ["src/adapters/engine_*.py"]
        }
      }
    ]
  }
}
```

Each consuming repo declares which of ITS code paths relate to imported knowledge. The harness uses `local-covers` for PostToolUse nudges. The knowledge file itself doesn't need to know about every consumer.

## Cross-Project: Tightly-Coupled Repos

For tightly-coupled project families (e.g., GDS core + modules), the simplest approach is symlinks or workspace linking — work from the core repo with modules linked in. The agent sees `docs/knowledge/` directly. No import mechanism needed.

QMD collections + `loaf kb import` are for loosely-coupled sharing where symlinks aren't practical.

## Optional QMD Integration

QMD works without models (BM25-only):
- `qmd collection add/remove`, `qmd search`, `qmd get`, `qmd update` — no models needed
- `qmd embed` (embeddings) and `qmd query` (hybrid+reranking) — need ~2GB of GGUF models
- Existing Loaf commands may integrate with QMD when it is available, but QMD is not a mandatory knowledge backend. Semantic search is opt-in.

## Cross-References

- [skill-architecture.md](skill-architecture.md) — how knowledge packages are authored and distributed
- [work-model.md](work-model.md) — how maintained knowledge relates to tracker work, Git, and private continuity
- [Knowledge and tool boundaries](../architecture/authority-boundaries.md#knowledge-and-tool-boundaries) — optional retrieval and user-managed memory
- [../ARCHITECTURE.md#authority-model](../ARCHITECTURE.md#authority-model) — authority model
- Full brainstorm: SQLite brainstorm/spark records or an explicitly durable report
