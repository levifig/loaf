package state

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPreviewStorageHomeMigrationPlansLegacyCopy(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	legacyStatus, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize(legacy) error = %v", err)
	}

	plan, err := PreviewStorageHomeMigration(root, PathResolver{})
	if err != nil {
		t.Fatalf("PreviewStorageHomeMigration() error = %v", err)
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
	if plan.LegacyDatabasePath != legacyStatus.DatabasePath {
		t.Fatalf("LegacyDatabasePath = %q, want %q", plan.LegacyDatabasePath, legacyStatus.DatabasePath)
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

	legacyStatus, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize(legacy) error = %v", err)
	}

	plan, err := ApplyStorageHomeMigration(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}

	if !plan.Applied {
		t.Fatal("Applied = false, want true")
	}
	if plan.Action != StorageHomeActionAlreadyMigrated {
		t.Fatalf("Action = %q, want %q", plan.Action, StorageHomeActionAlreadyMigrated)
	}
	if _, err := os.Stat(legacyStatus.DatabasePath); err != nil {
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
}

func TestApplyStorageHomeMigrationDoesNotOverwriteExistingDataHomeDatabase(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize(legacy) error = %v", err)
	}
	dataStatus, err := Initialize(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("Initialize(data) error = %v", err)
	}

	plan, err := ApplyStorageHomeMigration(context.Background(), root, PathResolver{})
	if err != nil {
		t.Fatalf("ApplyStorageHomeMigration() error = %v", err)
	}

	if plan.Applied {
		t.Fatal("Applied = true, want no overwrite")
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

func TestCopyFileExclusiveRemovesPartialDestinationOnCopyFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reading a directory as a file is platform-specific")
	}
	sourceDir := filepath.Join(t.TempDir(), "source-dir")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(source-dir) error = %v", err)
	}
	destination := filepath.Join(t.TempDir(), "state.db")

	err := copyFileExclusive(sourceDir, destination, 0o600)
	if err == nil {
		t.Fatal("copyFileExclusive() error = nil, want copy failure")
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("destination stat error = %v, want partial destination removed", statErr)
	}
}
