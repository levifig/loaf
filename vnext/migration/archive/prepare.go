package archive

import (
	"fmt"
	"github.com/levifig/loaf/vnext/continuity"
	continuitysqlite "github.com/levifig/loaf/vnext/continuity/sqlite"
)

// PrepareRecord validates and encodes a declared migration payload using the destination typed codec.
func PrepareRecord(
	projectID continuity.ProjectID,
	record Record,
) (continuitysqlite.RehearsalFact, error) {
	payloads := 0
	for _, set := range []bool{record.Project != nil, record.Journal != nil, record.Wrap != nil, record.Handoff != nil, record.Payload != nil} {
		if set {
			payloads++
		}
	}
	if payloads != 1 {
		return continuitysqlite.RehearsalFact{}, fmt.Errorf("record must contain exactly one payload")
	}
	if (record.Kind == RecordProject && record.Project == nil) || (record.Kind == RecordJournal && record.Journal == nil) || (record.Kind == RecordWrap && record.Wrap == nil) || (record.Kind == RecordHandoff && record.Handoff == nil) {
		return continuitysqlite.RehearsalFact{}, fmt.Errorf("record tag does not match payload")
	}
	observation := record.Observation.Continuity()

	switch record.Kind {
	case RecordProject:
		return continuitysqlite.NewProjectRehearsalFact(projectID, record.FactID, continuity.ProjectRegistrationPayload{
			Observation: observation,
			Label:       record.Project.Label,
		})
	case RecordJournal:
		return continuitysqlite.NewJournalRehearsalFact(projectID, record.FactID, record.SubjectID, continuity.JournalRecordedPayload{
			Observation: observation,
			Content: continuity.JournalContent{
				Category: record.Journal.Category,
				Scope:    record.Journal.Scope,
				Text:     record.Journal.Text,
			},
		})
	case RecordWrap:
		return continuitysqlite.NewWrapRehearsalFact(projectID, record.FactID, record.SubjectID, continuity.WrapRecordedPayload{
			Observation: observation,
			Scope:       record.Wrap.Scope,
			Synthesis:   record.Wrap.Synthesis,
		})
	case RecordHandoff:
		suggestedSkills := make([]string, len(record.Handoff.SuggestedSkills))
		copy(suggestedSkills, record.Handoff.SuggestedSkills)
		return continuitysqlite.NewHandoffRehearsalFact(projectID, record.FactID, record.SubjectID, continuity.HandoffRecordedPayload{
			Observation: observation,
			Purpose:     record.Handoff.Purpose, Situation: record.Handoff.Situation,
			NextActions: record.Handoff.NextActions, QuestionsAndRisks: record.Handoff.QuestionsAndRisks,
			SuggestedSkills: suggestedSkills,
		})
	default:
		return prepareTypedRecord(projectID, record)
	}
}

// Continuity converts legacy display provenance without inferring a harness.
func (o Observation) Continuity() continuity.Observation {
	harness, version := o.ObservedHarness, o.HarnessVersion
	if harness == "" {
		harness = "unknown"
	}
	if version == "" {
		version = "unknown"
	}
	return continuity.Observation{ObservedAtMillis: o.ObservedAtMillis, ObservedHarness: harness, HarnessVersion: version, HarnessSessionID: o.HarnessSessionID, Branch: o.Branch, Worktree: o.Worktree}
}
