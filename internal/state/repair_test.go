package state

import (
	"context"
	"os"
	"testing"
)

func TestRepairMissingRelationshipOriginsDryRunDoesNotWrite(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	projectID := projectIDForTest(t, store, root)
	insertRelationshipWithoutOrigin(t, store, projectID)
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	result, err := RepairMissingRelationshipOrigins(context.Background(), root, PathResolver{StateHome: stateHome}, RelationshipOriginRepairOptions{Origin: "imported"})
	if err != nil {
		t.Fatalf("RepairMissingRelationshipOrigins() error = %v", err)
	}
	if result.Applied {
		t.Fatal("Applied = true, want dry-run")
	}
	if result.Matched != 1 {
		t.Fatalf("Matched = %d, want 1", result.Matched)
	}
	if result.Updated != 0 {
		t.Fatalf("Updated = %d, want 0 for dry-run", result.Updated)
	}

	store = openTestStore(t, root, stateHome)
	defer store.Close()
	assertMissingRelationshipOrigins(t, store, projectID, 1)
}

func TestRepairMissingRelationshipOriginsApplyBackfillsCurrentProject(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	store := openTestStore(t, root, stateHome)
	projectID := projectIDForTest(t, store, root)
	insertRelationshipWithoutOrigin(t, store, projectID)
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	result, err := RepairMissingRelationshipOrigins(context.Background(), root, PathResolver{StateHome: stateHome}, RelationshipOriginRepairOptions{Origin: "imported", Apply: true})
	if err != nil {
		t.Fatalf("RepairMissingRelationshipOrigins() error = %v", err)
	}
	if !result.Applied {
		t.Fatal("Applied = false, want true")
	}
	if result.Matched != 1 {
		t.Fatalf("Matched = %d, want 1", result.Matched)
	}
	if result.Updated != 1 {
		t.Fatalf("Updated = %d, want 1", result.Updated)
	}
	if result.BackupPath == "" {
		t.Fatal("BackupPath is empty for applied repair")
	}
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Fatalf("repair backup does not exist: %v", err)
	}

	store = openTestStore(t, root, stateHome)
	defer store.Close()
	assertMissingRelationshipOrigins(t, store, projectID, 0)
	var origin string
	if err := store.db.QueryRowContext(context.Background(), `SELECT origin FROM relationships WHERE id = 'relationship-without-origin'`).Scan(&origin); err != nil {
		t.Fatalf("read repaired relationship origin error = %v", err)
	}
	if origin != "imported" {
		t.Fatalf("origin = %q, want imported", origin)
	}
}

func TestRepairMissingRelationshipOriginsRejectsUnknownOrigin(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	_, err := RepairMissingRelationshipOrigins(context.Background(), root, PathResolver{StateHome: stateHome}, RelationshipOriginRepairOptions{Origin: "guessed", Apply: true})
	if err == nil {
		t.Fatal("RepairMissingRelationshipOrigins() error = nil, want invalid origin error")
	}
}

func insertRelationshipWithoutOrigin(t *testing.T, store *Store, projectID string) {
	t.Helper()
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO relationships (id, project_id, from_entity_kind, from_entity_id, to_entity_kind, to_entity_id, relationship_type, reason, created_at, updated_at)
VALUES ('relationship-without-origin', ?, 'task', 'task-one', 'spec', 'spec-one', 'implements', 'legacy row', '2026-06-13T10:00:00Z', '2026-06-13T10:00:00Z')
`, projectID); err != nil {
		t.Fatalf("insert relationship without origin error = %v", err)
	}
}

func assertMissingRelationshipOrigins(t *testing.T, store *Store, projectID string, want int) {
	t.Helper()
	var got int
	if err := store.db.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM relationships
WHERE project_id = ?
  AND (origin IS NULL OR TRIM(origin) = '')
`, projectID).Scan(&got); err != nil {
		t.Fatalf("count missing relationship origins error = %v", err)
	}
	if got != want {
		t.Fatalf("missing relationship origins = %d, want %d", got, want)
	}
}
