package archive

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/levifig/loaf/vnext/continuity"
)

func fullFixtureV2(t *testing.T) Content {
	t.Helper()
	c := validContentV1()
	c.Families.Ideas = true
	c.Families.Sparks = true
	c.Families.Decisions = true
	c.Families.Explorations = true
	c.Families.Findings = true
	c.Source.JournalCategoryCounts = []CategoryCount{{Original: "discover", Mapped: "discover", Rows: 1}}
	for _, name := range []string{"journal_entries", "journal_revision_facts", "handoffs", "ideas", "sparks", "explorations", "exploration_checkpoints", "exploration_checkpoint_items", "issue_tracker_references", "issue_criterion_claims", "specs", "intents", "tasks", "brainstorms", "plans", "work_contracts", "issues"} {
		k := KindCount{Kind: name}
		if name == "journal_entries" || name == "journal_revision_facts" {
			k.SourceRows = 2
			k.MappedSourceRows = 2
			k.MappedRecords = 2
		}
		c.Source.Kinds = append(c.Source.Kinds, k)
	}
	observation := continuity.Observation{ObservedAtMillis: 2000, ObservedHarness: "codex", HarnessVersion: "fixture"}
	raw, err := json.Marshal(continuity.IdeaCreatedPayload{Observation: observation, Content: continuity.IdeaContent{Label: "An idea", Text: "Exact body\n"}})
	if err != nil {
		t.Fatal(err)
	}
	c.Records = append(c.Records, Record{Kind: RecordKind(continuity.FactIdeaCreated), SourceID: "legacy-idea", FactID: "idea-fact", SubjectID: "idea", Payload: raw})
	for i := range c.Source.Kinds {
		if c.Source.Kinds[i].Kind == "ideas" {
			c.Source.Kinds[i].SourceRows = 1
			c.Source.Kinds[i].MappedSourceRows = 1
			c.Source.Kinds[i].MappedRecords = 1
		}
	}
	return c
}

func TestFullDomainV2SealParsePreservesV1AndRejectsFalseCoverage(t *testing.T) {
	c := fullFixtureV2(t)
	a, err := SealV2(c)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(raw)
	if err != nil || again.ContentSHA256 != a.ContentSHA256 {
		t.Fatalf("roundtrip: %v", err)
	}
	if a.Content.Expected.IdeaCount != 1 {
		t.Fatal("idea absent from manifest")
	}
	if _, err := Seal(c); err == nil {
		t.Fatal("v1 accepted v2 coverage")
	}
	for name, mutate := range map[string]func(*Content){
		"missing kind": func(c *Content) { c.Source.Kinds = c.Source.Kinds[:len(c.Source.Kinds)-1] },
		"false idea count": func(c *Content) {
			for i := range c.Source.Kinds {
				if c.Source.Kinds[i].Kind == "ideas" {
					c.Source.Kinds[i].MappedRecords = 999
				}
			}
		},
		"negative categories": func(c *Content) { c.Source.JournalCategoryCounts[0].Rows = -1 },
		"impossible revision accounting": func(c *Content) {
			c.Source.JournalFactRows = 0
			c.Source.CollapsedRevisionRows = 99
			for i := range c.Source.Kinds {
				if c.Source.Kinds[i].Kind == "journal_revision_facts" {
					c.Source.Kinds[i].SourceRows = 0
					c.Source.Kinds[i].MappedSourceRows = 0
				}
			}
		},
		"invalid retained text": func(c *Content) {
			c.Source.Mappings = []Mapping{{Name: "retained", Field: "original", Original: string([]byte{0xff})}}
		},
		"invalid source id": func(c *Content) { c.Records[len(c.Records)-1].SourceID = string([]byte{0xff}) },
		"unknown payload member": func(c *Content) {
			r := &c.Records[len(c.Records)-1]
			r.Payload = []byte(strings.TrimSuffix(string(r.Payload), "}") + `,"Future":"silently lost"}`)
		},
		"dangling predecessor": func(c *Content) {
			raw, _ := json.Marshal(continuity.IdeaResolutionPayload{Observation: continuity.Observation{ObservedAtMillis: 2000}, Predecessor: "absent", Resolution: "done"})
			c.Records = append(c.Records, Record{Kind: RecordKind(continuity.FactIdeaResolved), SourceID: "legacy-idea", FactID: "terminal", SubjectID: "idea", Payload: raw})
			for i := range c.Source.Kinds {
				if c.Source.Kinds[i].Kind == "ideas" {
					c.Source.Kinds[i].MappedRecords++
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := fullFixtureV2(t)
			mutate(&c)
			if _, err := SealV2(c); err == nil {
				t.Fatal("invalid full-domain archive accepted")
			}
		})
	}
}

func TestPrepareRecordFailsClosedOnMismatchedPayload(t *testing.T) {
	for _, r := range []Record{{Kind: RecordProject}, {Kind: RecordJournal, Wrap: &WrapRecord{Synthesis: "wrong"}}} {
		if _, err := PrepareRecord("project", r); err == nil {
			t.Fatal("bad payload accepted")
		}
	}
}

func TestFullDomainArchiveRejectsAmplifyingTypedListsAndDuplicateMembers(t *testing.T) {
	for _, raw := range []string{`{"records":[` + strings.Repeat(`{},`, maxRecords) + `{}]}`, `{"payload":{"Text":"first","Text":"second"}}`} {
		if err := inspectJSONBounds([]byte(raw)); err == nil {
			t.Fatal("unsafe JSON accepted")
		}
	}
}
