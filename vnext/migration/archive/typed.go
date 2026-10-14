package archive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/levifig/loaf/vnext/continuity"
	continuitysqlite "github.com/levifig/loaf/vnext/continuity/sqlite"
)

func prepareTypedRecord(projectID continuity.ProjectID, r Record) (continuitysqlite.RehearsalFact, error) {
	if r.Payload == nil || r.Project != nil || r.Journal != nil || r.Wrap != nil || r.Handoff != nil {
		return continuitysqlite.RehearsalFact{}, fmt.Errorf("typed record must have exactly one payload")
	}
	switch continuity.FactKind(r.Kind) {
	case continuity.FactSparkCaptured:
		var payload continuity.SparkCapturedPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewSparkCapturedRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactSparkDismissed:
		var payload continuity.SparkDismissedPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewSparkDismissedRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactSparkPromotedToIdea:
		var payload continuity.SparkPromotionPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewSparkPromotionRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactIdeaCreated:
		var payload continuity.IdeaCreatedPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewIdeaCreatedRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactIdeaRevised:
		var payload continuity.IdeaRevisionPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewIdeaRevisionRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactIdeaResolved:
		var payload continuity.IdeaResolutionPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewIdeaResolutionRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactIdeaArchived:
		var payload continuity.IdeaArchivePayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewIdeaArchiveRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactIdeaPromotedToExternalReference:
		var payload continuity.IdeaPromotionPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewIdeaPromotionRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactExplorationStarted:
		var payload continuity.ExplorationStartedPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewExplorationStartedRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactCheckpointRecorded:
		var payload continuity.CheckpointRecordedPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewCheckpointRecordedRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactExternalReferenceRegistered:
		var payload continuity.ExternalReferenceRegistrationPayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewExternalReferenceRegistrationRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	case continuity.FactVerificationEvidenceRecorded:
		var payload continuity.VerificationEvidencePayload
		if err := decodeTypedPayload(r.Payload, &payload); err != nil {
			return continuitysqlite.RehearsalFact{}, err
		}
		return continuitysqlite.NewVerificationEvidenceRehearsalFact(projectID, r.FactID, r.SubjectID, payload)
	default:
		return continuitysqlite.RehearsalFact{}, fmt.Errorf("unsupported archive fact kind %q", r.Kind)
	}
}

func decodeTypedPayload(raw []byte, value any) error {
	if err := inspectJSONBounds(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := requireJSONEOFV1(decoder); err != nil {
		return err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return err
	}
	if !bytes.Equal(canonical, compact.Bytes()) {
		return fmt.Errorf("typed archive payload is not in canonical field form")
	}
	return nil
}
