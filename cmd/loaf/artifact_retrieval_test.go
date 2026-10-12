package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/state"
)

func TestPublicBinaryDistinguishesJournalCouncilAndExplicitFiles(t *testing.T) {
	binary := sharedTestLoafBinary(t, repoRoot(t))
	root := createMainRepo(t, "artifact-routing")
	env := isolatedInstallEnv(t)
	if output, err := runBinary(binary, root, env, "state", "init"); err != nil {
		t.Fatalf("initialize: %v\n%s", err, output)
	}
	message := "journal-only-fixture: Rejected a shared queue because retry scope is local."
	output, err := runBinary(binary, root, env, "journal", "log", "wrap(retry): "+message, "--json")
	if err != nil {
		t.Fatalf("capture checkpoint: %v\n%s", err, output)
	}
	var entry state.JournalLogResult
	if err := json.Unmarshal([]byte(output), &entry); err != nil || entry.ID == "" {
		t.Fatalf("checkpoint address: %v\n%s", err, output)
	}
	output, err = runBinary(binary, root, env, "journal", "search", "journal-only-fixture", "--json")
	if err != nil || !strings.Contains(output, entry.ID) {
		t.Fatalf("discover journal-only checkpoint: %v\n%s", err, output)
	}
	output, err = runBinary(binary, root, env, "journal", "show", entry.ID, "--json")
	var checkpoint state.JournalShow
	if err != nil || json.Unmarshal([]byte(output), &checkpoint) != nil || checkpoint.Entry.Message != message {
		t.Fatalf("read exact checkpoint: %v\n%s", err, output)
	}
	if output, err := runBinary(binary, root, env, "handoff", "show", entry.ID, "--json"); err == nil {
		t.Fatalf("journal entry falsely treated as a Handoff body: %s", output)
	}
	councilBody := "# Retry options\nKeep retries local; preserve the dissent about shared capacity."
	output, err = runBinary(binary, root, env, "council", "new", "--title", "Retry options", "--message", councilBody, "--json")
	var council state.ArtifactEntityCreateResult
	if err != nil || json.Unmarshal([]byte(output), &council) != nil || council.Entity.Alias == "" {
		t.Fatalf("create file council: %v\n%s", err, output)
	}
	for _, verb := range []string{"list", "show"} {
		args := []string{"council", verb}
		if verb == "show" {
			args = append(args, council.Entity.Alias)
		}
		output, err = runBinary(binary, root, env, append(args, "--json")...)
		if err != nil || !strings.Contains(output, council.Entity.Alias) {
			t.Fatalf("discover/read file council: %v\n%s", err, output)
		}
	}
	var shown state.ArtifactEntityShow
	if json.Unmarshal([]byte(output), &shown) != nil || shown.Entity.Body != councilBody || len(shown.Entity.Sources) != 1 {
		t.Fatalf("council body/source differs: %s", output)
	}
	councilPath := filepath.Join(root, filepath.FromSlash(shown.Entity.Sources[0].Path))
	fileBody, err := os.ReadFile(councilPath)
	if err != nil || !strings.Contains(string(fileBody), councilBody) {
		t.Fatalf("council body missing from authored file: %v", err)
	}
	snapshotPath := filepath.Join(t.TempDir(), "portable-transfer.md")
	snapshot := "# Portable snapshot\nSource store unavailable here; re-read current retry policy before acting.\n"
	if err := os.WriteFile(snapshotPath, []byte(snapshot), 0o600); err != nil {
		t.Fatal(err)
	}
	missingEnv := isolatedInstallEnv(t)
	for _, args := range [][]string{{"journal", "show", entry.ID}, {"council", "show", council.Entity.Alias}, {"handoff", "new", "--title", "Missing store", "--message", snapshot}} {
		if output, err := runBinary(binary, root, missingEnv, args...); err == nil {
			t.Fatalf("unavailable store silently substituted for %v: %s", args, output)
		}
	}
	for path, want := range map[string]string{councilPath: string(fileBody), snapshotPath: snapshot} {
		if got, err := os.ReadFile(path); err != nil || string(got) != want {
			t.Fatalf("explicit authored evidence changed or unreadable: %s: %v", path, err)
		}
	}
	if _, err := os.Stat(envValue(t, missingEnv, "LOAF_DB")); !os.IsNotExist(err) {
		t.Fatalf("missing-store command created fallback state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "handoffs")); !os.IsNotExist(err) {
		t.Fatalf("journal checkpoint manufactured a handoff directory: %v", err)
	}
}
