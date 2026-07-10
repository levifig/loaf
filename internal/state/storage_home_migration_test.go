package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levifig/loaf/internal/project"
)

func TestPreviewStorageHomeMigrationPlansLegacyCopy(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	legacyPath := initializeLegacyStateDatabase(t, root, PathResolver{})

	plan, err := PreviewStorageHomeMigration(root, PathResolver{})
	if err != nil {
		t.Fatalf("PreviewStorageHomeMigration() error = %v", err)
	}

	if plan.ContractVersion != StateJSONContractVersion {
		t.Fatalf("ContractVersion = %d, want %d", plan.ContractVersion, StateJSONContractVersion)
	}
	if plan.DatabaseScope != "global" {
		t.Fatalf("DatabaseScope = %q, want global", plan.DatabaseScope)
	}
	if plan.MigrationScope != "project" {
		t.Fatalf("MigrationScope = %q, want project", plan.MigrationScope)
	}
	if plan.ProjectID != "" {
		t.Fatalf("ProjectID = %q, want empty before apply", plan.ProjectID)
	}
	if plan.Action != StorageHomeActionCopy {
		t.Fatalf("Action = %q, want %q", plan.Action, StorageHomeActionCopy)
	}
	if plan.Applied {
		t.Fatal("Applied = true, want false for dry-run")
	}
	if plan.DatabaseExists {
		t.Fatal("DatabaseExists = true, want false before apply")
	}
	if !plan.LegacyDatabaseExists {
		t.Fatal("LegacyDatabaseExists = false, want true")
	}
	if plan.LegacyDatabasePath != legacyPath {
		t.Fatalf("LegacyDatabasePath = %q, want %q", plan.LegacyDatabasePath, legacyPath)
	}
	if !strings.HasPrefix(plan.DatabasePath, dataHome) {
		t.Fatalf("DatabasePath = %q, want under data home %q", plan.DatabasePath, dataHome)
	}
}

func TestApplyStorageHomeMigrationCopiesLegacyDatabaseWithoutDeletingIt(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	legacyPath := initializeLegacyStateDatabase(t, root, PathResolver{})

	plan, err := ApplyStorageHomeMigration(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}

	if plan.ContractVersion != StateJSONContractVersion {
		t.Fatalf("ContractVersion = %d, want %d", plan.ContractVersion, StateJSONContractVersion)
	}
	if plan.DatabaseScope != "global" {
		t.Fatalf("DatabaseScope = %q, want global", plan.DatabaseScope)
	}
	if plan.MigrationScope != "project" {
		t.Fatalf("MigrationScope = %q, want project", plan.MigrationScope)
	}
	if plan.ProjectID == "" {
		t.Fatal("ProjectID is empty after apply")
	}
	if plan.ProjectName != filepath.Base(root.Path()) {
		t.Fatalf("ProjectName = %q, want %q", plan.ProjectName, filepath.Base(root.Path()))
	}
	if plan.ProjectCurrentPath != root.Path() {
		t.Fatalf("ProjectCurrentPath = %q, want %q", plan.ProjectCurrentPath, root.Path())
	}
	if !plan.Applied {
		t.Fatal("Applied = false, want true")
	}
	if plan.Action != StorageHomeActionAlreadyMigrated {
		t.Fatalf("Action = %q, want %q", plan.Action, StorageHomeActionAlreadyMigrated)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy database stat error = %v, want legacy file preserved", err)
	}
	if _, err := os.Stat(plan.DatabasePath); err != nil {
		t.Fatalf("data-home database stat error = %v", err)
	}

	status, err := Inspect(root, PathResolver{})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q after migration", status.Mode, ModeSQLiteReady)
	}
	if status.DatabasePath != plan.DatabasePath {
		t.Fatalf("DatabasePath = %q, want %q", status.DatabasePath, plan.DatabasePath)
	}

	second, err := ApplyStorageHomeMigration(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("second ApplyStorageHomeMigration() error = %v", err)
	}
	if second.Applied {
		t.Fatal("second Applied = true, want idempotent no-op")
	}
	if second.Action != StorageHomeActionAlreadyMigrated {
		t.Fatalf("second Action = %q, want %q", second.Action, StorageHomeActionAlreadyMigrated)
	}
	if second.DatabaseScope != "global" || second.MigrationScope != "project" {
		t.Fatalf("second scopes = %q/%q, want global/project", second.DatabaseScope, second.MigrationScope)
	}
}

func TestStorageHomeMigrationUsesProjectDataDatabaseBeforeStateHome(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	projectPath := initializeProjectDataDatabase(t, root, PathResolver{})
	initializeLegacyStateDatabase(t, root, PathResolver{})

	plan, err := PreviewStorageHomeMigration(root, PathResolver{})
	if err != nil {
		t.Fatalf("PreviewStorageHomeMigration() error = %v", err)
	}
	if plan.Action != StorageHomeActionCopy {
		t.Fatalf("Action = %q, want %q", plan.Action, StorageHomeActionCopy)
	}
	if plan.DatabaseScope != "global" || plan.MigrationScope != "project" {
		t.Fatalf("scopes = %q/%q, want global/project", plan.DatabaseScope, plan.MigrationScope)
	}
	if plan.LegacyDatabasePath != projectPath {
		t.Fatalf("LegacyDatabasePath = %q, want project data path %q", plan.LegacyDatabasePath, projectPath)
	}

	applied, err := ApplyStorageHomeMigration(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}
	if !applied.Applied {
		t.Fatal("Applied = false, want true")
	}
	if applied.ProjectID == "" {
		t.Fatal("ProjectID is empty after project database migration")
	}
	if applied.ProjectCurrentPath != root.Path() {
		t.Fatalf("ProjectCurrentPath = %q, want %q", applied.ProjectCurrentPath, root.Path())
	}
	if applied.DatabasePath == projectPath {
		t.Fatalf("DatabasePath = %q, want global path distinct from project path", applied.DatabasePath)
	}
	status, err := Inspect(root, PathResolver{})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeSQLiteReady)
	}
}

func TestApplyStorageHomeMigrationUpgradesCopiedLegacySchema(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	resolver := PathResolver{}
	legacyPath, err := resolver.LegacyDatabasePath(root)
	if err != nil {
		t.Fatalf("LegacyDatabasePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("create legacy database dir error = %v", err)
	}
	legacyStore, err := OpenStore(legacyPath)
	if err != nil {
		t.Fatalf("OpenStore(legacy) error = %v", err)
	}
	if err := ApplyMigrations(ctx, legacyStore.db, SchemaMigrations()[:1]); err != nil {
		t.Fatalf("apply legacy schema migration error = %v", err)
	}
	if err := legacyStore.UpsertProject(ctx, root); err != nil {
		t.Fatalf("UpsertProject(legacy) error = %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("close legacy store error = %v", err)
	}

	plan, err := ApplyStorageHomeMigration(ctx, root, resolver)
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}
	if !plan.Applied {
		t.Fatal("Applied = false, want true")
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy database stat error = %v, want legacy file preserved", err)
	}

	status, err := Inspect(root, resolver)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeSQLiteReady)
	}
	if status.SchemaVersion != CurrentSchemaVersion() {
		t.Fatalf("SchemaVersion = %d, want %d", status.SchemaVersion, CurrentSchemaVersion())
	}
}

func TestApplyStorageHomeMigrationDoesNotOverwriteExistingDataHomeDatabase(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	initializeLegacyStateDatabase(t, root, PathResolver{})
	dataStatus, err := Initialize(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("Initialize(data) error = %v", err)
	}

	plan, err := ApplyStorageHomeMigration(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}

	if plan.Action != StorageHomeActionAlreadyMigrated {
		t.Fatalf("Action = %q, want %q", plan.Action, StorageHomeActionAlreadyMigrated)
	}
	if plan.DatabasePath != dataStatus.DatabasePath {
		t.Fatalf("DatabasePath = %q, want existing data path %q", plan.DatabasePath, dataStatus.DatabasePath)
	}
	if len(plan.Warnings) == 0 {
		t.Fatal("Warnings empty, want legacy-leftover warning")
	}
}

func TestApplyStorageHomeMigrationIncludesPendingWALFrames(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	legacyPath := initializeLegacyStateDatabase(t, root, PathResolver{})
	legacyStore, err := OpenStore(legacyPath)
	if err != nil {
		t.Fatalf("OpenStore(legacy) error = %v", err)
	}
	defer legacyStore.Close()

	if _, err := legacyStore.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatalf("wal checkpoint error = %v", err)
	}
	wantProjectID := "pending-wal-project"
	wantProjectPath := filepath.Join(root.Path(), "pending-wal-project")
	if _, err := legacyStore.db.ExecContext(ctx, `
INSERT INTO projects (id, identity_hash, friendly_name, current_path, last_seen_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, wantProjectID, wantProjectID, "Pending WAL Project", wantProjectPath, "2026-06-12T00:00:00Z", "2026-06-12T00:00:00Z", "2026-06-12T00:00:00Z"); err != nil {
		t.Fatalf("insert WAL-backed project error = %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `
INSERT INTO project_paths (id, project_id, path, is_current, first_seen_at, last_seen_at, created_at, updated_at)
VALUES (?, ?, ?, 1, ?, ?, ?, ?)
`, "pending-wal-project-path", wantProjectID, wantProjectPath, "2026-06-12T00:00:00Z", "2026-06-12T00:00:00Z", "2026-06-12T00:00:00Z", "2026-06-12T00:00:00Z"); err != nil {
		t.Fatalf("insert WAL-backed project path error = %v", err)
	}
	if info, err := os.Stat(legacyPath + "-wal"); err != nil {
		t.Fatalf("legacy WAL stat error = %v", err)
	} else if info.Size() == 0 {
		t.Fatal("legacy WAL is empty, want pending frames before migration")
	}

	plan, err := ApplyStorageHomeMigration(ctx, root, PathResolver{})
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}

	dataStore, err := OpenStore(plan.DatabasePath)
	if err != nil {
		t.Fatalf("OpenStore(data) error = %v", err)
	}
	defer dataStore.Close()
	var count int
	if err := dataStore.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id = ?`, wantProjectID).Scan(&count); err != nil {
		t.Fatalf("query migrated project error = %v", err)
	}
	if count != 1 {
		t.Fatalf("migrated project count = %d, want 1", count)
	}
}

func TestApplyStorageHomeMigrationCopyRebuildsExactJournalSearchWithoutLegacyContamination(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	legacyPath := initializeLegacyStateDatabase(t, root, PathResolver{})
	legacyStore, err := OpenStore(legacyPath)
	if err != nil {
		t.Fatalf("OpenStore(legacy) error = %v", err)
	}
	for _, message := range []string{"legacy-one", "legacy-two", "legacy-three"} {
		if _, err := legacyStore.LogJournal(ctx, root, JournalLogOptions{Entry: "decision(storage): " + message}); err != nil {
			legacyStore.Close()
			t.Fatalf("LogJournal(%s) error = %v", message, err)
		}
	}
	var firstID, secondID string
	if err := legacyStore.db.QueryRowContext(ctx, `SELECT id FROM journal_entries ORDER BY rowid LIMIT 1`).Scan(&firstID); err != nil {
		legacyStore.Close()
		t.Fatalf("read first legacy journal id: %v", err)
	}
	if err := legacyStore.db.QueryRowContext(ctx, `SELECT id FROM journal_entries ORDER BY rowid LIMIT 1 OFFSET 1`).Scan(&secondID); err != nil {
		legacyStore.Close()
		t.Fatalf("read second legacy journal id: %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `DELETE FROM journal_search WHERE journal_entry_id = ?`, firstID); err != nil {
		legacyStore.Close()
		t.Fatalf("delete legacy derived row: %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `UPDATE journal_search SET message = 'legacy-stale' WHERE journal_entry_id = ?`, secondID); err != nil {
		legacyStore.Close()
		t.Fatalf("change legacy derived row: %v", err)
	}
	correlationColumn, err := journalSearchCorrelationColumn(ctx, legacyStore)
	if err != nil {
		legacyStore.Close()
		t.Fatalf("read legacy correlation column: %v", err)
	}
	rogueSQL := fmt.Sprintf(`
INSERT INTO journal_search(rowid, project_id, journal_entry_id, %s, entry_type, scope, message)
VALUES ((SELECT COALESCE(MAX(rowid), 0) + 1 FROM journal_search), 'rogue-project', 'rogue-entry', '', 'rogue', '', 'rogue')`, correlationColumn)
	if _, err := legacyStore.db.ExecContext(ctx, rogueSQL); err != nil {
		legacyStore.Close()
		t.Fatalf("insert legacy rogue derived row: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("Close(legacy) error = %v", err)
	}

	plan, err := ApplyStorageHomeMigration(ctx, root, PathResolver{})
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}
	store, err := OpenStore(plan.DatabasePath)
	if err != nil {
		t.Fatalf("OpenStore(copied) error = %v", err)
	}
	defer store.Close()
	parity, err := InspectJournalSearchParity(ctx, store)
	if err != nil {
		t.Fatalf("InspectJournalSearchParity(copied) error = %v", err)
	}
	if !parity.Ready || parity.CanonicalRows != 3 || parity.IndexRows != 3 {
		t.Fatalf("copied journal parity = %#v, want three exact ready rows", parity)
	}
	if got := countIdentityRows(t, store, `SELECT COUNT(*) FROM journal_search WHERE journal_entry_id = 'rogue-entry'`); got != 0 {
		t.Fatalf("copied rogue derived rows = %d, want zero", got)
	}
	if got := countIdentityRows(t, store, `SELECT COUNT(*) FROM journal_search WHERE message = 'legacy-stale'`); got != 0 {
		t.Fatalf("copied stale derived rows = %d, want zero", got)
	}
}

func TestApplyStorageHomeMigrationMergeRebuildsGlobalJournalSearchAfterCanonicalCopy(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	otherRoot := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	resolver := PathResolver{}
	if _, err := Initialize(ctx, otherRoot, resolver); err != nil {
		t.Fatalf("Initialize(other) error = %v", err)
	}
	globalPath, err := resolver.DatabasePath(root)
	if err != nil {
		t.Fatalf("DatabasePath(global) error = %v", err)
	}
	globalStore, err := OpenStore(globalPath)
	if err != nil {
		t.Fatalf("OpenStore(global) error = %v", err)
	}
	if _, err := globalStore.LogJournal(ctx, otherRoot, JournalLogOptions{Entry: "decision(storage): global-row"}); err != nil {
		globalStore.Close()
		t.Fatalf("LogJournal(global) error = %v", err)
	}
	globalCorrelationColumn, err := journalSearchCorrelationColumn(ctx, globalStore)
	if err != nil {
		globalStore.Close()
		t.Fatalf("read global correlation column: %v", err)
	}
	globalRogueSQL := fmt.Sprintf(`
INSERT INTO journal_search(rowid, project_id, journal_entry_id, %s, entry_type, scope, message)
VALUES ((SELECT COALESCE(MAX(rowid), 0) + 1 FROM journal_search), 'rogue-global', 'rogue-global-entry', '', 'rogue', '', 'rogue-global')`, globalCorrelationColumn)
	if _, err := globalStore.db.ExecContext(ctx, globalRogueSQL); err != nil {
		globalStore.Close()
		t.Fatalf("seed global rogue derived row: %v", err)
	}
	if err := globalStore.Close(); err != nil {
		t.Fatalf("Close(global) error = %v", err)
	}

	legacyPath, err := resolver.LegacyDatabasePath(root)
	if err != nil {
		t.Fatalf("LegacyDatabasePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("create legacy database directory: %v", err)
	}
	legacyStore, err := OpenStore(legacyPath)
	if err != nil {
		t.Fatalf("OpenStore(legacy) error = %v", err)
	}
	if err := ApplyMigrations(ctx, legacyStore.db, SchemaMigrations()[:1]); err != nil {
		legacyStore.Close()
		t.Fatalf("apply legacy v1 migration: %v", err)
	}
	legacyID := ProjectID(root)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := legacyStore.db.ExecContext(ctx, `INSERT INTO projects (id, identity_hash, created_at, updated_at) VALUES (?, ?, ?, ?)`, legacyID, legacyID, now, now); err != nil {
		legacyStore.Close()
		t.Fatalf("insert legacy hash project: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("Close(legacy v1) error = %v", err)
	}
	legacyStore, err = OpenStore(legacyPath)
	if err != nil {
		t.Fatalf("reopen legacy error = %v", err)
	}
	if err := legacyStore.ApplyMigrations(ctx); err != nil {
		legacyStore.Close()
		t.Fatalf("upgrade legacy database: %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `
UPDATE projects SET friendly_name = 'Legacy Hash', current_path = ?, last_seen_at = ? WHERE id = ?`, root.Path(), now, legacyID); err != nil {
		legacyStore.Close()
		t.Fatalf("set legacy project path: %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `
INSERT INTO project_paths (id, project_id, path, is_current, first_seen_at, last_seen_at, created_at, updated_at)
VALUES ('legacy-root-path', ?, ?, 1, ?, ?, ?, ?)`, legacyID, root.Path(), now, now, now, now); err != nil {
		legacyStore.Close()
		t.Fatalf("insert legacy project path: %v", err)
	}
	for _, message := range []string{"merge-one", "merge-two"} {
		if _, err := legacyStore.LogJournal(ctx, root, JournalLogOptions{Entry: "decision(storage): " + message}); err != nil {
			legacyStore.Close()
			t.Fatalf("LogJournal(legacy %s) error = %v", message, err)
		}
	}
	var legacyJournalID string
	if err := legacyStore.db.QueryRowContext(ctx, `SELECT id FROM journal_entries ORDER BY rowid LIMIT 1`).Scan(&legacyJournalID); err != nil {
		legacyStore.Close()
		t.Fatalf("read legacy journal id: %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `DELETE FROM journal_search WHERE journal_entry_id = ?`, legacyJournalID); err != nil {
		legacyStore.Close()
		t.Fatalf("delete legacy derived row: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("Close(legacy final) error = %v", err)
	}

	plan, err := PreviewStorageHomeMigration(root, resolver)
	if err != nil {
		t.Fatalf("PreviewStorageHomeMigration(merge) error = %v", err)
	}
	if plan.Action != StorageHomeActionMerge {
		t.Fatalf("merge plan action = %q, want %q", plan.Action, StorageHomeActionMerge)
	}
	merged, err := ApplyStorageHomeMigration(ctx, root, resolver)
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration(merge) error = %v", err)
	}
	mergedStore, err := OpenStore(merged.DatabasePath)
	if err != nil {
		t.Fatalf("OpenStore(merged) error = %v", err)
	}
	defer mergedStore.Close()
	parity, err := InspectJournalSearchParity(ctx, mergedStore)
	if err != nil {
		t.Fatalf("InspectJournalSearchParity(merged) error = %v", err)
	}
	if !parity.Ready || parity.CanonicalRows != 3 || parity.IndexRows != 3 {
		t.Fatalf("merged journal parity = %#v, want three exact ready rows", parity)
	}
	if got := countIdentityRows(t, mergedStore, `SELECT COUNT(*) FROM journal_search WHERE journal_entry_id LIKE 'rogue-%'`); got != 0 {
		t.Fatalf("merged rogue derived rows = %d, want zero", got)
	}
}

func TestApplyStorageHomeMigrationCopyParityFailureRemovesDestination(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	legacyPath := initializeLegacyStateDatabase(t, root, PathResolver{})
	legacyStore, err := OpenStore(legacyPath)
	if err != nil {
		t.Fatalf("OpenStore(legacy) error = %v", err)
	}
	if _, err := legacyStore.LogJournal(ctx, root, JournalLogOptions{Entry: "decision(storage): original"}); err != nil {
		legacyStore.Close()
		t.Fatalf("LogJournal() error = %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `DROP TABLE journal_search`); err != nil {
		legacyStore.Close()
		t.Fatalf("drop legacy journal_search: %v", err)
	}
	if _, err := legacyStore.db.ExecContext(ctx, `
CREATE TABLE journal_search (
  rowid INTEGER PRIMARY KEY,
  project_id TEXT NOT NULL,
  journal_entry_id TEXT NOT NULL,
  session_id TEXT,
  entry_type TEXT NOT NULL,
  scope TEXT NOT NULL,
  message TEXT NOT NULL CHECK(message = 'prior')
)`); err != nil {
		legacyStore.Close()
		t.Fatalf("create failing legacy journal_search: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("Close(legacy) error = %v", err)
	}
	plan, err := PreviewStorageHomeMigration(root, PathResolver{})
	if err != nil {
		t.Fatalf("PreviewStorageHomeMigration() error = %v", err)
	}
	if plan.Action != StorageHomeActionCopy {
		t.Fatalf("plan action = %q, want %q", plan.Action, StorageHomeActionCopy)
	}
	if _, err := ApplyStorageHomeMigration(ctx, root, PathResolver{}); err == nil {
		t.Fatal("ApplyStorageHomeMigration() error = nil, want copied parity failure")
	}
	if _, err := os.Stat(plan.DatabasePath); !os.IsNotExist(err) {
		t.Fatalf("copied destination stat error = %v, want destination removed", err)
	}
}

func TestApplyStorageHomeMigrationMergeParityFailureRollsBackGlobalCanonicalRows(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	otherRoot := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	resolver := PathResolver{}
	if _, err := Initialize(ctx, otherRoot, resolver); err != nil {
		t.Fatalf("Initialize(other) error = %v", err)
	}
	globalPath, err := resolver.DatabasePath(root)
	if err != nil {
		t.Fatalf("DatabasePath(global) error = %v", err)
	}
	globalStore, err := OpenStore(globalPath)
	if err != nil {
		t.Fatalf("OpenStore(global) error = %v", err)
	}
	if _, err := globalStore.LogJournal(ctx, otherRoot, JournalLogOptions{Entry: "decision(storage): global-canonical"}); err != nil {
		globalStore.Close()
		t.Fatalf("LogJournal(global) error = %v", err)
	}
	var globalProjectID, globalJournalID, globalMessage, globalHarness string
	var globalRowID int64
	if err := globalStore.db.QueryRowContext(ctx, `
SELECT journal_entries.rowid, journal_entries.project_id, journal_entries.id, journal_entries.message,
       COALESCE(journal_entries.harness_session_id, '') FROM journal_entries`).Scan(&globalRowID, &globalProjectID, &globalJournalID, &globalMessage, &globalHarness); err != nil {
		globalStore.Close()
		t.Fatalf("read global journal row: %v", err)
	}
	if _, err := globalStore.db.ExecContext(ctx, `DROP TABLE journal_search`); err != nil {
		globalStore.Close()
		t.Fatalf("drop global journal_search: %v", err)
	}
	if _, err := globalStore.db.ExecContext(ctx, `
CREATE TABLE journal_search (
  rowid INTEGER PRIMARY KEY,
  project_id TEXT NOT NULL,
  journal_entry_id TEXT NOT NULL,
  session_id TEXT,
  entry_type TEXT NOT NULL,
  scope TEXT NOT NULL,
  message TEXT NOT NULL CHECK(message = 'prior derived')
)`); err != nil {
		globalStore.Close()
		t.Fatalf("create failing global journal_search: %v", err)
	}
	if _, err := globalStore.db.ExecContext(ctx, `
INSERT INTO journal_search(rowid, project_id, journal_entry_id, session_id, entry_type, scope, message)
VALUES (?, ?, ?, ?, 'decision', 'storage', 'prior derived')`, globalRowID, globalProjectID, globalJournalID, globalHarness); err != nil {
		globalStore.Close()
		t.Fatalf("seed global prior derived row: %v", err)
	}
	if err := globalStore.Close(); err != nil {
		t.Fatalf("Close(global) error = %v", err)
	}
	initializeLegacyStateDatabase(t, root, resolver)
	plan, err := PreviewStorageHomeMigration(root, resolver)
	if err != nil {
		t.Fatalf("PreviewStorageHomeMigration() error = %v", err)
	}
	if plan.Action != StorageHomeActionMerge {
		t.Fatalf("plan action = %q, want %q", plan.Action, StorageHomeActionMerge)
	}
	if _, err := ApplyStorageHomeMigration(ctx, root, resolver); err == nil {
		t.Fatal("ApplyStorageHomeMigration() error = nil, want merge parity failure")
	}
	check, err := OpenStore(globalPath)
	if err != nil {
		t.Fatalf("OpenStore(global after rollback) error = %v", err)
	}
	defer check.Close()
	var gotMessage string
	if err := check.db.QueryRowContext(ctx, `SELECT message FROM journal_entries WHERE id = ?`, globalJournalID).Scan(&gotMessage); err != nil {
		t.Fatalf("read global canonical row after rollback: %v", err)
	}
	if gotMessage != globalMessage {
		t.Fatalf("global canonical message after rollback = %q, want %q", gotMessage, globalMessage)
	}
	var priorCount int
	if err := check.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM journal_search WHERE journal_entry_id = ? AND message = 'prior derived'`, globalJournalID).Scan(&priorCount); err != nil {
		t.Fatalf("read global derived row after rollback: %v", err)
	}
	if priorCount != 1 {
		t.Fatalf("global prior derived rows after rollback = %d, want one", priorCount)
	}
	if got := countIdentityRows(t, check, `SELECT COUNT(*) FROM journal_entries WHERE project_id = ?`, ProjectID(root)); got != 0 {
		t.Fatalf("copied root canonical rows after rollback = %d, want zero", got)
	}
}

func initializeLegacyStateDatabase(t *testing.T, root project.Root, resolver PathResolver) string {
	t.Helper()
	ctx := context.Background()
	legacyPath, err := resolver.LegacyDatabasePath(root)
	if err != nil {
		t.Fatalf("LegacyDatabasePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("create legacy database dir error = %v", err)
	}
	store, err := OpenStore(legacyPath)
	if err != nil {
		t.Fatalf("OpenStore(legacy) error = %v", err)
	}
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations(legacy) error = %v", err)
	}
	if err := store.UpsertProject(ctx, root); err != nil {
		t.Fatalf("UpsertProject(legacy) error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close(legacy) error = %v", err)
	}
	return legacyPath
}

func initializeProjectDataDatabase(t *testing.T, root project.Root, resolver PathResolver) string {
	t.Helper()
	projectPath, err := resolver.ProjectDatabasePath(root)
	if err != nil {
		t.Fatalf("ProjectDatabasePath() error = %v", err)
	}
	initializeDatabaseAtPath(t, root, projectPath)
	return projectPath
}

func initializeDatabaseAtPath(t *testing.T, root project.Root, path string) {
	t.Helper()
	ctx := context.Background()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create database dir error = %v", err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore(%s) error = %v", path, err)
	}
	if err := store.ApplyMigrations(ctx); err != nil {
		t.Fatalf("ApplyMigrations(%s) error = %v", path, err)
	}
	if err := store.UpsertProject(ctx, root); err != nil {
		t.Fatalf("UpsertProject(%s) error = %v", path, err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close(%s) error = %v", path, err)
	}
}
