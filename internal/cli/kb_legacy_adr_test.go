package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func legacyADRFixture(status, extra, sections string) string {
	return "---\nid: ADR-001\ntitle: Runtime choice\nstatus: " + status + "\ndate: 2026-01-01\n" + extra + "---\n# ADR-001: Runtime choice\n\n## Context\nContext.\n## Decision\nChoice.\n## Consequences\nTradeoffs.\n## Alternatives Considered\nAlternatives.\n" + sections
}

func TestRunnerKbLegacyADRValidationIsExplicitAndNative(t *testing.T) {
	for _, tc := range []struct{ name, status, extra, sections string }{
		{name: "proposed", status: "Proposed"},
		{name: "accepted", status: "Accepted"},
		{name: "rejected", status: "Rejected", extra: "rejected_date: 2026-02-01\n", sections: "## Rejected\nReason.\n"},
		{name: "deprecated", status: "Deprecated", extra: "deprecated_date: 2026-02-01\n", sections: "## Deprecated\nReason.\n"},
		{name: "superseded", status: "Superseded", extra: "superseded_by:\n  - ADR-002\n  - ADR-003\n", sections: "## Superseded\nReplacement.\n"},
		{name: "revised", status: "Accepted", extra: "revised: 2026-02-01\n", sections: "## Revisions\n2026-02-01: Clarified.\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			name := "ADR-001-runtime.md"
			writeFile(t, filepath.Join(root, name), legacyADRFixture(tc.status, tc.extra, tc.sections))
			// Explicit file validation must need neither Git nor an interpreter.
			t.Setenv("PATH", t.TempDir())
			var stdout bytes.Buffer
			err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"kb", "validate", "--legacy-adr", name, "--json"})
			if err != nil {
				t.Fatalf("native legacy validation: %v\n%s", err, stdout.String())
			}
			var results []kbValidationResult
			if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
				t.Fatal(err)
			}
			if len(results) != 1 || results[0].File != name || len(results[0].Errors) != 0 {
				t.Fatalf("results = %#v", results)
			}
		})
	}
}

func TestRunnerKbLegacyADRValidationReportsAllFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ADR-001-runtime.md"), legacyADRFixture("Rejected", "revised: invalid-date\n", ""))
	writeFile(t, filepath.Join(root, "topic.md"), "# A living topic\n")
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"kb", "validate", "--legacy-adr", "ADR-001-runtime.md", "topic.md", "missing.md", "--json"})
	if err == nil {
		t.Fatal("invalid records accepted")
	}
	var results []kbValidationResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || !hasValidationIssue(results[2].Errors, "read") {
		t.Fatalf("results = %#v", results)
	}
	for _, field := range []string{"revised", "rejected_date", "sections"} {
		if !hasValidationIssue(results[0].Errors, field) {
			t.Fatalf("missing %s error: %#v", field, results[0])
		}
	}
	if !hasValidationIssue(results[1].Errors, "frontmatter") || !hasValidationIssue(results[1].Errors, "filename") {
		t.Fatalf("topic accepted as legacy ADR: %#v", results[1])
	}
}

func TestRunnerKbLegacyADRValidationRequiresFiles(t *testing.T) {
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: t.TempDir()}).Run([]string{"kb", "validate", "--legacy-adr"})
	if err == nil || !strings.Contains(err.Error(), "file") {
		t.Fatalf("missing file: %v", err)
	}
}

func TestLegacyADRValidationPreservesHistoricalChecks(t *testing.T) {
	valid := legacyADRFixture("Accepted", "", "")
	for _, tc := range []struct{ name, old, replacement, field string }{
		{"id format", "id: ADR-001", "id: ADR-1", "id"},
		{"id mismatch", "id: ADR-001", "id: ADR-002", "id"},
		{"title", "title: Runtime choice", "title:", "title"},
		{"heading", "# ADR-001: Runtime choice", "# Runtime choice", "heading"},
		{"status", "status: Accepted", "status: shipped", "status"},
		{"missing date", "date: 2026-01-01\n", "", "date"},
		{"date format", "date: 2026-01-01", "date: yesterday", "date"},
		{"context", "## Context", "## Background", "sections"},
		{"decision", "## Decision", "## Choice", "sections"},
		{"consequences", "## Consequences", "## Effects", "sections"},
		{"alternatives", "## Alternatives Considered", "## Alternatives", "sections"},
		{"superseded reference", "status: Accepted", "status: Superseded", "superseded_by"},
		{"unsupported metadata", "status: Accepted", "status: Accepted\ninvalid line", "frontmatter"},
		{"unterminated metadata", "\n---\n#", "\n#", "frontmatter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errors := validateLegacyADR("ADR-001-runtime.md", strings.Replace(valid, tc.old, tc.replacement, 1))
			if !hasValidationIssue(errors, tc.field) {
				t.Fatalf("missing %s error: %#v", tc.field, errors)
			}
		})
	}
	// These are deliberately shape-only checks, not new calendar or YAML rules.
	for _, date := range []string{"", "2026-99-99"} {
		if errors := validateLegacyADR("ADR-001-runtime.md", strings.Replace(valid, "date: 2026-01-01", "date: "+date, 1)); len(errors) != 0 {
			t.Fatalf("historically accepted date %q: %#v", date, errors)
		}
	}
}

func TestLegacyADRFrontmatterSubset(t *testing.T) {
	content := "---\nid: 'ADR-001' # comment\ntitle: \"Runtime choice\"\nsuperseded_by:\n  - 'ADR-002' # comment\n  - ADR-003\n# ignored\n  nested: ignored\n---\n"
	values, errors := legacyADRFrontmatter(content)
	if len(errors) != 0 || values["id"] != "ADR-001" || values["title"] != "Runtime choice" || values["superseded_by"] != "ADR-002, ADR-003" {
		t.Fatalf("values=%#v errors=%#v", values, errors)
	}
}

func TestLegacyADRFileReadBoundaries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ADR-001-crlf.md"), strings.ReplaceAll(legacyADRFixture("Accepted", "", ""), "\n", "\r\n"))
	writeFile(t, filepath.Join(root, "invalid.md"), string([]byte{0xff}))
	writeFile(t, filepath.Join(root, "large.md"), strings.Repeat("x", projectFileReadLimit+1))
	if err := os.Mkdir(filepath.Join(root, "directory.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	results := validateLegacyADRFiles(root, []string{"ADR-001-crlf.md", "invalid.md", "large.md", "directory.md"})
	if len(results[0].Errors) != 0 {
		t.Fatalf("CRLF: %#v", results[0])
	}
	for _, result := range results[1:] {
		if !hasValidationIssue(result.Errors, "read") {
			t.Fatalf("unreadable file accepted: %#v", result)
		}
	}
}

func TestKbValidateOptions(t *testing.T) {
	for _, args := range [][]string{{"file.md"}, {"--legacy-adr", "--unknown"}} {
		if _, err := parseKbValidateArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	options, err := parseKbValidateArgs([]string{"--legacy-adr", "--json", "--", "-file.md"})
	if err != nil || len(options.files) != 1 || options.files[0] != "-file.md" || !options.jsonOutput {
		t.Fatalf("options=%#v err=%v", options, err)
	}
}
