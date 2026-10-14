package archive

import (
	"encoding/json"
	"fmt"
	"github.com/levifig/loaf/vnext/continuity"
)

// Validate references before sealing, rather than allowing an ineligible fact
// to disappear from both expected and actual projections.
func validateCausalRecordsV2(records []Record) error {
	facts := map[continuity.FactID]Record{}
	subjects := map[continuity.SubjectRef]bool{}
	for _, r := range records {
		if r.Payload == nil {
			kind := continuity.RecordJournalEntry
			switch r.Kind {
			case RecordProject:
				kind = continuity.RecordProjectIdentity
			case RecordWrap:
				kind = continuity.RecordWrap
			case RecordHandoff:
				kind = continuity.RecordHandoff
			}
			subjects[continuity.SubjectRef{Kind: kind, ID: r.SubjectID}] = true
			facts[r.FactID] = r
			continue
		}
		definition, ok := continuity.DefinitionFor(continuity.FactKind(r.Kind))
		if !ok {
			return fmt.Errorf("unknown archive fact kind")
		}
		subject := continuity.SubjectRef{Kind: definition.Record, ID: r.SubjectID}
		var links struct {
			Predecessor   continuity.FactID
			Revises       continuity.FactID
			IdeaID        continuity.SubjectID
			ExplorationID continuity.SubjectID
			ReferenceID   continuity.SubjectID
			Target        *continuity.SubjectRef
		}
		if err := json.Unmarshal(r.Payload, &links); err != nil {
			return err
		}
		parent := links.Predecessor
		if links.Revises != "" {
			parent = links.Revises
		}
		if parent != "" {
			prior, found := facts[parent]
			parentDefinition, parentKnown := continuity.DefinitionFor(continuity.FactKind(prior.Kind))
			if !found || !parentKnown || parentDefinition.Record != definition.Record || prior.SubjectID != r.SubjectID {
				return fmt.Errorf("record %s has absent or foreign predecessor", r.FactID)
			}
		} else if subjects[subject] {
			return fmt.Errorf("record %s duplicates a subject root", r.FactID)
		}
		for _, target := range []continuity.SubjectRef{{Kind: continuity.RecordIdea, ID: links.IdeaID}, {Kind: continuity.RecordExploration, ID: links.ExplorationID}, {Kind: continuity.RecordExternalReference, ID: links.ReferenceID}} {
			if target.ID != "" && !subjects[target] {
				return fmt.Errorf("record %s has absent or foreign target", r.FactID)
			}
		}
		if links.Target != nil && !subjects[*links.Target] {
			return fmt.Errorf("record %s has absent verification target", r.FactID)
		}
		subjects[subject] = true
		facts[r.FactID] = r
	}
	return nil
}
