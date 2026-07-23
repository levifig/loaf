---
change: markdown-reimport-safety
created: 2026-07-23
branch: markdown-reimport-safety
---

# Markdown Re-import Safety

## Problem

Re-importing `.agents` Markdown into SQLite (`loaf migrate markdown`) is broken in two independent ways, both surfaced by the gridsight-core-gds migration on 2026-07-22 (journal findings `journal:453d67cf2bd7bf131a1b5f09`, `journal:8038077b8a7e19e3abe77533`).

First, the dry-run and apply paths disagree because they share no code: `PreviewMarkdownMigration` only counts files and never opens the database, while apply runs the whole import in one transaction and hard-fails on the first journal entry whose `journal_origins` row carries a non-`migration` `capture_mechanism` (`markdown_import.go:740`). Schema migration 0011 backfilled every pre-existing entry with mechanism `unknown`, so any project that imported Markdown before 0011 now fails wholesale on re-import — gridsight previewed 108 importable sessions, then apply refused everything (2,841 `unknown`-origin rows).

Second, the import treats frontmatter as status authority: `upsertSpec`/`upsertTask`/`upsertSimpleEntity`/`upsertReport` all overwrite `status` unconditionally on conflict, with no archived guard and no event trail. Gridsight's SPEC-007/008 were archived in SQLite on 2026-07-08 (events intact) and silently flipped back on the 2026-07-22 import, and out-of-vocabulary spellings like `accepted` landed raw in the database. PR #130 closed the archive-gate vocabulary and normalizes spec status at import, but the clobber itself, the missing normalization for other entity kinds, and the preview blindness all remain.

## Hypothesis

If import decisions become deterministic functions of database state (reclaim provably-import-created origins, skip foreign provenance, never overwrite an existing lifecycle status) and the dry-run executes the same code path as apply inside a rolled-back transaction, then re-import becomes idempotent and safe, preview and apply can never disagree again, and the class of silent lifecycle regressions is closed rather than patched.

## Scope

**In**

- Origin-collision semantics in the importer: `unknown`-mechanism origins on ID-matched entries are reclaimed as `migration` and re-imported; `manual`/`skill`/`hook` origins cause the entry to be skipped untouched, counted, and listed. Apply never aborts on a collision.
- Insert-only status: frontmatter status (normalized) is written only when a row is first created or when it fills a placeholder `unknown`; existing SQLite statuses are never overwritten, and divergences are reported in both preview and apply results.
- Lifecycle vocabulary normalization at insert for tasks, reports, ideas, and brainstorms via `CanonicalLifecycleStatus` (specs landed in PR #130); out-of-vocabulary values are stored raw on first insert and surfaced as warnings.
- Preview-as-simulation: when the global database and project row exist, `--dry-run` runs the real import in a transaction and rolls it back, reporting exactly what apply would do; when they don't, the current file-count preview remains.
- Additive report fields on `MarkdownMigrationPlan`/`MarkdownMigrationResult` (reclaimed-origin count, skipped-entry list with mechanisms, status-divergence list) rendered in human and `--json` output; `StateJSONContractVersion` stays 1 per the additive-fields convention.
- Regression coverage shaped like gridsight: pre-0011 `unknown` backfill, an archived spec whose Markdown still says a terminal legacy status, one foreign-mechanism entry, and a double-apply idempotency check.

**Out** (deferred, not rejected)

- A doctor check that detects historical archived-flip damage in existing databases — follow-up candidate once these semantics land.
- Repairing gridsight's data in code: SPEC-007/008 re-archive is an operational step (`loaf spec archive` accepts them post-#130) noted for the gridsight session, not shipped here.
- Any markdown-export or rollback-manifest changes beyond carrying the new report fields.

**Cut** (explicitly rejected)

- A `--skip-conflicts` flag: the deterministic reclaim/skip semantics make apply total, so there is no refusal left to skip and no flag to carry forever.
- Overwriting `manual`/`skill`/`hook` provenance under any setting — deliberately re-stamped origins are facts the importer must not erase.
- Timestamp- or merge-based status reconciliation between Markdown and SQLite — SQLite is the source of truth once a row exists; import is a one-way legacy migration, not sync.

## Observable Workflow

On a pre-0011 project, `loaf migrate markdown --dry-run` opens the database read-only, simulates the import, and reports per-kind counts plus `reclaimed origins: N`, `skipped (foreign provenance): N` with mechanisms, and `status divergences: N` naming each kept SQLite status. `loaf migrate markdown --apply` then succeeds with the same numbers — the gridsight scenario imports its 108 sessions instead of refusing. Archived specs stay archived through any number of re-imports, and running `--apply` twice in a row produces an identical report with nothing newly mutated. `--json` carries the same new fields for both modes.

## Rabbit Holes and No-Gos

- Two-way Markdown↔SQLite sync: divergence reporting will tempt "why not write the DB status back to frontmatter" — import stays one-way, and export is a different surface.
- Redesigning the journal provenance model: this Change consumes `capture_mechanism` as-is; vocabulary or envelope changes belong to the origin-hygiene lineage (PR #127), not here.
- In-code repair of historical damage: detecting and fixing rows already flipped in other projects' databases is the deferred doctor follow-up, not import logic.
- Reworking `--resume`: with idempotent apply it is behaviorally identical to `--apply`; it stays an accepted alias and its removal is somebody else's cleanup.

## Decisions

Provenance: decisions 1–3 accepted by interview on 2026-07-23 (one question at a time, recommended option chosen each time); 4–7 made autonomously during shaping per granularity convention.

1. **Reclaim `unknown` origins, skip foreign, no new flag.** Import entry IDs are deterministic (`stableMigrationID` over project, path, line), so an ID match proves prior import provenance; the 0011 backfill's `unknown` rows are reclaimable as `migration` without lying. Genuinely foreign mechanisms mean deliberate re-stamping, so those entries are skipped whole and reported. Forecloses `--skip-conflicts` and wholesale-abort behavior.
2. **Insert-only status.** Frontmatter status is written on first insert (or over a placeholder `unknown`) and never over an existing row; title, body, and source still refresh. Matches ARCHITECTURE.md's SQLite-owns-operational-state boundary and makes archived terminal by construction. Forecloses status_changed events for import (import no longer changes statuses) and any frontmatter-wins semantics.
3. **Preview simulates apply in a rolled-back transaction.** One code path serves both modes, so drift between preview and apply becomes structurally impossible; the file-count preview remains only when there is no database or project row to consult, and preview never creates either.
4. **Out-of-vocabulary statuses are stored raw with a warning on first insert.** Inventing a mapping would fabricate lifecycle state; honest storage plus a visible warning matches the provenance-honesty principle.
5. **Normalization parity for the remaining entity kinds.** Tasks, reports, ideas, and brainstorms get the same `CanonicalLifecycleStatus` treatment specs received in PR #130, at insert time only.
6. **`--resume` remains an alias of `--apply`.** Idempotent semantics make the distinction cosmetic; the flag surface does not change in either direction.
7. **Report fields are additive under JSON contract version 1.** The plan/result structs have taken additive fields without a bump before; existing fields keep their meaning.

## Planning Contract

### Approach

Refactor `Store.ImportMarkdown` so the importer produces an `ImportReport` (reclaimed, skipped, divergences, warnings) and takes a commit/rollback mode; `ApplyMarkdownMigration` commits, the new simulation path rolls back, and `PreviewMarkdownMigration` becomes a thin dispatcher between simulation and the file-count fallback. Centralize the two decision points as small pure helpers — origin disposition from `capture_mechanism`, and status disposition from (existing status, incoming normalized status) — so tests pin the semantics directly. All four `upsertX` statements drop `status = excluded.status`; the insert-only rule is expressed in SQL (`ON CONFLICT ... SET status = CASE WHEN specs.status = 'unknown' THEN excluded.status ELSE specs.status END` or an equivalent read-then-write) with the divergence recorded when the kept status differs from the incoming one. The foreign-origin skip happens before `upsertJournalEntry` so skipped entries are never mutated at all, and it covers the line's derived writes too — sparks, spark aliases, and promoted-to relationships parsed from a skipped session line are skipped with it.

### Placement

`internal/state/markdown_import.go` (decision helpers, report, skip-before-upsert ordering), `internal/state/markdown_migration.go` (preview dispatch and result envelopes), `internal/cli/cli.go` (human rendering of the new report lines for dry-run/apply/resume, JSON passthrough). Tests in `internal/state/markdown_import_test.go`, `markdown_migration_test.go`, and CLI-level parity assertions in `internal/cli/cli_test.go`.

### Preview/database boundary

Preview must have zero side effects: it never calls `Initialize`, never creates the database file, the project row, or WAL siblings, and skips the FTS rebuild (`rebuildAndVerifyJournalSearch`) that apply runs before commit. Simulation engages only when the global database file exists and the project identity resolves to a registered row; otherwise the file-count plan is returned with a note that conflict detection needs an initialized project.

### Risks

- The FTS rebuild inside the apply transaction is wasted work under rollback; preview must skip it explicitly rather than inherit it.
- Insert-only status changes semantics for anyone still treating Markdown as live status authority after first import; the divergence report is the mitigation — nothing silently changes, and the kept-vs-incoming pair is printed.
- SQLite `CASE`-based conditional upserts are easy to get subtly wrong per table; the shared decision helper plus per-kind tests guard against copy-paste drift across the four entity upserts.
- Concurrent writers during a long simulation hold a read transaction; acceptable because import already takes a full write transaction of the same shape.

### Sequencing

U1 defines the report type and semantics that U2 and U4 consume, so it lands first; U3 is independent of U1/U2 and can proceed in parallel; U5 grows alongside each unit rather than trailing; U6 is mechanical once the CLI surface settles.

## Implementation Units

- **U1 — Import decision semantics and report contract.** Reclaim/skip origin disposition, insert-only status with placeholder fill, divergence recording, and the additive `ImportReport` fields on plan/result structs.
- **U2 — Preview as simulation.** Shared run path with commit/rollback modes, side-effect-free preview dispatch, file-count fallback when the database or project row is absent, FTS rebuild skipped under rollback.
- **U3 — Vocabulary normalization parity.** `CanonicalLifecycleStatus` at insert for tasks, reports, ideas, brainstorms; raw-plus-warning for out-of-vocabulary values.
- **U4 — CLI rendering.** Human output lines and `--json` fields for reclaimed/skipped/divergences in dry-run, apply, and resume; usage/help text updated to describe the report.
- **U5 — Regression fixtures and tests.** Gridsight-shaped fixture (pre-0011 `unknown` backfill, archived spec with terminal legacy frontmatter, one foreign-mechanism entry), preview==apply parity assertions, double-apply idempotency, per-kind status-disposition unit tests.
- **U6 — Docs.** CLI reference regeneration and the loaf-reference/skill passages describing `migrate markdown` semantics.

## Verification Contract

<!-- Executable (machine-checkable): -->

- **V1.** `npm run test` (go test ./...) passes on the branch.
- **V2.** New state tests pass by name: reclaim-unknown-origins, skip-foreign-origins, insert-only-status (including placeholder fill and archived-stays-archived), preview-equals-apply parity, double-apply idempotency.
- **V3.** `loaf build` and `npm run typecheck` succeed; regenerated CLI reference committed if it changes.
- **V4.** Isolated smoke under `LOAF_DB=$(mktemp -d)/loaf.sqlite`: seed the gridsight-shaped fixture, assert `--dry-run --json` and `--apply --json` report identical counts, then a second `--apply` reports zero new mutations.

<!-- Human review: -->

- **H1.** A reviewer confirms no code path can write `migration` over a `manual`/`skill`/`hook` origin, and that skipped entries are never partially mutated.
- **H2.** No new CLI flags exist; `--resume` behavior is unchanged as an alias.
- **H3.** JSON output changes are additive only — every pre-existing field keeps its name, type, and meaning.

## Definition of Done

- All six units landed on `markdown-reimport-safety`, V1–V4 green, H1–H3 confirmed in review.
- `loaf change check` reports zero violations.
- PR squash-merged per convention, with the gridsight ops note (re-archive SPEC-007/008) and the doctor-check follow-up filed where triage will see them.

## Durable Outputs

- An ADR in `docs/decisions/` recording the import authority model: deterministic-ID origin reclaim, insert-only status, and preview-as-simulation as the permanent contract for one-way Markdown migration.
- Updated `loaf-reference` skill passage for `loaf migrate markdown` describing the report fields and the no-abort semantics.

## Open Questions

- [KU] Do any plugin or skill surfaces parse `MarkdownMigrationPlan`/`Result` JSON strictly enough to notice new fields? → owner: U4 (verify consumers during CLI wiring; Go's JSON decoding is tolerant, but rendered docs may enumerate fields).
- [KU] When the database exists but the project row does not, is the file-count fallback acceptable, or should simulation bootstrap a throwaway in-memory copy? → owner: U2 (default is the fallback; revisit only if dogfooding shows preview lying on first-ever imports).

## Source Inputs

- Journal findings `journal:453d67cf2bd7bf131a1b5f09` (origin-collision dry-run/apply asymmetry) and `journal:8038077b8a7e19e3abe77533` (lifecycle status clobber), triaged 2026-07-23 from the gridsight-core-gds session `loaf-migrate-markdown-workflow`.
- Empirical evidence from the global database: 2,841 `unknown`-mechanism origins in gridsight, SPEC-007/008 archived→complete flip with 2026-07-08 archive events intact.
- PR #130 (spec terminal-status vocabulary close) as the landed base this builds on; PR #127 and schema migration 0011 as the origin-vocabulary context.
- Shaping interview in this conversation (2026-07-23), decisions 1–3.
