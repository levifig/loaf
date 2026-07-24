---
change: markdown-reimport-safety
created: 2026-07-23
branch: markdown-reimport-safety
---

# Markdown Re-import Safety

## Problem

Re-importing `.agents` Markdown into SQLite (`loaf migrate markdown`) is broken in two independent ways, both surfaced by the gridsight-core-gds migration on 2026-07-22 (journal findings `journal:453d67cf2bd7bf131a1b5f09`, `journal:8038077b8a7e19e3abe77533`).

First, the dry-run and apply paths disagree because they share no code: `PreviewMarkdownMigration` only counts files and never opens the database, while apply runs the whole import in one transaction and hard-fails on the first journal entry whose `journal_origins` row carries a non-`migration` `capture_mechanism` (`markdown_import.go:740`). Schema migration 0011 backfilled every pre-existing entry with mechanism `unknown`, so any project that imported Markdown before 0011 now fails wholesale on re-import — gridsight previewed 108 importable sessions, then apply refused everything (2,841 `unknown`-origin rows).

Second, the import treats source status as authority: `upsertSpec`/`upsertTask`/`upsertSimpleEntity`/`upsertReport` all overwrite `status` unconditionally on conflict, with no archived guard and no event trail. Gridsight's SPEC-007/008 were archived in SQLite on 2026-07-08 (events intact) and silently flipped back on the 2026-07-22 import, and out-of-vocabulary spellings like `accepted` landed raw in the database. PR #130 closed the archive-gate vocabulary and normalizes spec status at import, but the clobber itself, the missing normalization for other entity kinds, and the preview blindness all remain.

## Hypothesis

If import decisions become deterministic functions of database state (reclaim exactly the origin rows that carry no information beyond their paired journal row, skip all other foreign provenance, never overwrite a real lifecycle status) and the dry-run executes the identical apply pipeline against a verified disposable snapshot of the database, then re-import becomes safe and business-state idempotent, and dry-run and apply cannot disagree whenever both run against the same database and `.agents` state. Where no database or registered project exists, the dry-run is an honestly labeled file inventory and parity is inapplicable — there is no apply outcome to disagree with until state exists.

## Scope

**In**

- Origin-collision semantics in the importer: an origin row is reclaimed as `migration` only when it matches the full 0011-compatible fingerprint — `capture_mechanism = 'unknown'`, `envelope_version = 1`, NULL in every field 0011 wrote NULL (`observed_harness`, `observed_harness_version`, `agent_id`, `source_event`, `head`, `change_path`, `change_sha256`, `dirty`, `reconstructable`, `durable_result_kind`, `durable_result_id`), and null-safe equality with the paired journal row for every field 0011 copied (`harness_session_id`, `branch`↔`observed_branch`, `worktree`↔`observed_worktree`, `created_at`). Every other non-`migration` origin — `manual`, `skill`, `hook`, custom mechanisms, future envelope versions, and any `unknown` row carrying evidence or mismatched copies — causes the entry to be skipped untouched, counted, and listed. Apply never aborts on an origin collision.
- Status authority: stored `unknown` is never authoritative — the importer is the only writer of `unknown` (status mutations are vocabulary-gated in `spec_status.go` and `unknown` is outside both canonical and legacy vocabularies), so it definitionally records the absence of lifecycle information. A real normalized incoming status fills a stored `unknown`; any other stored status is never overwritten, and kept-vs-incoming divergences are reported. Archived can never flip back because `archived ≠ unknown`.
- Lifecycle vocabulary normalization at insert for tasks, reports, ideas, and brainstorms via `CanonicalLifecycleStatus` (specs landed in PR #130), governed by the normative per-kind status table in the Planning Contract; explicit out-of-vocabulary values are stored raw on first insert and surfaced as warnings; absent and explicitly-`unknown` source status are no-opinion — no warning, no divergence.
- Dry-run as snapshot simulation: when the database file exists and the project resolves to a registered row (read-only lookup), `--dry-run` snapshots the database with the existing `state.Backup` primitive (`VACUUM INTO` plus its verification pass — integrity check, foreign-key check, journal-search parity, provenance integrity), runs the complete apply pipeline — including `Initialize` preflight (`RequireCurrentSchema`, registration refresh) and the FTS rebuild — against the snapshot, reports, and removes the snapshot. Under a quiescent fixture the live main database and `-wal` bytes are unchanged; `-shm` is ephemeral SQLite coordination state and excluded, matching the precedent in `backup_test.go:477`. When no database or registered project exists, the cheap file-inventory preview runs, explicitly labeled inventory-only.
- A new `import_report` object (reclaimed-origin count, skipped-entry list with mechanisms, status-divergence list, and status/provenance outcome warnings) computed inside the importer's own transaction and carried on both simulate and apply results; inventory I/O and parser warnings stay in the existing plan `warnings` field with its meaning unchanged. Dry-run results carry a `mode` discriminant (`simulation` or `inventory`); `import_report` is present exactly when `mode` is `simulation`.
- Apply no longer calls preview: `ApplyMarkdownMigration` produces inventory and `import_report` from its single committed transaction, eliminating the current preview-then-import TOCTOU window (`markdown_import.go:70`).
- Spark target-refresh semantics: spark-derived imported relationships are delete-then-reinserted per spark (parity with `deleteImportedRelationships` on artifacts), so a changed promoted-to target no longer strands the old edge while the line remains a spark.
- Inventory counter fixes: the cheap spark counter learns `sessions/archive/*.md` (parity with apply's file set), and counter read failures surface as plan warnings instead of being swallowed.
- Snapshot hygiene: snapshots live in a loaf-owned `0700` temp subdirectory with `0600` files named with pid and timestamp, cleanup is registered before creation so creation failure, cancellation, or ENOSPC leaves no partial file, and removal covers the snapshot's `-wal`/`-shm` siblings; deletion is best-effort on process crash, with the residue pattern named so a future sweep can collect strays.
- Regression coverage shaped like gridsight: 0011-fingerprint backfill rows, evidence-bearing and copy-mismatched `unknown` rows, an archived spec whose Markdown still says a terminal legacy status, foreign-mechanism and future-envelope entries, and a second-apply business-state idempotence check over the full import-touched table set.

**Out** (deferred, not rejected)

- A doctor check that detects historical archived-flip damage in existing databases, and a residue sweep for crash-orphaned snapshots — follow-up candidates once these semantics land.
- Repairing gridsight's data in code: SPEC-007/008 re-archive is an operational step (`loaf spec archive` accepts them post-#130) noted for the gridsight session, not shipped here.
- An alias provenance model: import-created alias rows remain import-refreshable, and hand-curated aliases are not a supported surface today; protecting them needs provenance on `aliases`, which is its own Change.
- Reconciling removed or type-changed session lines: a line that stops being a `spark` (edited to another type, or deleted) leaves its previously derived spark, alias, and relationships in place — orphaned-derivative garbage collection is legacy-content hygiene for the doctor follow-up, not import logic.
- Any markdown-export or rollback-manifest changes beyond carrying the new report fields.

**Cut** (explicitly rejected)

- A `--skip-conflicts` flag: the deterministic reclaim/skip semantics make apply total with respect to origin collisions, so there is no refusal left to skip and no flag to carry forever. (Apply can still fail for real reasons — schema preflight, FTS rebuild, I/O — and those failures are identical in simulation.)
- Reclaiming any origin outside the full 0011-compatible fingerprint — the fingerprint cannot prove 0011 authored the row, and does not try to; it proves the row is information-free above its paired journal row, which is the actual safety requirement. Anything with evidence or mismatched copies is skipped, never rewritten.
- Timestamp- or merge-based status reconciliation between Markdown and SQLite — SQLite is the source of truth once a real status exists; import is a one-way legacy migration, not sync.
- Byte-identical idempotence and `-shm` byte guarantees: `updated_at`/`imported_at` refresh on re-import by design, and the WAL index is mutable coordination state; idempotence is defined over logical business state.

## Observable Workflow

On a pre-0011 project, `loaf migrate markdown --dry-run` reports `mode: simulation`, the file inventory, and an import report: `reclaimed origins: N`, `skipped (foreign provenance): N` with mechanisms, `status divergences: N` naming each kept SQLite status, and the registered project identity — computed by really running the import against a verified throwaway snapshot. `loaf migrate markdown --apply` then reports the same numbers from its committed transaction, provided the database and `.agents` did not change between the two commands (loaf takes no cross-command lock; the parity precondition is stated, not assumed). The gridsight scenario imports its 108 sessions instead of refusing. Archived specs stay archived through any number of re-imports. A second `--apply` reports `reclaimed origins: 0`, the same skipped list, and unchanged business state across every import-touched table. On a fresh machine with no database, `--dry-run` reports `mode: inventory` with an explicit note that conflict detection needs an initialized project, and no `import_report`. `--json` carries the same fields; all pre-existing fields keep their meanings.

## Rabbit Holes and No-Gos

- Two-way Markdown↔SQLite sync: divergence reporting will tempt "why not write the DB status back to frontmatter" — import stays one-way, and export is a different surface.
- Redesigning the journal provenance model: this Change consumes `capture_mechanism` as-is; vocabulary or envelope changes belong to the origin-hygiene lineage (PR #127), not here.
- In-code repair of historical damage: detecting and fixing rows already flipped in other projects' databases is the deferred doctor follow-up, not import logic.
- Reworking `--resume`: with business-state idempotent apply it is behaviorally identical to `--apply`; it stays an accepted alias and its removal is somebody else's cleanup.
- Proving snapshot-provenance beyond the Backup primitive's existing verification contract: integrity, foreign keys, and journal-search parity are already asserted there; do not build a second verification layer.

## Decisions

Provenance: decisions 1–3 were accepted by interview on 2026-07-23 and hardened across two adversarial review rounds (Codex, gpt-5.6 xhigh, 2026-07-23/24 — see Source Inputs); 4–9 were made autonomously during shaping and review disposition.

1. **Reclaim requires the full 0011-compatible fingerprint, evaluated against both the origin row and its paired journal row.** The decision helper receives both rows and requires: mechanism `unknown`, envelope version 1, NULL in all eleven fields 0011 wrote NULL, and null-safe equality for the four values 0011 copied from the journal row. This does not prove 0011 authored the row — nothing can — and the claim is deliberately weaker: a matching row is information-free above the journal row the import already owns by deterministic ID, so rewriting it as `migration` destroys no evidence that is not being deliberately refreshed from the source file. Rows failing any predicate — including `dirty = 1`, populated `observed_harness_version`, or a branch value differing from the journal row — are skipped whole. Hardened per review round 2 finding 1; forecloses `--skip-conflicts` and wholesale-abort behavior.
2. **Stored `unknown` is never authoritative; everything else is insert-only.** The importer is the only writer of status `unknown` — the status-mutation surface is vocabulary-gated (`spec_status.go:32`) and `unknown` is in neither the canonical nor the legacy vocabulary — so a stored `unknown` definitionally records "no lifecycle information", and filling it with a real normalized status is strict information gain on any import. Every other stored status is never overwritten; divergences are reported with both sides. This replaces the structural-placeholder rule (`body_source_id IS NULL` is demonstrably not placeholder-exclusive — review round 2 finding 2, `spec_lifecycle_test.go:535`) and needs no placeholder concept at all: `ensureSpecPlaceholder` rows are covered because they are created at `unknown`. Forecloses status_changed events for import and any frontmatter-wins semantics.
3. **Dry-run runs the real apply pipeline against a snapshot taken by the existing Backup primitive.** Simulation is `ApplyMarkdownMigration` pointed at a verified copy: `state.Backup`'s `VACUUM INTO` plus its existing verification (integrity, foreign keys, journal-search parity — which also covers the VACUUM rowid/FTS-coupling concern) produces the snapshot, and the full pipeline including `Initialize` preflight and the FTS rebuild runs there, so a behind-schema database or FTS corruption fails simulation and apply identically. Registration and schema writes land on the disposable copy. Live-file preservation is asserted over main database and WAL bytes under a quiescent fixture; `-shm` is excluded as ephemeral coordination state (`backup_test.go:477` precedent). Snapshot-creation failure is its own clearly reported error class, not claimed identical to any apply failure. Hardened per review round 2 findings 3, 4, and 5.
4. **Idempotence means equivalent business state over every import-touched table.** After a second apply, the logical dump of `specs`, `tasks`, `ideas`, `brainstorms`, `shaping_drafts`, `reports`, `sparks`, `journal_entries`, `journal_origins`, `aliases`, `relationships`, `sources`, and `artifact_bodies` — all columns except `created_at`/`updated_at`/`imported_at` — is identical, and the report shows `reclaimed origins: 0` with a stable skipped list. Widened from the round-1 four-table comparison per review round 2 finding 9.
5. **Inventory and effects are separate contracts with separate warning ownership.** Existing plan fields keep their file-inventory meanings and the existing plan `warnings` field keeps inventory I/O/parser warnings; `import_report.warnings` carries status and provenance outcomes. The `mode` discriminant (`simulation`/`inventory`) makes the dry-run response shape deterministic: `import_report` present exactly when simulation ran. Resolves review round 2 finding 8 and round 1 finding 10.
6. **Source status is defined per kind by the normative table in the Planning Contract.** Specs and tasks take `TASKS.json` index status over frontmatter (PR #130 precedence); reports derive `archived` from the archive directory as a real directory-derived source status and frontmatter otherwise; ideas and brainstorms default to `open`, their canonical base state — deliberately not fillable later, unlike `unknown`. Absent status and explicit `status: unknown` are both no-opinion. Resolves review round 2 finding 7.
7. **Spark derivation has target-refresh semantics, not full lifecycle ownership.** Delete-then-reinsert of a spark's imported relationships repairs changed promoted-to targets while the line remains a spark; removed or type-changed lines are explicitly deferred (Scope Out) rather than silently half-handled. Narrowed per review round 2 finding 10.
8. **The inventory fallback is honest, not apply-equivalent.** When no database or registered project exists there is nothing to collide with; the fallback's counters are fixed (archive sparks, surfaced read errors) and its output is labeled, and no parity claim is made for it. Preview never creates the database or project row; simulation requires both via read-only resolution (`LookupProjectIdentityForRoot` errors on unregistered — verified read-only).
9. **`--resume` remains an alias of `--apply`; JSON contract version stays 1.** Idempotent semantics make the distinction cosmetic; report fields are additive under the existing convention.

## Planning Contract

### Approach

The importer produces an `ImportReport` (reclaimed, skipped with mechanisms, divergences, outcome warnings) inside its own transaction, and `ApplyMarkdownMigration` stops calling preview — its result carries inventory counted during the same walk plus the in-transaction report, closing the TOCTOU window. `SimulateMarkdownMigration(ctx, root, resolver)` takes the Backup-primitive snapshot, then invokes the same apply entry point with a resolver pointed at the snapshot — simulation and apply are one code path by construction, differing only in which database file they address and in the result envelope (`applied: false`, `action: simulate`, `mode: simulation`, project identity populated from the snapshot's registration). The origin decision is a pure helper over (origin row, paired journal row) implementing the Decision 1 fingerprint; the status decision is a pure helper over (stored status, incoming normalized status) implementing Decision 2. The foreign-origin skip happens before `upsertJournalEntry` and covers the line's derived writes — sparks, spark aliases, and promoted-to relationships from a skipped line are skipped with it. All four `upsertX` statements drop `status = excluded.status`; the insert-only rule reads the existing row first so divergences are recorded with both sides.

### Per-kind status table (normative)

| Kind | Absent / explicit `unknown` on insert | Canonical or legacy on insert | Out-of-vocabulary on insert | Existing row on re-import |
|------|--------------------------------------|-------------------------------|-----------------------------|---------------------------|
| spec | `unknown` (fillable later) | normalized canonical | raw + `import_report` warning | fill iff stored `unknown`; else keep + divergence |
| task | `unknown` (fillable later) | normalized canonical | raw + warning | fill iff stored `unknown`; else keep + divergence |
| report | `unknown` (fillable later); archive-directory reports insert `archived` (directory-derived source status) | normalized canonical | raw + warning | fill iff stored `unknown`; else keep + divergence |
| idea / brainstorm | `open` (canonical base state; not fillable later) | normalized canonical | raw + warning | keep + divergence (stored value is never `unknown`) |
| spark | `open` (existing behavior, unchanged) | n/a | n/a | already insert-only |
| shaping draft | `draft` default, raw otherwise (no lifecycle vocabulary, no normalization, no warning) | n/a | n/a | fill iff stored `unknown`; else keep + divergence |

### Placement

`internal/state/markdown_import.go` (decision helpers, report, skip-before-upsert ordering, spark relationship refresh), `internal/state/markdown_migration.go` (inventory function, counter fixes, simulate dispatch and result envelopes), a snapshot helper wrapping `state.Backup` near `backup.go` (loaf-owned `0700` temp dir, `0600` files, pre-registered cleanup), `internal/cli/cli.go` (dry-run dispatch with the resolver passed in — today it resolves the database path only after preview (`cli.go:3565`) — plus human rendering and JSON passthrough). The doctor caller `inspectUnimportedLocalMarkdown` in `internal/state/status.go` keeps calling the cheap inventory function and must not trigger simulation. Tests in `internal/state/markdown_import_test.go`, `markdown_migration_test.go`, and CLI-level parity assertions in `internal/cli/cli_test.go`.

### Snapshot mechanics

The snapshot is produced by the existing `state.Backup` primitive — `VACUUM INTO` from the live store followed by its verification pass (integrity check, foreign-key check, journal-search parity, provenance integrity), which is already production-tested by `--backup` and already asserts the FTS/rowid coupling survives the copy. The snapshot and its `-wal`/`-shm` siblings live in a loaf-owned `0700` directory with `0600` files named `markdown-simulate-<pid>-<timestamp>.sqlite`; cleanup is registered before `VACUUM INTO` starts so creation failure, ENOSPC, or cancellation leaves no partial file, and teardown removes siblings. Deletion is best-effort under process crash; the naming pattern is the contract that lets a future sweep identify strays. Concurrent-writer behavior is the Backup primitive's existing busy/timeout semantics; simulation fails with a clear snapshot-creation error rather than retrying indefinitely.

### Risks

- Snapshot cost scales with the global database, not the project; accepted because dry-run is interactive and rare, `VACUUM INTO` compacts, and the failure mode (ENOSPC) is a clean pre-registered-cleanup abort.
- The fingerprint can false-negative (a genuine backfill row that gained evidence or drifted from its journal row is skipped, not reclaimed) — safe direction, visible in the skipped list rather than silent.
- Insert-only status changes semantics for anyone still treating Markdown as live status authority after first import; the divergence report is the mitigation — nothing silently changes, and the kept-vs-incoming pair is printed.
- Per-table conditional status writes are easy to get subtly wrong; the shared decision helper plus the normative table's per-kind tests guard against copy-paste drift.

### Sequencing

U1 defines the report type and semantics that U2 and U4 consume, so it lands first; U3 is independent of U1/U2 and can proceed in parallel; U5 grows alongside each unit rather than trailing; U6 is mechanical once the CLI surface settles.

## Implementation Units

- **U1 — Import decision semantics and report contract.** Fingerprint reclaim over (origin, journal) row pairs, unknown-is-never-authoritative status disposition per the normative table, divergence recording, spark-derived relationship delete-then-reinsert, and the `import_report` object computed in-transaction with apply no longer calling preview.
- **U2 — Snapshot simulation.** Snapshot helper wrapping `state.Backup` with hygiene contract (permissions, pre-registered cleanup, sibling teardown), `SimulateMarkdownMigration(ctx, root, resolver)` invoking the full apply pipeline against the snapshot, `mode` discriminant, inventory-fallback dispatch labeled inventory-only, and the inventory counter fixes.
- **U3 — Vocabulary normalization parity.** `CanonicalLifecycleStatus` at insert for tasks, reports, ideas, brainstorms per the normative table; raw-plus-warning for explicit out-of-vocabulary values; absent/explicit-`unknown` as no-opinion. Shaping drafts excluded from normalization (no lifecycle vocabulary); they get the status-disposition rule from U1 and nothing more.
- **U4 — CLI rendering.** Dry-run dispatch passing the resolver, human output for `mode`, `import_report`, and project identity on simulate (no more hardcoded `project: (not initialized)` when a registered project simulated), `--json` passthrough for dry-run, apply, and resume.
- **U5 — Regression fixtures and tests.** Gridsight-shaped fixture (0011-fingerprint rows, `dirty = 1` and copy-mismatched `unknown` rows, archived spec with terminal legacy source status, foreign-mechanism entry, future-envelope entry), simulate/apply parity, second-apply idempotence over the full table dump, per-kind status matrix from the normative table including `TASKS.json`/frontmatter conflicts, changed spark targets, live-file byte preservation (main + WAL, quiescent, `-shm` excluded), FTS-failure parity, snapshot-creation-failure hygiene, and dispatch routing (no DB, DB without project, behind-schema DB, checksum drift, explicit `StateHome`, `LOAF_DB`).
- **U6 — Docs.** CLI reference regeneration and the loaf-reference/skill passages describing `loaf migrate markdown` semantics, the `mode` discriminant, and the inventory-vs-simulation distinction.

## Verification Contract

<!-- Executable (machine-checkable): -->

- **V1.** `npm run test` (go test ./...) passes on the branch.
- **V2.** Named state tests pass: fingerprint-reclaim (clean backfill rows reclaimed; `manual`/`skill`/`hook`, custom mechanisms, envelope-v2 rows, `dirty = 1` rows, populated-`observed_harness_version` rows, and journal-copy-mismatched rows all skipped with rows byte-preserved), status disposition per the normative table for every kind (fill over stored `unknown`, archived-stays-archived, divergence reporting, out-of-vocabulary warning), and spark-target-change cleanup.
- **V3.** `loaf build` and `npm run typecheck` succeed; regenerated CLI reference committed if it changes.
- **V4.** Isolated smoke under `LOAF_DB=$(mktemp -d)/loaf.sqlite`: seed the gridsight-shaped fixture; `--dry-run --json` and `--apply --json` report equal `import_report` values with no intervening state change; a second `--apply` reports `reclaimed_origins: 0` and the logical dump of all import-touched tables (timestamps excluded) is identical before and after it.
- **V5.** Byte test under a quiescent fixture: sha256 of the live main database and `-wal` are identical before and after `--dry-run`; `-shm` is excluded as ephemeral; the snapshot and its siblings are gone afterward.
- **V6.** FTS-failure parity: with a corrupted `journal_search` table, simulate and apply fail with the same error; the live database is untouched by the failed simulate. Snapshot-creation failure (e.g. unwritable temp dir) reports its own error class and leaves no partial snapshot.
- **V7.** Dispatch routing: no database → inventory fallback (`mode: inventory`, no `import_report`); database without registered project → inventory fallback; behind-schema database → simulation fails with the same schema-preflight error apply gives; registered current-schema project → simulation; explicit `StateHome` and `LOAF_DB` both honored by the resolver passed from the CLI.

<!-- Human review: -->

- **H1.** A reviewer confirms no code path writes `migration` over any origin failing the Decision 1 fingerprint, that the fingerprint check reads both the origin and journal rows, and that skipped entries are never partially mutated (entry, origin, sparks, aliases, relationships all untouched).
- **H2.** No new CLI flags exist; `--resume` behavior is unchanged as an alias.
- **H3.** JSON output changes are additive only — every pre-existing field keeps its name, type, and inventory meaning; plan `warnings` keeps inventory warnings; effects and outcome warnings live in `import_report`.
- **H4.** The doctor diagnostic (`inspectUnimportedLocalMarkdown`) still calls only the cheap inventory function, and the snapshot helper's cleanup registration precedes snapshot creation in the code.

## Definition of Done

- All six units landed on `markdown-reimport-safety`, V1–V7 green, H1–H4 confirmed in review.
- `loaf change check` reports zero violations.
- PR squash-merged per convention, with the gridsight ops note (re-archive SPEC-007/008) and the doctor-check follow-up (archived-flip damage detection, orphaned-derivative GC, snapshot-residue sweep) filed where triage will see them.

## Durable Outputs

- An ADR in `docs/decisions/` recording the import authority model: information-free-above-journal-row origin reclaim, unknown-is-never-authoritative status disposition, and full-pipeline snapshot simulation as the permanent contract for one-way Markdown migration.
- Updated `loaf-reference` skill passage for `loaf migrate markdown` describing the report fields, the `mode` discriminant, the no-abort collision semantics, and the parity precondition.

## Open Questions

- [KU] Do any plugin or skill surfaces parse the migration JSON strictly enough to notice new fields? → owner: U4 (verify consumers during CLI wiring; Go's JSON decoding is tolerant, but rendered docs may enumerate fields).
- [KU] Does the Backup primitive's busy/timeout behavior need a tighter bound for interactive dry-run use? → owner: U2 (measure under the fixture; fail-fast with a clear error beats waiting).

## Source Inputs

- Journal findings `journal:453d67cf2bd7bf131a1b5f09` (origin-collision dry-run/apply asymmetry) and `journal:8038077b8a7e19e3abe77533` (lifecycle status clobber), triaged 2026-07-23 from the gridsight-core-gds session `loaf-migrate-markdown-workflow`.
- Empirical evidence from the global database: 2,841 `unknown`-mechanism origins in gridsight matching the 0011 backfill shape; SPEC-007/008 archived→complete flip with 2026-07-08 archive events intact.
- PR #130 (spec terminal-status vocabulary close) as the landed base this builds on; PR #127 and schema migration 0011 as the origin-vocabulary context.
- Shaping interview in this conversation (2026-07-23), decisions 1–3 as originally accepted.
- Adversarial review round 1 (Codex, gpt-5.6 xhigh, 2026-07-23): 10 findings. Round 2 (same thread, 2026-07-24): 10 findings — the fingerprint completed and re-argued as information-free-above-journal-row, the placeholder rule replaced by unknown-is-never-authoritative, the SHM byte guarantee dropped, simulation unified onto the apply pipeline with shared preflight, the snapshot method pinned to the Backup primitive, and the verification contract widened accordingly.
