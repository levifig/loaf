# Loaf Operational State Schema

This packet preserves SQL mirrors and selected diagrams for the shipped SQLite compatibility runtime. The executable migrations in `internal/state/migrations/` remain the source of truth; this is not the destination private-continuity schema.

## SQL mirrors

The 27 SQL files mirror the auto-applied migrations returned by `SchemaMigrations()`: `0001`–`0009` and `0011`–`0028`. [The schema documentation test](../../internal/state/schema_test.go) checks each mirror against its executable migration; migration `0020` uses normalized SQL comparison.

Migration `0010_journal_first.sql` is an explicit, out-of-band transformation. Its source lives at `internal/state/migrations/0010_journal_first.sql`; it has no mirror here and is excluded from `SchemaMigrations()` so opening a store never applies it automatically. It runs through the backup-first `loaf state migrate journal-first --apply` operation.

## Selected schema diagrams

- [operational-state.dbml](operational-state.dbml) provides a relational view.
- [operational-state.mmd](operational-state.mmd) provides a Mermaid ER view.

These are selected compatibility views, not exhaustive snapshots of an executed database. `TestSchemaDocumentationMirrorsExecutableMigration` checks the tables and columns selected by `requiredInitialTables`, except the issue, work-contract, and release tables listed in `issueFoundationTables`, plus an explicit set of Mermaid relationship labels. The expected columns come from migration declarations with gone-table declarations removed; the check does not replay every SQL alteration into a complete diagram.

The diagrams include later continuity, synchronization, and gateway tables while retaining the pre-`0010` session structures. Their headers explain the journal-first transformation separately. Session tables shown for migration compatibility do not establish a current session lifecycle.

Migration 0011 deliberately omits foreign keys from `journal_origins` and `journal_deferrals` to `journal_entries` and `sparks`: the explicit 0010 transformation may rebuild `journal_entries` after 0011 has applied, so optional provenance is audited by the owning workflows instead of creating a migration-order dependency.

When changing the compatibility schema, update the executable migrations and their checked mirrors together. Before release, every new schema migration must include data-preservation tests and defined operator steps for upgrade and repair. [Private Continuity](../architecture/private-continuity.md) owns the current model, recovery guarantees, and pending cutover boundary.
