# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0-dev.5] - 2026-03-29

### Added
- `loaf cleanup` command — scan `.agents/` artifacts and recommend cleanup actions (SPEC-012)
  - Covers all 7 artifact types: sessions, tasks, specs, plans, drafts, councils, reports
  - `--dry-run` and `--sessions`/`--specs`/`--plans`/`--drafts` filters
  - Non-TTY pipe-safe output (behaves like `--dry-run` when piped)
  - Interactive per-item confirmation with delete previews
  - Nested frontmatter support (`session.*`, `council.*`, `report.*`)
  - Dual council schema support (council-session + orchestration formats)
  - Detects drafts promoted to specs via `source` field cross-reference
- Shared prompt helpers (`askYesNo`, `askChoice`, `isTTY`) in `cli/lib/prompts.ts`
- Pre-merge prompt hook for squash merge conventions (clean body, no auto-dump)
- Prompt hook support in build system (Claude Code target; filtered for other targets)
- Advisory `/reflect` suggestion in `/implement` AFTER phase when session has extractable learnings (SPEC-011)
- Post-implementation reflection flag in `/shape` Step 9 for sessions with strategic tensions (SPEC-011)
- `/reflect` recommendation in `/cleanup` extraction checks before archiving decision-rich sessions (SPEC-011)

### Changed
- Spec cleanup (task archival, spec archival) moved to pre-merge on the feature branch instead of post-merge on main
- Post-merge housekeeping reduced to: pull main, delete branch, suggest reflection
- `/cleanup` skill updated to reference CLI as execution engine (skill = policy + Linear, CLI = filesystem)

### Fixed
- Pre-push hook changed from unconditionally blocking (exit 2) to advisory (exit 0)
- Stale `docs/specs/` paths in `/reflect`, `/shape`, and spec template — now `.agents/specs/`

## [2.0.0-dev.4] - 2026-03-27

### Added
- `loaf task archive` command — move completed tasks to archive and update TASKS.json atomically
- `loaf spec archive` command — same for completed specs
- `loaf task sync --push` — push JSON metadata to .md frontmatter (reverse sync)
- Tasks section in `/cleanup` skill with drift detection and CLI-based archival
- Archive step in post-merge housekeeping hook
- SPEC-016 draft: Council Advisory Redesign
- `loaf version` subcommand showing version, Node.js, built targets, and content stats (TASK-020)

### Changed
- Post-merge hook split into pre-merge checklist (changelog, version, build) and post-merge housekeeping (archival, cleanup)
- `/cleanup` archival process now uses CLI commands instead of raw `mv`
- Skills and references replaced `.agents/` path references with CLI commands and IDs
- `council-session` skill changed to model-invoked (not user-invocable)

## [2.0.0-dev.3] - 2026-03-27

### Added
- Workflow enforcement hooks: pre-PR (conditional blocker), post-merge (housekeeping checklist), pre-push (branch safety) (SPEC-015)
- Project-level CHANGELOG.md in Keep a Changelog format with retroactive entries
- Hook library functions `parse_command` and `parse_exit_code` in json-parser.sh

## [2.0.0-dev.2] - 2026-03-27

### Added
- `/bootstrap` skill and `loaf setup` CLI command for 0-to-1 project setup (SPEC-013)

## [2.0.0-dev.1] - 2026-03-25

### Added
- Knowledge management system with staleness tracking and lifecycle hooks (SPEC-009)
- `loaf task` and `loaf spec` CLI commands with managed markdown data model
- `loaf task list --active` flag for filtering in-progress tasks
- `loaf release` command with pre-release versioning support
- `loaf init` command with safe project scaffolding
- `loaf install` command replacing the shell-based installer
- Vitest test infrastructure and task management tests
- TypeScript build system replacing the shell-based builder (SPEC-008)
- Loaf CLI v2.0.0 skeleton and source reorganization

### Fixed
- Post-merge housekeeping steps added to implement skill
- Code review findings from SPEC-008 implementation addressed
- Redundant root CLAUDE.md symlink removed
