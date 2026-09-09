package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerKbReviewRespectsResolvedDocumentRole(t *testing.T) {
	for _, tc := range []struct {
		name, target                    string
		directoryLink, outside, allowed bool
	}{
		{name: "topic alias", target: "docs/architecture/model.md"},
		{name: "overview alias", target: "docs/ARCHITECTURE.md"},
		{name: "decision alias", target: "docs/decisions/ADR-001-model.md"},
		{name: "directory alias", target: "docs/architecture/model.md", directoryLink: true},
		{name: "outside repository", target: "model.md", outside: true},
		{name: "knowledge alias", target: "docs/knowledge/model.md", allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := initCLIGitRepo(t)
			targetRoot := repo
			if tc.outside {
				targetRoot = t.TempDir()
			}
			target := filepath.Join(targetRoot, tc.target)
			mkdirAll(t, filepath.Dir(target))
			original := "---\ntopics: [runtime]\n---\n# Model\n"
			writeFile(t, target, original)
			alias := filepath.Join(repo, "docs", "knowledge", "alias.md")
			mkdirAll(t, filepath.Dir(alias))
			link, destination := alias, target
			if tc.directoryLink {
				link = filepath.Join(repo, "docs", "knowledge", "linked")
				destination = filepath.Dir(target)
				alias = filepath.Join(link, filepath.Base(target))
			}
			if err := os.Symlink(destination, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			var stdout bytes.Buffer
			err := (Runner{Stdout: &stdout, WorkingDir: repo}).Run([]string{"kb", "review", alias, "--json"})
			body, readErr := os.ReadFile(target)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tc.allowed {
				if err != nil || !strings.Contains(string(body), "last_reviewed:") {
					t.Fatalf("knowledge alias review: %v, %s", err, stdout.String())
				}
			} else if err == nil || string(body) != original {
				t.Fatalf("review should refuse without changing target: err=%v body=%s output=%s", err, body, stdout.String())
			}
		})
	}
}

func TestRunnerKbValidateReportsReadFailures(t *testing.T) {
	for _, tc := range []struct {
		name, path                  string
		brokenLink, deniedDirectory bool
	}{
		{name: "broken architecture link", path: "docs/architecture/broken.md", brokenLink: true},
		{name: "oversized architecture", path: "docs/architecture/large.md"},
		{name: "oversized knowledge", path: "docs/knowledge/large.md"},
		{name: "unreadable subtree", path: "docs/architecture/denied", deniedDirectory: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := initCLIGitRepo(t)
			path := filepath.Join(repo, tc.path)
			mkdirAll(t, filepath.Dir(path))
			switch {
			case tc.brokenLink:
				if err := os.Symlink("missing.md", path); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			case tc.deniedDirectory:
				skipWithoutEnforcedPermissions(t)
				mkdirAll(t, path)
				writeFile(t, filepath.Join(path, "model.md"), "# Model\n")
				chmodForTest(t, path, 0o000)
			default:
				writeFile(t, path, strings.Repeat("x", projectFileReadLimit+1))
			}
			var stdout, stderr bytes.Buffer
			err := (Runner{Stdout: &stdout, Stderr: &stderr, WorkingDir: repo}).Run([]string{"kb", "validate", "--json"})
			if err == nil {
				t.Fatalf("validation silently accepted unreadable input: %s", stdout.String())
			}
			var results []kbValidationResult
			if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
				t.Fatal(err)
			}
			if len(results) != 1 || results[0].File != tc.path || !hasValidationIssue(results[0].Errors, "read") {
				t.Fatalf("results must identify the failed read: %#v", results)
			}
		})
	}
}

func TestKnowledgeConfigRepairUsesCurrentDefaults(t *testing.T) {
	knowledge := map[string]any{}
	var problems []string
	ensureKnowledgeConfigDefaults(knowledge, &problems)
	local, ok := knowledge["local"].([]string)
	if !ok || strings.Join(local, ",") != "docs/knowledge,docs/architecture,docs/decisions" || len(problems) != 0 {
		t.Fatalf("repaired config = %#v, errors = %v", knowledge, problems)
	}
}
