package archive

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/levifig/loaf/vnext/continuity"
	continuitysqlite "github.com/levifig/loaf/vnext/continuity/sqlite"
)

// Mapping names a lossy or coercing transformation and preserves its source value.
type Mapping struct {
	Name     string `json:"name"`
	Field    string `json:"field"`
	Original string `json:"original"`
}

// KindCount separates source rows from destination facts. One source can emit
// lifecycle facts; journal revisions and duplicate spark entries can collapse.
// RetiredRows is an inventory only, not a claim that their bytes were archived.
type CategoryCount struct {
	Original string `json:"original"`
	Mapped   string `json:"mapped"`
	Rows     int    `json:"rows"`
}

type KindCount struct {
	Kind             string `json:"kind"`
	SourceRows       int    `json:"source_rows"`
	MappedSourceRows int    `json:"mapped_source_rows"`
	MappedRecords    int    `json:"mapped_records"`
	RetiredRows      int    `json:"retired_rows"`
}

const JournalTextControlCharactersStrippedV1 = "legacy_journal_control_characters_stripped_v1"
const LegacyEffectiveStateMappingV1 = "legacy_effective_state_with_declared_lifecycle_v1"
const LegacyMissingBodyMappingV1 = "legacy_missing_idea_body_v1"
const LegacyCriterionClaimMappingV1 = "legacy_criterion_claim_is_not_execution_evidence_v1"

// SealV2 seals a full supported-domain archive. Retired byte preservation,
// tracker disposition, and activation remain separate gates.
func SealV2(content Content) (Archive, error) {
	content.Expected = ProjectionManifest{}
	if err := validateContentV2(content, false); err != nil {
		return Archive{}, err
	}
	expected, err := expectedProjectionV2(content)
	if err != nil {
		return Archive{}, err
	}
	content.Expected = expected
	a := Archive{Format: Format, Version: Version2, Content: content}
	a.ContentSHA256, err = archiveDigestV2(a)
	return a, err
}

func archiveDigestV2(a Archive) (string, error) {
	preimage := struct {
		Format  string  `json:"format"`
		Version int     `json:"version"`
		Content Content `json:"content"`
	}{a.Format, a.Version, a.Content}
	encoded, err := canonicalJSONV1(preimage)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(append([]byte("loaf:vnext:migration:archive:v2\x00"), encoded...))
	return hex.EncodeToString(digest[:]), nil
}

func validateArchiveV2(a Archive) error {
	digest, err := archiveDigestV2(a)
	if err != nil {
		return err
	}
	if digest != a.ContentSHA256 {
		return fmt.Errorf("archive content digest does not match canonical content")
	}
	return validateContentV2(a.Content, true)
}

func validateContentV2(c Content, verify bool) error {
	if c.Source.JournalFactRows != c.Source.JournalProjectionRows+c.Source.CollapsedRevisionRows || c.Source.CollapsedRevisionRows < 0 {
		return fmt.Errorf("source journal revision counts do not reconcile")
	}
	if len(c.Records) > maxRecords {
		return fmt.Errorf("archive record count exceeds %d", maxRecords)
	}
	if !c.Families.Project || !c.Families.Journal || !c.Families.Wrap || !c.Families.Sparks || !c.Families.Ideas || !c.Families.Explorations || !c.Families.Decisions || !c.Families.Findings {
		return fmt.Errorf("archive v2 must declare every supported family")
	}
	if c.Families.CompleteForCutover {
		return fmt.Errorf("supported-domain archive alone cannot claim cutover completeness")
	}
	base := c
	base.Records = nil
	base.Expected = ProjectionManifest{}
	base.Families = FamilyManifest{Project: true, Journal: true, Wrap: true, Handoffs: c.Families.Handoffs}
	base.Source.Kinds = nil
	payloadBytes := len(c.Project.Label)
	for _, m := range c.Source.Mappings {
		if err := validateMappingV2(m); err != nil {
			return err
		}
		payloadBytes += len(m.Name) + len(m.Field) + len(m.Original)
	}
	for _, k := range c.Source.JournalCategoryCounts {
		if !utf8.ValidString(k.Original) || !utf8.ValidString(k.Mapped) {
			return fmt.Errorf("journal category contains invalid UTF-8")
		}
		payloadBytes += len(k.Original) + len(k.Mapped)
	}
	ids := map[continuity.FactID]bool{}
	journalRows := 0
	for i, r := range c.Records {
		if !utf8.ValidString(r.SourceID) {
			return fmt.Errorf("record %d source identity contains invalid UTF-8", i)
		}
		payloadBytes += recordPayloadBytesV1(r)
		if payloadBytes > maxAggregatePayloadBytes {
			return fmt.Errorf("archive aggregate payload exceeds %d bytes", maxAggregatePayloadBytes)
		}
		if ids[r.FactID] {
			return fmt.Errorf("record %d duplicates fact id", i)
		}
		ids[r.FactID] = true
		if err := r.FactID.Validate(); err != nil {
			return err
		}
		if err := r.SubjectID.Validate(); err != nil {
			return err
		}
		for _, m := range r.Mappings {
			if err := validateMappingV2(m); err != nil {
				return fmt.Errorf("record %d: %w", i, err)
			}
		}
		if r.Payload == nil {
			base.Records = append(base.Records, r)
			if r.Kind == RecordJournal || r.Kind == RecordWrap {
				journalRows++
			}
		} else {
			if strings.TrimSpace(r.SourceID) == "" {
				return fmt.Errorf("record %d lacks source identity", i)
			}
			if _, err := prepareTypedRecord(c.Project.ProjectID, r); err != nil {
				return fmt.Errorf("record %d: %w", i, err)
			}
		}
	}
	if c.Source.MatchedSparkJournalRows < 0 || c.Source.UnmatchedSparkJournalRows < 0 || journalRows+c.Source.MatchedSparkJournalRows+c.Source.UnmatchedSparkJournalRows != c.Source.JournalProjectionRows {
		return fmt.Errorf("journal/spark source reconciliation mismatch")
	}
	base.Source.JournalProjectionRows = journalRows
	base.Source.JournalFactRows = journalRows + base.Source.CollapsedRevisionRows
	if err := validateContentV1(base, false); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, k := range c.Source.Kinds {
		if k.Kind == "" || seen[k.Kind] || k.SourceRows < 0 || k.MappedRecords < 0 || k.RetiredRows < 0 || k.RetiredRows > k.SourceRows {
			return fmt.Errorf("invalid source kind counts")
		}
		seen[k.Kind] = true
	}
	if err := validateSourceCountsV2(c); err != nil {
		return err
	}
	if err := validateCausalRecordsV2(c.Records); err != nil {
		return err
	}
	expected, err := expectedProjectionV2(c)
	if err != nil {
		return err
	}
	if verify && expected != c.Expected {
		return fmt.Errorf("expected projection manifest does not match archive records")
	}
	return nil
}

func validateMappingV2(m Mapping) error {
	if strings.TrimSpace(m.Name) == "" || m.Field == "" {
		return fmt.Errorf("incomplete mapping declaration")
	}
	if !utf8.ValidString(m.Name) || !utf8.ValidString(m.Field) || !utf8.ValidString(m.Original) {
		return fmt.Errorf("mapping contains invalid UTF-8")
	}
	return nil
}

func expectedProjectionV2(c Content) (ProjectionManifest, error) {
	facts := make([]continuitysqlite.RehearsalFact, 0, len(c.Records))
	for i, r := range c.Records {
		f, err := PrepareRecord(c.Project.ProjectID, r)
		if err != nil {
			return ProjectionManifest{}, fmt.Errorf("prepare record %d: %w", i, err)
		}
		facts = append(facts, f)
	}
	snapshot, err := continuitysqlite.PreviewRehearsalFacts(c.Project.ProjectID, facts)
	if err != nil {
		return ProjectionManifest{}, err
	}
	return ProjectionManifestForSnapshotV2(snapshot)
}

// ProjectionManifestForSnapshotV2 binds every supported projection and its
// payload/provenance while excluding only writer-local clocks and sequences.
// Exact staged-fact inventory verification independently prevents omissions,
// duplicate imports, unrelated facts, or a contaminated destination.
func ProjectionManifestForSnapshotV2(snapshot continuity.Snapshot) (ProjectionManifest, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return ProjectionManifest{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var v any
	if err := decoder.Decode(&v); err != nil {
		return ProjectionManifest{}, err
	}
	stripWriterClocks(v)
	encoded, err := canonicalJSONV1(v)
	if err != nil {
		return ProjectionManifest{}, err
	}
	digest := sha256.Sum256(append([]byte("loaf:vnext:migration:projection:v2\x00"), encoded...))
	return ProjectionManifest{ProjectCount: 1, EffectiveJournalCount: len(snapshot.EffectiveJournal.Entries), WrapCount: len(snapshot.LatestWraps.Wraps), HandoffCount: len(snapshot.LatestHandoffs.Handoffs), SparkCount: len(snapshot.ActiveSparks.Sparks), IdeaCount: len(snapshot.CurrentIdeas.Ideas), ExplorationCount: len(snapshot.Explorations.Explorations), CheckpointCount: len(snapshot.LatestCheckpoints.Checkpoints), ExternalReferenceCount: len(snapshot.ExternalReferences.References), VerificationCount: len(snapshot.VerificationEvidence.Evidence), SHA256: hex.EncodeToString(digest[:])}, nil
}

func stripWriterClocks(v any) {
	switch value := v.(type) {
	case map[string]any:
		// These keys belong only to typed FactStamp values in a Snapshot.
		delete(value, "Clock")
		delete(value, "EnvironmentID")
		delete(value, "EnvironmentSequence")
		for _, item := range value {
			stripWriterClocks(item)
		}
	case []any:
		for _, item := range value {
			stripWriterClocks(item)
		}
	}
}
