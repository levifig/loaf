package sqlite

import (
	"context"
	"github.com/levifig/loaf/vnext/continuity"
)

// NewSparkCapturedRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewSparkCapturedRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.SparkCapturedPayload) (RehearsalFact, error) {
	content, err := encodeSparkCapturedV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordSpark, ID: subjectID}, kind: continuity.FactSparkCaptured, content: content})
}

// NewSparkDismissedRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewSparkDismissedRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.SparkDismissedPayload) (RehearsalFact, error) {
	content, err := encodeSparkDismissedV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordSpark, ID: subjectID}, kind: continuity.FactSparkDismissed, content: content})
}

// NewSparkPromotionRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewSparkPromotionRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.SparkPromotionPayload) (RehearsalFact, error) {
	content, err := encodeSparkPromotionV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordSpark, ID: subjectID}, kind: continuity.FactSparkPromotedToIdea, content: content})
}

// NewIdeaCreatedRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewIdeaCreatedRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.IdeaCreatedPayload) (RehearsalFact, error) {
	content, err := encodeIdeaCreatedV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordIdea, ID: subjectID}, kind: continuity.FactIdeaCreated, content: content})
}

// NewIdeaRevisionRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewIdeaRevisionRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.IdeaRevisionPayload) (RehearsalFact, error) {
	content, err := encodeIdeaRevisionV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordIdea, ID: subjectID}, kind: continuity.FactIdeaRevised, content: content})
}

// NewIdeaResolutionRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewIdeaResolutionRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.IdeaResolutionPayload) (RehearsalFact, error) {
	content, err := encodeIdeaResolutionV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordIdea, ID: subjectID}, kind: continuity.FactIdeaResolved, content: content})
}

// NewIdeaArchiveRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewIdeaArchiveRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.IdeaArchivePayload) (RehearsalFact, error) {
	content, err := encodeIdeaArchiveV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordIdea, ID: subjectID}, kind: continuity.FactIdeaArchived, content: content})
}

// NewIdeaPromotionRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewIdeaPromotionRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.IdeaPromotionPayload) (RehearsalFact, error) {
	content, err := encodeIdeaPromotionV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordIdea, ID: subjectID}, kind: continuity.FactIdeaPromotedToExternalReference, content: content})
}

// NewExplorationStartedRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewExplorationStartedRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.ExplorationStartedPayload) (RehearsalFact, error) {
	content, err := encodeExplorationStartedV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordExploration, ID: subjectID}, kind: continuity.FactExplorationStarted, content: content})
}

// NewCheckpointRecordedRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewCheckpointRecordedRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.CheckpointRecordedPayload) (RehearsalFact, error) {
	content, err := encodeCheckpointRecordedV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordCheckpoint, ID: subjectID}, kind: continuity.FactCheckpointRecorded, content: content})
}

// NewExternalReferenceRegistrationRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewExternalReferenceRegistrationRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.ExternalReferenceRegistrationPayload) (RehearsalFact, error) {
	content, err := encodeExternalReferenceRegistrationV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordExternalReference, ID: subjectID}, kind: continuity.FactExternalReferenceRegistered, content: content})
}

// NewVerificationEvidenceRehearsalFact prepares one typed migration fact; no raw insertion is exposed.
func NewVerificationEvidenceRehearsalFact(projectID continuity.ProjectID, factID continuity.FactID, subjectID continuity.SubjectID, payload continuity.VerificationEvidencePayload) (RehearsalFact, error) {
	content, err := encodeVerificationEvidenceV1(payload)
	if err != nil {
		return RehearsalFact{}, err
	}
	return newRehearsalFactV1(appendIntentV1{projectID: projectID, factID: factID, subject: continuity.SubjectRef{Kind: continuity.RecordVerificationEvidence, ID: subjectID}, kind: continuity.FactVerificationEvidenceRecorded, content: content})
}

// PreviewRehearsalFacts derives the expected semantic projection without opening
// a store. Ordinal clocks describe archive order only; they are never persisted.
func PreviewRehearsalFacts(projectID continuity.ProjectID, facts []RehearsalFact) (continuity.Snapshot, error) {
	if len(facts) == 0 || len(facts) > maxRehearsalFactsV1 {
		return continuity.Snapshot{}, &continuity.Problem{Code: continuity.ProblemInvalid, Field: "facts", Detail: "must contain a bounded project-first sequence"}
	}
	if err := projectID.Validate(); err != nil {
		return continuity.Snapshot{}, err
	}
	records := make([]storedFactV1, 0, len(facts))
	for i, f := range facts {
		if f.intent.projectID != projectID {
			return continuity.Snapshot{}, rehearsalContaminationProblemV1()
		}
		if err := validateAppendIntentV1(f.intent); err != nil {
			return continuity.Snapshot{}, err
		}
		records = append(records, storedFactV1{factID: f.intent.factID, projectID: projectID, subject: f.intent.subject, kind: f.intent.kind, payloadVersion: payloadVersionV1, content: f.intent.content, environmentID: "environment-rehearsal-preview", environmentSequence: int64(i + 1), clock: continuity.HybridTime{WallMillis: int64(i + 1)}, envelopeVersion: 1})
	}
	return foldProjectSnapshotV1(context.Background(), projectID, 0, records)
}
