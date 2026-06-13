package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/project"
)

func TestInspectReportsMarkdownOnlyWithoutCreatingFiles(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeMarkdownOnly {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeMarkdownOnly)
	}
	if status.DatabaseExists {
		t.Fatal("DatabaseExists = true, want false")
	}
	if status.DatabaseParentExists {
		t.Fatal("DatabaseParentExists = true, want false before init")
	}
	if status.ProjectID != "" {
		t.Fatalf("ProjectID = %q, want empty before SQLite records durable identity", status.ProjectID)
	}
	if status.LegacyProjectKey != ProjectID(root) {
		t.Fatalf("LegacyProjectKey = %q, want %q", status.LegacyProjectKey, ProjectID(root))
	}
	if _, err := os.Stat(filepath.Dir(status.DatabasePath)); !os.IsNotExist(err) {
		t.Fatalf("database parent exists after Inspect(); err = %v", err)
	}
	assertDiagnostic(t, status.Diagnostics, "database-missing")
	assertDiagnostic(t, status.Diagnostics, "markdown-fallback-active")
}

func TestInspectReportsLegacyStateDatabaseWhenDataHomeIsMissing(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	legacyPath := initializeLegacyStateDatabase(t, root, PathResolver{})

	status, err := Inspect(root, PathResolver{})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeMarkdownOnly {
		t.Fatalf("Mode = %q, want %q before storage-home migration", status.Mode, ModeMarkdownOnly)
	}
	if status.DatabaseExists {
		t.Fatal("DatabaseExists = true, want false for new data home before migration")
	}
	if !status.LegacyDatabaseExists {
		t.Fatal("LegacyDatabaseExists = false, want true")
	}
	if status.LegacyDatabasePath != legacyPath {
		t.Fatalf("LegacyDatabasePath = %q, want %q", status.LegacyDatabasePath, legacyPath)
	}
	if !strings.HasPrefix(status.DatabasePath, dataHome+string(filepath.Separator)) {
		t.Fatalf("DatabasePath = %q, want under XDG_DATA_HOME %q", status.DatabasePath, dataHome)
	}
	assertDiagnostic(t, status.Diagnostics, "legacy-state-database-detected")
}

func TestRepairPlanRecommendsSafeInitializationForMissingDatabase(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	plan := RepairPlanForStatus(status)
	action := findRepairAction(t, plan, "initialize-database")
	if !action.Safe {
		t.Fatalf("initialize action Safe = false, want true")
	}
	if action.Applied {
		t.Fatalf("initialize action Applied = true, want false for dry-run plan")
	}
	if action.Command != "loaf state doctor --fix" {
		t.Fatalf("initialize action Command = %q, want doctor --fix", action.Command)
	}
	if action.Path != status.DatabasePath {
		t.Fatalf("initialize action Path = %q, want %q", action.Path, status.DatabasePath)
	}
}

func TestRepairPlanTreatsLegacyLeftoverAsManualReview(t *testing.T) {
	root := projectRoot(t)
	dataHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)

	legacyPath := initializeLegacyStateDatabase(t, root, PathResolver{})
	if _, err := Initialize(context.Background(), root, PathResolver{}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	status, err := Inspect(root, PathResolver{})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	assertDiagnostic(t, status.Diagnostics, "legacy-project-database-leftover")

	action := findRepairAction(t, RepairPlanForStatus(status), "review-legacy-project-database")
	if action.Safe {
		t.Fatal("legacy leftover action Safe = true, want manual review")
	}
	if action.Applied {
		t.Fatal("legacy leftover action Applied = true, want false")
	}
	if action.Command != "loaf state repair legacy-project-database --dry-run --json" {
		t.Fatalf("legacy leftover action Command = %q, want legacy archive dry-run", action.Command)
	}
	if action.Path != legacyPath {
		t.Fatalf("legacy leftover action Path = %q, want %q", action.Path, legacyPath)
	}
}

func TestRepairPlanDeduplicatesRepeatedActions(t *testing.T) {
	status := Status{
		DatabasePath: "/tmp/loaf.sqlite",
		Diagnostics: []Diagnostic{
			{Severity: "error", Code: "backend-mapping-entity-missing", Message: "first missing backend mapping"},
			{Severity: "error", Code: "backend-mapping-entity-missing", Message: "second missing backend mapping"},
		},
	}

	actions := RepairPlanForStatus(status)
	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1: %#v", len(actions), actions)
	}
	if actions[0].Code != "audit-backend-mappings" {
		t.Fatalf("action Code = %q, want audit-backend-mappings", actions[0].Code)
	}
}

func TestRepairPlanPreservesDistinctDiagnosticActions(t *testing.T) {
	status := Status{
		DatabasePath: "/tmp/loaf.sqlite",
		Diagnostics: []Diagnostic{
			{Severity: "error", Code: "backend-mapping-entity-missing", Message: "missing backend mapping"},
			{Severity: "warn", Code: "backend-mapping-entity-ambiguous", Message: "ambiguous backend mapping"},
		},
	}

	actions := RepairPlanForStatus(status)
	if len(actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2: %#v", len(actions), actions)
	}
	if actions[0].DiagnosticCode == actions[1].DiagnosticCode {
		t.Fatalf("diagnostic codes should remain distinct: %#v", actions)
	}
}

func TestInspectReportsSQLiteReadyWhenDatabaseIsInitialized(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	initialized, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeSQLiteReady)
	}
	if status.DatabasePath != initialized.DatabasePath {
		t.Fatalf("DatabasePath = %q, want %q", status.DatabasePath, initialized.DatabasePath)
	}
	if !status.DatabaseExists {
		t.Fatal("DatabaseExists = false, want true")
	}
	if !status.DatabaseParentExists {
		t.Fatal("DatabaseParentExists = false, want true")
	}
	if status.SchemaVersion != CurrentSchemaVersion() {
		t.Fatalf("SchemaVersion = %d, want %d", status.SchemaVersion, CurrentSchemaVersion())
	}
	if status.ProjectID == "" {
		t.Fatal("ProjectID is empty after SQLite records durable identity")
	}
	if status.ProjectID == ProjectID(root) {
		t.Fatalf("ProjectID = legacy path key %q, want generated durable identity", status.ProjectID)
	}
	if status.LegacyProjectKey != ProjectID(root) {
		t.Fatalf("LegacyProjectKey = %q, want %q", status.LegacyProjectKey, ProjectID(root))
	}
	if status.ProjectName != filepath.Base(root.Path()) {
		t.Fatalf("ProjectName = %q, want folder name", status.ProjectName)
	}
	if status.ProjectCurrentPath != root.Path() {
		t.Fatalf("ProjectCurrentPath = %q, want %q", status.ProjectCurrentPath, root.Path())
	}
	assertDiagnostic(t, status.Diagnostics, "sqlite-ready")
}

func TestInspectReportsInvalidWhenDatabaseFileIsNotSQLite(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	path, err := (PathResolver{StateHome: stateHome}).DatabasePath(root)
	if err != nil {
		t.Fatalf("DatabasePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("not sqlite"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "database-open-failed")
}

func TestInspectReportsInvalidSchemaVersionMismatch(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	_, err := store.db.ExecContext(context.Background(), `
INSERT INTO schema_migrations (version, name, checksum, applied_at)
VALUES (99, 'future_schema', 'future', '2026-05-28T10:00:00Z')
`)
	if err != nil {
		t.Fatalf("insert future schema migration error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "schema-version-mismatch")
}

func TestInspectReportsInvalidSchemaChecksumMismatch(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	if _, err := store.db.ExecContext(context.Background(), `UPDATE schema_migrations SET checksum = 'drifted' WHERE version = 1`); err != nil {
		t.Fatalf("drift schema migration checksum error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "schema-checksum-mismatch")
}

func TestInspectReportsStaleCompatibilityExportsAsWarnings(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	projectID := projectIDForTest(t, store, root)
	_, err := store.db.ExecContext(context.Background(), `
INSERT INTO ideas (id, project_id, title, status, created_at, updated_at)
VALUES ('idea-stale-export', ?, 'Stale Export Idea', 'open', '2026-05-28T10:00:00Z', '2026-05-28T12:00:00Z');
`, projectID)
	if err != nil {
		t.Fatalf("insert stale export idea fixture error = %v", err)
	}
	_, err = store.db.ExecContext(context.Background(), `
INSERT INTO exports (id, project_id, export_kind, format, path, state_version, source_entity_kind, source_entity_id, generated_at, created_at, updated_at)
VALUES ('export-stale', ?, 'triage', 'markdown', '.agents/exports/triage.md', 1, 'idea', 'idea-stale-export', '2026-05-28T11:00:00Z', '2026-05-28T11:00:00Z', '2026-05-28T11:00:00Z');
`, projectID)
	if err != nil {
		t.Fatalf("insert stale export fixture error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q despite stale export warning", status.Mode, ModeSQLiteReady)
	}
	assertDiagnostic(t, status.Diagnostics, "sqlite-ready")
	assertDiagnostic(t, status.Diagnostics, "stale-compatibility-export")
}

func TestInspectReportsInvalidProjectPathInvariants(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	projectID := projectIDForTest(t, store, root)
	if _, err := store.db.ExecContext(context.Background(), `
UPDATE projects
SET current_path = ?
WHERE id = ?
`, filepath.Join(root.Path(), "stale"), projectID); err != nil {
		t.Fatalf("drift project current_path error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "project-current-path-mismatch")

	action := findRepairAction(t, RepairPlanForStatus(status), "repair-project-path-invariants")
	if action.Safe {
		t.Fatalf("repair action Safe = true, want manual project path repair")
	}
	if action.Command != "loaf project list --json" {
		t.Fatalf("repair action Command = %q, want project list", action.Command)
	}
}

func TestInspectReportsSQLiteForeignKeyViolations(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	if _, err := store.db.ExecContext(context.Background(), `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys error = %v", err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO aliases (id, project_id, entity_kind, entity_id, namespace, alias, created_at, updated_at)
VALUES ('alias-orphaned-project', 'project-missing', 'task', 'task-missing', 'task', 'TASK-MISSING', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`); err != nil {
		t.Fatalf("insert orphaned alias fixture error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "sqlite-foreign-key-violation")

	action := findRepairAction(t, RepairPlanForStatus(status), "inspect-state-invariants")
	if action.Safe {
		t.Fatalf("repair action Safe = true, want manual integrity inspection")
	}
	if action.Command != "loaf state doctor --json" {
		t.Fatalf("repair action Command = %q, want state doctor JSON inspection", action.Command)
	}
}

func TestInspectUsesReadOnlyConnection(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	status, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	var journalMode string
	if err := store.db.QueryRowContext(context.Background(), `PRAGMA journal_mode = DELETE`).Scan(&journalMode); err != nil {
		t.Fatalf("set rollback journal mode error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	removeSQLiteSidecars(t, status.DatabasePath)
	if err := os.Chmod(status.DatabasePath, 0o400); err != nil {
		t.Fatalf("chmod database read-only error = %v", err)
	}
	defer os.Chmod(status.DatabasePath, 0o600)
	databaseDir := filepath.Dir(status.DatabasePath)
	if err := os.Chmod(databaseDir, 0o500); err != nil {
		t.Fatalf("chmod database directory read-only error = %v", err)
	}
	defer os.Chmod(databaseDir, 0o700)

	inspected, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if inspected.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q", inspected.Mode, ModeSQLiteReady)
	}
}

func TestInspectReportsMissingRelationshipOriginAsWarning(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	projectID := projectIDForTest(t, store, root)
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO relationships (id, project_id, from_entity_kind, from_entity_id, to_entity_kind, to_entity_id, relationship_type, reason, created_at, updated_at)
VALUES ('relationship-without-origin', ?, 'task', 'task-one', 'spec', 'spec-one', 'implements', 'legacy row', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID); err != nil {
		t.Fatalf("insert relationship without origin error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q for relationship provenance warning", status.Mode, ModeSQLiteReady)
	}
	assertDiagnostic(t, status.Diagnostics, "relationship-origin-missing")

	action := findRepairAction(t, RepairPlanForStatus(status), "audit-relationship-origin")
	if action.Safe {
		t.Fatalf("repair action Safe = true, want manual relationship audit")
	}
	if action.Command != "loaf state repair relationship-origin --origin imported --dry-run --json" {
		t.Fatalf("repair action Command = %q, want relationship origin repair dry-run", action.Command)
	}
}

func TestInspectReportsInvalidBackendMappingMissingEntity(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	projectID := projectIDForTest(t, store, root)
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO backend_mappings (id, project_id, backend, entity_kind, entity_id, external_kind, external_id, external_url, sync_status, created_at, updated_at)
VALUES ('backend-mapping-orphaned', ?, 'linear', 'task', 'task-missing', 'issue', 'ENG-123', 'https://linear.app/workspace/issue/ENG-123', 'linked', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID); err != nil {
		t.Fatalf("insert orphaned backend mapping error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "backend-mapping-entity-missing")

	action := findRepairAction(t, RepairPlanForStatus(status), "audit-backend-mappings")
	if action.Safe {
		t.Fatalf("repair action Safe = true, want manual backend mapping audit")
	}
	if action.Command != "loaf state export all --format json" {
		t.Fatalf("repair action Command = %q, want export all JSON", action.Command)
	}
}

func TestInspectReportsUnknownBackendMappingEntityKind(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	projectID := projectIDForTest(t, store, root)
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO backend_mappings (id, project_id, backend, entity_kind, entity_id, external_kind, external_id, external_url, sync_status, created_at, updated_at)
VALUES ('backend-mapping-unknown-kind', ?, 'linear', 'milestone', 'milestone-one', 'project_milestone', 'milestone-123', NULL, 'linked', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID); err != nil {
		t.Fatalf("insert unknown-kind backend mapping error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "backend-mapping-entity-kind-unknown")
	assertNoDiagnostic(t, status.Diagnostics, "backend-mapping-entity-missing")
}

func TestInspectReportsAmbiguousBackendMappingAsWarning(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	projectID := projectIDForTest(t, store, root)
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO tasks (id, project_id, spec_id, title, status, priority, body_source_id, created_at, updated_at)
VALUES ('task-linear', ?, NULL, 'Linear-backed task', 'todo', 'P2', NULL, '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID); err != nil {
		t.Fatalf("insert task fixture error = %v", err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO backend_mappings (id, project_id, backend, entity_kind, entity_id, external_kind, external_id, external_url, sync_status, created_at, updated_at)
VALUES
  ('backend-mapping-linear-one', ?, 'linear', 'task', 'task-linear', 'issue', 'ENG-123', 'https://linear.app/workspace/issue/ENG-123', 'linked', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z'),
  ('backend-mapping-linear-two', ?, 'linear', 'task', 'task-linear', 'issue', 'ENG-124', 'https://linear.app/workspace/issue/ENG-124', 'linked', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID, projectID); err != nil {
		t.Fatalf("insert ambiguous backend mapping fixtures error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q for ambiguous backend mapping warning", status.Mode, ModeSQLiteReady)
	}
	assertDiagnostic(t, status.Diagnostics, "backend-mapping-entity-ambiguous")

	action := findRepairAction(t, RepairPlanForStatus(status), "audit-backend-mappings")
	if action.Safe {
		t.Fatalf("repair action Safe = true, want manual backend mapping audit")
	}
}

func TestInspectWarnsOnUnmappedLocalTasksWhenLinearEnabled(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root.Path(), ".agents"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.agents) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root.Path(), ".agents", "loaf.json"), []byte(`{"integrations":{"linear":{"enabled":true}}}`+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(loaf.json) error = %v", err)
	}
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	projectID := projectIDForTest(t, store, root)
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO tasks (id, project_id, spec_id, title, status, priority, body_source_id, created_at, updated_at)
VALUES
  ('task-active-unmapped', ?, NULL, 'Active unmapped task', 'todo', 'P2', NULL, '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z'),
  ('task-archived-unmapped', ?, NULL, 'Archived unmapped task', 'archived', 'P2', NULL, '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z'),
  ('task-active-mapped', ?, NULL, 'Active mapped task', 'todo', 'P2', NULL, '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID, projectID, projectID); err != nil {
		t.Fatalf("insert task fixtures error = %v", err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO backend_mappings (id, project_id, backend, entity_kind, entity_id, external_kind, external_id, external_url, sync_status, created_at, updated_at)
VALUES ('backend-mapping-linear-task', ?, 'linear', 'task', 'task-active-mapped', 'issue', 'ENG-125', 'https://linear.app/workspace/issue/ENG-125', 'linked', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID); err != nil {
		t.Fatalf("insert mapped task backend fixture error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeSQLiteReady {
		t.Fatalf("Mode = %q, want %q for Linear-mode local task warning", status.Mode, ModeSQLiteReady)
	}
	diagnostic := findDiagnostic(t, status.Diagnostics, "linear-mode-local-task-unmapped")
	if !strings.Contains(diagnostic.Message, "1 active local task row") {
		t.Fatalf("diagnostic Message = %q, want count of only active unmapped tasks", diagnostic.Message)
	}

	action := findRepairAction(t, RepairPlanForStatus(status), "audit-backend-mappings")
	if action.Safe {
		t.Fatalf("repair action Safe = true, want manual backend mapping audit")
	}
	if action.Command != "loaf state export all --format json" {
		t.Fatalf("repair action Command = %q, want export all JSON", action.Command)
	}
}

func TestInspectReportsInvalidWhenOperationalInvariantsAreUnreadable(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	defer store.Close()

	if _, err := store.db.ExecContext(context.Background(), `DROP TABLE relationships`); err != nil {
		t.Fatalf("drop relationships error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "state-invariants-unreadable")

	action := findRepairAction(t, RepairPlanForStatus(status), "inspect-state-invariants")
	if action.Safe {
		t.Fatalf("repair action Safe = true, want manual invariant inspection")
	}
	if action.Command != "loaf state doctor --json" {
		t.Fatalf("repair action Command = %q, want state doctor JSON inspection", action.Command)
	}
}

func TestInspectReportsInvalidEmptyFileWithoutOpeningSQLite(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	path, err := (PathResolver{StateHome: stateHome}).DatabasePath(root)
	if err != nil {
		t.Fatalf("DatabasePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "database-file-empty")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("empty file size = %d, want 0 after Inspect", info.Size())
	}
}

func TestInspectReportsInvalidWhenDatabasePathIsDirectory(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	path, err := (PathResolver{StateHome: stateHome}).DatabasePath(root)
	if err != nil {
		t.Fatalf("DatabasePath() error = %v", err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	status, err := Inspect(root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if status.Mode != ModeInvalid {
		t.Fatalf("Mode = %q, want %q", status.Mode, ModeInvalid)
	}
	assertDiagnostic(t, status.Diagnostics, "database-path-is-directory")
}

func projectRoot(t *testing.T) project.Root {
	t.Helper()
	root, err := project.ResolveRoot(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	return root
}

func openTestStore(t *testing.T, root project.Root, stateHome string) *Store {
	t.Helper()
	path, err := (PathResolver{StateHome: stateHome}).DatabasePath(root)
	if err != nil {
		t.Fatalf("DatabasePath() error = %v", err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	return store
}

func projectIDForTest(t *testing.T, store *Store, root project.Root) string {
	t.Helper()
	identity, err := store.ProjectIdentityForRoot(context.Background(), root)
	if err != nil {
		t.Fatalf("ProjectIdentityForRoot() error = %v", err)
	}
	return identity.ID
}

func assertDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	_ = findDiagnostic(t, diagnostics, code)
}

func findDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) Diagnostic {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, diagnostics)
	return Diagnostic{}
}

func assertNoDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			t.Fatalf("diagnostic %q found in %#v", code, diagnostics)
		}
	}
}

func removeSQLiteSidecars(t *testing.T, path string) {
	t.Helper()
	for _, suffix := range []string{"-wal", "-shm"} {
		sidecar := path + suffix
		if err := os.Remove(sidecar); err != nil && !os.IsNotExist(err) {
			t.Fatalf("remove SQLite sidecar %s error = %v", sidecar, err)
		}
	}
}

func findRepairAction(t *testing.T, actions []RepairAction, code string) RepairAction {
	t.Helper()
	for _, action := range actions {
		if action.Code == code {
			return action
		}
	}
	t.Fatalf("repair action %q not found in %#v", code, actions)
	return RepairAction{}
}
