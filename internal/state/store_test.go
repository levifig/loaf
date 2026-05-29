package state

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestInitializeAppliesMigrationsAndRecordsProject(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()

	status, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeSQLiteReady)
	}
	if status.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", status.SchemaVersion)
	}
	if !status.DatabaseExists {
		t.Fatal("DatabaseExists = false, want true")
	}
	if !filepath.IsAbs(status.DatabasePath) {
		t.Fatalf("DatabasePath = %q, want absolute path", status.DatabasePath)
	}
	if _, err := os.Stat(status.DatabasePath); err != nil {
		t.Fatalf("database was not created: %v", err)
	}

	store, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	count, err := store.AppliedMigrationCount(context.Background())
	if err != nil {
		t.Fatalf("AppliedMigrationCount() error = %v", err)
	}
	if count != len(SchemaMigrations()) {
		t.Fatalf("AppliedMigrationCount() = %d, want %d", count, len(SchemaMigrations()))
	}

	var projectID string
	err = store.db.QueryRowContext(context.Background(), `SELECT id FROM projects WHERE id = ?`, ProjectID(root)).Scan(&projectID)
	if err != nil {
		t.Fatalf("project row missing: %v", err)
	}
}

func TestInitializeIsIdempotent(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()

	first, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("first Initialize() error = %v", err)
	}
	second, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("second Initialize() error = %v", err)
	}

	if first.DatabasePath != second.DatabasePath {
		t.Fatalf("DatabasePath changed: %q -> %q", first.DatabasePath, second.DatabasePath)
	}
	if second.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", second.SchemaVersion)
	}
}

func TestApplyMigrationsDetectsChecksumDrift(t *testing.T) {
	db, err := sql.Open(sqliteDriverName, filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	migration := SchemaMigration{Version: 1, Name: "one", SQL: "CREATE TABLE one (id TEXT PRIMARY KEY NOT NULL);\n"}
	if err := ApplyMigrations(context.Background(), db, []SchemaMigration{migration}); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	drifted := SchemaMigration{Version: 1, Name: "one", SQL: "CREATE TABLE two (id TEXT PRIMARY KEY NOT NULL);\n"}
	if err := ApplyMigrations(context.Background(), db, []SchemaMigration{drifted}); err == nil {
		t.Fatal("ApplyMigrations() error = nil, want checksum mismatch")
	}
}
