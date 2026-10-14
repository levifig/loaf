package archive

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/levifig/loaf/vnext/continuity"
)

func validateSourceCountsV2(c Content) error {
	expected := map[string]KindCount{}
	for _, name := range []string{"journal_entries", "journal_revision_facts", "handoffs", "ideas", "sparks", "explorations", "exploration_checkpoints", "exploration_checkpoint_items", "issue_tracker_references", "issue_criterion_claims", "specs", "intents", "tasks", "brainstorms", "plans", "work_contracts", "issues"} {
		expected[name] = KindCount{Kind: name}
	}
	categories := map[string]CategoryCount{}
	addCategory := func(original, mapped string, rows int) {
		count := categories[original]
		count.Original = original
		count.Mapped = mapped
		count.Rows += rows
		categories[original] = count
	}
	add := func(name string, source, mapped, retired int) {
		k := expected[name]
		k.SourceRows += source
		k.MappedRecords += mapped
		k.RetiredRows += retired
		expected[name] = k
	}
	journalRows, wrapRows, normalized, normalizedText := 0, 0, 0, 0
	unmatched, matched, origins, revisionDeclarations := 0, 0, 0, 0
	sparkKeys := map[string]bool{}
	for _, r := range c.Records {
		for _, m := range r.Mappings {
			if m.Name == JournalTextControlCharactersStrippedV1 {
				normalizedText++
			}
		}
		switch r.Kind {
		case RecordJournal:
			journalRows++
			original := string(r.Journal.Category)
			for _, m := range r.Mappings {
				if m.Name == JournalCategoryMappingUnsupportedToNoteV1 && m.Field == "category" {
					original = m.Original
					normalized++
				}
			}
			addCategory(original, string(r.Journal.Category), 1)
		case RecordWrap:
			wrapRows++
		case RecordHandoff:
			add("handoffs", 1, 1, 0)
		default:
			kind := continuity.FactKind(r.Kind)
			switch kind {
			case continuity.FactIdeaCreated:
				add("ideas", 1, 1, 0)
			case continuity.FactIdeaRevised, continuity.FactIdeaResolved, continuity.FactIdeaArchived, continuity.FactIdeaPromotedToExternalReference:
				add("ideas", 0, 1, 0)
			case continuity.FactSparkCaptured:
				if !strings.HasPrefix(r.SourceID, "journal:") {
					add("sparks", 1, 1, 0)
					var p continuity.SparkCapturedPayload
					if err := json.Unmarshal(r.Payload, &p); err != nil {
						return err
					}
					sparkKeys[p.Scope+"\x00"+p.Text+"\x00"+strconv.FormatInt(p.Observation.ObservedAtMillis, 10)] = true
				} else {
					unmatched++
				}
			case continuity.FactSparkDismissed, continuity.FactSparkPromotedToIdea:
				add("sparks", 0, 1, 0)
			case continuity.FactExplorationStarted:
				add("explorations", 1, 1, 0)
			case continuity.FactCheckpointRecorded:
				add("exploration_checkpoints", 1, 1, 0)
				var payload continuity.CheckpointRecordedPayload
				if err := json.Unmarshal(r.Payload, &payload); err != nil {
					return err
				}
				add("exploration_checkpoint_items", len(payload.Items), len(payload.Items), 0)
			case continuity.FactExternalReferenceRegistered:
				add("issue_tracker_references", 1, 1, 0)
			case continuity.FactVerificationEvidenceRecorded:
				add("issue_criterion_claims", 1, 1, 0)
			}
		}
	}
	if c.Source.MatchedSparkJournalRows+c.Source.UnmatchedSparkJournalRows > 0 {
		addCategory("spark", "spark.captured", c.Source.MatchedSparkJournalRows+c.Source.UnmatchedSparkJournalRows)
	}
	add("journal_entries", c.Source.JournalProjectionRows, journalRows+wrapRows+c.Source.UnmatchedSparkJournalRows, 0)
	add("journal_revision_facts", c.Source.JournalFactRows, c.Source.JournalProjectionRows, 0)
	for _, m := range c.Source.Mappings {
		switch m.Name {
		case JournalTextControlCharactersStrippedV1:
			normalizedText++
		case "legacy_journal_revisions_folded_to_effective_state_v1":
			revisionDeclarations++
			if m.Field != "collapsed_revision_rows" || m.Original != strconv.Itoa(c.Source.CollapsedRevisionRows) {
				return fmt.Errorf("journal revision declaration mismatch")
			}
		case "legacy_journal_origin_metadata_preserved_v1":
			origins++
		case "legacy_duplicate_journal_spark_v1":
			matched++
			if !sparkKeys[m.Original] {
				return fmt.Errorf("deduplicated journal spark lacks its source capture")
			}
		case "legacy_issue_link_without_durable_tracker_locator_v1":
			add("issue_tracker_references", 1, 0, 1)
		case "legacy_duplicate_tracker_locator_v1":
			add("issue_tracker_references", 1, 0, 0)
		}
	}
	if (c.Source.CollapsedRevisionRows == 0 && revisionDeclarations != 0) || (c.Source.CollapsedRevisionRows > 0 && revisionDeclarations != 1) {
		return fmt.Errorf("journal revision declaration count mismatch")
	}
	if unmatched != c.Source.UnmatchedSparkJournalRows || matched != c.Source.MatchedSparkJournalRows || origins != c.Source.JournalOriginRows {
		return fmt.Errorf("source origin or spark mapping counts do not reconcile")
	}
	if len(c.Source.Kinds) != len(expected) {
		return fmt.Errorf("source kind inventory must be complete")
	}
	seen := map[string]bool{}
	for _, k := range c.Source.Kinds {
		want, ok := expected[k.Kind]
		want.MappedSourceRows = want.SourceRows - want.RetiredRows
		if !ok || seen[k.Kind] {
			return fmt.Errorf("unknown or duplicate source kind %q", k.Kind)
		}
		seen[k.Kind] = true
		switch k.Kind {
		case "specs", "intents", "tasks", "brainstorms", "plans", "work_contracts", "issues":
			if k.MappedSourceRows != 0 || k.MappedRecords != 0 || k.SourceRows < 0 || k.RetiredRows != k.SourceRows {
				return fmt.Errorf("retired inventory must be unmapped and explicitly retired")
			}
		default:
			if k != want {
				return fmt.Errorf("source kind %s counts do not reconcile with archive mappings", k.Kind)
			}
		}
	}
	if normalizedText != c.Source.NormalizedJournalTextRows {
		return fmt.Errorf("journal control-character count does not reconcile")
	}
	if (normalizedText == 0 && c.Source.JournalTextMapping != "") || (normalizedText > 0 && c.Source.JournalTextMapping != JournalTextControlCharactersStrippedV1) {
		return fmt.Errorf("journal text mapping declaration mismatch")
	}
	if normalized != c.Source.NormalizedJournalCategoryRows {
		return fmt.Errorf("journal category normalization count does not reconcile")
	}
	if len(c.Source.JournalCategoryCounts) != len(categories) {
		return fmt.Errorf("journal category inventory does not reconcile")
	}
	seen = map[string]bool{}
	for _, k := range c.Source.JournalCategoryCounts {
		if seen[k.Original] || categories[k.Original] != k {
			return fmt.Errorf("journal category counts do not reconcile")
		}
		seen[k.Original] = true
	}
	return nil
}
