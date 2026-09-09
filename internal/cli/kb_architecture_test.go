package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerKbDiscoversArchitectureWithoutKnowledgeMetadata(t *testing.T) {
	for _, legacyConfig := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "legacy defaults"}[legacyConfig], func(t *testing.T) {
			repo := writeKbValidateValidFixture(t)
			originalConfig := `{"knowledge":{"local":["docs/knowledge","docs/decisions"]}}`
			if legacyConfig {
				mkdirAll(t, filepath.Join(repo, ".agents"))
				writeFile(t, filepath.Join(repo, ".agents", "loaf.json"), originalConfig)
			}
			architecture := map[string]string{
				"docs/architecture/README.md":             "# Architecture index\n",
				"docs/architecture/legacy-metadata.md":    "---\ntopics: [runtime]\nlast_reviewed: 2000-01-01\ncovers: [src/main.go]\n---\n# Runtime\n",
				"docs/ARCHITECTURE.md":                    "# System Map\n\nRead the owning topics.\n",
				"docs/architecture/runtime.md":            "# Runtime\n\nGo owns deterministic operations.\n",
				"docs/architecture/storage/identity.md":   "# Identity\n\nIdentifiers are stable.\n",
				"docs/decisions/ADR-001-runtime.md":       "---\nid: ADR-001\ntitle: Go runtime\nstatus: Accepted\ndate: 2026-01-01\n---\n# ADR-001: Go runtime\n",
				"docs/decisions/keep-the-relay-opaque.md": "# Keep the relay opaque\n\nDecision status: accepted\n",
			}
			for name, body := range architecture {
				mkdirAll(t, filepath.Dir(filepath.Join(repo, name)))
				writeFile(t, filepath.Join(repo, name), body)
			}
			var stdout, stderr bytes.Buffer
			runner := Runner{Stdout: &stdout, Stderr: &stderr, WorkingDir: repo}
			if err := runner.Run([]string{"kb", "validate", "--json"}); err != nil {
				t.Fatalf("validate: %v\n%s", err, stdout.String())
			}
			var results []kbValidationResult
			if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
				t.Fatal(err)
			}
			if len(results) != len(architecture)+1 {
				t.Fatalf("results = %#v, want all architecture and knowledge files", results)
			}
			for _, result := range results {
				if len(result.Errors) != 0 {
					t.Fatalf("unexpected errors: %#v", result)
				}
			}
			stdout.Reset()
			if err := runner.Run([]string{"kb", "status", "--json"}); err != nil {
				t.Fatal(err)
			}
			var summary kbStatusSummary
			if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
				t.Fatal(err)
			}
			if summary.TotalFiles != len(architecture)+1 || summary.ArchitectureFiles != len(architecture) || summary.FilesWithCovers != 1 || summary.FilesWithoutCovers != 0 || summary.AvgReviewAgeDays != 0 {
				t.Fatalf("summary = %#v", summary)
			}
			stdout.Reset()
			if err := runner.Run([]string{"kb", "check", "--json"}); err != nil {
				t.Fatal(err)
			}
			var stale []kbStalenessResult
			if err := json.Unmarshal(stdout.Bytes(), &stale); err != nil {
				t.Fatal(err)
			}
			if len(stale) != 1 || stale[0].File != "docs/knowledge/go.md" {
				t.Fatalf("staleness = %#v, want knowledge only", stale)
			}
			for name, original := range architecture {
				body, err := os.ReadFile(filepath.Join(repo, name))
				if err != nil || string(body) != original {
					t.Fatalf("read-only discovery changed %s: %v", name, err)
				}
			}
			if legacyConfig {
				body, err := os.ReadFile(filepath.Join(repo, ".agents", "loaf.json"))
				if err != nil || string(body) != originalConfig {
					t.Fatalf("legacy config mutated: %v", err)
				}
			}
		})
	}
}

func TestRunnerKbArchitectureValidationRejectsBrokenDocuments(t *testing.T) {
	repo := initCLIGitRepo(t)
	for name, body := range map[string]string{
		"docs/architecture/empty.md": " \n",
		"docs/decisions/broken.md":   "---\nstatus: Accepted\n# Unclosed frontmatter\n",
	} {
		mkdirAll(t, filepath.Dir(filepath.Join(repo, name)))
		writeFile(t, filepath.Join(repo, name), body)
	}
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: repo}).Run([]string{"kb", "validate", "--json"})
	if err == nil {
		t.Fatal("validate accepted empty or unterminated documents")
	}
	var results []kbValidationResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	for _, result := range results {
		if len(result.Errors) == 0 || hasValidationIssue(result.Errors, "topics") || hasValidationIssue(result.Errors, "last_reviewed") {
			t.Fatalf("wrong document validation: %#v", result)
		}
	}
}

func TestRunnerKbReviewRefusesArchitectureEvenWithTopics(t *testing.T) {
	for _, name := range []string{"docs/ARCHITECTURE.md", "docs/architecture/runtime.md", "docs/decisions/runtime.md"} {
		t.Run(name, func(t *testing.T) {
			repo := initCLIGitRepo(t)
			path := filepath.Join(repo, name)
			mkdirAll(t, filepath.Dir(path))
			original := "---\ntopics: [runtime]\n---\n# Runtime\n"
			writeFile(t, path, original)
			var stdout bytes.Buffer
			err := (Runner{Stdout: &stdout, WorkingDir: repo}).Run([]string{"kb", "review", name, "--json"})
			if err == nil || !strings.Contains(stdout.String(), "architecture") {
				t.Fatalf("review = %v, output = %s", err, stdout.String())
			}
			body, err := os.ReadFile(path)
			if err != nil || string(body) != original {
				t.Fatalf("review changed architecture metadata: %v", err)
			}
		})
	}
}

func TestKbArchitectureDiscoveryPreservesCustomPaths(t *testing.T) {
	repo := initCLIGitRepo(t)
	mkdirAll(t, filepath.Join(repo, ".agents"))
	original := `{"knowledge":{"local":["guides"],"staleness_threshold_days":14}}`
	writeFile(t, filepath.Join(repo, ".agents", "loaf.json"), original)
	config := loadNativeKbConfig(repo)
	if strings.Join(config.Local, ",") != "guides" || config.StalenessThresholdDays != 14 {
		t.Fatalf("custom config replaced: %#v", config)
	}
	for name, body := range map[string]string{
		"guides/domain.md":         "---\ntopics: [domain]\nlast_reviewed: 2026-01-01\n---\n# Domain\n",
		"docs/ARCHITECTURE.md":     "# Overview\n",
		"docs/architecture/app.md": "# App\n",
	} {
		mkdirAll(t, filepath.Dir(filepath.Join(repo, name)))
		writeFile(t, filepath.Join(repo, name), body)
	}
	var stderr bytes.Buffer
	files := loadNativeKnowledgeFiles(repo, config, &stderr, true)
	if len(files) != 1 || files[0].RelativePath != "guides/domain.md" {
		t.Fatalf("custom discovery escaped configured paths: %#v", files)
	}
	body, err := os.ReadFile(filepath.Join(repo, ".agents", "loaf.json"))
	if err != nil || string(body) != original {
		t.Fatalf("config mutated: %v", err)
	}
}

func TestKbArchitectureDocumentStructure(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "plain", body: "# Model\n"},
		{name: "CRLF", body: "---\r\nstatus: accepted\r\n---\r\n# Decision\r\n"},
		{name: "no metadata required", body: "---\n---\n# Topic\n"},
		{name: "empty", want: "Document body must not be empty"},
		{name: "metadata only", body: "---\nstatus: accepted\n---\n", want: "Document body must not be empty"},
		{name: "unclosed", body: "---", want: "Unterminated frontmatter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := architectureDocumentError([]byte(tc.body)); got != tc.want {
				t.Fatalf("error = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKbArchitectureDiscoveryDeduplicatesAndDoesNotFollowDirectorySymlinks(t *testing.T) {
	repo := initCLIGitRepo(t)
	mkdirAll(t, filepath.Join(repo, "docs", "architecture", "nested"))
	writeFile(t, filepath.Join(repo, "docs", "ARCHITECTURE.md"), "# Overview\n")
	writeFile(t, filepath.Join(repo, "docs", "architecture", "nested", "app.md"), "# App\n")
	external := t.TempDir()
	writeFile(t, filepath.Join(external, "outside.md"), "# Outside\n")
	if err := os.Symlink(external, filepath.Join(repo, "docs", "architecture", "linked")); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	var stderr bytes.Buffer
	config := kbConfig{Local: []string{"docs/architecture", "docs/architecture/nested", "docs/architecture"}}
	files := loadNativeKnowledgeFiles(repo, config, &stderr, true)
	if len(files) != 2 || files[0].RelativePath != "docs/ARCHITECTURE.md" || files[1].RelativePath != "docs/architecture/nested/app.md" {
		t.Fatalf("discovery = %#v, warnings = %s", files, stderr.String())
	}
}
