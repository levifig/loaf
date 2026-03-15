---
topics: [cli, commands, distribution, multi-harness]
last_reviewed: 2026-03-14
---

# CLI Design

Loaf evolves from a build-time framework to a unified CLI tool.

## Command Structure

```sh
loaf kb init                    # Scaffold knowledge base + QMD collections
loaf kb import                  # Interactive fuzzy import from another project
loaf kb check                   # Staleness report (covers: + git)
loaf kb validate                # Frontmatter consistency check
loaf kb status                  # Quick summary
loaf kb review <file>           # Mark as reviewed (reset last_reviewed)

loaf task list                  # Task board (from TASKS.json)
loaf task status                # Summary
loaf spec list                  # Spec status

loaf build                      # Build all targets
loaf build --target <name>      # Specific target
loaf install --to <target>      # Install to specific harness
loaf install --to all           # Auto-detect all harnesses

loaf skill list                 # List installed skills
loaf skill add <name>           # Add from marketplace
loaf skill validate             # Check skill structure

loaf init                       # Full project setup
```

### Additional Flags (Discussed)

```sh
loaf kb check --verbose                # Detailed: shows specific commits since review
loaf kb check --file <path>            # Check coverage for a specific file
loaf kb check --modified-this-session  # Check only paths modified in current session
```

**`--verbose` output format:**
```
  engine-registry.md  ⚠ STALE
    Last reviewed: 2026-02-15 (27 days ago)
    Covers: src/pipeline/registry.py, src/models/engine_*.py
    Changes since review:
      abc1234 feat: add FallbackEngine type (3 days ago)
      def5678 fix: engine priority ordering (12 days ago)
```

## Design Principles

- **Agent-creates, human-curates.** Most authoring commands are for agents. CLI is for human management and health checks.
- **CLI as cross-harness equalizer.** Agents on any harness can call `loaf kb check` via Bash.
- **Thin wrappers over QMD.** `loaf kb init/import` wrap `qmd collection add`. Don't reinvent retrieval.
- **Progressive.** Works without QMD (basic frontmatter parsing). Better with QMD (semantic search).

## Import UX

Interactive fuzzy search, ranked by name similarity to current project. Projects grouped (not separate knowledge/decisions). KB imported by default, decisions opt-in.

## Inspiration

- EveryIn's compound-engineering-plugin: `install --to <target>` pattern for multi-harness distribution
- QMD: collections + context model for knowledge retrieval

## Cross-References

- [knowledge-management-design.md](knowledge-management-design.md) — the `loaf kb` feature design
- [agent-harness-vision.md](agent-harness-vision.md) — how agents use the CLI
