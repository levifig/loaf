package state

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/levifig/loaf/vnext/continuity"
	continuitysqlite "github.com/levifig/loaf/vnext/continuity/sqlite"
	"github.com/levifig/loaf/vnext/migration/archive"
	"github.com/levifig/loaf/vnext/migration/rehearsal"
)

func TestFullDomainArchiveLifecycleBodiesSparkDedupAndReplay(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	idea, err := CaptureIdea(ctx, root, resolver, IdeaCaptureOptions{Title: "Preserved idea"})
	if err != nil {
		t.Fatal(err)
	}
	spark, err := CaptureSpark(ctx, root, resolver, SparkCaptureOptions{Text: "matched spark", Scope: "migration"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PromoteSpark(ctx, root, resolver, SparkPromoteOptions{Spark: spark.Spark.ID, ToIdea: idea.Idea.ID}); err != nil {
		t.Fatal(err)
	}
	archived, err := CaptureIdea(ctx, root, resolver, IdeaCaptureOptions{Title: "Archived idea"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ArchiveIdeas(ctx, root, resolver, IdeaArchiveOptions{Refs: []string{archived.Idea.ID}, Reason: "kept as evidence"}); err != nil {
		t.Fatal(err)
	}
	exploration, err := CreateExploration(ctx, root, resolver, ExplorationCreateOptions{Title: "Migration exploration"})
	if err != nil {
		t.Fatal(err)
	}
	for _, conclusion := range []string{"old conclusion", "latest conclusion"} {
		if _, err := AppendExplorationCheckpoint(ctx, root, resolver, ExplorationCheckpointOptions{ExplorationRef: exploration.Exploration.ID, Purpose: "framing", Conclusions: conclusion, Unresolved: "question", NextAction: "next", Items: []CheckpointItemInput{{Type: "evidence", Content: "observed"}}}); err != nil {
			t.Fatal(err)
		}
	}
	store, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	body := "# Original body\n\nExact bytes  \n"
	now := time.Now().UTC().Format(time.RFC3339Nano)
	mustExec(t, store, `INSERT INTO artifact_bodies (id,project_id,entity_kind,entity_id,body_kind,content,content_hash,created_at,updated_at) VALUES ('body',?,'idea',?,'markdown',?,?,?,?)`, status.ProjectID, idea.Idea.ID, body, artifactBodyHash(body), now, now)
	var created string
	if err := store.db.QueryRow(`SELECT created_at FROM sparks WHERE id=?`, spark.Spark.ID).Scan(&created); err != nil {
		t.Fatal(err)
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for id, p := range map[string]JournalFactPayload{
		"matched":   {EntryType: "spark", Scope: "migration", Message: "matched spark", CreatedAt: created, UpdatedAt: created},
		"unmatched": {EntryType: "spark", Message: "unmatched spark", CreatedAt: created, UpdatedAt: created},
		"control":   {EntryType: "review", Message: "keep \x1b[31mred\x1b[0m\x00 text\n", CreatedAt: created, UpdatedAt: created},
		"category":  {EntryType: "task", Message: "retired label", CreatedAt: created, UpdatedAt: created},
	} {
		observed, err := time.Parse(time.RFC3339Nano, p.CreatedAt)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := appendJournalFactTx(ctx, tx, status.ProjectID, id, p, observed); err != nil {
			t.Fatal(err)
		}
		if err := insertJournalProjectionTx(ctx, tx, status.ProjectID, id, p); err != nil {
			t.Fatal(err)
		}
		if err := insertJournalSearchTx(ctx, tx, status.ProjectID, id, "", p.EntryType, p.Scope, p.Message); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: backup, ProjectID: status.ProjectID, Root: root, Resolver: resolver})
	if err != nil {
		t.Fatal(err)
	}
	a, err := archive.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if a.Version != 2 || a.Content.Source.MatchedSparkJournalRows != 1 || a.Content.Source.UnmatchedSparkJournalRows != 1 {
		t.Fatalf("wrong v2/dedup receipt: %+v", a.Content.Source)
	}
	destination, err := continuitysqlite.Open(filepath.Join(t.TempDir(), "destination"), "environment-full-domain")
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	first, err := rehearsal.Import(ctx, encoded, destination)
	if err != nil {
		t.Fatal(err)
	}
	again, err := rehearsal.Import(ctx, encoded, destination)
	if err != nil || again != first {
		t.Fatalf("non-idempotent replay: %v", err)
	}
	snapshot, err := destination.Snapshot(ctx, continuity.ProjectID(status.ProjectID), continuity.SnapshotRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.ActiveSparks.Sparks) != 1 || snapshot.ActiveSparks.Sparks[0].Text != "unmatched spark" {
		t.Fatalf("spark dedup/lifecycle: %+v", snapshot.ActiveSparks)
	}
	if len(snapshot.CurrentIdeas.Ideas) != 2 {
		t.Fatal("lost ideas")
	}
	foundBody, foundArchive := false, false
	for _, i := range snapshot.CurrentIdeas.Ideas {
		if i.Content.Label == "Preserved idea" {
			foundBody = i.Content.Text == body
		}
		if i.Content.Label == "Archived idea" {
			foundArchive = i.Disposition == continuity.IdeaArchived
		}
	}
	if !foundBody || !foundArchive {
		t.Fatal("body or archive lifecycle lost")
	}
	if len(snapshot.LatestCheckpoints.Checkpoints) != 1 || snapshot.LatestCheckpoints.Checkpoints[0].Conclusions != "latest conclusion" {
		t.Fatal("latest checkpoint wrong")
	}
	if len(snapshot.EffectiveJournal.Entries) != 2 {
		t.Fatal("spark leaked into journal")
	}
	for _, j := range snapshot.EffectiveJournal.Entries {
		if j.Content.Category == continuity.JournalReview && j.Content.Text != "keep red text\n" {
			t.Fatalf("control mapping wrong: %q", j.Content.Text)
		}
	}
	if !json.Valid(encoded) {
		t.Fatal("invalid JSON")
	}
}

func TestFullDomainArchiveNormalizesWrapAndJournalSparkControls(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{"wrap(archive): retained \x1b[31msynthesis\x1b[0m\x00", "spark(archive): retained \x1b[31mcapture\x1b[0m\x00"} {
		if _, err := LogJournal(ctx, root, resolver, JournalLogOptions{Entry: entry}); err != nil {
			t.Fatal(err)
		}
	}
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: backup, ProjectID: status.ProjectID, Root: root, Resolver: resolver})
	if err != nil {
		t.Fatal(err)
	}
	a, err := archive.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if a.Content.Source.NormalizedJournalTextRows != 2 || a.Content.Source.UnmatchedSparkJournalRows != 1 {
		t.Fatal("missing normalization counts")
	}
	retained := 0
	for _, record := range a.Content.Records {
		for _, mapping := range record.Mappings {
			if mapping.Name == archive.JournalTextControlCharactersStrippedV1 {
				retained++
			}
		}
		if record.Wrap != nil && record.Wrap.Synthesis != "retained synthesis" {
			t.Fatal("wrap text was not normalized")
		}
		if record.Kind == archive.RecordKind(continuity.FactSparkCaptured) {
			var payload continuity.SparkCapturedPayload
			if err := json.Unmarshal(record.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Text != "retained capture" {
				t.Fatal("journal spark text was not normalized")
			}
		}
	}
	if retained != 2 {
		t.Fatal("original journal text declaration lost during conversion")
	}
	destination, err := continuitysqlite.Open(filepath.Join(t.TempDir(), "destination"), "environment-journal-controls")
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	if _, err := rehearsal.Import(ctx, encoded, destination); err != nil {
		t.Fatal(err)
	}
}

func TestFullDomainArchiveRecoversFactBodyAndNativeObservationAndDurableReference(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	idea, err := CaptureIdea(ctx, root, resolver, IdeaCaptureOptions{Title: "Fact-backed idea"})
	if err != nil {
		t.Fatal(err)
	}
	spark, err := CaptureSpark(ctx, root, resolver, SparkCaptureOptions{Text: "promoted later"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy fixtures retain trustworthy attribution and a body available only
	// in the immutable event payload. Backdate capture, then use native promotion.
	old := time.Now().Add(-10 * time.Second).UTC().Format(time.RFC3339)
	for _, entity := range []struct{ kind, id string }{{"idea", idea.Idea.ID}, {"spark", spark.Spark.ID}} {
		rows, err := store.db.Query(`SELECT id,payload FROM facts WHERE project_id=? AND kind=?`, status.ProjectID, entity.kind+map[string]string{"idea": ".created", "spark": ".captured"}[entity.kind])
		if err != nil {
			t.Fatal(err)
		}
		var factID, raw string
		if !rows.Next() {
			t.Fatal("missing fixture fact")
		}
		if err := rows.Scan(&factID, &raw); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		payload, err := decodeCoreEventPayload(raw)
		if err != nil {
			t.Fatal(err)
		}
		payload.ObservedHarness = "amp"
		payload.ObservedHarnessVersion = "fixture-version"
		payload.HarnessSessionID = "fixture-conversation"
		payload.Branch = "fixture-branch"
		payload.Worktree = "/fixture/worktree"
		if entity.kind == "idea" {
			payload.Body = "Exact fact-backed body\n"
		} else {
			payload.CreatedAt = old
			payload.UpdatedAt = old
			mustExec(t, store, `UPDATE sparks SET created_at=?,updated_at=? WHERE id=?`, old, old, entity.id)
		}
		encoded, err := encodeCoreEventPayload(payload)
		if err != nil {
			t.Fatal(err)
		}
		mustExec(t, store, `UPDATE facts SET payload=? WHERE id=?`, encoded, factID)
	}
	mapped, err := store.CreateIssue(ctx, root, IssueCreateOptions{Title: "Mapped legacy issue", Alias: "EX-287", Criteria: []IssueCriterionInput{{Text: "parent criterion", Tier: "H"}, {Text: "child criterion", Tier: "H"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.BindLinearIssue(ctx, root, mapped.ID, "EX-287", "https://linear.app/loaf/issue/EX-287"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	mustExec(t, store, `INSERT INTO issue_criterion_claims (id,project_id,child_criterion_id,parent_criterion_id,created_at,updated_at) VALUES ('claim',?,?,?,?,?)`, status.ProjectID, mapped.Criteria[1].ID, mapped.Criteria[0].ID, now, now)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := PromoteSpark(ctx, root, resolver, SparkPromoteOptions{Spark: spark.Spark.ID, ToIdea: idea.Idea.ID}); err != nil {
		t.Fatal(err)
	}
	exploration, err := CreateExploration(ctx, root, resolver, ExplorationCreateOptions{Title: "Native provenance"})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := AppendExplorationCheckpoint(ctx, root, resolver, ExplorationCheckpointOptions{ExplorationRef: exploration.Exploration.ID, Purpose: "purpose", Conclusions: "conclusion", Unresolved: "question", NextAction: "action"})
	if err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(status.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, store, `UPDATE explorations SET observed_harness='cursor',observed_harness_version='fixture' WHERE id=?`, exploration.Exploration.ID)
	mustExec(t, store, `UPDATE exploration_checkpoints SET observed_harness='opencode',observed_harness_version='fixture' WHERE id=?`, checkpoint.Checkpoint.ID)
	var promotionRaw string
	if err := store.db.QueryRow(`SELECT payload FROM facts WHERE project_id=? AND kind=?`, status.ProjectID, FactKindSparkPromoted).Scan(&promotionRaw); err != nil {
		t.Fatal(err)
	}
	promotionSource, err := decodeCoreEventPayload(promotionRaw)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: backup, ProjectID: status.ProjectID, Root: root, Resolver: resolver})
	if err != nil {
		t.Fatal(err)
	}
	a, err := archive.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	foundPromotion := false
	for _, record := range a.Content.Records {
		if record.Kind == archive.RecordKind(continuity.FactSparkPromotedToIdea) {
			var payload continuity.SparkPromotionPayload
			if err := json.Unmarshal(record.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			want, err := parseVNextRehearsalTimeV1(promotionSource.UpdatedAt, "updated_at")
			if err != nil || payload.Observation.ObservedAtMillis != want {
				t.Fatal("promotion observation used the unchanged capture-row timestamp")
			}
			foundPromotion = true
		}
	}
	if !foundPromotion {
		t.Fatal("missing promotion fact")
	}
	destination, err := continuitysqlite.Open(filepath.Join(t.TempDir(), "destination"), "environment-fidelity")
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	if _, err := rehearsal.Import(ctx, encoded, destination); err != nil {
		t.Fatal(err)
	}
	snapshot, err := destination.Snapshot(ctx, continuity.ProjectID(status.ProjectID), continuity.SnapshotRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.CurrentIdeas.Ideas) != 1 || snapshot.CurrentIdeas.Ideas[0].Content.Text != "Exact fact-backed body\n" || snapshot.CurrentIdeas.Ideas[0].ContentObservation.ObservedHarness != "amp" || snapshot.CurrentIdeas.Ideas[0].ContentObservation.Branch != "fixture-branch" {
		t.Fatal("fact body or native provenance lost")
	}
	if len(snapshot.ActiveSparks.Sparks) != 0 {
		t.Fatal("later promotion remained active")
	}
	if len(snapshot.ExternalReferences.References) != 1 || snapshot.ExternalReferences.References[0].Locator != "linear:loaf/EX-287" {
		t.Fatal("durable native locator lost")
	}
	if len(snapshot.VerificationEvidence.Evidence) != 1 || snapshot.VerificationEvidence.Evidence[0].Outcome != continuity.VerificationIndeterminate {
		t.Fatal("criterion claim became execution proof")
	}
	if snapshot.Explorations.Explorations[0].HeadObservation.ObservedHarness != "cursor" || snapshot.LatestCheckpoints.Checkpoints[0].HeadObservation.ObservedHarness != "opencode" {
		t.Fatal("stored exploration provenance lost")
	}
}

func TestFullDomainArchiveRejectsInvalidUTF8BeforeJSONEncoding(t *testing.T) {
	e := fullDomainExportV2{projectID: "project-fixture"}
	observation := continuity.Observation{ObservedHarness: "unknown", HarnessVersion: "unknown"}
	invalid := string([]byte{0xff})
	for _, tc := range []struct{ name, source, text string }{
		{"body", "idea", invalid},
		{"source", invalid, "valid body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := e.append(continuity.FactIdeaCreated, tc.source, "subject-fixture", continuity.IdeaCreatedPayload{Observation: observation, Content: continuity.IdeaContent{Label: "Idea", Text: tc.text}}); err == nil {
				t.Fatal("invalid source bytes were silently replaced during JSON encoding")
			}
		})
	}
}

func TestFullDomainArchiveRejectsInvalidOriginBeforeJSONEncoding(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LogJournal(ctx, root, resolver, JournalLogOptions{Entry: "note: valid text", Origin: &JournalOriginInput{EnvelopeVersion: 1, CaptureMechanism: JournalOriginMechanismManual, AgentID: string([]byte{0xff})}}); err != nil {
		t.Fatal(err)
	}
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: backup, ProjectID: status.ProjectID, Root: root, Resolver: resolver}); err == nil {
		t.Fatal("invalid origin bytes silently changed")
	}
}

func TestTrackerArchiveLocatorsPreserveNativeReferenceMeaning(t *testing.T) {
	for input, want := range map[string]string{"https://github.com/levifig/loaf/issues/287": "github:levifig/loaf#287", "https://linear.app/loaf/issue/EX-287/title": "linear:loaf/EX-287"} {
		got, ok := trackerLocatorV2(input)
		if !ok || got != want {
			t.Fatalf("locator %q: %q", input, got)
		}
	}
	if _, ok := trackerLocatorV2("https://example.com/issues/1?access_token=private"); ok {
		t.Fatal("query-bearing URL accepted")
	}
	a, _ := trackerLocatorV2("https://linear.app/alpha/issue/ENG-1")
	b, _ := trackerLocatorV2("https://linear.app/beta/issue/ENG-1")
	if a == b {
		t.Fatal("distinct Linear workspaces collapsed")
	}
	a, _ = trackerLocatorV2("https://github.com/levifig/loaf/issues/287")
	b, _ = trackerLocatorV2("https://github.com:8443/levifig/loaf/issues/287")
	if a == b {
		t.Fatal("distinct tracker origins collapsed")
	}
}

func TestFullDomainArchiveUsesLatestRepeatedPromotion(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	status, err := Initialize(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	spark, err := CaptureSpark(ctx, root, resolver, SparkCaptureOptions{Text: "Repeated promotion"})
	if err != nil {
		t.Fatal(err)
	}
	var latest string
	for _, title := range []string{"First target", "Effective target"} {
		idea, err := CaptureIdea(ctx, root, resolver, IdeaCaptureOptions{Title: title})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PromoteSpark(ctx, root, resolver, SparkPromoteOptions{Spark: spark.Spark.ID, ToIdea: idea.Idea.ID}); err != nil {
			t.Fatal(err)
		}
		latest = idea.Idea.ID
	}
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: backup, ProjectID: status.ProjectID, Root: root, Resolver: resolver})
	if err != nil {
		t.Fatal(err)
	}
	a, err := archive.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	want := (&fullDomainExportV2{projectID: status.ProjectID}).id("idea", latest)
	found := false
	for _, record := range a.Content.Records {
		if record.Kind == archive.RecordKind(continuity.FactSparkPromotedToIdea) {
			var payload continuity.SparkPromotionPayload
			if err := json.Unmarshal(record.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload.IdeaID != want {
				t.Fatal("promotion selected stale relationship")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing promotion")
	}
}

func TestFullDomainArchivePreservesMarkdownImportedPromotion(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	resolver := PathResolver{StateHome: t.TempDir()}
	writeAgentsFile(t, root.Path(), "ideas/20260528-idea.md", "# Imported Idea\n")
	writeAgentsFile(t, root.Path(), "sessions/20260528-session.md", "[2026-05-28 10:00] spark(scope): insight captured to .agents/ideas/20260528-idea.md\n")
	result, err := ApplyMarkdownMigration(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(result.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	var relationships, promotionFacts int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM relationships WHERE relationship_type='promoted_to' AND from_entity_kind='spark'`).Scan(&relationships); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM facts WHERE kind=?`, FactKindSparkPromoted).Scan(&promotionFacts); err != nil {
		t.Fatal(err)
	}
	var sparkID, ideaID string
	if err := store.db.QueryRow(`SELECT id FROM sparks`).Scan(&sparkID); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT id FROM ideas`).Scan(&ideaID); err != nil {
		t.Fatal(err)
	}
	// Model an older completed import before exercising today's native writes.
	mustExec(t, store, `UPDATE sparks SET created_at='2026-05-28T10:00:00Z'`)
	mustExec(t, store, `UPDATE ideas SET created_at='2026-05-28T10:00:00Z'`)
	store.Close()
	if relationships != 1 || promotionFacts != 0 {
		t.Fatalf("fixture did not exercise relationship-only legacy promotion: %d/%d", relationships, promotionFacts)
	}
	backup, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: backup, ProjectID: result.ProjectID, Root: root, Resolver: resolver})
	if err != nil {
		t.Fatal(err)
	}
	a, err := archive.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, record := range a.Content.Records {
		if record.Kind != archive.RecordKind(continuity.FactSparkPromotedToIdea) {
			continue
		}
		var payload continuity.SparkPromotionPayload
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Observation.ObservedHarness != "unknown" {
			t.Fatal("fabricated native promotion attribution")
		}
		for _, mapping := range record.Mappings {
			found = found || mapping.Name == "legacy_relationship_only_promotion_with_unknown_provenance_v1"
		}
	}
	if !found {
		t.Fatal("missing declared relationship-only promotion")
	}
	store, err = OpenStore(result.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, store, `UPDATE relationships SET to_entity_id='absent' WHERE relationship_type='promoted_to'`)
	store.Close()
	broken, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: broken, ProjectID: result.ProjectID, Root: root, Resolver: resolver}); err == nil {
		t.Fatal("dangling relationship-only promotion silently omitted")
	}
	store, err = OpenStore(result.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, store, `UPDATE relationships SET to_entity_id=? WHERE relationship_type='promoted_to'`, ideaID)
	store.Close()
	if _, err := PromoteSpark(ctx, root, resolver, SparkPromoteOptions{Spark: sparkID, ToIdea: ideaID}); err != nil {
		t.Fatal(err)
	}
	if _, err := ArchiveIdeas(ctx, root, resolver, IdeaArchiveOptions{Refs: []string{ideaID}, Reason: "later native lifecycle"}); err != nil {
		t.Fatal(err)
	}
	later, err := Backup(ctx, root, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExportVNextRehearsalArchive(ctx, VNextRehearsalExportOptions{Backup: later, ProjectID: result.ProjectID, Root: root, Resolver: resolver}); err != nil {
		t.Fatalf("partial creation lineage rejected later native lifecycle: %v", err)
	}
}
