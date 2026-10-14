package state

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"testing"

	"github.com/levifig/loaf/vnext/migration/retired"
)

func TestRetiredArchiveExactBodiesAndClosedTableReconciliation(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	body := []byte{0xff, 0, 13, 10, 32, 32}
	for _, kind := range []string{"spec", "task", "intent"} {
		switch kind {
		case "intent":
			mustExec(t, s, `INSERT INTO intents(id,project_id,created_at) VALUES(?,?,?)`, kind, status.ProjectID, "2026-10-14T00:00:00Z")
		default:
			mustExec(t, s, `INSERT INTO `+kind+`s(id,project_id,title,status,created_at,updated_at) VALUES(?,?,?,'archived',?,?)`, kind, status.ProjectID, "Original", "2026-10-14T00:00:00Z", "2026-10-14T00:00:00Z")
		}
		mustExec(t, s, `INSERT INTO artifact_bodies(id,project_id,entity_kind,entity_id,content,content_hash,created_at,updated_at) VALUES(?,?,?,?,CAST(? AS TEXT),'old-hash',?,?)`, "body-"+kind, status.ProjectID, kind, kind, body, "2026-10-14T00:00:00Z", "2026-10-14T00:00:00Z")
	}
	mustExec(t, s, `INSERT INTO hook_events(id,project_id,hook_id,hook_phase,command,exit_code,created_at,updated_at) VALUES('typed',?,'fixture','test',?,1.25,?,?)`, status.ProjectID, []byte{}, "2026-10-14T00:00:00Z", "2026-10-14T00:00:00Z")
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	before := testFileSHA256(t, backup.BackupPath)
	exported, err := ExportVNextRehearsalBundle(ctx, VNextRehearsalExportOptions{Backup: backup, ProjectID: status.ProjectID, Root: root, Resolver: resolver})
	if err != nil {
		t.Fatal(err)
	}
	a, err := retired.Parse(exported.RetiredArchive)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Manifest.Tables) != len(ProjectScopedLiveTables()) {
		t.Fatal("incomplete table inventory")
	}
	for _, count := range a.Manifest.Tables {
		if count.SourceRows != count.MigratedRows+count.RetiredRows {
			t.Fatal("unreconciled table")
		}
	}
	typed, err := a.Record("hook_events", "typed")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range a.Content.Tables {
		if table.Name == "hook_events" {
			command, _ := table.Value(typed, "command")
			exit, _ := table.Value(typed, "exit_code")
			message, _ := table.Value(typed, "message")
			if command.Type != "blob" || command.Data == nil || len(command.Data) != 0 || exit.Type != "real" || binary.BigEndian.Uint64(exit.Data) != math.Float64bits(1.25) || message.Type != "null" || message.Data != nil {
				t.Fatal("SQLite storage classes changed")
			}
		}
	}
	for _, kind := range []string{"spec", "task", "intent"} {
		if _, err := a.Record(kind+"s", kind); err != nil {
			t.Fatal(err)
		}
		r, err := a.Record("artifact_bodies", "body-"+kind)
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range a.Content.Tables {
			if table.Name == "artifact_bodies" {
				v, ok := table.Value(r, "content")
				if !ok || v.Type != "text" || !bytes.Equal(v.Data, body) {
					t.Fatal("body bytes changed")
				}
			}
		}
	}
	if testFileSHA256(t, backup.BackupPath) != before {
		t.Fatal("backup changed")
	}
	mustExec(t, s, `INSERT INTO artifact_bodies(id,project_id,entity_kind,entity_id,content,content_hash,created_at,updated_at) VALUES('orphan',?,'idea','absent','orphan','hash',?,?)`, status.ProjectID, "2026-10-14T00:00:00Z", "2026-10-14T00:00:00Z")
	txBody, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	bodyArchive, err := readRetiredArchive(ctx, txBody, status.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range bodyArchive.Tables {
		if table.Name == "artifact_bodies" {
			for _, row := range table.Rows {
				id, _ := table.Value(row, "id")
				if string(id.Data) == "orphan" && row.Disposition != "retired" {
					t.Fatal("orphan body claimed migrated")
				}
			}
		}
	}
	if err := txBody.Rollback(); err != nil {
		t.Fatal(err)
	}
	// Schema extensions and removals must not disappear from the closed inventory.
	mustExec(t, s, `CREATE TABLE future_private_rows(id TEXT,project_id TEXT)`)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readRetiredArchive(ctx, tx, status.ProjectID); err == nil {
		t.Fatal("accepted unknown project table")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
}

func TestRetiredArchiveRecognizesOnlyJournalFirstSessionTableAbsence(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	mustExec(t, s, `DROP TABLE session_state_snapshots`)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readRetiredArchive(ctx, tx, status.ProjectID); err == nil {
		t.Fatal("accepted lone missing session table")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	// Rebuild the empty fixture journal to its known post-migration shape.
	mustExec(t, s, `DROP TABLE journal_search`)
	mustExec(t, s, `DROP TABLE journal_entries`)
	mustExec(t, s, `CREATE TABLE journal_entries(id TEXT PRIMARY KEY,project_id TEXT)`)
	mustExec(t, s, `DROP TABLE sessions`)
	tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := readRetiredArchive(ctx, tx, status.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range c.Tables {
		if table.Name == "sessions" || table.Name == "session_state_snapshots" {
			if !table.Absent || table.SourceRows != 0 {
				t.Fatal("missing absence evidence")
			}
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
}
