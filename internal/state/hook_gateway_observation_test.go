package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestHookGatewayCountsMigrationFoldsPerCallRowsIntoMonths(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// Pin to migration 28 so later migrations cannot take its place.
	migrations := SchemaMigrations()
	if err := ApplyMigrations(ctx, store.db, migrations[:26]); err != nil {
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
	if _, err := store.db.ExecContext(ctx, `
INSERT INTO hook_events(id, project_id, hook_id, hook_phase, command, message, created_at, updated_at)
VALUES ('historic', ?, 'old-check', 'pre-tool', 'old command', 'old message', '2026-09-15T00:00:00Z', '2026-09-15T00:00:00Z')
`, projectID); err != nil {
		t.Fatal(err)
	}
	perCall := []struct{ id, decision, reason, at string }{
		{"sep-1", "allow", "passed", "2026-09-02T08:00:00Z"},
		{"sep-2", "allow", "passed", "2026-09-30T23:59:59.999999Z"},
		{"oct-1", "allow", "passed", "2026-10-01T00:00:00Z"},
		{"oct-2", "allow", "passed", "2026-10-01T00:00:00.000000001Z"},
		{"oct-3", "allow", "passed", "2026-10-31T23:59:59Z"},
		{"oct-4", "block", "policy", "2026-10-20T12:00:00Z"},
	}
	for _, row := range perCall {
		if _, err := store.db.ExecContext(ctx, `
INSERT INTO hook_events(id, project_id, hook_id, hook_phase, tool_name, decision, reason_class,
                        observed_harness, observed_harness_version, created_at, updated_at)
VALUES (?, ?, 'check-secrets', 'pre-tool', 'Bash', ?, ?, 'claude-code', '2.1.0', ?, ?)
`, row.id, projectID, row.decision, row.reason, row.at, row.at); err != nil {
			t.Fatal(err)
		}
	}

	if err := ApplyMigrations(ctx, store.db, migrations[26:27]); err != nil {
		t.Fatal(err)
	}

	var remaining int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hook_events`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	var command, message string
	if err := store.db.QueryRowContext(ctx, `SELECT command, message FROM hook_events WHERE id = 'historic'`).Scan(&command, &message); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 || command != "old command" || message != "old message" {
		t.Fatalf("hook_events after fold: rows=%d command=%q message=%q, want only the unchanged historic row", remaining, command, message)
	}
	type bucket struct {
		month, decision string
		count           int
	}
	rows, err := store.db.QueryContext(ctx, `
SELECT month, decision, count FROM hook_gateway_counts
WHERE project_id = ? AND hook_id = 'check-secrets' AND tool_name = 'Bash'
  AND observed_harness = 'claude-code' AND observed_harness_version = '2.1.0'
ORDER BY month, decision`, projectID)
	if err != nil {
		t.Fatal(err)
	}
	var got []bucket
	for rows.Next() {
		var b bucket
		if err := rows.Scan(&b.month, &b.decision, &b.count); err != nil {
			t.Fatal(err)
		}
		got = append(got, b)
	}
	rows.Close()
	want := []bucket{
		{"2026-09", "allow", 2},
		{"2026-10", "allow", 3},
		{"2026-10", "block", 1},
	}
	if len(got) != len(want) {
		t.Fatalf("monthly counts = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("monthly counts = %#v, want %#v", got, want)
		}
	}

	status, err := store.HookGatewayStatusForMonth(ctx, time.Date(2026, time.October, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 2 || status[0].Decision != "allow" || status[0].Count != 3 || status[1].Decision != "block" || status[1].Count != 1 {
		t.Fatalf("October gateway status = %#v", status)
	}

	observation := HookGatewayObservation{
		HookID: "check-secrets", ToolName: "Bash", Decision: "allow", ReasonClass: "passed",
		ObservedHarness: "unknown", HarnessVersion: "unknown",
	}
	for range 2 {
		if err := store.RecordHookGatewayObservation(ctx, projectID, observation); err != nil {
			t.Fatal(err)
		}
	}
	status, err = store.HookGatewayStatusForMonth(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 || status[0].ObservedHarness != "unknown" || status[0].Decision != "allow" || status[0].Count != 2 {
		t.Fatalf("current-month gateway status = %#v", status)
	}
}
