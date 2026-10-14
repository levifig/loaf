package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/levifig/loaf/vnext/continuity"
	"github.com/levifig/loaf/vnext/migration/archive"
)

type fullDomainExportV2 struct {
	projectID             string
	records               []archive.Record
	counts                []archive.KindCount
	mappings              []archive.Mapping
	sparkMatches          map[string]bool
	bytes                 int
	rootObservations      map[string]CoreEventPayload
	headObservations      map[string]CoreEventPayload
	headFactKinds         map[string]string
	promotionObservations map[string]CoreEventPayload
	journalObservations   map[string]archive.Observation
}

func (e *fullDomainExportV2) id(role, id string) continuity.SubjectID {
	return continuity.SubjectID(vnextRehearsalIDV1(role+"_migration_", role+"-subject", e.projectID, id))
}
func (e *fullDomainExportV2) fact(role, id string) continuity.FactID {
	return continuity.FactID(vnextRehearsalIDV1("fact_migration_", role+"-fact", e.projectID, id))
}
func legacyObservationV2(created string) (continuity.Observation, error) {
	at, err := parseVNextRehearsalTimeV1(created, "created_at")
	return continuity.Observation{ObservedAtMillis: at, ObservedHarness: "unknown", HarnessVersion: "unknown"}, err
}
func (e *fullDomainExportV2) append(kind continuity.FactKind, source string, subject continuity.SubjectID, payload interface{ Validate() error }, mappings ...archive.Mapping) error {
	if !utf8.ValidString(source) {
		return fmt.Errorf("%s source identity is not valid UTF-8", kind)
	}
	if err := payload.Validate(); err != nil {
		return fmt.Errorf("map %s %q: %w", kind, source, err)
	}
	for _, mapping := range mappings {
		if !utf8.ValidString(mapping.Original) {
			return fmt.Errorf("%s mapping %q is not valid UTF-8", kind, mapping.Name)
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	r := archive.Record{Kind: archive.RecordKind(kind), SourceID: source, FactID: e.fact(string(kind), source), SubjectID: subject, Payload: raw, Mappings: mappings}
	if _, err := archive.PrepareRecord(continuity.ProjectID(e.projectID), r); err != nil {
		return fmt.Errorf("map %s %q: %w", kind, source, err)
	}
	e.bytes += len(raw) + len(source)
	if len(e.records) >= vnextRehearsalMaxRecordsV1 || e.bytes > vnextRehearsalMaxAggregatePayloadBytesV1 {
		return fmt.Errorf("full-domain archive exceeds resource limits")
	}
	e.records = append(e.records, r)
	return nil
}
func (e *fullDomainExportV2) count(kind string, rows, start, retired int) {
	e.counts = append(e.counts, archive.KindCount{Kind: kind, SourceRows: rows, MappedRecords: len(e.records) - start, MappedSourceRows: rows - retired, RetiredRows: retired})
}

// Read every retained source in the same verified read-only snapshot as journal
// and handoff. Legacy mutable lifecycles have no destination causal lineage;
// effective-state mappings explicitly declare that boundary instead of inventing it.
func readFullDomainV2(ctx context.Context, tx *sql.Tx, projectID string) (fullDomainExportV2, error) {
	e := fullDomainExportV2{projectID: projectID, sparkMatches: map[string]bool{}, rootObservations: map[string]CoreEventPayload{}, headObservations: map[string]CoreEventPayload{}, headFactKinds: map[string]string{}, promotionObservations: map[string]CoreEventPayload{}, journalObservations: map[string]archive.Observation{}}
	if err := e.journalOrigins(ctx, tx); err != nil {
		return e, err
	}
	if err := e.legacyLifecycleEvidence(ctx, tx); err != nil {
		return e, err
	}
	if err := e.ideas(ctx, tx); err != nil {
		return e, err
	}
	if err := e.sparks(ctx, tx); err != nil {
		return e, err
	}
	if err := e.explorations(ctx, tx); err != nil {
		return e, err
	}
	if err := e.references(ctx, tx); err != nil {
		return e, err
	}
	if err := e.retiredInventory(ctx, tx); err != nil {
		return e, err
	}
	if err := e.claims(ctx, tx); err != nil {
		return e, err
	}
	return e, nil
}

func effectiveMappingV2(status string) archive.Mapping {
	return archive.Mapping{Name: archive.LegacyEffectiveStateMappingV1, Field: "legacy_lifecycle", Original: status}
}

func (e *fullDomainExportV2) ideas(ctx context.Context, tx *sql.Tx) error {
	folded, err := foldIdeaFacts(ctx, tx, e.projectID)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT i.id,i.title,i.status,i.created_at,i.updated_at,COALESCE(i.body_source_id,''),b.content,b.content_hash,b.body_kind,COALESCE(src.path,''),COALESCE(src.hash,'') FROM ideas i LEFT JOIN sources src ON src.id=i.body_source_id AND src.project_id=i.project_id LEFT JOIN artifact_bodies b ON b.project_id=i.project_id AND b.entity_kind='idea' AND b.entity_id=i.id WHERE i.project_id=? ORDER BY i.created_at,i.id`, e.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	start, n, projectionOnly := len(e.records), 0, 0
	seen := map[string]bool{}
	for rows.Next() {
		var id, title, status, created, updated, bodySource, sourcePath, sourceHash string
		var body, hash, bodyKind sql.NullString
		if err := rows.Scan(&id, &title, &status, &created, &updated, &bodySource, &body, &hash, &bodyKind, &sourcePath, &sourceHash); err != nil {
			return err
		}
		if seen[id] {
			return fmt.Errorf("idea %q has multiple bodies", id)
		}
		seen[id] = true
		n++
		if !utf8.ValidString(title) {
			return fmt.Errorf("idea %q title is not valid UTF-8", id)
		}
		f, ok := folded[id]
		_, hasRoot := e.rootObservations["idea\x00"+id]
		comparison := f.Payload
		if !hasRoot {
			comparison.CreatedAt = created
		}
		if ok && !ideaProjectionMatches(title, status, created, updated, comparison) {
			return fmt.Errorf("idea %q fact/projection mismatch", id)
		}
		observation, err := e.observation(created, "idea", id, false)
		if err != nil {
			return err
		}
		mappings := []archive.Mapping{effectiveMappingV2(status)}
		if !ok {
			projectionOnly++
		}
		if !hasRoot {
			mappings = append(mappings, archive.Mapping{Name: "legacy_projection_root_without_creation_fact_v1", Field: "created_at", Original: created})
		}
		if !body.Valid && f.Payload.Body != "" {
			body = sql.NullString{String: f.Payload.Body, Valid: true}
			hash = sql.NullString{String: artifactBodyHash(body.String), Valid: true}
			bodyKind = sql.NullString{String: ArtifactBodyKindMarkdown, Valid: true}
			mappings = append(mappings, archive.Mapping{Name: "legacy_idea_body_recovered_from_fact_v1", Field: "body", Original: id})
		}
		if !body.Valid {
			mappings = append(mappings, archive.Mapping{Name: archive.LegacyMissingBodyMappingV1, Field: "body_source_id", Original: bodySource + "\n" + sourcePath + "\n" + sourceHash})
		} else if !bodyKind.Valid || bodyKind.String != ArtifactBodyKindMarkdown || !hash.Valid || hash.String != artifactBodyHash(body.String) {
			return fmt.Errorf("idea %q body checksum mismatch", id)
		}
		if body.Valid && f.Payload.Body != "" && f.Payload.Body != body.String {
			mappings = append(mappings, archive.Mapping{Name: "legacy_idea_artifact_body_precedence_v1", Field: "body", Original: f.Payload.Body})
		}
		label := title
		if len(label) > 512 {
			label = label[:512]
			for !utf8.ValidString(label) {
				label = label[:len(label)-1]
			}
			label = strings.TrimSpace(label)
			mappings = append(mappings, archive.Mapping{Name: "legacy_long_idea_label_prefix_v1", Field: "label", Original: title})
		}
		subject := e.id("idea", id)
		if err := e.append(continuity.FactIdeaCreated, id, subject, continuity.IdeaCreatedPayload{Observation: observation, Content: continuity.IdeaContent{Label: label, Text: body.String}}, mappings...); err != nil {
			return err
		}
		observation, err = e.observation(updated, "idea", id, true)
		if err != nil {
			return err
		}
		predecessor := e.fact(string(continuity.FactIdeaCreated), id)
		switch status {
		case LifecycleStatusOpen, LifecycleStatusDraft, "raw", "captured", "intake":
		case LifecycleStatusDone, "resolved", "absorbed":
			err = e.append(continuity.FactIdeaResolved, id, subject, continuity.IdeaResolutionPayload{Observation: observation, Predecessor: predecessor, Resolution: f.Payload.Note}, effectiveMappingV2(status))
		case LifecycleStatusArchived, "rejected":
			err = e.append(continuity.FactIdeaArchived, id, subject, continuity.IdeaArchivePayload{Observation: observation, Predecessor: predecessor, Reason: f.Payload.Note}, effectiveMappingV2(status))
		default:
			return fmt.Errorf("idea %q unsupported legacy status %q", id, status)
		}
		if err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if n != len(folded)+projectionOnly {
		return fmt.Errorf("idea fact/projection cardinality mismatch")
	}
	e.count("ideas", n, start, 0)
	return nil
}

func (e *fullDomainExportV2) sparks(ctx context.Context, tx *sql.Tx) error {
	folded, err := foldSparkFacts(ctx, tx, e.projectID)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,text,COALESCE(scope,''),status,created_at,updated_at FROM sparks WHERE project_id=? ORDER BY created_at,id`, e.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	start, n, projectionOnly := len(e.records), 0, 0
	for rows.Next() {
		var id, text, scope, status, created, updated string
		if err := rows.Scan(&id, &text, &scope, &status, &created, &updated); err != nil {
			return err
		}
		n++
		f, ok := folded[id]
		_, hasRoot := e.rootObservations["spark\x00"+id]
		promotionMetadataOnly := ok && e.headFactKinds["spark\x00"+id] == FactKindSparkPromoted && f.Payload.RelatedKind == "idea" && f.Payload.RelatedID != ""
		comparison := f.Payload
		if !hasRoot {
			comparison.CreatedAt = created
			if comparison.Scope == "" {
				comparison.Scope = scope
			}
		}
		if promotionMetadataOnly {
			comparison.UpdatedAt = updated
		}
		if ok && !sparkProjectionMatches(scope, status, text, created, updated, comparison) {
			return fmt.Errorf("spark %q fact/projection mismatch", id)
		}
		observation, err := e.observation(created, "spark", id, false)
		if err != nil {
			return err
		}
		e.sparkMatches[scope+"\x00"+text+"\x00"+fmt.Sprint(observation.ObservedAtMillis)] = true
		subject := e.id("spark", id)
		mappings := []archive.Mapping{effectiveMappingV2(status)}
		if !ok {
			projectionOnly++
		}
		if !hasRoot {
			mappings = append(mappings, archive.Mapping{Name: "legacy_projection_root_without_creation_fact_v1", Field: "created_at_and_scope", Original: created + "\n" + scope})
		}
		if err := e.append(continuity.FactSparkCaptured, id, subject, continuity.SparkCapturedPayload{Observation: observation, Scope: scope, Text: text}, mappings...); err != nil {
			return err
		}
		relations, err := tx.QueryContext(ctx, `SELECT r.id,r.to_entity_id,r.created_at,r.updated_at,COALESCE(r.origin,''),COALESCE(r.reason,''),EXISTS(SELECT 1 FROM ideas i WHERE i.project_id=r.project_id AND i.id=r.to_entity_id) FROM relationships r WHERE r.project_id=? AND r.from_entity_kind='spark' AND r.from_entity_id=? AND r.relationship_type='promoted_to' AND r.to_entity_kind='idea' ORDER BY r.to_entity_id`, e.projectID, id)
		if err != nil {
			return err
		}
		var targets []string
		var relationCreated string
		for relations.Next() {
			var relationID, target, createdAt, updatedAt, origin, reason string
			var targetExists bool
			if err := relations.Scan(&relationID, &target, &createdAt, &updatedAt, &origin, &reason, &targetExists); err != nil {
				relations.Close()
				return err
			}
			if !targetExists {
				relations.Close()
				return fmt.Errorf("spark %q promotion target is absent from the source project", id)
			}
			values := []string{relationID, id, target, createdAt, updatedAt, origin, reason}
			for _, value := range values {
				if !utf8.ValidString(value) {
					relations.Close()
					return fmt.Errorf("spark promotion relationship contains invalid UTF-8")
				}
			}
			raw, err := json.Marshal(map[string]string{"id": relationID, "spark_id": id, "idea_id": target, "created_at": createdAt, "updated_at": updatedAt, "origin": origin, "reason": reason})
			if err != nil {
				relations.Close()
				return err
			}
			e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_spark_promotion_relationship_evidence_v1", Field: "relationships." + relationID, Original: string(raw)})
			e.bytes += len(raw)
			if e.bytes > vnextRehearsalMaxAggregatePayloadBytesV1 {
				relations.Close()
				return fmt.Errorf("spark promotion evidence exceeds resource limits")
			}
			targets = append(targets, target)
			relationCreated = createdAt
		}
		if err := relations.Err(); err != nil {
			relations.Close()
			return err
		}
		relations.Close()
		observation, parseErr := e.observation(updated, "spark", id, true)
		if parseErr != nil {
			return parseErr
		}
		predecessor := e.fact(string(continuity.FactSparkCaptured), id)
		promotion, promoted := e.promotionObservations["spark\x00"+id]
		if promoted || len(targets) > 0 {
			selectionMapping := "legacy_latest_promotion_fact_selects_effective_target_v1"
			if !promoted && len(targets) == 1 {
				promotion = CoreEventPayload{RelatedKind: "idea", RelatedID: targets[0], UpdatedAt: relationCreated}
				selectionMapping = "legacy_relationship_only_promotion_with_unknown_provenance_v1"
			}
			found := false
			for _, target := range targets {
				found = found || target == promotion.RelatedID
			}
			if promotion.RelatedKind != "idea" || !found {
				return fmt.Errorf("spark %q promotion fact/relationship mismatch", id)
			}
			observation, err = observationFromLegacyPayloadV2(promotion.UpdatedAt, promotion)
			if err != nil {
				return err
			}
			if err := e.append(continuity.FactSparkPromotedToIdea, id, subject, continuity.SparkPromotionPayload{Observation: observation, Predecessor: predecessor, IdeaID: e.id("idea", promotion.RelatedID)}, effectiveMappingV2(status), archive.Mapping{Name: selectionMapping, Field: "promotion_targets", Original: strings.Join(targets, "\n")}, archive.Mapping{Name: "legacy_promotion_event_timestamp_without_row_touch_v1", Field: "updated_at", Original: updated}); err != nil {
				return err
			}
		} else {
			switch status {
			case LifecycleStatusOpen:
			case LifecycleStatusDone, LifecycleStatusArchived:
				if err := e.append(continuity.FactSparkDismissed, id, subject, continuity.SparkDismissedPayload{Observation: observation, Predecessor: predecessor, Reason: f.Payload.Note}, effectiveMappingV2(status)); err != nil {
					return err
				}
			default:
				return fmt.Errorf("spark %q unsupported legacy status %q", id, status)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if n != len(folded)+projectionOnly {
		return fmt.Errorf("spark fact/projection cardinality mismatch")
	}
	e.count("sparks", n, start, 0)
	return nil
}

func (e *fullDomainExportV2) explorations(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT id,title,created_at,COALESCE(NULLIF(observed_harness,''),'unknown'),COALESCE(NULLIF(observed_harness_version,''),'unknown') FROM explorations WHERE project_id=? ORDER BY created_at,id`, e.projectID)
	if err != nil {
		return err
	}
	start, n := len(e.records), 0
	for rows.Next() {
		var id, title, created, harness, version string
		if err := rows.Scan(&id, &title, &created, &harness, &version); err != nil {
			rows.Close()
			return err
		}
		n++
		observation, err := legacyObservationV2(created)
		if err != nil {
			rows.Close()
			return err
		}
		observation.ObservedHarness = harness
		observation.HarnessVersion = version
		if err := e.append(continuity.FactExplorationStarted, id, e.id("exploration", id), continuity.ExplorationStartedPayload{Observation: observation, Label: title}, archive.Mapping{Name: "legacy_exploration_title_only_v1", Field: "purpose", Original: ""}); err != nil {
			rows.Close()
			return err
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	e.count("explorations", n, start, 0)
	rows, err = tx.QueryContext(ctx, `SELECT id,exploration_id,seq,purpose,conclusions,unresolved,next_action,created_at,COALESCE(operation_key,''),content_digest,COALESCE(NULLIF(observed_harness,''),'unknown'),COALESCE(NULLIF(observed_harness_version,''),'unknown') FROM exploration_checkpoints WHERE project_id=? ORDER BY exploration_id,seq,id`, e.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	start, n = len(e.records), 0
	sequences := map[string]int{}
	itemCount := 0
	for rows.Next() {
		var id, exploration, purpose, conclusions, unresolved, next, created, operation, digest, harness, version string
		var seq int
		if err := rows.Scan(&id, &exploration, &seq, &purpose, &conclusions, &unresolved, &next, &created, &operation, &digest, &harness, &version); err != nil {
			return err
		}
		n++
		sequences[exploration]++
		if sequences[exploration] != seq {
			return fmt.Errorf("checkpoint sequence gap for %q", exploration)
		}
		observation, err := legacyObservationV2(created)
		if err != nil {
			return err
		}
		items, err := tx.QueryContext(ctx, `SELECT item_type,position,content FROM exploration_checkpoint_items WHERE project_id=? AND checkpoint_id=? ORDER BY position`, e.projectID, id)
		if err != nil {
			return err
		}
		observation.ObservedHarness = harness
		observation.HarnessVersion = version
		payload := continuity.CheckpointRecordedPayload{Observation: observation, ExplorationID: e.id("exploration", exploration), CurrentFraming: purpose, Conclusions: conclusions, UnresolvedQuestion: unresolved, NextAction: next, Items: []continuity.CheckpointItem{}}
		for items.Next() {
			var kind, text string
			var position int
			if err := items.Scan(&kind, &position, &text); err != nil {
				items.Close()
				return err
			}
			if position != len(payload.Items)+1 {
				items.Close()
				return fmt.Errorf("checkpoint item order gap")
			}
			payload.Items = append(payload.Items, continuity.CheckpointItem{Kind: continuity.CheckpointItemKind(kind), Text: text})
			itemCount++
		}
		if err := items.Err(); err != nil {
			items.Close()
			return err
		}
		if err := items.Close(); err != nil {
			return err
		}
		parts := []string{purpose, conclusions, unresolved, next}
		for _, item := range payload.Items {
			parts = append(parts, string(item.Kind)+":"+item.Text)
		}
		if intentDigest(strings.Join(parts, "\x00")) != digest {
			return fmt.Errorf("checkpoint %q content checksum mismatch", id)
		}
		metadata, err := json.Marshal(struct {
			Seq           int
			OperationKey  string
			ContentDigest string
		}{seq, operation, digest})
		if err != nil {
			return err
		}
		if err := e.append(continuity.FactCheckpointRecorded, id, e.id("checkpoint", id), payload, archive.Mapping{Name: "legacy_checkpoint_operation_metadata_not_portable_v1", Field: "operation_metadata", Original: string(metadata)}); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	e.count("exploration_checkpoints", n, start, 0)
	e.counts = append(e.counts, archive.KindCount{Kind: "exploration_checkpoint_items", SourceRows: itemCount, MappedSourceRows: itemCount, MappedRecords: itemCount})
	return nil
}

func durableTrackerLocatorV2(value string) bool {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	return strings.Contains(u.Path, "/issues/") || strings.Contains(u.Path, "/issue/")
}

func (e *fullDomainExportV2) references(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT id,COALESCE(external_url,''),created_at FROM backend_mappings WHERE project_id=? AND entity_kind='issue' ORDER BY created_at,id`, e.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	start, n, retired := len(e.records), 0, 0
	locators := map[string]bool{}
	for rows.Next() {
		var id, locator, created string
		if err := rows.Scan(&id, &locator, &created); err != nil {
			return err
		}
		n++
		opaqueLocator, validLocator := trackerLocatorV2(locator)
		if !validLocator {
			retired++
			e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_issue_link_without_durable_tracker_locator_v1", Field: "backend_mappings." + id, Original: locator})
			continue
		}
		if locators[opaqueLocator] {
			e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_duplicate_tracker_locator_v1", Field: "backend_mappings." + id, Original: locator})
			continue
		}
		locators[opaqueLocator] = true
		observation, err := legacyObservationV2(created)
		if err != nil {
			return err
		}
		if err := e.append(continuity.FactExternalReferenceRegistered, id, e.id("external-reference", opaqueLocator), continuity.ExternalReferenceRegistrationPayload{Observation: observation, Locator: opaqueLocator}, archive.Mapping{Name: "legacy_tracker_url_to_opaque_locator_v1", Field: "locator", Original: locator}, archive.Mapping{Name: "legacy_issue_link_registration_without_work_mirror_v1", Field: "legacy_issue_mapping", Original: id}); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	e.count("issue_tracker_references", n, start, retired)
	return nil
}

func (e *fullDomainExportV2) claims(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT id,child_criterion_id,parent_criterion_id,created_at FROM issue_criterion_claims WHERE project_id=? ORDER BY created_at,id`, e.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	start, n := len(e.records), 0
	for rows.Next() {
		var id, child, parent, created string
		if err := rows.Scan(&id, &child, &parent, &created); err != nil {
			return err
		}
		n++
		observation, err := legacyObservationV2(created)
		if err != nil {
			return err
		}
		detail, err := json.Marshal(struct {
			Child  string `json:"child_criterion_id"`
			Parent string `json:"parent_criterion_id"`
		}{child, parent})
		if err != nil {
			return err
		}
		payload := continuity.VerificationEvidencePayload{Observation: observation, Target: continuity.SubjectRef{Kind: continuity.RecordProjectIdentity, ID: continuity.SubjectID(e.projectID)}, Check: "legacy criterion coverage claim", Method: "legacy declaration; execution was not observed", Outcome: continuity.VerificationIndeterminate, Detail: string(detail)}
		if err := e.append(continuity.FactVerificationEvidenceRecorded, id, e.id("verification-evidence", id), payload, archive.Mapping{Name: archive.LegacyCriterionClaimMappingV1, Field: "legacy_claim", Original: id}); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	e.count("issue_criterion_claims", n, start, 0)
	return nil
}

var legacyANSISequencesV2 = regexp.MustCompile("\\x1b\\[[0-?]*[ -/]*[@-~]|\\x1b\\][^\\x07\\x1b]*(?:\\x07|\\x1b\\\\)|\\x1b[@-_]")

func cleanJournalTextV2(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\u2028' || r == '\u2029' {
			return '\n'
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, legacyANSISequencesV2.ReplaceAllString(value, ""))
}

func augmentFullDomainV2(c archive.Content, e fullDomainExportV2) (archive.Content, error) {
	categories := map[string]archive.CategoryCount{}
	for _, r := range c.Records {
		if o, ok := e.journalObservations[r.SourceID]; ok && (r.Journal != nil || r.Wrap != nil) {
			r.Observation.ObservedHarness = o.ObservedHarness
			r.Observation.HarnessVersion = o.HarnessVersion
		}
		if r.Journal != nil {
			original := string(r.Journal.Category)
			for _, m := range r.Mappings {
				if m.Field == "category" {
					original = m.Original
				}
			}
			count := categories[original]
			count.Original = original
			count.Mapped = string(r.Journal.Category)
			if r.Journal.Category == continuity.JournalSpark {
				count.Mapped = "spark.captured"
			}
			count.Rows++
			categories[original] = count
		}
	}
	names := make([]string, 0, len(categories))
	for name := range categories {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c.Source.JournalCategoryCounts = append(c.Source.JournalCategoryCounts, categories[name])
	}
	records := make([]archive.Record, 0, len(c.Records)+len(e.records))
	for _, r := range c.Records {
		if o, ok := e.journalObservations[r.SourceID]; ok && (r.Journal != nil || r.Wrap != nil) {
			r.Observation.ObservedHarness = o.ObservedHarness
			r.Observation.HarnessVersion = o.HarnessVersion
		}
		if r.Journal != nil && r.Journal.Category == continuity.JournalSpark {
			key := r.Journal.Scope + "\x00" + r.Journal.Text + "\x00" + fmt.Sprint(r.Observation.ObservedAtMillis)
			if e.sparkMatches[key] {
				e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_duplicate_journal_spark_v1", Field: "journal_entries." + r.SourceID, Original: key})
				c.Source.MatchedSparkJournalRows++
				continue
			}
			c.Source.UnmatchedSparkJournalRows++
			if err := e.append(continuity.FactSparkCaptured, "journal:"+r.SourceID, e.id("spark-journal", r.SourceID), continuity.SparkCapturedPayload{Observation: r.Observation.Continuity(), Scope: r.Journal.Scope, Text: r.Journal.Text}, archive.Mapping{Name: "legacy_unmatched_journal_spark_v1", Field: "journal_entry", Original: r.SourceID}); err != nil {
				return archive.Content{}, err
			}
			continue
		}
		if r.Journal != nil {
			clean := cleanJournalTextV2(r.Journal.Text)
			if clean != r.Journal.Text {
				c.Source.NormalizedJournalTextRows++
				c.Source.JournalTextMapping = archive.JournalTextControlCharactersStrippedV1
				r.Mappings = append(r.Mappings, archive.Mapping{Name: archive.JournalTextControlCharactersStrippedV1, Field: "text", Original: r.Journal.Text})
				if strings.TrimSpace(clean) == "" {
					clean = "Legacy journal text contained only stripped control sequences."
				}
				r.Journal.Text = clean
			}
		}
		records = append(records, r)
	}
	c.Records = append(records, e.records...)
	c.Source.Kinds = append([]archive.KindCount{{Kind: "journal_entries", SourceRows: c.Source.JournalProjectionRows, MappedRecords: len(records) - 1 - c.Source.HandoffRows + c.Source.UnmatchedSparkJournalRows}, {Kind: "journal_revision_facts", SourceRows: c.Source.JournalFactRows, MappedRecords: c.Source.JournalProjectionRows}, {Kind: "handoffs", SourceRows: c.Source.HandoffRows, MappedRecords: c.Source.HandoffRows}}, e.counts...)
	if c.Source.CollapsedRevisionRows > 0 {
		e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_journal_revisions_folded_to_effective_state_v1", Field: "collapsed_revision_rows", Original: strconv.Itoa(c.Source.CollapsedRevisionRows)})
	}
	if c.Source.DroppedSpecLinks+c.Source.DroppedTaskLinks > 0 {
		e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_journal_retired_links_omitted_v1", Field: "spec_task_links", Original: fmt.Sprintf("spec=%d task=%d", c.Source.DroppedSpecLinks, c.Source.DroppedTaskLinks)})
	}
	for i := range c.Source.Kinds {
		c.Source.Kinds[i].MappedSourceRows = c.Source.Kinds[i].SourceRows - c.Source.Kinds[i].RetiredRows
	}
	c.Source.Mappings = e.mappings
	c.Families.Ideas = true
	c.Families.Sparks = true
	c.Families.Explorations = true
	c.Families.Decisions = true
	c.Families.Findings = true
	return c, nil
}

func (e *fullDomainExportV2) retiredInventory(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"specs", "intents", "tasks", "brainstorms", "plans", "work_contracts", "issues"} {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE project_id=?", e.projectID).Scan(&count); err != nil {
			return err
		}
		e.counts = append(e.counts, archive.KindCount{Kind: table, SourceRows: count, RetiredRows: count})
	}
	e.mappings = append(e.mappings, archive.Mapping{Name: "retired_inventory_is_not_byte_archive_or_tracker_disposition_v1", Field: "retired_rows", Original: "No bytes archived; open local work still requires tracker disposition before freeze."})
	return nil
}

func (e *fullDomainExportV2) legacyLifecycleEvidence(ctx context.Context, tx *sql.Tx) error {
	allKinds := append(sparkFactKinds(), ideaFactKinds()...)
	args := []any{e.projectID}
	for _, kind := range allKinds {
		args = append(args, kind)
	}
	var count, bytes int64
	query := `SELECT COUNT(*), COALESCE(SUM(length(CAST(payload AS BLOB))+length(CAST(id AS BLOB))+length(CAST(env_id AS BLOB))+length(CAST(hlc AS BLOB))),0) FROM facts WHERE project_id=? AND kind IN (` + strings.TrimSuffix(strings.Repeat("?,", len(allKinds)), ",") + `)`
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count, &bytes); err != nil {
		return err
	}
	if count > vnextRehearsalMaxRecordsV1 || bytes+int64(e.bytes) > vnextRehearsalMaxAggregatePayloadBytesV1 {
		return fmt.Errorf("legacy lifecycle source exceeds resource limits")
	}
	for _, kinds := range [][]string{sparkFactKinds(), ideaFactKinds()} {
		facts, err := listCoreEventFacts(ctx, tx, e.projectID, kinds)
		if err != nil {
			return err
		}
		for _, fact := range facts {
			for _, value := range []string{fact.ID, fact.ProjectID, fact.Kind, fact.Payload, fact.EnvID, fact.HLC, fact.Permanence} {
				if !utf8.ValidString(value) {
					return fmt.Errorf("legacy lifecycle fact contains invalid UTF-8")
				}
			}
			payload, err := decodeCoreEventPayload(fact.Payload)
			if err != nil {
				return err
			}
			key := payload.SubjectKind + "\x00" + coreEventSubjectID(fact.ID, payload)
			e.headObservations[key] = payload
			e.headFactKinds[key] = fact.Kind
			if fact.Kind == FactKindSparkPromoted {
				e.promotionObservations[key] = payload
			}
			if fact.Kind == FactKindSparkCaptured || fact.Kind == FactKindIdeaCreated {
				if _, exists := e.rootObservations[key]; !exists {
					e.rootObservations[key] = payload
				}
			}
			raw, err := json.Marshal(fact)
			if err != nil {
				return err
			}
			e.bytes += len(raw)
			if e.bytes > vnextRehearsalMaxAggregatePayloadBytesV1 {
				return fmt.Errorf("legacy lifecycle archive exceeds resource limits")
			}
			e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_causal_history_preserved_without_replaying_v1", Field: "facts." + fact.ID, Original: string(raw)})
		}
	}
	return nil
}

func (e *fullDomainExportV2) observation(created, kind, id string, head bool) (continuity.Observation, error) {
	payload := e.rootObservations[kind+"\x00"+id]
	if head {
		payload = e.headObservations[kind+"\x00"+id]
		if payload.UpdatedAt != "" {
			created = payload.UpdatedAt
		}
	}
	return observationFromLegacyPayloadV2(created, payload)
}

func observationFromLegacyPayloadV2(created string, payload CoreEventPayload) (continuity.Observation, error) {
	observation, err := legacyObservationV2(created)
	if err != nil {
		return observation, err
	}
	if payload.ObservedHarness != "" {
		observation.ObservedHarness = payload.ObservedHarness
	}
	if payload.ObservedHarnessVersion != "" {
		observation.HarnessVersion = payload.ObservedHarnessVersion
	}
	observation.HarnessSessionID = payload.HarnessSessionID
	observation.Branch = payload.Branch
	observation.Worktree = payload.Worktree
	return observation, nil
}

func trackerLocatorV2(value string) (string, bool) {
	if !durableTrackerLocatorV2(value) {
		return "", false
	}
	u, err := url.Parse(value)
	if err != nil {
		return "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if u.Host == "github.com" && len(parts) == 4 && parts[2] == "issues" {
		if _, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
			return "github:" + parts[0] + "/" + parts[1] + "#" + parts[3], true
		}
	}
	if u.Host == "linear.app" && len(parts) >= 3 && parts[1] == "issue" {
		return "linear:" + parts[0] + "/" + parts[2], true
	}
	return "legacy-tracker:" + vnextRehearsalIDV1("reference_", "tracker-locator", value), true
}

func (e *fullDomainExportV2) journalOrigins(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT * FROM journal_origins WHERE project_id=? ORDER BY journal_entry_id`, e.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return err
		}
		fields := map[string]any{}
		for i, name := range columns {
			switch value := values[i].(type) {
			case string:
				if !utf8.ValidString(value) {
					return fmt.Errorf("journal origin contains invalid UTF-8")
				}
			case []byte:
				if !utf8.Valid(value) {
					return fmt.Errorf("journal origin contains invalid UTF-8")
				}
			}
			fields[name] = values[i]
		}
		raw, err := json.Marshal(fields)
		if err != nil {
			return err
		}
		e.bytes += len(raw)
		if e.bytes > vnextRehearsalMaxAggregatePayloadBytesV1 {
			return fmt.Errorf("journal origin archive exceeds payload bounds")
		}
		entry, _ := fields["journal_entry_id"].(string)
		id := entry
		e.mappings = append(e.mappings, archive.Mapping{Name: "legacy_journal_origin_metadata_preserved_v1", Field: "journal_origins." + id, Original: string(raw)})
		harness, _ := fields["observed_harness"].(string)
		version, _ := fields["observed_harness_version"].(string)
		if harness == "" {
			harness = "unknown"
		}
		if version == "" {
			version = "unknown"
		}
		observation := archive.Observation{ObservedHarness: harness, HarnessVersion: version}
		if seen[entry] {
			previous := e.journalObservations[entry]
			if previous.ObservedHarness != harness {
				observation.ObservedHarness = "unknown"
			}
			if previous.HarnessVersion != version {
				observation.HarnessVersion = "unknown"
			}
		}
		seen[entry] = true
		e.journalObservations[entry] = observation
	}
	return rows.Err()
}
