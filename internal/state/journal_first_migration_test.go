package state

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// seedJournalFirstFixture builds a production-shaped v9 database: sessions with
// and without harness ids, journal entries with session_id and NULL
// harness_session_id, entry_type='session' lifecycle noise, entry_type='wrap'
// synthesis, events/aliases with entity_kind='session', and a handoff carrying
// a session_id. It returns the project id.
func seedJournalFirstFixture(t *testing.T, databasePath string, projectID string) {
	t.Helper()
	store, err := OpenStore(databasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	ctx := context.Background()
	now := "2026-07-01T00:00:00Z"

	statements := []struct {
		sql  string
		args []any
	}{
		// Sessions: one with a harness id, one without.
		{`INSERT INTO sessions (id, project_id, harness_session_id, branch, status, body_source_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, ?, ?)`, []any{"session-with-hsid", projectID, "hsid-abc", "feat/journal-first", "stopped", now, now}},
		{`INSERT INTO sessions (id, project_id, harness_session_id, branch, status, body_source_id, created_at, updated_at) VALUES (?, ?, NULL, ?, ?, NULL, ?, ?)`, []any{"session-no-hsid", projectID, "main", "active", now, now}},

		// Journal entries.
		// (a) real entry linked to a session that has a harness id, NULL hsid -> backfilled.
		{journalInsertSQL, journalArgs("j-backfill", projectID, "decision", "arch", "chose journal-first", "feat/journal-first", "session-with-hsid", nil, now)},
		// (b) real wrap synthesis linked to the hsid session -> preserved, backfilled.
		{journalInsertSQL, journalArgs("j-wrap", projectID, "wrap", "", "tried X abandoned Y next Z", "feat/journal-first", "session-with-hsid", nil, now)},
		// (c) entry linked to a session WITHOUT a harness id -> not backfillable, survives.
		{journalInsertSQL, journalArgs("j-no-hsid", projectID, "note", "", "no harness id available", "main", "session-no-hsid", nil, now)},
		// (d) entry that already has a harness id -> untouched.
		{journalInsertSQL, journalArgs("j-preset", projectID, "note", "", "preset harness id", "main", "session-no-hsid", strPtr("hsid-preset"), now)},
		// (e) untagged project-scoped entry (no session) -> survives.
		{journalInsertSQL, journalArgs("j-untagged", projectID, "note", "", "untagged project entry", "", "", nil, now)},
		// noise: entry_type='session' lifecycle markers -> purged.
		{journalInsertSQL, journalArgs("j-noise-1", projectID, "session", "start", "=== SESSION STARTED ===", "main", "session-with-hsid", nil, now)},
		{journalInsertSQL, journalArgs("j-noise-2", projectID, "session", "stop", "=== SESSION STOPPED ===", "main", "session-with-hsid", nil, now)},
		{journalInsertSQL, journalArgs("j-noise-3", projectID, "session", "end", "session ended", "main", "session-no-hsid", nil, now)},

		// Handoff carrying a session_id -> retargeted to harness id.
		{`INSERT INTO handoffs (id, project_id, session_id, task_id, title, status, body_source_id, created_at, updated_at) VALUES (?, ?, ?, NULL, ?, ?, NULL, ?, ?)`, []any{"handoff-1", projectID, "session-with-hsid", "Handoff", "done", now, now}},

		// events/aliases with entity_kind='session' -> deleted.
		{`INSERT INTO events (id, project_id, entity_kind, entity_id, event_type, from_status, to_status, note, created_at, updated_at) VALUES (?, ?, 'session', ?, 'status_changed', 'active', 'stopped', 'noise', ?, ?)`, []any{"event-session-1", projectID, "session-with-hsid", now, now}},
		{`INSERT INTO events (id, project_id, entity_kind, entity_id, event_type, from_status, to_status, note, created_at, updated_at) VALUES (?, ?, 'spec', ?, 'status_changed', 'todo', 'done', 'kept', ?, ?)`, []any{"event-spec-1", projectID, "spec-x", now, now}},
		{`INSERT INTO aliases (id, project_id, entity_kind, entity_id, namespace, alias, created_at, updated_at) VALUES (?, ?, 'session', ?, 'session', 'session-1', ?, ?)`, []any{"alias-session-1", projectID, "session-with-hsid", now, now}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("seed statement %q error = %v", statement.sql, err)
		}
	}
	// Mirror the journal rows into the v9 FTS index so we can prove it is rebuilt.
	if _, err := store.db.ExecContext(ctx, `
INSERT INTO journal_search(rowid, project_id, journal_entry_id, session_id, entry_type, scope, message)
SELECT rowid, project_id, id, COALESCE(session_id, ''), entry_type, COALESCE(scope, ''), message FROM journal_entries
`); err != nil {
		t.Fatalf("seed journal_search error = %v", err)
	}
}

const journalInsertSQL = `INSERT INTO journal_entries (
  id, project_id, entry_type, scope, message, observed_branch, observed_worktree, harness_session_id, session_id, spec_id, task_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?, NULL, NULL, ?, ?)`

func journalArgs(id, projectID, entryType, scope, message, branch, sessionID string, harnessSessionID *string, now string) []any {
	return []any{
		id, projectID, entryType, emptyToNil(scope), message, emptyToNil(branch), harnessSessionID, emptyToNil(sessionID), now, now,
	}
}

func strPtr(s string) *string { return &s }

func TestPreviewJournalFirstMigrationUsesCopyRun(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	status, err := Initialize(ctx, root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedJournalFirstFixture(t, status.DatabasePath, status.ProjectID)

	result, err := PreviewJournalFirstMigration(ctx, root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("PreviewJournalFirstMigration() error = %v", err)
	}
	if result.Action != JournalFirstMigrationActionDryRun || result.Applied || !result.CopyRun {
		t.Fatalf("preview action/applied/copy_run = %q/%t/%t, want dry-run/false/true", result.Action, result.Applied, result.CopyRun)
	}
	if result.BackupPath != "" {
		t.Fatalf("preview backup path = %q, want empty (dry-run must not back up)", result.BackupPath)
	}
	// 8 journal rows seeded, 3 are entry_type='session' noise -> 5 survive.
	if result.JournalEntriesBefore != 8 || result.NoiseEntriesPurged != 3 || result.JournalEntriesAfter != 5 {
		t.Fatalf("preview journal counts before/purged/after = %d/%d/%d, want 8/3/5", result.JournalEntriesBefore, result.NoiseEntriesPurged, result.JournalEntriesAfter)
	}
	// Backfill is measured pre-purge: j-backfill, j-wrap, j-noise-1, j-noise-2 all
	// link to session-with-hsid with a NULL harness_session_id.
	if result.HarnessSessionBackfill != 4 {
		t.Fatalf("preview backfillable = %d, want 4", result.HarnessSessionBackfill)
	}
	if result.SessionEventsDeleted != 1 || result.SessionAliasesDeleted != 1 {
		t.Fatalf("preview session events/aliases = %d/%d, want 1/1", result.SessionEventsDeleted, result.SessionAliasesDeleted)
	}
	if result.SessionsDropped != 2 {
		t.Fatalf("preview sessions dropped = %d, want 2", result.SessionsDropped)
	}
	if result.JournalSearchRows != 5 {
		t.Fatalf("preview journal_search rows = %d, want 5", result.JournalSearchRows)
	}
	if result.SchemaVersion != journalFirstMigrationVersion {
		t.Fatalf("preview schema version = %d, want %d", result.SchemaVersion, journalFirstMigrationVersion)
	}

	// The live database must be untouched by a dry-run.
	if got := rawTableExists(t, status.DatabasePath, "sessions"); !got {
		t.Fatalf("live sessions table missing after preview; dry-run must not mutate live state")
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_entries`); got != 8 {
		t.Fatalf("live journal_entries after preview = %d, want 8 (untouched)", got)
	}
}

func TestApplyJournalFirstMigrationTransformsLiveDatabase(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	status, err := Initialize(ctx, root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedJournalFirstFixture(t, status.DatabasePath, status.ProjectID)

	applied, err := ApplyJournalFirstMigration(ctx, root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("ApplyJournalFirstMigration() error = %v", err)
	}
	if !applied.Applied || applied.CopyRun || applied.Action != JournalFirstMigrationActionApply {
		t.Fatalf("apply action/applied/copy_run = %q/%t/%t, want apply/true/false", applied.Action, applied.Applied, applied.CopyRun)
	}
	if applied.BackupPath == "" {
		t.Fatalf("apply backup path is empty; a pre-migration backup is mandatory")
	}
	if _, err := os.Stat(applied.BackupPath); err != nil {
		t.Fatalf("stat backup path %q: %v", applied.BackupPath, err)
	}
	if applied.NoiseEntriesPurged != 3 || applied.JournalEntriesAfter != 5 || applied.HarnessSessionBackfill != 4 {
		t.Fatalf("apply counts purged/after/backfill = %d/%d/%d, want 3/5/4", applied.NoiseEntriesPurged, applied.JournalEntriesAfter, applied.HarnessSessionBackfill)
	}
	if applied.SchemaVersion != journalFirstMigrationVersion {
		t.Fatalf("apply schema version = %d, want %d", applied.SchemaVersion, journalFirstMigrationVersion)
	}

	// sessions and session_state_snapshots must be gone.
	if rawTableExists(t, status.DatabasePath, "sessions") {
		t.Fatalf("sessions table still present after apply")
	}
	if rawTableExists(t, status.DatabasePath, "session_state_snapshots") {
		t.Fatalf("session_state_snapshots table still present after apply")
	}

	// journal_entries.session_id column must be gone; harness_session_id present.
	if rawColumnExists(t, status.DatabasePath, "journal_entries", "session_id") {
		t.Fatalf("journal_entries.session_id column still present after apply")
	}
	if !rawColumnExists(t, status.DatabasePath, "journal_entries", "harness_session_id") {
		t.Fatalf("journal_entries.harness_session_id column missing after apply")
	}

	// handoffs.session_id gone; harness_session_id present and backfilled.
	if rawColumnExists(t, status.DatabasePath, "handoffs", "session_id") {
		t.Fatalf("handoffs.session_id column still present after apply")
	}
	if !rawColumnExists(t, status.DatabasePath, "handoffs", "harness_session_id") {
		t.Fatalf("handoffs.harness_session_id column missing after apply")
	}
	if got := rawString(t, status.DatabasePath, `SELECT harness_session_id FROM handoffs WHERE id = 'handoff-1'`); got != "hsid-abc" {
		t.Fatalf("handoff harness_session_id after apply = %q, want hsid-abc", got)
	}

	// Backfilled harness ids on journal rows.
	if got := rawString(t, status.DatabasePath, `SELECT harness_session_id FROM journal_entries WHERE id = 'j-backfill'`); got != "hsid-abc" {
		t.Fatalf("j-backfill harness_session_id after apply = %q, want hsid-abc", got)
	}
	if got := rawString(t, status.DatabasePath, `SELECT harness_session_id FROM journal_entries WHERE id = 'j-wrap'`); got != "hsid-abc" {
		t.Fatalf("j-wrap harness_session_id after apply = %q, want hsid-abc", got)
	}
	// Preset harness id untouched.
	if got := rawString(t, status.DatabasePath, `SELECT harness_session_id FROM journal_entries WHERE id = 'j-preset'`); got != "hsid-preset" {
		t.Fatalf("j-preset harness_session_id after apply = %q, want hsid-preset", got)
	}

	// Noise purged; wrap synthesis preserved; non-noise entries all survive.
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_entries WHERE entry_type = 'session'`); got != 0 {
		t.Fatalf("session-type journal rows after apply = %d, want 0", got)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_entries WHERE id = 'j-wrap'`); got != 1 {
		t.Fatalf("j-wrap survived count = %d, want 1", got)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_entries`); got != 5 {
		t.Fatalf("journal_entries after apply = %d, want 5", got)
	}

	// events/aliases session rows deleted; non-session rows kept.
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM events WHERE entity_kind = 'session'`); got != 0 {
		t.Fatalf("session events after apply = %d, want 0", got)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM events WHERE entity_kind = 'spec'`); got != 1 {
		t.Fatalf("spec events after apply = %d, want 1 (preserved)", got)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM aliases WHERE entity_kind = 'session'`); got != 0 {
		t.Fatalf("session aliases after apply = %d, want 0", got)
	}

	// FTS rebuilt with harness_session_id column, repopulated, still matches.
	if !rawColumnExists(t, status.DatabasePath, "journal_search", "harness_session_id") {
		t.Fatalf("journal_search.harness_session_id column missing after apply")
	}
	if rawColumnExists(t, status.DatabasePath, "journal_search", "session_id") {
		t.Fatalf("journal_search.session_id column still present after apply")
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_search`); got != 5 {
		t.Fatalf("journal_search rows after apply = %d, want 5 (== surviving journal rows)", got)
	}
	if got := rawString(t, status.DatabasePath, `SELECT journal_entry_id FROM journal_search WHERE journal_search MATCH 'abandoned'`); got != "j-wrap" {
		t.Fatalf("FTS match for surviving wrap = %q, want j-wrap", got)
	}

	// Integrity + FK checks clean.
	if got := rawString(t, status.DatabasePath, `PRAGMA integrity_check`); got != "ok" {
		t.Fatalf("integrity_check after apply = %q, want ok", got)
	}
	if n := rawForeignKeyViolations(t, status.DatabasePath); n != 0 {
		t.Fatalf("foreign_key_check violations after apply = %d, want 0", n)
	}

	// schema_migrations records version 10.
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM schema_migrations WHERE version = 10`); got != 1 {
		t.Fatalf("schema_migrations version 10 rows = %d, want 1", got)
	}
}

func TestApplyJournalFirstMigrationIsIdempotent(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	status, err := Initialize(ctx, root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedJournalFirstFixture(t, status.DatabasePath, status.ProjectID)

	if _, err := ApplyJournalFirstMigration(ctx, root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("first ApplyJournalFirstMigration() error = %v", err)
	}
	// Re-open the migrated store directly and re-run the shared migration set:
	// migration 10 is checksum-recorded, so re-applying must be a no-op.
	store, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	if err := ApplyMigrations(ctx, store.db, append(SchemaMigrations(), JournalFirstMigration())); err != nil {
		t.Fatalf("idempotent re-apply error = %v", err)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM schema_migrations WHERE version = 10`); got != 1 {
		t.Fatalf("schema_migrations version 10 rows after re-apply = %d, want 1", got)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_entries`); got != 5 {
		t.Fatalf("journal_entries after re-apply = %d, want 5 (unchanged)", got)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_search`); got != 5 {
		t.Fatalf("journal_search after re-apply = %d, want 5 (unchanged)", got)
	}
}

func TestInspectAcceptsMigratedJournalFirstDatabase(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedJournalFirstFixture(t, status.DatabasePath, status.ProjectID)

	if _, err := ApplyJournalFirstMigration(ctx, root, resolver); err != nil {
		t.Fatalf("ApplyJournalFirstMigration() error = %v", err)
	}

	migrated, err := Inspect(root, resolver)
	if err != nil {
		t.Fatalf("Inspect() after migration error = %v", err)
	}
	if migrated.Mode != ModeSQLiteReady {
		t.Fatalf("Inspect().Mode after migration = %q, want %q; diagnostics = %+v", migrated.Mode, ModeSQLiteReady, migrated.Diagnostics)
	}
	if migrated.SchemaVersion != journalFirstMigrationVersion {
		t.Fatalf("Inspect().SchemaVersion after migration = %d, want %d", migrated.SchemaVersion, journalFirstMigrationVersion)
	}
	for _, diagnostic := range migrated.Diagnostics {
		if diagnostic.Code == "schema-version-mismatch" || diagnostic.Code == "schema-checksum-mismatch" || diagnostic.Code == "schema-migration-missing" {
			t.Fatalf("Inspect() after migration reported schema diagnostic %q: %s", diagnostic.Code, diagnostic.Message)
		}
	}
}

func TestApplyJournalFirstMigrationReRunIsCleanNoOp(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedJournalFirstFixture(t, status.DatabasePath, status.ProjectID)

	if _, err := ApplyJournalFirstMigration(ctx, root, resolver); err != nil {
		t.Fatalf("first ApplyJournalFirstMigration() error = %v", err)
	}

	// The real command entry point must remain reachable on an already-migrated
	// (v10) database: Inspect must report ModeSQLiteReady, not ModeInvalid, so
	// the re-run is an idempotent no-op rather than an "invalid database" error.
	second, err := ApplyJournalFirstMigration(ctx, root, resolver)
	if err != nil {
		t.Fatalf("second ApplyJournalFirstMigration() error = %v", err)
	}
	if second.JournalEntriesAfter != 5 || second.NoiseEntriesPurged != 0 {
		t.Fatalf("re-run counts after/purged = %d/%d, want 5/0 (noise already purged)", second.JournalEntriesAfter, second.NoiseEntriesPurged)
	}
	if second.SchemaVersion != journalFirstMigrationVersion {
		t.Fatalf("re-run schema version = %d, want %d", second.SchemaVersion, journalFirstMigrationVersion)
	}

	// Dry-run must also stay reachable on a migrated database.
	preview, err := PreviewJournalFirstMigration(ctx, root, resolver)
	if err != nil {
		t.Fatalf("PreviewJournalFirstMigration() on migrated database error = %v", err)
	}
	if !preview.CopyRun || preview.JournalEntriesAfter != 5 {
		t.Fatalf("preview copy_run/after on migrated database = %t/%d, want true/5", preview.CopyRun, preview.JournalEntriesAfter)
	}

	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM journal_entries`); got != 5 {
		t.Fatalf("journal_entries after re-run = %d, want 5 (unchanged)", got)
	}
	if got := rawCount(t, status.DatabasePath, `SELECT COUNT(*) FROM schema_migrations WHERE version = 10`); got != 1 {
		t.Fatalf("schema_migrations version 10 rows after re-run = %d, want 1", got)
	}
}

func TestJournalExportSucceedsAfterJournalFirstMigration(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedJournalFirstFixture(t, status.DatabasePath, status.ProjectID)

	if _, err := ApplyJournalFirstMigration(ctx, root, resolver); err != nil {
		t.Fatalf("ApplyJournalFirstMigration() error = %v", err)
	}

	// Spec Test Condition (line 91): `loaf journal export` produces valid
	// markdown and JSONL for a project — including on a migrated (v10) database.
	md, err := ExportJournalMarkdown(ctx, root, resolver)
	if err != nil {
		t.Fatalf("ExportJournalMarkdown() after migration error = %v", err)
	}
	if !strings.Contains(md.Content, "tried X abandoned Y next Z") {
		t.Fatalf("markdown export missing surviving wrap synthesis; got:\n%s", md.Content)
	}
	if strings.Contains(md.Content, "=== SESSION STARTED ===") {
		t.Fatalf("markdown export contains purged noise; got:\n%s", md.Content)
	}

	jsonl, err := ExportJournalJSONL(ctx, root, resolver)
	if err != nil {
		t.Fatalf("ExportJournalJSONL() after migration error = %v", err)
	}
	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(jsonl.Content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("JSONL export line is not valid JSON: %v\nline: %s", err, line)
		}
		lines++
	}
	if lines != 5 {
		t.Fatalf("JSONL export line count after migration = %d, want 5 (surviving journal rows)", lines)
	}
}

func TestBackupVerifiesMigratedJournalFirstDatabase(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	stateHome := t.TempDir()
	resolver := PathResolver{StateHome: stateHome}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedJournalFirstFixture(t, status.DatabasePath, status.ProjectID)

	if _, err := ApplyJournalFirstMigration(ctx, root, resolver); err != nil {
		t.Fatalf("ApplyJournalFirstMigration() error = %v", err)
	}

	// Backup of a migrated (v10) database must verify, not reject on version.
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatalf("Backup() on migrated database error = %v", err)
	}
	if backup.SchemaVersion != journalFirstMigrationVersion {
		t.Fatalf("Backup().SchemaVersion = %d, want %d", backup.SchemaVersion, journalFirstMigrationVersion)
	}
	verification, err := VerifyBackup(ctx, backup.BackupPath)
	if err != nil {
		t.Fatalf("VerifyBackup() on migrated backup error = %v", err)
	}
	if verification.SchemaVersion != journalFirstMigrationVersion {
		t.Fatalf("VerifyBackup().SchemaVersion = %d, want %d", verification.SchemaVersion, journalFirstMigrationVersion)
	}
}

func TestJournalFirstMigrationExcludedFromAutoApply(t *testing.T) {
	if CurrentSchemaVersion() != 9 {
		t.Fatalf("CurrentSchemaVersion() = %d, want 9 (migration 10 must not auto-apply on store open)", CurrentSchemaVersion())
	}
	for _, m := range SchemaMigrations() {
		if m.Version == journalFirstMigrationVersion {
			t.Fatalf("journal-first migration %d must be excluded from SchemaMigrations()", journalFirstMigrationVersion)
		}
	}
	m := JournalFirstMigration()
	if m.Version != 10 || m.Name != "journal_first" {
		t.Fatalf("JournalFirstMigration() = %d/%q, want 10/journal_first", m.Version, m.Name)
	}
	if len(m.Checksum()) != 64 {
		t.Fatalf("JournalFirstMigration().Checksum() length = %d, want 64", len(m.Checksum()))
	}
}

// --- raw read helpers (temp-DB house pattern) ---

func rawCount(t *testing.T, databasePath string, query string, args ...any) int {
	t.Helper()
	store, err := OpenStore(databasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	var n int
	if err := store.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count query %q: %v", query, err)
	}
	return n
}

func rawString(t *testing.T, databasePath string, query string, args ...any) string {
	t.Helper()
	store, err := OpenStore(databasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	var s string
	if err := store.db.QueryRow(query, args...).Scan(&s); err != nil {
		t.Fatalf("string query %q: %v", query, err)
	}
	return s
}

func rawTableExists(t *testing.T, databasePath string, table string) bool {
	t.Helper()
	store, err := OpenStore(databasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	exists, err := sqliteTableExists(context.Background(), store.db, table)
	if err != nil {
		t.Fatalf("table exists %s: %v", table, err)
	}
	return exists
}

func rawColumnExists(t *testing.T, databasePath string, table string, column string) bool {
	t.Helper()
	store, err := OpenStore(databasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	var n int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n); err != nil {
		t.Fatalf("pragma_table_info %s: %v", table, err)
	}
	return n > 0
}

func rawForeignKeyViolations(t *testing.T, databasePath string) int {
	t.Helper()
	store, err := OpenStore(databasePath)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	defer store.Close()
	rows, err := store.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	return n
}
