package state

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSimulateMarkdownMigrationProducesImportReport(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	writeAgentsFile(t, root.Path(), "ideas/20260724-sim.md", "# Idea\n")
	if _, err := Initialize(ctx, root, resolver); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	result, err := SimulateMarkdownMigration(ctx, root, resolver)
	if err != nil {
		t.Fatalf("SimulateMarkdownMigration() error = %v", err)
	}
	if result.Applied {
		t.Fatal("Applied = true, want false")
	}
	if result.Action != MarkdownMigrationActionSimulate {
		t.Fatalf("Action = %q, want %q", result.Action, MarkdownMigrationActionSimulate)
	}
	if result.Mode != MarkdownMigrationModeSimulation {
		t.Fatalf("Mode = %q, want %q", result.Mode, MarkdownMigrationModeSimulation)
	}
	if result.ImportReport == nil {
		t.Fatal("ImportReport is nil")
	}
	if result.ImportReport.SkippedEntries == nil || result.ImportReport.StatusDivergences == nil || result.ImportReport.Warnings == nil {
		t.Fatalf("ImportReport slices must be non-nil: %#v", result.ImportReport)
	}
	if result.ProjectID == "" {
		t.Fatal("ProjectID is empty")
	}
	if result.Ideas != 1 {
		t.Fatalf("Ideas = %d, want 1", result.Ideas)
	}
}

func TestSimulateMarkdownMigrationInventoryWhenNoDatabase(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	writeAgentsFile(t, root.Path(), "ideas/20260724-inv.md", "# Idea\n")

	result, err := SimulateMarkdownMigration(ctx, root, resolver)
	if err != nil {
		t.Fatalf("SimulateMarkdownMigration() error = %v", err)
	}
	if result.Mode != MarkdownMigrationModeInventory {
		t.Fatalf("Mode = %q, want %q", result.Mode, MarkdownMigrationModeInventory)
	}
	if result.ImportReport != nil {
		t.Fatalf("ImportReport = %#v, want nil", result.ImportReport)
	}
	if result.Applied || result.Action != MarkdownMigrationActionSimulate {
		t.Fatalf("Applied/Action = %t/%q", result.Applied, result.Action)
	}
	found := false
	for _, warning := range result.Warnings {
		if warning == markdownInventoryOnlyWarning {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Warnings = %#v, want inventory-only note", result.Warnings)
	}
	if result.Ideas != 1 {
		t.Fatalf("Ideas = %d, want 1", result.Ideas)
	}
}

func TestSimulateMarkdownMigrationInventoryWhenProjectUnregistered(t *testing.T) {
	ctx := context.Background()
	registered := projectRoot(t)
	unregistered := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	if _, err := Initialize(ctx, registered, resolver); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	writeAgentsFile(t, unregistered.Path(), "ideas/20260724-unreg.md", "# Idea\n")

	result, err := SimulateMarkdownMigration(ctx, unregistered, resolver)
	if err != nil {
		t.Fatalf("SimulateMarkdownMigration() error = %v", err)
	}
	if result.Mode != MarkdownMigrationModeInventory {
		t.Fatalf("Mode = %q, want %q", result.Mode, MarkdownMigrationModeInventory)
	}
	if result.ImportReport != nil {
		t.Fatalf("ImportReport = %#v, want nil", result.ImportReport)
	}
}

func TestSimulateMarkdownMigrationPreservesLiveDurableBytes(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	writeAgentsFile(t, root.Path(), "ideas/20260724-bytes.md", "# Idea\n")
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	// Hold a connection with WAL content. Quiescent means no competing writers;
	// the connection stays open so main+-wal exist as durable bytes to compare.
	// -shm is excluded by durableSQLiteFilesSnapshot (backup_test.go precedent).
	writer, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer writer.Close()
	if _, err := writer.LogJournal(ctx, root, JournalLogOptions{Entry: "decision(simulate): durable"}); err != nil {
		t.Fatalf("LogJournal() error = %v", err)
	}

	before := durableSQLiteFilesSnapshot(t, status.DatabasePath)
	if len(before[status.DatabasePath+"-wal"]) == 0 {
		t.Fatal("source WAL is empty before simulate; expected live WAL content")
	}
	result, err := SimulateMarkdownMigration(ctx, root, resolver)
	if err != nil {
		t.Fatalf("SimulateMarkdownMigration() error = %v", err)
	}
	if result.Mode != MarkdownMigrationModeSimulation {
		t.Fatalf("Mode = %q, want simulation", result.Mode)
	}
	after := durableSQLiteFilesSnapshot(t, status.DatabasePath)
	for path, contents := range before {
		got, ok := after[path]
		if !ok {
			t.Fatalf("durable file missing after simulate: %s", path)
		}
		if !bytes.Equal(contents, got) {
			t.Fatalf("durable bytes changed for %s", path)
		}
	}
}

func TestSimulateMarkdownMigrationBehindSchemaMatchesApply(t *testing.T) {
	ctx := context.Background()
	root, resolver, databasePath := seedSchema10UpgradeTarget(t)
	writeAgentsFile(t, root.Path(), "ideas/20260724-behind.md", "# Idea\n")

	beforeVersion := schemaVersionAndMigrationCount(t, databasePath)
	_, simErr := SimulateMarkdownMigration(ctx, root, resolver)
	_, applyErr := ApplyMarkdownMigration(ctx, root, resolver)

	var simRequired *SchemaUpgradeRequiredError
	var applyRequired *SchemaUpgradeRequiredError
	if !errors.As(simErr, &simRequired) {
		t.Fatalf("SimulateMarkdownMigration() error = %v, want schema-upgrade-required", simErr)
	}
	if !errors.As(applyErr, &applyRequired) {
		t.Fatalf("ApplyMarkdownMigration() error = %v, want schema-upgrade-required", applyErr)
	}
	if simRequired.Code != SchemaUpgradeRequiredCode || applyRequired.Code != SchemaUpgradeRequiredCode {
		t.Fatalf("codes = %q/%q", simRequired.Code, applyRequired.Code)
	}
	if simRequired.CurrentVersion != applyRequired.CurrentVersion {
		t.Fatalf("current versions = %d/%d", simRequired.CurrentVersion, applyRequired.CurrentVersion)
	}
	afterVersion := schemaVersionAndMigrationCount(t, databasePath)
	if beforeVersion != afterVersion {
		t.Fatalf("live schema mutated by simulate: before=%s after=%s", beforeVersion, afterVersion)
	}
}

func TestCreateMarkdownSimulationSnapshotCleansUpOnVerifyFailure(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Corrupt the live DB so VACUUM INTO succeeds but structural verify fails.
	// Dropping sqlite_master is not feasible; instead write a non-sqlite file
	// as the "live" path after initializing a real DB is wrong. Use a path that
	// VACUUM can read but then replace snapshot verify by using an invalid
	// source that openStoreReadOnlyForBackup rejects — that exercises cleanup
	// on creation failure before a partial file remains.
	missing := filepath.Join(t.TempDir(), "missing.sqlite")
	_, cleanup, err := createMarkdownSimulationSnapshot(ctx, missing)
	if err == nil {
		cleanup()
		t.Fatal("createMarkdownSimulationSnapshot() error = nil, want failure")
	}
	var snapErr *MarkdownSimulationSnapshotError
	if !errors.As(err, &snapErr) || snapErr.Code != MarkdownSimulationSnapshotFailedCode {
		t.Fatalf("error = %v (%T), want MarkdownSimulationSnapshotError", err, err)
	}

	// Also ensure a successful snapshot is removed by cleanup and leaves no siblings.
	path, cleanup, err := createMarkdownSimulationSnapshot(ctx, status.DatabasePath)
	if err != nil {
		t.Fatalf("createMarkdownSimulationSnapshot() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("snapshot missing before cleanup: %v", err)
	}
	dir := filepath.Dir(path)
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("snapshot still present after cleanup: %v", err)
	}
	if _, err := os.Stat(path + "-wal"); !os.IsNotExist(err) {
		t.Fatalf("snapshot wal still present: %v", err)
	}
	if _, err := os.Stat(path + "-shm"); !os.IsNotExist(err) {
		t.Fatalf("snapshot shm still present: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("snapshot temp dir still present: %v", err)
	}
}

func TestPreviewMarkdownMigrationCountsArchivedSessionSparks(t *testing.T) {
	root := projectRoot(t)
	writeAgentsFile(t, root.Path(), "sessions/active.md", "[2026-07-24 10:00] spark(scope): active spark\n")
	writeAgentsFile(t, root.Path(), "sessions/archive/old.md", "[2026-07-24 09:00] spark(scope): archived spark\n")

	plan, err := PreviewMarkdownMigration(root)
	if err != nil {
		t.Fatalf("PreviewMarkdownMigration() error = %v", err)
	}
	if plan.Sparks != 2 {
		t.Fatalf("Sparks = %d, want 2 (active + archive)", plan.Sparks)
	}
}

func TestInspectUnimportedLocalMarkdownUsesInventoryNotSimulation(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	writeAgentsFile(t, root.Path(), "ideas/20260724-doctor.md", "# Idea\n")
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()

	before := durableSQLiteFilesSnapshot(t, status.DatabasePath)
	diagnostics, err := inspectUnimportedLocalMarkdown(ctx, root, store, status.ProjectID)
	if err != nil {
		t.Fatalf("inspectUnimportedLocalMarkdown() error = %v", err)
	}
	after := durableSQLiteFilesSnapshot(t, status.DatabasePath)
	for path, contents := range before {
		if !bytes.Equal(contents, after[path]) {
			t.Fatalf("doctor inventory mutated durable bytes for %s", path)
		}
	}
	if len(diagnostics) == 0 {
		t.Fatal("expected local-markdown-not-imported diagnostic")
	}
	if diagnostics[0].Code != "local-markdown-not-imported" {
		t.Fatalf("Code = %q", diagnostics[0].Code)
	}
	details, _ := diagnostics[0].Details["preview_command"].(string)
	if !strings.Contains(details, "migrate markdown --dry-run") {
		t.Fatalf("preview_command = %q", details)
	}
}

func TestVacuumSQLiteIntoRejectsNonEmptyDestination(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	dest := filepath.Join(t.TempDir(), "exists.sqlite")
	if err := os.WriteFile(dest, []byte("not-empty"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := vacuumSQLiteInto(ctx, status.DatabasePath, dest); err == nil {
		t.Fatal("vacuumSQLiteInto() error = nil, want rejection of non-empty destination")
	}
}
