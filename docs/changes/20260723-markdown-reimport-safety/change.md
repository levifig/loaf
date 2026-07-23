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

If import decisions become deterministic functions of database state (reclaim exactly the schema-0011 backfill origins, skip all other foreign provenance, never overwrite an existing lifecycle status) and the dry-run executes the full apply pipeline against a disposable snapshot of the database, then re-import becomes safe and business-state idempotent, dry-run and apply cannot disagree whenever there is database state to consult, and the class of silent lifecycle regressions is closed rather than patched. Where no database or project row exists, the dry-run is an honestly labeled file inventory — nothing exists to collide with, so the parity guarantee is vacuously preserved.

## Scope

**In**

- Origin-collision semantics in the importer: origins matching the schema-0011 backfill fingerprint (`capture_mechanism = 'unknown'`, `envelope_version = 1`, and every evidence field — `source_event`, `observed_harness`, `agent_id`, `head`, `change_path`, `change_sha256`, `durable_result_*` — NULL) are reclaimed as `migration` and re-imported; every other non-`migration` origin — `manual`, `skill`, `hook`, custom mechanisms, future envelope versions, and `unknown` rows carrying evidence — causes the entry to be skipped untouched, counted, and listed. Apply never aborts on a collision.
- Insert-only status: source status (normalized) is written when a row is first created, or over a structural placeholder — a spec row with `body_source_id IS NULL`, the only shape `ensureSpecPlaceholder` creates. Existing sourced statuses are never overwritten, including a sourced `unknown`; divergences are reported in both simulate and apply results.
- Lifecycle vocabulary normalization at insert for tasks, reports, ideas, and brainstorms via `CanonicalLifecycleStatus` (specs landed in PR #130); explicit out-of-vocabulary values are stored raw on first insert and surfaced as warnings; absent or explicitly-`unknown` status is no-opinion — no warning, no divergence.
- Dry-run as snapshot simulation: when the database file exists and the project resolves to a registered row (read-only lookup), `--dry-run` snapshots the database (`VACUUM INTO` a temp file from a read-only connection, or the online backup API — implementer's choice, gated by V5), runs the complete apply pipeline including the FTS rebuild against the snapshot, reports, and deletes the snapshot. The live database file, `-wal`, and `-shm` are untouched byte-for-byte. When no database or project row exists, the cheap file-inventory preview runs, explicitly labeled inventory-only.
- A new `import_report` object (reclaimed-origin count, skipped-entry list with mechanisms, status-divergence list, warnings) computed inside the importer's own transaction and carried on both simulate and apply results; existing plan fields keep their file-inventory meaning untouched.
- Apply no longer calls preview: `ApplyMarkdownMigration` produces inventory and `import_report` from its single committed transaction, eliminating the current preview-then-import TOCTOU window (`markdown_import.go:70`).
- Accepted session lines own their derived rows: spark-derived imported relationships are delete-then-reinserted per spark (parity with `deleteImportedRelationships` on artifacts), so a changed promoted-to target no longer strands the old edge.
- Inventory counter fixes: the cheap spark counter learns `sessions/archive/*.md` (parity with apply's file set), and counter read failures surface as warnings instead of being swallowed.
- Regression coverage shaped like gridsight: 0011-fingerprint backfill rows, an archived spec whose Markdown still says a terminal legacy status, foreign-mechanism and future-envelope entries, and a second-apply business-state idempotence check.

**Out** (deferred, not rejected)

- A doctor check that detects historical archived-flip damage in existing databases — follow-up candidate once these semantics land.
- Repairing gridsight's data in code: SPEC-007/008 re-archive is an operational step (`loaf spec archive` accepts them post-#130) noted for the gridsight session, not shipped here.
- An alias provenance model: import-created alias rows remain import-refreshable, and hand-curated aliases are not a supported surface today; protecting them needs provenance on `aliases`, which is its own Change.
- Any markdown-export or rollback-manifest changes beyond carrying the new report fields.

**Cut** (explicitly rejected)

- A `--skip-conflicts` flag: the deterministic reclaim/skip semantics make apply total, so there is no refusal left to skip and no flag to carry forever.
- Reclaiming any origin outside the 0011 backfill fingerprint — a deterministic entry ID proves which row the importer addresses, not who created it; everything with evidence or an unexpected shape is skipped, never rewritten.
- Timestamp- or merge-based status reconciliation between Markdown and SQLite — SQLite is the source of truth once a row exists; import is a one-way legacy migration, not sync.
- Byte-identical idempotence: `updated_at`/`imported_at` refresh on re-import by design; idempotence is defined over business state, not row bytes.

## Observable Workflow

On a pre-0011 project, `loaf migrate markdown --dry-run` reports the file inventory plus an import report: `reclaimed origins: N`, `skipped (foreign provenance): N` with mechanisms, `status divergences: N` naming each kept SQLite status, and the registered project identity — computed by really running the import against a throwaway snapshot. `loaf migrate markdown --apply` then reports the same numbers from its committed transaction — the gridsight scenario imports its 108 sessions instead of refusing. Archived specs stay archived through any number of re-imports. A second `--apply` reports `reclaimed origins: 0`, the same skipped list, and unchanged business state — statuses, mechanisms, aliases, and relationships are identical before and after. On a fresh machine with no database, `--dry-run` prints the inventory with an explicit note that conflict detection needs an initialized project. `--json` carries `import_report` on both modes; all pre-existing fields keep their meanings.

## Rabbit Holes and No-Gos

- Two-way Markdown↔SQLite sync: divergence reporting will tempt "why not write the DB status back to frontmatter" — import stays one-way, and export is a different surface.
- Redesigning the journal provenance model: this Change consumes `capture_mechanism` as-is; vocabulary or envelope changes belong to the origin-hygiene lineage (PR #127), not here.
- In-code repair of historical damage: detecting and fixing rows already flipped in other projects' databases is the deferred doctor follow-up, not import logic.
- Reworking `--resume`: with business-state idempotent apply it is behaviorally identical to `--apply`; it stays an accepted alias and its removal is somebody else's cleanup.
- Chasing byte-identical dry-runs: the snapshot approach makes live-file preservation testable; do not extend that ambition to WAL checkpointing behavior on the snapshot copy itself.

## Decisions

Provenance: decisions 1–3 were accepted by interview on 2026-07-23 and refined the same day by adversarial review (Codex, gpt-5.6 xhigh, 10 findings — see Source Inputs); 4–9 were made autonomously during shaping and review disposition.

1. **Reclaim is restricted to the schema-0011 backfill fingerprint; everything else non-`migration` is skipped.** The fingerprint (`unknown` + envelope v1 + all evidence fields NULL) is exactly what migration 0011 wrote, so reclaiming it as `migration` restores true provenance and can erase nothing — the fields the reclaim would overwrite are required to be empty for the row to match. All 2,841 gridsight rows match it (verified against the live database). Custom mechanisms, future envelope versions, and evidence-bearing `unknown` rows are facts the importer must not touch: their entries are skipped whole. Refines interview decision "reclaim unknown, skip foreign" per review finding 6; forecloses `--skip-conflicts` and wholesale-abort behavior.
2. **Insert-only status with structural placeholder fill.** Source status is written on first insert or over a spec row with `body_source_id IS NULL` — the one shape `ensureSpecPlaceholder` creates and the only placeholder in the system. A sourced `unknown` is authoritative and kept (reported as divergence when the incoming status is real). Refines the interview decision per review finding 7; forecloses status_changed events for import (import never changes an existing sourced status) and any frontmatter-wins semantics.
3. **Dry-run simulates the full apply against a disposable snapshot, not the live database.** A rolled-back transaction on the live store cannot be side-effect-free (WAL/`-shm` growth, writer lock, checkpoint-on-close — review finding 1) and skipping the FTS rebuild would reintroduce divergence (finding 2). The snapshot runs the entire pipeline including `rebuildAndVerifyJournalSearch`, so an FTS corruption fails preview and apply identically, and live-file preservation becomes byte-testable (V5). Replaces the interview decision's rolled-back-transaction mechanism; the user-visible promise (dry-run tells the truth) is unchanged.
4. **Idempotence means equivalent business state.** After a second apply: statuses, origin mechanisms, aliases, and relationships are identical; the report shows `reclaimed origins: 0` and a stable skipped list. Row timestamps (`updated_at`, `imported_at`) and artifact-body/FTS rewrites are excluded by design and documented. Resolves review finding 5's contradiction — the two reports are not byte-identical, and were wrong to be promised as such.
5. **Inventory and effects are separate contracts.** Existing plan fields (`specs`, `sessions`, `relationships`, …) keep their file-inventory meanings — `relationships` remains a syntactic declaration count. The new `import_report` object carries effects. H3 (additive-only JSON) holds because nothing existing changes meaning. Resolves review finding 10.
6. **Source status is defined per kind, and "no opinion" is precise.** Specs and tasks take `TASKS.json` index status over frontmatter (PR #130 precedence); reports derive `archived` from the archive directory and frontmatter otherwise. Absent status and explicit `status: unknown` are both no-opinion: never a warning, never a divergence. Only explicit out-of-vocabulary spellings warn. Resolves review finding 9.
7. **Accepted session lines own their derived rows.** Spark-derived imported relationships get delete-then-reinsert per spark, matching artifact imports, so changed promoted-to targets clean up after themselves (review finding 8). Alias rows stay import-refreshable — protecting hand-curated aliases needs alias provenance, deferred to its own Change and named in Scope Out.
8. **The inventory fallback is honest, not apply-equivalent.** When no database or project row exists there is nothing to collide with, so the fallback's blindness is harmless for this defect class — but it is labeled inventory-only, its spark counter gains archive parity, and its swallowed read errors become warnings (review finding 3). Preview never creates the database or project row; simulation requires both to already exist via read-only resolution (`LookupProjectIdentityForRoot` errors on unregistered — verified read-only).
9. **`--resume` remains an alias of `--apply`; JSON contract version stays 1.** Idempotent semantics make the distinction cosmetic; report fields are additive under the existing convention.

## Planning Contract

### Approach

The importer produces an `ImportReport` (reclaimed, skipped with mechanisms, divergences, warnings) inside its own transaction, and `ApplyMarkdownMigration` stops calling preview — its result carries inventory counted during the same walk plus the in-transaction report, closing the TOCTOU window (review finding 4). A new `SimulateMarkdownMigration(ctx, root, resolver)` snapshots the database, opens the snapshot as a normal store, runs the identical import-plus-FTS pipeline, extracts the same result shape (`applied: false`, `action: simulate`, project identity populated), and deletes the snapshot under `defer`. `PreviewMarkdownMigration` survives as the cheap inventory function for the doctor diagnostic and the no-database fallback. The origin decision is a pure helper over the full origins row (fingerprint → reclaim; `migration` → refresh; anything else → skip), and the status decision is a pure helper over (existing row shape, incoming normalized status). The foreign-origin skip happens before `upsertJournalEntry` and covers the line's derived writes — sparks, spark aliases, and promoted-to relationships from a skipped line are skipped with it. All four `upsertX` statements drop `status = excluded.status`; the insert-only rule reads the existing row first so divergences are recorded with both sides.

### Placement

`internal/state/markdown_import.go` (decision helpers, report, skip-before-upsert ordering, spark relationship refresh), `internal/state/markdown_migration.go` (inventory function, counter fixes, simulate dispatch and result envelopes), a small snapshot helper in `internal/state` near `store.go`, `internal/cli/cli.go` (dry-run dispatch with the resolver passed in — today it resolves the database path only after preview (`cli.go:3565`) — plus human rendering and JSON passthrough). The doctor caller `inspectUnimportedLocalMarkdown` in `internal/state/status.go` keeps calling the cheap inventory function and must not trigger simulation. Tests in `internal/state/markdown_import_test.go`, `markdown_migration_test.go`, and CLI-level parity assertions in `internal/cli/cli_test.go`.

### Snapshot mechanics

The snapshot is taken from a read-only source connection (`VACUUM INTO` a temp path, or the serialized backup API), which cannot mutate the live file and captures WAL content consistently. The snapshot lives in the OS temp dir, is opened with the normal `OpenStore` path (WAL mode on the copy is fine — it is disposable), and is removed with its `-wal`/`-shm` siblings when simulation ends, success or failure. Live-file preservation is proven by V5's byte comparison, not assumed.

### Risks

- Snapshot cost scales with the global database, not the project (V5 fixture keeps this visible); accepted because dry-run is interactive and rare, and `VACUUM INTO` compacts.
- The fingerprint can false-negative (a legitimate backfill row that somehow gained evidence would be skipped, not reclaimed) — safe direction, reported in the skipped list rather than silent.
- Insert-only status changes semantics for anyone still treating Markdown as live status authority after first import; the divergence report is the mitigation — nothing silently changes, and the kept-vs-incoming pair is printed.
- Per-table conditional status writes are easy to get subtly wrong; the shared decision helper plus per-kind tests guard against copy-paste drift.

### Sequencing

U1 defines the report type and semantics that U2 and U4 consume, so it lands first; U3 is independent of U1/U2 and can proceed in parallel; U5 grows alongside each unit rather than trailing; U6 is mechanical once the CLI surface settles.

## Implementation Units

- **U1 — Import decision semantics and report contract.** Fingerprint reclaim / skip disposition over full origin rows, insert-only status with structural placeholder fill, divergence recording, spark-derived relationship delete-then-reinsert, and the `import_report` object computed in-transaction with apply no longer calling preview.
- **U2 — Snapshot simulation.** `SimulateMarkdownMigration(ctx, root, resolver)` with snapshot create/dispose, full pipeline including FTS rebuild, inventory-fallback dispatch labeled inventory-only, and the inventory counter fixes (archive sparks, error warnings).
- **U3 — Vocabulary normalization parity.** `CanonicalLifecycleStatus` at insert for tasks, reports, ideas, brainstorms; raw-plus-warning for explicit out-of-vocabulary values; absent/explicit-`unknown` as no-opinion. Shaping drafts are excluded — no lifecycle vocabulary exists for that kind; they get insert-only status from U1 and nothing more.
- **U4 — CLI rendering.** Dry-run dispatch passing the resolver, human output for `import_report` and project identity on simulate (no more hardcoded `project: (not initialized)` when a registered project simulated), `--json` passthrough for dry-run, apply, and resume.
- **U5 — Regression fixtures and tests.** Gridsight-shaped fixture (0011-fingerprint rows, archived spec with terminal legacy source status, foreign-mechanism entry, future-envelope `unknown` entry), simulate/apply parity, second-apply business-state idempotence snapshot, per-kind status-disposition matrix including `TASKS.json`/frontmatter conflicts, changed spark targets on re-import, live-file byte preservation, FTS-failure parity, and path routing (no DB, DB without project, explicit `StateHome`, `LOAF_DB`).
- **U6 — Docs.** CLI reference regeneration and the loaf-reference/skill passages describing `loaf migrate markdown` semantics and the inventory-vs-simulation distinction.

## Verification Contract

<!-- Executable (machine-checkable): -->

- **V1.** `npm run test` (go test ./...) passes on the branch.
- **V2.** Named state tests pass: fingerprint-reclaim (backfill rows reclaimed; `manual`/`skill`/`hook`, custom mechanisms, envelope-v2 `unknown`, and evidence-bearing `unknown` all skipped with rows byte-preserved), insert-only status (structural placeholder filled, sourced `unknown` kept and divergence-reported, archived-stays-archived), spark-target-change cleanup, and the per-kind status matrix (missing, explicit `unknown`, out-of-vocabulary, `TASKS.json`-vs-frontmatter conflict).
- **V3.** `loaf build` and `npm run typecheck` succeed; regenerated CLI reference committed if it changes.
- **V4.** Isolated smoke under `LOAF_DB=$(mktemp -d)/loaf.sqlite`: seed the gridsight-shaped fixture; `--dry-run --json` and `--apply --json` report equal `import_report` values; a second `--apply` reports `reclaimed_origins: 0` with an identical business-state snapshot (statuses, mechanisms, aliases, relationships dumped and compared, timestamps excluded).
- **V5.** Byte test: sha256 and existence of the live database, `-wal`, and `-shm` are identical before and after `--dry-run`; the snapshot and its siblings are gone afterward.
- **V6.** FTS-failure parity: with a corrupted `journal_search` table, simulate and apply fail with the same error; the live database is untouched by the failed simulate.
- **V7.** Dispatch routing: no database → inventory fallback (labeled); database without registered project → inventory fallback; registered project → simulation; explicit `StateHome` and `LOAF_DB` both honored by the resolver passed from the CLI.

<!-- Human review: -->

- **H1.** A reviewer confirms no code path writes `migration` over any origin outside the 0011 fingerprint, and that skipped entries are never partially mutated (entry, origin, sparks, aliases, relationships all untouched).
- **H2.** No new CLI flags exist; `--resume` behavior is unchanged as an alias.
- **H3.** JSON output changes are additive only — every pre-existing field keeps its name, type, and inventory meaning; effects live exclusively in `import_report`.
- **H4.** The doctor diagnostic (`inspectUnimportedLocalMarkdown`) still calls only the cheap inventory function.

## Definition of Done

- All six units landed on `markdown-reimport-safety`, V1–V7 green, H1–H4 confirmed in review.
- `loaf change check` reports zero violations.
- PR squash-merged per convention, with the gridsight ops note (re-archive SPEC-007/008) and the doctor-check follow-up filed where triage will see them.

## Durable Outputs

- An ADR in `docs/decisions/` recording the import authority model: fingerprint-bounded origin reclaim, insert-only status, and snapshot simulation as the permanent contract for one-way Markdown migration.
- Updated `loaf-reference` skill passage for `loaf migrate markdown` describing the report fields, the no-abort semantics, and the inventory-vs-simulation distinction.

## Open Questions

- [KU] Do any plugin or skill surfaces parse the migration JSON strictly enough to notice new fields? → owner: U4 (verify consumers during CLI wiring; Go's JSON decoding is tolerant, but rendered docs may enumerate fields).
- [KU] Snapshot temp placement and free-space handling when the global database is large → owner: U2 (default OS temp dir; fail with a clear error naming the required space rather than a partial snapshot).

## Source Inputs

- Journal findings `journal:453d67cf2bd7bf131a1b5f09` (origin-collision dry-run/apply asymmetry) and `journal:8038077b8a7e19e3abe77533` (lifecycle status clobber), triaged 2026-07-23 from the gridsight-core-gds session `loaf-migrate-markdown-workflow`.
- Empirical evidence from the global database: 2,841 `unknown`-mechanism origins in gridsight all matching the 0011 backfill fingerprint; SPEC-007/008 archived→complete flip with 2026-07-08 archive events intact.
- PR #130 (spec terminal-status vocabulary close) as the landed base this builds on; PR #127 and schema migration 0011 as the origin-vocabulary context.
- Shaping interview in this conversation (2026-07-23), decisions 1–3 as originally accepted.
- Adversarial review by Codex (gpt-5.6 xhigh, 2026-07-23): 10 findings, all dispositioned in Decisions 1–8 and the expanded Verification Contract; findings 1, 2, 4, 5, 6, and 7 changed the contract materially.
