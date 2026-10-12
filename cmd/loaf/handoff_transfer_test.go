package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/state"
)

func TestPublicBinaryHandoffTransferSurvivesFreshProcesses(t *testing.T) {
	binary := sharedTestLoafBinary(t, repoRoot(t))
	workingDir := createMainRepo(t, "handoff-source")
	env := isolatedInstallEnv(t)
	legacyDir := filepath.Join(workingDir, ".agents", "handoffs")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(legacyDir, "legacy-transfer.md")
	legacy := "---\nstatus: deprecated\n---\nExisting user transfer.\n"
	if err := os.WriteFile(legacyPath, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := runBinary(binary, workingDir, env, "state", "init", "--json"); err != nil {
		t.Fatalf("initialize isolated store: %v\n%s", err, output)
	}
	packet := "## Purpose and Scope\nInspect retry-evidence-fixture; read-only, no implementation.\n\n## Current State and Evidence\nRead README.md again; this snapshot may be stale.\n\n## Next Action\nIdentify the current retry recommendation.\n"
	packetPath := filepath.Join(t.TempDir(), "transfer-body.md")
	if err := os.WriteFile(packetPath, []byte(packet), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := runBinary(binary, workingDir, env, "handoff", "new", "--title", "Retry evidence", "--body-file", packetPath, "--json")
	if err != nil {
		t.Fatalf("persist transfer: %v\n%s", err, output)
	}
	var created state.ArtifactEntityCreateResult
	if err := json.Unmarshal([]byte(output), &created); err != nil {
		t.Fatal(err)
	}
	if created.Entity.ID == "" || created.Entity.Alias == "" || created.ProjectID == "" {
		t.Fatalf("incomplete transfer address: %+v", created)
	}
	// Each invocation is a new CLI process; neither the staging file nor an
	// authored handoff file is necessary for a recipient to retrieve the body.
	if err := os.Remove(packetPath); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{created.Entity.ID, created.Entity.Alias} {
		output, err := runBinary(binary, workingDir, env, "handoff", "show", ref, "--json")
		if err != nil {
			t.Fatalf("fresh recipient show %s: %v\n%s", ref, err, output)
		}
		var got state.ArtifactEntityShow
		if err := json.Unmarshal([]byte(output), &got); err != nil {
			t.Fatal(err)
		}
		if got.Entity.ID != created.Entity.ID || got.ProjectID != created.ProjectID || got.Entity.Body != packet || len(got.Entity.Sources) != 0 {
			t.Fatalf("retrieved transfer differs: %+v", got)
		}
	}
	for _, discovery := range []struct {
		args []string
		ref  string
	}{
		{[]string{"handoff", "list", "--json"}, created.Entity.Alias},
		{[]string{"search", "retry-evidence-fixture", "--json"}, created.Entity.ID},
	} {
		output, err := runBinary(binary, workingDir, env, discovery.args...)
		if err != nil || !strings.Contains(output, discovery.ref) {
			t.Fatalf("discover %v: %v\n%s", discovery.args, err, output)
		}
	}
	if got, err := os.ReadFile(legacyPath); err != nil || string(got) != legacy {
		t.Fatalf("legacy transfer changed: %v\n%s", err, got)
	}
	entries, err := os.ReadDir(legacyDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected handoff files: %v, %v", entries, err)
	}
	otherProject := createMainRepo(t, "handoff-recipient")
	if output, err := runBinary(binary, otherProject, env, "handoff", "show", created.Entity.ID, "--json"); err == nil {
		t.Fatalf("local ID falsely retrieved from another project: %s", output)
	}
	missingEnv := isolatedInstallEnv(t)
	if output, err := runBinary(binary, workingDir, missingEnv, "handoff", "show", created.Entity.ID, "--json"); err == nil {
		t.Fatalf("unavailable store falsely retrieved transfer: %s", output)
	}
	if _, err := os.Stat(envValue(t, missingEnv, "LOAF_DB")); !os.IsNotExist(err) {
		t.Fatalf("unavailable-store read created fallback state: %v", err)
	}
}
