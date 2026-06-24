package state

import (
	"context"
	"strings"
	"testing"

	"github.com/levifig/loaf/internal/project"
)

func TestFindingCRUDVerdictFiltersAndReportShow(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	report, err := CreateReport(context.Background(), root, PathResolver{StateHome: stateHome}, ReportCreateOptions{
		Slug:   "security-audit",
		Kind:   "audit",
		Source: "test",
	})
	if err != nil {
		t.Fatalf("CreateReport() error = %v", err)
	}

	created, err := CreateFinding(context.Background(), root, PathResolver{StateHome: stateHome}, FindingCreateOptions{
		Report:     report.Report.Alias,
		Title:      "Missing authorization check",
		Severity:   "critical",
		Confidence: "high",
		Dimension:  "auth",
		Path:       "internal/auth.go",
		LineStart:  42,
		LineEnd:    44,
		Body:       "The handler accepts caller-controlled account IDs.",
		SetBody:    true,
	})
	if err != nil {
		t.Fatalf("CreateFinding() error = %v", err)
	}
	if !strings.HasPrefix(created.Finding.Alias, "FINDING-") || created.Finding.Status != "open" || created.Finding.Report.Alias != report.Report.Alias {
		t.Fatalf("created finding = %#v, want open finding linked to report", created.Finding)
	}
	if created.Finding.Body != "The handler accepts caller-controlled account IDs." {
		t.Fatalf("Body = %q, want finding body", created.Finding.Body)
	}
	assertReportProjectContext(t, root, created.ContractVersion, created.DatabaseScope, created.DatabasePath, created.ProjectID, created.ProjectName, created.ProjectCurrentPath)

	openFindings, err := ListFindings(context.Background(), root, PathResolver{StateHome: stateHome}, FindingListOptions{Severity: "critical", Status: "open"})
	if err != nil {
		t.Fatalf("ListFindings(open critical) error = %v", err)
	}
	if len(openFindings.Findings) != 1 {
		t.Fatalf("open critical findings = %#v, want one", openFindings.Findings)
	}

	verdict, err := RecordFindingVerdict(context.Background(), root, PathResolver{StateHome: stateHome}, FindingVerdictOptions{
		Finding:   created.Finding.Alias,
		Outcome:   "confirmed",
		Rationale: "Reproduced with a cross-account request.",
	})
	if err != nil {
		t.Fatalf("RecordFindingVerdict() error = %v", err)
	}
	if verdict.Finding.Status != "confirmed" || verdict.Verdict.Outcome != "confirmed" || verdict.Verdict.Finding.Alias != created.Finding.Alias {
		t.Fatalf("verdict = %#v, want confirmed finding verdict", verdict)
	}

	show, err := ShowFinding(context.Background(), root, PathResolver{StateHome: stateHome}, created.Finding.Alias)
	if err != nil {
		t.Fatalf("ShowFinding() error = %v", err)
	}
	if show.Finding.Status != "confirmed" || len(show.Finding.Verdicts) != 1 || show.Finding.Verdicts[0].Rationale == "" {
		t.Fatalf("show finding = %#v, want confirmed finding with verdict", show.Finding)
	}

	confirmed, err := ListFindings(context.Background(), root, PathResolver{StateHome: stateHome}, FindingListOptions{Status: "confirmed", Severity: "critical"})
	if err != nil {
		t.Fatalf("ListFindings(confirmed critical) error = %v", err)
	}
	if len(confirmed.Findings) != 1 {
		t.Fatalf("confirmed critical findings = %#v, want one", confirmed.Findings)
	}

	reportShow, err := ShowReport(context.Background(), root, PathResolver{StateHome: stateHome}, report.Report.Alias)
	if err != nil {
		t.Fatalf("ShowReport() error = %v", err)
	}
	if len(reportShow.Report.Findings) != 1 || reportShow.Report.Findings[0].Status != "confirmed" || reportShow.Report.Findings[0].Alias != created.Finding.Alias {
		t.Fatalf("report findings = %#v, want confirmed child finding", reportShow.Report.Findings)
	}
}

func TestFindingRunRelationships(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	report, err := CreateReport(context.Background(), root, PathResolver{StateHome: stateHome}, ReportCreateOptions{Slug: "run-audit"})
	if err != nil {
		t.Fatalf("CreateReport() error = %v", err)
	}
	runAlias := insertRunFixture(t, root, stateHome, "RUN-001")

	finding, err := CreateFinding(context.Background(), root, PathResolver{StateHome: stateHome}, FindingCreateOptions{
		Report:     report.Report.Alias,
		Run:        runAlias,
		Title:      "Pipeline finding",
		Severity:   "high",
		Confidence: "medium",
	})
	if err != nil {
		t.Fatalf("CreateFinding(run) error = %v", err)
	}
	if finding.Finding.Run == nil || finding.Finding.Run.Alias != runAlias {
		t.Fatalf("finding run = %#v, want %s", finding.Finding.Run, runAlias)
	}
	if _, err := RecordFindingVerdict(context.Background(), root, PathResolver{StateHome: stateHome}, FindingVerdictOptions{
		Finding:   finding.Finding.Alias,
		Run:       runAlias,
		Outcome:   "partial",
		Rationale: "Pipeline evidence was incomplete.",
	}); err != nil {
		t.Fatalf("RecordFindingVerdict(run) error = %v", err)
	}

	links, err := ListLinks(context.Background(), root, PathResolver{StateHome: stateHome}, runAlias)
	if err != nil {
		t.Fatalf("ListLinks(run) error = %v", err)
	}
	seen := map[string]bool{}
	for _, relationship := range links.Relationships {
		seen[relationship.Type+" "+relationship.Entity.Kind] = true
	}
	if !seen["produces finding"] || !seen["records verdict"] {
		t.Fatalf("run relationships = %#v, want produces finding and records verdict", links.Relationships)
	}
}

func TestFindingValidationRejectsInvalidVocabulary(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	report, err := CreateReport(context.Background(), root, PathResolver{StateHome: stateHome}, ReportCreateOptions{Slug: "validation"})
	if err != nil {
		t.Fatalf("CreateReport() error = %v", err)
	}
	if _, err := CreateFinding(context.Background(), root, PathResolver{StateHome: stateHome}, FindingCreateOptions{
		Report:     report.Report.Alias,
		Title:      "Bad severity",
		Severity:   "blocker",
		Confidence: "high",
	}); err == nil || !strings.Contains(err.Error(), "invalid finding severity") {
		t.Fatalf("CreateFinding(invalid severity) error = %v, want invalid severity", err)
	}
	if _, err := ListFindings(context.Background(), root, PathResolver{StateHome: stateHome}, FindingListOptions{Status: "done"}); err == nil || !strings.Contains(err.Error(), "invalid finding status") {
		t.Fatalf("ListFindings(invalid status) error = %v, want invalid status", err)
	}
}

func insertRunFixture(t *testing.T, root project.Root, stateHome string, alias string) string {
	t.Helper()
	store := openTestStore(t, root, stateHome)
	defer store.Close()
	projectID := projectIDForTest(t, store, root)
	now := "2026-06-24T12:00:00Z"
	runID := stableMigrationID("run", projectID, alias)
	if _, err := store.db.ExecContext(context.Background(), `
INSERT INTO runs (id, project_id, generator_ref, generator_version, generator_hash, status, metadata, started_at, completed_at, created_at, updated_at)
VALUES (?, ?, 'pipeline/findings', 'v1', 'hash-run', 'completed', NULL, ?, ?, ?, ?)
`, runID, projectID, now, now, now, now); err != nil {
		t.Fatalf("insert run fixture error = %v", err)
	}
	tx, err := store.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin run alias transaction error = %v", err)
	}
	defer tx.Rollback()
	if err := insertAlias(context.Background(), tx, projectID, "run", runID, "run", alias, now); err != nil {
		t.Fatalf("insert run alias error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit run alias transaction error = %v", err)
	}
	return alias
}
