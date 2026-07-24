package state

import (
	"strings"
)

// importLifecycleStatusResult is the insert-time classification of a source status
// for lifecycle-managed entity kinds (Change markdown-reimport-safety U3).
//
// U1 owns the full stored×incoming matrix in markdown_import_decisions.go
// (decideImportStatus). This helper covers the insert-only half so U3 can land
// in parallel; on merge, call sites should switch to
// decideImportStatus(kind, id, nil, raw, noOpinionInsert, true) and append
// decision.Warning onto ImportReport.Warnings.
type importLifecycleStatusResult struct {
	// Status is the value to store on first insert.
	Status string
	// OutOfVocabulary is true when Status is the raw explicit source value
	// and an import_report warning should be recorded.
	OutOfVocabulary bool
	// Warning is non-empty exactly when OutOfVocabulary is true.
	Warning string
}

// classifyImportLifecycleStatus maps an incoming source status for insert.
//
// raw is the source status before defaults: empty means absent. Absent and
// explicit "unknown" are no-opinion and insert as noOpinionInsert with no
// warning. Canonical and legacy spellings map through CanonicalLifecycleStatus.
// Anything else is out-of-vocabulary: stored raw with a warning.
//
// Shaping drafts have no lifecycle vocabulary and must not call this helper.
func classifyImportLifecycleStatus(kind string, entityID string, raw string, noOpinionInsert string) importLifecycleStatusResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "unknown" {
		return importLifecycleStatusResult{Status: noOpinionInsert}
	}
	if canonical, ok := CanonicalLifecycleStatus(kind, trimmed); ok {
		return importLifecycleStatusResult{Status: canonical}
	}
	return importLifecycleStatusResult{
		Status:          trimmed,
		OutOfVocabulary: true,
		// Wording matches U1 importStatusOOVWarning for merge convergence.
		Warning: "out-of-vocabulary status " + trimmed + " for " + kind + " " + entityID,
	}
}

// noteImportStatusWarning records an outcome warning for U1's ImportReport.
// Plan.Warnings remains inventory-only; status/provenance outcomes belong on
// import_report.warnings once that type lands.
//
// outcomeWarnings is a shared slice pointer so value-receiver import methods
// can append without converting the whole importer to pointer receivers.
func (m markdownImporter) noteImportStatusWarning(warning string) {
	if warning == "" || m.outcomeWarnings == nil {
		return
	}
	*m.outcomeWarnings = append(*m.outcomeWarnings, warning)
}

// applyImportLifecycleStatus classifies raw, records any OOV warning, and
// returns the insert status.
func (m markdownImporter) applyImportLifecycleStatus(kind string, entityID string, raw string, noOpinionInsert string) string {
	result := classifyImportLifecycleStatus(kind, entityID, raw, noOpinionInsert)
	if result.OutOfVocabulary {
		m.noteImportStatusWarning(result.Warning)
	}
	return result.Status
}
