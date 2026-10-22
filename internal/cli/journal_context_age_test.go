package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/levifig/loaf/internal/state"
)

func TestJournalContextDigestShowsAgesAndDemotesAStaleSynthesis(t *testing.T) {
	age := func(days int) *int { return &days }
	synthesis := state.JournalEntryRecord{ID: "s1", EntryType: "wrap", Scope: "project", Message: "July synthesis", CreatedAt: "2026-07-30T17:25:00Z", AgeDays: age(84)}
	scoped := state.JournalEntryRecord{ID: "w1", EntryType: "wrap", Scope: "digest", Message: "scoped wrap", CreatedAt: "2026-10-12T21:42:00Z", AgeDays: age(9)}
	recent := state.JournalEntryRecord{ID: "e1", EntryType: "discover", Scope: "x", Message: "just now", CreatedAt: "2026-10-22T00:00:00Z", AgeDays: age(0)}
	yesterday := state.JournalEntryRecord{ID: "e2", EntryType: "discover", Scope: "x", Message: "yesterday", CreatedAt: "2026-10-21T09:00:00Z", AgeDays: age(1)}
	digest := func(stale bool) state.JournalContext {
		result := state.JournalContext{
			ProjectName: "loaf", Branch: "main",
			ProjectSynthesis: state.JournalContextJournalLayer{Available: true, AvailableCount: 1, ShownCount: 1, Stale: stale, EntriesSince: 1169, Items: []state.JournalEntryRecord{synthesis}},
			BranchRecency:    state.JournalContextJournalLayer{Available: true, AvailableCount: 2, ShownCount: 2, Items: []state.JournalEntryRecord{recent, yesterday}},
			// Empty, available layers stay out of the digest.
			LatestCheckpoint:       state.JournalContextCheckpointLayer{Available: true},
			ActiveLineage:          state.JournalContextJournalLayer{Available: true},
			UnresolvedBlockers:     state.JournalContextBlockerLayer{Available: true},
			DeferredIntent:         state.JournalContextDeferralLayer{Available: true},
			ExplorationCheckpoints: state.JournalContextCheckpointsLayer{Available: true},
			TransitionalTasks:      state.JournalContextTaskLayer{Available: true},
		}
		if stale {
			result.LatestCheckpoint = state.JournalContextCheckpointLayer{Available: true, AvailableCount: 1, ShownCount: 1, Items: []state.JournalContextCheckpointItem{{Entry: scoped, Scope: "digest", Label: "latest checkpoint (not project synthesis)"}}}
		}
		return result
	}
	render := func(result state.JournalContext) string {
		return renderJournalContextDigest(composeJournalContextCLIResult(result, activeChangeLayer{Available: true}, ""), journalContextDigestBudget)
	}

	stale := render(digest(true))
	want := strings.Join([]string{
		"loaf journal context: project loaf, branch main",
		"Entries are shortened to one line; read one in full with: loaf journal show <id>",
		"scoped-checkpoint: showing 1 of 1",
		"  latest checkpoint (not project synthesis): [2026-10-12 21:42, 9 days ago] w1 wrap(digest): scoped wrap",
		"branch-recency: showing 2 of 2",
		"  [2026-10-22 00:00, today] e1 discover(x): just now",
		"  [2026-10-21 09:00, 1 day ago] e2 discover(x): yesterday",
		"project-synthesis: showing 1 of 1 (stale, 1169 entries since; write a fresh wrap(project) when you finish substantial work)",
		"  [2026-07-30 17:25, 84 days ago] s1 wrap(project): July synthesis",
	}, "\n")
	if stale != want {
		t.Fatalf("stale digest mismatch\n--- got ---\n%s\n--- want ---\n%s", stale, want)
	}

	fresh := render(digest(false))
	if !strings.HasPrefix(fresh, "loaf journal context: project loaf, branch main\nEntries are shortened to one line; read one in full with: loaf journal show <id>\nproject-synthesis: showing 1 of 1\n  [2026-07-30 17:25, 84 days ago] s1 wrap(project): July synthesis\n") || strings.Contains(fresh, "stale") {
		t.Fatalf("fresh digest = %q, want the synthesis first with no stale label", fresh)
	}

	var layer bytes.Buffer
	writeJournalContextHuman(&layer, composeJournalContextCLIResult(digest(true), activeChangeLayer{}, journalContextLayerProjectSynthesis))
	if got := layer.String(); !strings.Contains(got, "    stale, 1169 entries since; write a fresh wrap(project) when you finish substantial work\n    [2026-07-30 17:25, 84 days ago] wrap(project): July synthesis") {
		t.Fatalf("project-synthesis layer = %q, want the stale label and the aged entry", got)
	}
}

func TestJournalContextDefaultPathAgesTheSynthesisByTheReadClock(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	log := func(entry string) {
		t.Helper()
		var output bytes.Buffer
		if err := (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", entry}); err != nil {
			t.Fatalf("journal log %q error = %v\n%s", entry, err, output.String())
		}
	}
	context := func(now func() time.Time, args ...string) string {
		t.Helper()
		var output bytes.Buffer
		if err := (Runner{Stdout: &output, WorkingDir: workingDir, StateHome: stateHome, Now: now}).Run(append([]string{"journal", "context"}, args...)); err != nil {
			t.Fatalf("journal context error = %v\n%s", err, output.String())
		}
		return output.String()
	}
	log("wrap(project): the first synthesis")
	for i := 0; i < state.JournalContextSynthesisStaleEntries; i++ {
		log("discover(work): progress")
	}
	later := func() time.Time { return time.Now().AddDate(0, 0, state.JournalContextSynthesisStaleDays+2) }

	stale := context(later)
	synthesisAt := strings.Index(stale, "project-synthesis: showing 1 of 1 (stale, 50 entries since; write a fresh wrap(project) when you finish substantial work)")
	if synthesisAt < 0 || synthesisAt < strings.Index(stale, "branch-recency:") || !strings.Contains(stale, "32 days ago]") {
		t.Fatalf("digest a month on = %q, want the stale synthesis last and aged entries", stale)
	}

	var jsonOut bytes.Buffer
	if err := (Runner{Stdout: &jsonOut, WorkingDir: workingDir, StateHome: stateHome, Now: later}).Run([]string{"journal", "context", "--json"}); err != nil {
		t.Fatalf("journal context --json error = %v", err)
	}
	var result journalContextCLIResult
	if err := json.Unmarshal(jsonOut.Bytes(), &result); err != nil {
		t.Fatalf("decode context JSON: %v", err)
	}
	synthesis := result.Layers.ProjectSynthesis
	if synthesis == nil || !synthesis.Stale || synthesis.EntriesSince != state.JournalContextSynthesisStaleEntries || synthesis.Items[0].AgeDays == nil || *synthesis.Items[0].AgeDays != state.JournalContextSynthesisStaleDays+2 {
		t.Fatalf("JSON synthesis = %#v, want stale with entries_since and age_days", synthesis)
	}

	log("wrap(project): a fresh synthesis")
	fresh := context(later)
	if !strings.Contains(fresh, "\nproject-synthesis: showing 1 of 1\n  [") || !strings.Contains(fresh, "wrap(project): a fresh synthesis") || strings.Contains(fresh, "stale,") {
		t.Fatalf("digest after a fresh wrap = %q, want the new synthesis first with no stale label", fresh)
	}
	if strings.Index(fresh, "project-synthesis:") > strings.Index(fresh, "branch-recency:") {
		t.Fatalf("digest after a fresh wrap = %q, want the synthesis to lead", fresh)
	}
}
