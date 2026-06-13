package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/levifig/loaf/internal/project"
)

const (
	ModeMarkdownOnly = "markdown-only"
	ModeSQLiteReady  = "sqlite-ready"
	ModeInvalid      = "invalid"
)

// Diagnostic describes a state-runtime observation without mutating state.
type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// RepairAction describes an explicit repair recommendation from diagnostics.
type RepairAction struct {
	Code           string `json:"code"`
	DiagnosticCode string `json:"diagnostic_code"`
	Description    string `json:"description"`
	Command        string `json:"command,omitempty"`
	Path           string `json:"path,omitempty"`
	Safe           bool   `json:"safe"`
	Applied        bool   `json:"applied"`
}

// Status is the pre-init state view exposed by `loaf state status`.
type Status struct {
	ProjectRoot          string         `json:"project_root"`
	ProjectID            string         `json:"project_id"`
	ProjectName          string         `json:"project_name,omitempty"`
	ProjectCurrentPath   string         `json:"project_current_path,omitempty"`
	DatabasePath         string         `json:"database_path"`
	LegacyDatabasePath   string         `json:"legacy_database_path,omitempty"`
	DatabaseExists       bool           `json:"database_exists"`
	LegacyDatabaseExists bool           `json:"legacy_database_exists"`
	DatabaseParentExists bool           `json:"database_parent_exists"`
	SchemaVersion        int            `json:"schema_version"`
	Mode                 string         `json:"mode"`
	Diagnostics          []Diagnostic   `json:"diagnostics"`
	RepairPlan           []RepairAction `json:"repair_plan,omitempty"`
}

// Inspect returns the current state-runtime status without creating files.
func Inspect(root project.Root, resolver PathResolver) (Status, error) {
	databasePath, err := resolver.DatabasePath(root)
	if err != nil {
		return Status{}, err
	}

	status := Status{
		ProjectRoot:  root.Path(),
		ProjectID:    ProjectID(root),
		DatabasePath: databasePath,
	}
	if legacyPath, err := migrationSourceDatabasePath(root, resolver); err == nil && legacyPath != databasePath {
		status.LegacyDatabasePath = legacyPath
		if info, err := os.Stat(legacyPath); err == nil && !info.IsDir() {
			status.LegacyDatabaseExists = true
		}
	}

	parent := filepath.Dir(databasePath)
	if info, err := os.Stat(parent); err == nil && info.IsDir() {
		status.DatabaseParentExists = true
	}

	info, err := os.Stat(databasePath)
	switch {
	case err == nil && info.IsDir():
		status.DatabaseExists = true
		status.Mode = ModeInvalid
		status.Diagnostics = append(status.Diagnostics, Diagnostic{
			Severity: "error",
			Code:     "database-path-is-directory",
			Message:  fmt.Sprintf("database path is a directory: %s", databasePath),
		})
	case err == nil && info.Size() == 0:
		status.DatabaseExists = true
		status.Mode = ModeInvalid
		status.Diagnostics = append(status.Diagnostics, Diagnostic{
			Severity: "error",
			Code:     "database-file-empty",
			Message:  fmt.Sprintf("database file is empty: %s", databasePath),
		})
	case err == nil:
		status.DatabaseExists = true
		store, err := OpenStore(databasePath)
		if err != nil {
			status.Mode = ModeInvalid
			status.Diagnostics = append(status.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "database-open-failed",
				Message:  err.Error(),
			})
			return status, nil
		}
		defer store.Close()
		version, err := store.SchemaVersion(context.Background())
		if err != nil {
			status.Mode = ModeInvalid
			status.Diagnostics = append(status.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "schema-version-unreadable",
				Message:  err.Error(),
			})
			return status, nil
		}
		status.SchemaVersion = version
		schemaDiagnostics, schemaValid := inspectSchemaMigrations(context.Background(), store, version)
		status.Diagnostics = append(status.Diagnostics, schemaDiagnostics...)
		if !schemaValid {
			status.Mode = ModeInvalid
			return status, nil
		}
		status.Mode = ModeSQLiteReady
		if identity, err := store.LookupProjectIdentityForRoot(context.Background(), root); err == nil {
			status.ProjectID = identity.ID
			status.ProjectName = identity.FriendlyName
			status.ProjectCurrentPath = identity.CurrentPath
		} else {
			status.Diagnostics = append(status.Diagnostics, Diagnostic{
				Severity: "warn",
				Code:     "project-identity-unreadable",
				Message:  err.Error(),
			})
		}
		status.Diagnostics = append(status.Diagnostics, Diagnostic{
			Severity: "info",
			Code:     "sqlite-ready",
			Message:  fmt.Sprintf("SQLite state database is ready at schema version %d", version),
		})
		if status.LegacyDatabaseExists {
			status.Diagnostics = append(status.Diagnostics, Diagnostic{
				Severity: "warn",
				Code:     "legacy-project-database-leftover",
				Message:  fmt.Sprintf("legacy project database remains at %s after global DB initialization", status.LegacyDatabasePath),
			})
		}
		exportDiagnostics, err := inspectStaleExports(context.Background(), store)
		if err != nil {
			status.Diagnostics = append(status.Diagnostics, Diagnostic{
				Severity: "warn",
				Code:     "export-staleness-unreadable",
				Message:  err.Error(),
			})
		} else {
			status.Diagnostics = append(status.Diagnostics, exportDiagnostics...)
		}
	case os.IsNotExist(err):
		status.Mode = ModeMarkdownOnly
		status.Diagnostics = append(status.Diagnostics,
			Diagnostic{
				Severity: "warn",
				Code:     "database-missing",
				Message:  "SQLite state database does not exist yet",
			},
			Diagnostic{
				Severity: "info",
				Code:     "markdown-fallback-active",
				Message:  "Markdown compatibility remains active for projects without initialized SQLite state",
			},
		)
		if status.LegacyDatabaseExists {
			status.Diagnostics = append(status.Diagnostics, Diagnostic{
				Severity: "warn",
				Code:     "legacy-state-database-detected",
				Message:  fmt.Sprintf("legacy SQLite state database exists at %s; run `loaf state migrate storage-home --apply` to copy it to %s", status.LegacyDatabasePath, status.DatabasePath),
			})
		}
	default:
		return Status{}, fmt.Errorf("inspect state database: %w", err)
	}

	return status, nil
}

// RepairPlanForStatus turns diagnostics into explicit, non-surprising repair actions.
func RepairPlanForStatus(status Status) []RepairAction {
	actions := []RepairAction{}
	for _, diagnostic := range status.Diagnostics {
		switch diagnostic.Code {
		case "database-missing":
			actions = append(actions, RepairAction{
				Code:           "initialize-database",
				DiagnosticCode: diagnostic.Code,
				Description:    "Initialize the global SQLite database for this project.",
				Command:        "loaf state doctor --fix",
				Path:           status.DatabasePath,
				Safe:           true,
			})
		case "legacy-state-database-detected":
			actions = append(actions, RepairAction{
				Code:           "migrate-storage-home",
				DiagnosticCode: diagnostic.Code,
				Description:    "Preview and then apply storage-home migration to copy legacy state into the global database.",
				Command:        "loaf state migrate storage-home --dry-run",
				Path:           status.LegacyDatabasePath,
				Safe:           false,
			})
		case "legacy-project-database-leftover":
			actions = append(actions, RepairAction{
				Code:           "review-legacy-project-database",
				DiagnosticCode: diagnostic.Code,
				Description:    "Review the leftover legacy project database and remove it manually only after verifying the global database.",
				Command:        "loaf state migrate storage-home --dry-run",
				Path:           status.LegacyDatabasePath,
				Safe:           false,
			})
		case "schema-version-mismatch", "schema-checksum-mismatch", "schema-migration-missing":
			actions = append(actions, RepairAction{
				Code:           "inspect-schema-migrations",
				DiagnosticCode: diagnostic.Code,
				Description:    "Inspect schema migration drift before applying any repair.",
				Command:        "loaf state doctor --json",
				Path:           status.DatabasePath,
				Safe:           false,
			})
		case "stale-compatibility-export":
			actions = append(actions, RepairAction{
				Code:           "regenerate-export",
				DiagnosticCode: diagnostic.Code,
				Description:    "Regenerate the stale compatibility export from SQLite state.",
				Safe:           false,
			})
		}
	}
	return actions
}

func inspectSchemaMigrations(ctx context.Context, store *Store, version int) ([]Diagnostic, bool) {
	diagnostics := []Diagnostic{}
	valid := true
	current := CurrentSchemaVersion()
	if version != current {
		valid = false
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "schema-version-mismatch",
			Message:  fmt.Sprintf("schema version %d does not match expected version %d", version, current),
		})
	}

	for _, migration := range SchemaMigrations() {
		var checksum string
		err := store.db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = ?`, migration.Version).Scan(&checksum)
		switch {
		case err == nil && checksum != migration.Checksum():
			valid = false
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "schema-checksum-mismatch",
				Message:  fmt.Sprintf("schema migration %d checksum does not match Go-owned migration", migration.Version),
			})
		case errorsIsNoRows(err):
			valid = false
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "schema-migration-missing",
				Message:  fmt.Sprintf("schema migration %d is missing", migration.Version),
			})
		case err != nil:
			valid = false
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "schema-version-unreadable",
				Message:  err.Error(),
			})
		}
	}
	return diagnostics, valid
}

func inspectStaleExports(ctx context.Context, store *Store) ([]Diagnostic, error) {
	rows, err := store.db.QueryContext(ctx, `
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, specs.updated_at
FROM exports JOIN specs ON exports.source_entity_kind = 'spec' AND specs.project_id = exports.project_id AND specs.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, tasks.updated_at
FROM exports JOIN tasks ON exports.source_entity_kind = 'task' AND tasks.project_id = exports.project_id AND tasks.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, ideas.updated_at
FROM exports JOIN ideas ON exports.source_entity_kind = 'idea' AND ideas.project_id = exports.project_id AND ideas.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, sparks.updated_at
FROM exports JOIN sparks ON exports.source_entity_kind = 'spark' AND sparks.project_id = exports.project_id AND sparks.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, brainstorms.updated_at
FROM exports JOIN brainstorms ON exports.source_entity_kind = 'brainstorm' AND brainstorms.project_id = exports.project_id AND brainstorms.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, shaping_drafts.updated_at
FROM exports JOIN shaping_drafts ON exports.source_entity_kind = 'shaping_draft' AND shaping_drafts.project_id = exports.project_id AND shaping_drafts.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, sessions.updated_at
FROM exports JOIN sessions ON exports.source_entity_kind = 'session' AND sessions.project_id = exports.project_id AND sessions.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, reports.updated_at
FROM exports JOIN reports ON exports.source_entity_kind = 'report' AND reports.project_id = exports.project_id AND reports.id = exports.source_entity_id
UNION ALL
SELECT exports.id, exports.path, exports.source_entity_kind, exports.source_entity_id, exports.generated_at, journal_entries.updated_at
FROM exports JOIN journal_entries ON exports.source_entity_kind = 'journal_entry' AND journal_entries.project_id = exports.project_id AND journal_entries.id = exports.source_entity_id
`)
	if err != nil {
		return nil, fmt.Errorf("inspect stale exports: %w", err)
	}
	defer rows.Close()

	diagnostics := []Diagnostic{}
	for rows.Next() {
		var id, path, kind, entityID, generatedAt, updatedAt string
		if err := rows.Scan(&id, &path, &kind, &entityID, &generatedAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan stale export: %w", err)
		}
		if updatedAt > generatedAt {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "warn",
				Code:     "stale-compatibility-export",
				Message:  fmt.Sprintf("export %s at %s is stale for %s %s", id, path, kind, entityID),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stale exports: %w", err)
	}
	return diagnostics, nil
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
