package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/state"
	"github.com/levifig/loaf/vnext/migration/retired"
)

func TestRunnerRetiredRecoveryWithoutLiveDatabaseAndCorruptionRefusal(t *testing.T) {
	f := prepareVNextRehearsalCLIFixture(t)
	nativeIntent, err := state.CreateIntent(context.Background(), f.projectRoot, state.PathResolver{StateHome: f.stateHome}, state.IntentCreateOptions{Title: "Native intent", Body: "Original native snapshot body", Disposition: "tracked", OperationID: "retired-native-intent"})
	if err != nil {
		t.Fatal(err)
	}
	s, err := state.OpenStore(f.database)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte{0xff, 0, 13, 10, 32, 32}
	for _, kind := range []string{"spec", "intent", "task"} {
		if kind == "intent" {
			_, err = s.DB().Exec(`INSERT INTO intents(id,project_id,created_at) SELECT ?,project_id,created_at FROM journal_entries LIMIT 1`, kind)
		} else {
			_, err = s.DB().Exec(`INSERT INTO `+kind+`s(id,project_id,title,status,created_at,updated_at) SELECT ?,project_id,'Original','done',created_at,updated_at FROM journal_entries LIMIT 1`, kind)
		}
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.DB().Exec(`INSERT INTO artifact_bodies(id,project_id,entity_kind,entity_id,content,content_hash,created_at,updated_at) SELECT ?,project_id,?,?,CAST(? AS TEXT),'old-hash',created_at,updated_at FROM journal_entries LIMIT 1`, "body-"+kind, kind, kind, body)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	backup, err := state.Backup(context.Background(), f.projectRoot, state.PathResolver{StateHome: f.stateHome})
	if err != nil {
		t.Fatal(err)
	}
	before := rehearsalTestFileSHA256(t, f.database)
	var out bytes.Buffer
	destination := filepath.Join(t.TempDir(), "rehearsal")
	if err := (Runner{WorkingDir: f.workingDir, StateHome: f.stateHome, Stdout: &out}).Run([]string{"migrate", "vnext-rehearsal", "--backup", backup.BackupPath, "--to", destination, "--json"}); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	var result migrateVNextRehearsalResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.RetiredArchive == nil || len(result.RetiredArchive.Manifest.Tables) != len(state.ProjectScopedLiveTables()) {
		t.Fatal("missing recovery receipt")
	}
	archivePath := result.RetiredArchive.Path
	out.Reset()
	if err := (Runner{WorkingDir: t.TempDir(), Stdout: &out}).Run([]string{"migrate", "retired-show", "--archive", archivePath, "--kind", "intent", "--id", nativeIntent.Intent.ID, "--body"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != nativeIntent.Intent.Body {
		t.Fatal("native intent body unavailable")
	}
	out.Reset()
	if err := (Runner{WorkingDir: t.TempDir(), Stdout: &out}).Run([]string{"migrate", "retired-show", "--archive", archivePath, "--kind", "intent", "--id", nativeIntent.Intent.ID}); err != nil {
		t.Fatal(err)
	}
	var recovery struct {
		Related []struct {
			Name string `json:"name"`
		} `json:"related_tables"`
	}
	if err := json.Unmarshal(out.Bytes(), &recovery); err != nil {
		t.Fatal(err)
	}
	foundSnapshot := false
	for _, table := range recovery.Related {
		foundSnapshot = foundSnapshot || table.Name == "intent_snapshots"
	}
	if !foundSnapshot {
		t.Fatal("native intent histories unavailable")
	}
	if info, err := os.Stat(archivePath); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("archive permissions: %v", err)
	}
	// Deliberately use a different uninitialized state home and working directory.
	for _, kind := range []string{"spec", "intent", "task"} {
		out.Reset()
		if err := (Runner{WorkingDir: t.TempDir(), StateHome: filepath.Join(t.TempDir(), "absent"), Stdout: &out}).Run([]string{"migrate", "retired-show", "--archive", archivePath, "--kind", kind, "--id", kind, "--body"}); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.Bytes(), body) {
			t.Fatal("recovered bytes differ")
		}
	}
	encoded, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	encoded[len(encoded)/2] ^= 1
	corrupt := filepath.Join(t.TempDir(), "corrupt.json")
	if err := os.WriteFile(corrupt, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := (Runner{WorkingDir: t.TempDir(), Stdout: &out}).Run([]string{"migrate", "retired-show", "--archive", corrupt, "--kind", "spec", "--id", "spec", "--body"}); err == nil || out.Len() != 0 {
		t.Fatal("corrupt archive emitted content")
	}
	if rehearsalTestFileSHA256(t, f.database) != before {
		t.Fatal("source database changed")
	}
}

func TestRetiredRecoverySelectsSnapshotAndKeepsContractReceiptIdentity(t *testing.T) {
	cell := func(s string) retired.Cell { return retired.Cell{Type: "text", Data: []byte(s)} }
	table := func(name string, columns []string, values ...[]string) retired.Table {
		tb := retired.Table{Name: name, Columns: columns, SourceRows: len(values), Rows: []retired.Row{}}
		for _, values := range values {
			row := retired.Row{Disposition: "retired", Values: []retired.Cell{}}
			for _, v := range values {
				row.Values = append(row.Values, cell(v))
			}
			tb.Rows = append(tb.Rows, row)
		}
		return tb
	}
	content := retired.Content{ProjectID: "project", BackupSHA256: strings.Repeat("a", 64), ContinuitySHA256: strings.Repeat("b", 64), SchemaVersion: 27, Tables: []retired.Table{
		table("intents", []string{"id", "project_id"}, []string{"intent", "project"}),
		table("intent_snapshots", []string{"id", "project_id", "intent_id", "body"}, []string{"snapshot1", "project", "intent", "original\r\n"}, []string{"snapshot2", "project", "intent", "later\n"}),
		table("work_contracts", []string{"id", "project_id", "provider", "provider_ref"}, []string{"contract", "project", "pr", "public-ref"}),
		table("artifact_bodies", []string{"id", "project_id", "entity_kind", "entity_id", "content"}, []string{"contract-body", "project", "work_contract", "contract", "original contract body"}),
		table("work_contract_receipts", []string{"id", "project_id", "provider", "provider_ref", "receipt_value"}, []string{"receipt", "project", "pr", "public-ref", "original receipt"}, []string{"foreign", "project", "pr", "other-ref", "unrelated"}),
		table("issue_identity", []string{"id", "project_id", "authority"}, []string{"identity", "project", "github"}),
	}}
	encoded, err := retired.Seal(content)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "retired.json")
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	args := []string{"migrate", "retired-show", "--archive", path, "--kind", "intent", "--id", "intent", "--body"}
	if err := (Runner{WorkingDir: t.TempDir(), Stdout: &out}).Run(args); err == nil || out.Len() != 0 {
		t.Fatal("ambiguous snapshot emitted body")
	}
	if err := (Runner{WorkingDir: t.TempDir(), Stdout: &out}).Run(append(args, "--body-id", "snapshot1")); err != nil || out.String() != "original\r\n" {
		t.Fatalf("explicit snapshot recovery: %v", err)
	}
	a, err := retired.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	related := retiredRelatedTables(a, "work-contract", "contract")
	if len(related) != 1 || related[0].Name != "work_contract_receipts" || len(related[0].Rows) != 1 {
		t.Fatal("provider-qualified receipt lost or crossed authority")
	}
	related = retiredRelatedTables(a, "issue", "any-issue")
	if len(related) != 1 || related[0].Name != "issue_identity" {
		t.Fatal("project-wide issue identity lost")
	}
	out.Reset()
	if err := (Runner{WorkingDir: t.TempDir(), Stdout: &out}).Run([]string{"migrate", "retired-show", "--archive", path, "--kind", "work-contract", "--id", "contract", "--body"}); err != nil || out.String() != "original contract body" {
		t.Fatalf("original contract body: %v", err)
	}
}
