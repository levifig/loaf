package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveContinuityWritersRecordHarnessProvenance(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "test-thread")
	t.Setenv("CODEX_VERSION", "1.2.3")
	t.Setenv("CLAUDECODE", "")
	root, _, store := explorationFixture(t)
	ctx := context.Background()
	if _, err := store.LogJournal(ctx, root, JournalLogOptions{Origin: func() *JournalOriginInput { o := newJournalOrigin(); return &o }(), Entry: "discover(test): direct state writer"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateArtifactEntity(ctx, root, ArtifactEntityCreateOptions{Kind: "handoff", Title: "Saved context", Body: "Recipient context"}); err != nil {
		t.Fatal(err)
	}
	_, err := store.CaptureIdea(ctx, root, IdeaCaptureOptions{Title: "A captured idea"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.CreateExploration(ctx, root, ExplorationCreateOptions{Title: "A bounded inquiry"})
	if err != nil {
		t.Fatal(err)
	}
	options := ExplorationCheckpointOptions{ExplorationRef: created.Exploration.Alias, OperationID: "once", Purpose: "p", Conclusions: "c", Unresolved: "u", NextAction: "n"}
	if _, err := store.AppendExplorationCheckpoint(ctx, root, options); err != nil {
		t.Fatal(err)
	}
	// Reusing the operation is a readback, not a second observation.
	if _, err := store.AppendExplorationCheckpoint(ctx, root, options); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`SELECT COUNT(*) FROM facts WHERE kind IN ('handoff.recorded','idea.created') AND json_extract(payload,'$.observed_harness')='codex' AND json_extract(payload,'$.observed_harness_version')='1.2.3'`,
		`SELECT COUNT(*) FROM (SELECT observed_harness,observed_harness_version FROM explorations UNION ALL SELECT observed_harness,observed_harness_version FROM exploration_checkpoints) WHERE observed_harness='codex' AND observed_harness_version='1.2.3'`,
	} {
		var count int
		if err := store.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("observed rows=%d, want 2 for %s", count, query)
		}
	}
	var harness, version string
	if err := store.db.QueryRowContext(ctx, `SELECT observed_harness,observed_harness_version FROM journal_origins`).Scan(&harness, &version); err != nil {
		t.Fatal(err)
	}
	if harness != "codex" || version != "1.2.3" {
		t.Fatalf("journal provenance=%q/%q", harness, version)
	}
}

func TestLiveContinuityUnknownAndHistoricFactBytes(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CLAUDECODE", "")
	root, _, store := explorationFixture(t)
	ctx := context.Background()
	projectID := projectIDForTest(t, store, root)
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	payload := CoreEventPayload{SubjectKind: "handoff", SubjectID: "historic", Title: "Retained bytes"}
	before, err := encodeCoreEventPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	old, err := appendCoreEventFactTx(ctx, tx, projectID, FactKindHandoffRecorded, "", payload, parseCoreEventTime("2026-01-01T00:00:00Z"), legacyFactEnvID)
	if err != nil {
		t.Fatal(err)
	}
	if old.Payload != before {
		t.Fatalf("historic payload changed: %s", old.Payload)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateArtifactEntity(ctx, root, ArtifactEntityCreateOptions{Kind: "handoff", Title: "Unknown origin", Body: "Context"}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facts WHERE kind='handoff.recorded' AND json_extract(payload,'$.observed_harness')='unknown' AND json_extract(payload,'$.observed_harness_version')='unknown'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("explicit unknown writes=%d, want 1", count)
	}
}

func TestExplorationObservationMigrationPreservesHistory(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CLAUDECODE", "")
	ctx := context.Background()
	store, err := OpenStore(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	migrations := SchemaMigrations()
	if err := ApplyMigrations(ctx, store.db, migrations[:len(migrations)-1]); err != nil {
		t.Fatal(err)
	}
	root := projectRoot(t)
	if err := store.UpsertProject(ctx, root); err != nil {
		t.Fatal(err)
	}
	projectID := projectIDForTest(t, store, root)
	if _, err := store.db.ExecContext(ctx, `INSERT INTO explorations(id,project_id,title,created_at) VALUES('old-exploration',?,'Retained title','2026-01-01T00:00:00Z')`, projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO exploration_checkpoints(id,project_id,exploration_id,seq,purpose,conclusions,unresolved,next_action,content_digest,created_at) VALUES('old-checkpoint',?,'old-exploration',1,'p','c','u','n','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-01-01T00:00:00Z')`, projectID); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, store.db, migrations[len(migrations)-1:]); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (SELECT observed_harness,observed_harness_version FROM explorations UNION ALL SELECT observed_harness,observed_harness_version FROM exploration_checkpoints) WHERE observed_harness IS NULL AND observed_harness_version IS NULL`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("historical unknown rows=%d, want unchanged 2", count)
	}
	result, err := store.CreateExploration(ctx, root, ExplorationCreateOptions{Title: "New inquiry"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendExplorationCheckpoint(ctx, root, ExplorationCheckpointOptions{ExplorationRef: result.Exploration.ID, Purpose: "p", Conclusions: "c", Unresolved: "u", NextAction: "n"}); err != nil {
		t.Fatal(err)
	}
	status, err := store.ContinuityHarnessStatusSince(ctx, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 || status[0].ObservedHarness != "unknown" || status[0].Count != 2 {
		t.Fatalf("new-only status=%#v", status)
	}
}

func TestContinuityStatusCountsProjectsAtMonthBoundary(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "thread")
	t.Setenv("CODEX_VERSION", "1.2.3")
	t.Setenv("CLAUDECODE", "")
	root, _, store := explorationFixture(t)
	ctx := context.Background()
	if _, err := store.CreateArtifactEntity(ctx, root, ArtifactEntityCreateOptions{Kind: "handoff", Title: "Context", Body: "Body"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CaptureIdea(ctx, root, IdeaCaptureOptions{Title: "Idea"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LogJournal(ctx, root, JournalLogOptions{Origin: func() *JournalOriginInput { o := newJournalOrigin(); return &o }(), Entry: "discover(test): usage"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	since := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	// A fractional timestamp at the first second belongs to this month. Earlier
	// writes do not; neither event count depends on a gateway being installed.
	if _, err := store.db.ExecContext(ctx, `UPDATE journal_origins SET created_at=?`, since.Format("2006-01-02T15:04:05")+".000000001Z"); err != nil {
		t.Fatal(err)
	}
	other := projectRoot(t)
	if err := store.UpsertProject(ctx, other); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CLAUDECODE", "1")
	if _, err := store.LogJournal(ctx, other, JournalLogOptions{Origin: func() *JournalOriginInput { o := newJournalOrigin(); return &o }(), Entry: "discover(test): another project"}); err != nil {
		t.Fatal(err)
	}
	status, err := store.ContinuityHarnessStatusSince(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int64{}
	for _, row := range status {
		counts[row.ObservedHarness] = row.Count
	}
	if len(status) != 2 || counts["codex"] != 3 || counts["claude-code"] != 1 {
		t.Fatalf("project counts=%#v", status)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE journal_origins SET created_at=? WHERE observed_harness='codex'`, since.Add(-time.Second).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	status, err = store.ContinuityHarnessStatusSince(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range status {
		if row.ObservedHarness == "codex" && row.Count != 2 {
			t.Fatalf("previous-month journal counted: %#v", status)
		}
	}
}
