package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/levifig/loaf/internal/project"
)

const (
	// JournalFirstMigrationActionDryRun previews the migration against a copy.
	JournalFirstMigrationActionDryRun = "dry-run"
	// JournalFirstMigrationActionApply runs the migration against the live database.
	JournalFirstMigrationActionApply = "apply"
)

// JournalFirstMigrationResult reports the outcome of a journal-first (SPEC-056)
// migration run. Counts are measured across the whole global database because
// the migration is schema-wide and touches every project's rows.
type JournalFirstMigrationResult struct {
	ContractVersion    int    `json:"contract_version"`
	DatabaseScope      string `json:"database_scope"`
	DatabasePath       string `json:"database_path"`
	ProjectID          string `json:"project_id,omitempty"`
	ProjectName        string `json:"project_name,omitempty"`
	ProjectCurrentPath string `json:"project_current_path,omitempty"`
	Action             string `json:"action"`
	Applied            bool   `json:"applied"`
	CopyRun            bool   `json:"copy_run"`
	BackupPath         string `json:"backup_path,omitempty"`
	SchemaVersion      int    `json:"schema_version"`

	JournalEntriesBefore   int `json:"journal_entries_before"`
	JournalEntriesAfter    int `json:"journal_entries_after"`
	NoiseEntriesPurged     int `json:"noise_entries_purged"`
	HarnessSessionBackfill int `json:"harness_session_ids_backfilled"`
	HandoffsRetargeted     int `json:"handoffs_retargeted"`
	SessionEventsDeleted   int `json:"session_events_deleted"`
	SessionAliasesDeleted  int `json:"session_aliases_deleted"`
	SessionsDropped        int `json:"sessions_dropped"`
	SnapshotsDropped       int `json:"snapshots_dropped"`
	JournalSearchRows      int `json:"journal_search_rows"`
}

// PreviewJournalFirstMigration runs the journal-first migration against a
// temporary copy of the live database and reports row counts without mutating
// live state.
func PreviewJournalFirstMigration(ctx context.Context, root project.Root, resolver PathResolver) (JournalFirstMigrationResult, error) {
	status, err := requireJournalFirstMigrationStatus(root, resolver)
	if err != nil {
		return JournalFirstMigrationResult{}, err
	}
	source, err := OpenStoreReadOnly(status.DatabasePath)
	if err != nil {
		return JournalFirstMigrationResult{}, err
	}
	defer source.Close()

	tempDir, err := os.MkdirTemp("", "loaf-journal-first-migration-*")
	if err != nil {
		return JournalFirstMigrationResult{}, fmt.Errorf("create journal-first migration temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)
	copyPath := filepath.Join(tempDir, "state.sqlite")
	if err := copySQLiteDatabase(ctx, source, copyPath, 0o600); err != nil {
		return JournalFirstMigrationResult{}, err
	}
	copyStore, err := OpenStore(copyPath)
	if err != nil {
		return JournalFirstMigrationResult{}, err
	}
	defer copyStore.Close()

	result := journalFirstMigrationBaseResult(status, JournalFirstMigrationActionDryRun)
	if err := runJournalFirstMigration(ctx, copyStore, &result); err != nil {
		return JournalFirstMigrationResult{}, err
	}
	result.CopyRun = true
	return result, nil
}

// ApplyJournalFirstMigration takes a mandatory pre-migration backup and then
// applies the journal-first migration to the live database.
func ApplyJournalFirstMigration(ctx context.Context, root project.Root, resolver PathResolver) (JournalFirstMigrationResult, error) {
	status, err := requireJournalFirstMigrationStatus(root, resolver)
	if err != nil {
		return JournalFirstMigrationResult{}, err
	}
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		return JournalFirstMigrationResult{}, err
	}
	store, err := openInitializedStore(root, resolver)
	if err != nil {
		return JournalFirstMigrationResult{}, err
	}
	defer store.Close()

	result := journalFirstMigrationBaseResult(status, JournalFirstMigrationActionApply)
	result.BackupPath = backup.BackupPath
	if err := runJournalFirstMigration(ctx, store, &result); err != nil {
		return JournalFirstMigrationResult{}, err
	}
	result.Applied = true
	return result, nil
}

func journalFirstMigrationBaseResult(status Status, action string) JournalFirstMigrationResult {
	return JournalFirstMigrationResult{
		ContractVersion:    StateJSONContractVersion,
		DatabaseScope:      "global",
		DatabasePath:       status.DatabasePath,
		ProjectID:          status.ProjectID,
		ProjectName:        status.ProjectName,
		ProjectCurrentPath: status.ProjectCurrentPath,
		Action:             action,
	}
}

func requireJournalFirstMigrationStatus(root project.Root, resolver PathResolver) (Status, error) {
	status, err := Inspect(root, resolver)
	if err != nil {
		return Status{}, err
	}
	switch status.Mode {
	case ModeSQLiteReady:
		return status, nil
	case ModeMarkdownOnly:
		return Status{}, fmt.Errorf("SQLite state database is not initialized; run `loaf state init` first")
	case ModeInvalid:
		return Status{}, fmt.Errorf("state database is invalid; run `loaf state doctor`")
	default:
		return Status{}, fmt.Errorf("state database is not ready; run `loaf state status`")
	}
}

// runJournalFirstMigration measures pre-migration counts, applies migration 10
// through the shared migration runner (recording it in schema_migrations), and
// measures the post-migration shape.
func runJournalFirstMigration(ctx context.Context, store *Store, result *JournalFirstMigrationResult) error {
	before, err := measureJournalFirstBefore(ctx, store)
	if err != nil {
		return err
	}
	result.JournalEntriesBefore = before.journalEntries
	result.NoiseEntriesPurged = before.noiseEntries
	result.HarnessSessionBackfill = before.backfillable
	result.HandoffsRetargeted = before.handoffs
	result.SessionEventsDeleted = before.sessionEvents
	result.SessionAliasesDeleted = before.sessionAliases
	result.SessionsDropped = before.sessions
	result.SnapshotsDropped = before.snapshots

	// Apply migrations 1-9 (idempotent, checksum-skipped) plus migration 10.
	// The shared runner records version 10 in schema_migrations.
	migrations := append(SchemaMigrations(), JournalFirstMigration())
	if err := ApplyMigrations(ctx, store.db, migrations); err != nil {
		return fmt.Errorf("apply journal-first migration: %w", err)
	}

	after, err := measureJournalFirstAfter(ctx, store)
	if err != nil {
		return err
	}
	result.JournalEntriesAfter = after.journalEntries
	result.JournalSearchRows = after.journalSearch
	result.SchemaVersion = after.schemaVersion
	return nil
}

type journalFirstBeforeCounts struct {
	journalEntries int
	noiseEntries   int
	backfillable   int
	handoffs       int
	sessionEvents  int
	sessionAliases int
	sessions       int
	snapshots      int
}

func measureJournalFirstBefore(ctx context.Context, store *Store) (journalFirstBeforeCounts, error) {
	var counts journalFirstBeforeCounts

	// The pre-migration counts query legacy tables and columns (sessions,
	// session_state_snapshots, journal_entries.session_id) that migration 10
	// drops. On an already-migrated database those are gone, so there is
	// nothing left to transform: report all-zero counts for a clean no-op
	// re-run instead of failing on the missing tables.
	sessionsExists, err := sqliteTableExists(ctx, store.db, "sessions")
	if err != nil {
		return journalFirstBeforeCounts{}, fmt.Errorf("measure journal-first pre-migration counts: %w", err)
	}
	if !sessionsExists {
		if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM journal_entries`).Scan(&counts.journalEntries); err != nil {
			return journalFirstBeforeCounts{}, fmt.Errorf("measure journal-first pre-migration counts: %w", err)
		}
		return counts, nil
	}

	scans := []struct {
		dst *int
		sql string
	}{
		{&counts.journalEntries, `SELECT COUNT(*) FROM journal_entries`},
		{&counts.noiseEntries, `SELECT COUNT(*) FROM journal_entries WHERE entry_type = 'session'`},
		{&counts.backfillable, `
SELECT COUNT(*) FROM journal_entries j
WHERE j.harness_session_id IS NULL
  AND j.session_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM sessions s WHERE s.id = j.session_id AND s.harness_session_id IS NOT NULL)`},
		{&counts.handoffs, `SELECT COUNT(*) FROM handoffs`},
		{&counts.sessionEvents, `SELECT COUNT(*) FROM events WHERE entity_kind = 'session'`},
		{&counts.sessionAliases, `SELECT COUNT(*) FROM aliases WHERE entity_kind = 'session'`},
		{&counts.sessions, `SELECT COUNT(*) FROM sessions`},
		{&counts.snapshots, `SELECT COUNT(*) FROM session_state_snapshots`},
	}
	for _, scan := range scans {
		if err := store.db.QueryRowContext(ctx, scan.sql).Scan(scan.dst); err != nil {
			return journalFirstBeforeCounts{}, fmt.Errorf("measure journal-first pre-migration counts: %w", err)
		}
	}
	return counts, nil
}

type journalFirstAfterCounts struct {
	journalEntries int
	journalSearch  int
	schemaVersion  int
}

func measureJournalFirstAfter(ctx context.Context, store *Store) (journalFirstAfterCounts, error) {
	var counts journalFirstAfterCounts
	scans := []struct {
		dst *int
		sql string
	}{
		{&counts.journalEntries, `SELECT COUNT(*) FROM journal_entries`},
		{&counts.journalSearch, `SELECT COUNT(*) FROM journal_search`},
	}
	for _, scan := range scans {
		if err := store.db.QueryRowContext(ctx, scan.sql).Scan(scan.dst); err != nil {
			return journalFirstAfterCounts{}, fmt.Errorf("measure journal-first post-migration counts: %w", err)
		}
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil {
		return journalFirstAfterCounts{}, err
	}
	counts.schemaVersion = version
	return counts, nil
}
