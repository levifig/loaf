package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestHookGatewayMigrationPreservesHistoricRowsAndAggregatesNewOnes(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	migrations := SchemaMigrations()[:25]
	if err := ApplyMigrations(ctx, store.db, migrations[:len(migrations)-1]); err != nil {
		t.Fatal(err)
	}
	root := projectRoot(t)
	if err := store.UpsertProject(ctx, root); err != nil {
		t.Fatal(err)
	}
	projectID, err := store.projectID(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	const historicTimestamp = "2026-10-01T00:00:00Z"
	if _, err := store.db.ExecContext(ctx, `
INSERT INTO hook_events(id, project_id, hook_id, hook_phase, command, message, created_at, updated_at)
VALUES ('historic', ?, 'old-check', 'pre-tool', 'old command', 'old message', ?, ?)
`, projectID, historicTimestamp, historicTimestamp); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, store.db, migrations[len(migrations)-1:]); err != nil {
		t.Fatal(err)
	}
	var command, message string
	var decision *string
	if err := store.db.QueryRowContext(ctx, `SELECT command, message, decision FROM hook_events WHERE id = 'historic'`).Scan(&command, &message, &decision); err != nil {
		t.Fatal(err)
	}
	if command != "old command" || message != "old message" || decision != nil {
		t.Fatalf("historic hook row changed: command=%q message=%q decision=%v", command, message, decision)
	}
	if err := store.RecordHookGatewayObservation(ctx, projectID, HookGatewayObservation{
		HookID: "check-secrets", ToolName: "Bash", Decision: "allow", ReasonClass: "passed",
		ObservedHarness: "unknown", HarnessVersion: "unknown",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
INSERT INTO hook_events(id, project_id, hook_id, hook_phase, tool_name, decision, reason_class,
                        observed_harness, observed_harness_version, created_at, updated_at)
VALUES ('boundary', ?, 'check-secrets', 'pre-tool', 'Bash', 'allow', 'passed',
        'unknown', 'unknown', '2026-10-01T00:00:00.000000001Z', '2026-10-01T00:00:00.000000001Z')
`, projectID); err != nil {
		t.Fatal(err)
	}
	status, err := store.HookGatewayStatusSince(ctx, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 || status[0].ProjectID != projectID || status[0].ObservedHarness != "unknown" || status[0].Decision != "allow" || status[0].ReasonClass != "passed" || status[0].Count != 2 {
		t.Fatalf("gateway status = %#v", status)
	}
}
