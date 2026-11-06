package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/levifig/loaf/internal/state"
)

func digestEntry(id, entryType, scope, message, created, branch string) state.JournalEntryRecord {
	return state.JournalEntryRecord{ID: id, EntryType: entryType, Scope: scope, Message: message, CreatedAt: created, ObservedBranch: branch}
}

// digestFixture is a small, fully populated default context: every layer kind,
// an empty layer, an unavailable layer, a truncated layer, and a diagnostic.
func digestFixture() journalContextCLIResult {
	synthesis := state.JournalContextJournalLayer{Available: true, AvailableCount: 1, ShownCount: 1, Items: []state.JournalEntryRecord{
		digestEntry("e-synth", "wrap", "project", "shipped the digest budget;\nnext: mark stale items", "2026-10-20T10:00:00Z", "main"),
	}}
	checkpoint := state.JournalContextCheckpointLayer{Available: true, Items: []state.JournalContextCheckpointItem{}}
	lineage := state.JournalContextJournalLayer{Available: false, Items: []state.JournalEntryRecord{}}
	blockers := state.JournalContextBlockerLayer{Available: true, AvailableCount: 5, ShownCount: 2, Truncated: true, Cursor: "opaque-cursor", Items: []state.JournalContextBlockerItem{
		{Key: "ship", Block: digestEntry("e-block-2", "block", "ship", "waiting on review", "2026-10-19T09:30:00Z", "feature/x")},
		{Key: "pr-12", Block: digestEntry("e-block-1", "block", "pr-12", "rebase approval required", "2026-10-18T08:15:00Z", "main")},
	}}
	deferrals := state.JournalContextDeferralLayer{Available: true, AvailableCount: 1, ShownCount: 1, Items: []state.JournalContextDeferralItem{
		{Intent: &state.JournalContextDeferredIntent{ID: "intent-1", Alias: "INTENT-1", Disposition: "deferred", Body: "revisit prompt recall"}},
	}}
	explorations := state.JournalContextCheckpointsLayer{Available: true, AvailableCount: 1, ShownCount: 1, Items: []state.JournalContextExplorationCheckpointItem{
		{ExplorationID: "expl-1", Alias: "EXPL-1", Title: "Digest shape", NextAction: "compare budgets"},
	}}
	active := activeChangeLayer{Available: true, Items: []activeChangeItem{}}
	branch := state.JournalContextJournalLayer{Available: true, AvailableCount: 2, ShownCount: 2, Items: []state.JournalEntryRecord{
		digestEntry("e-branch-2", "decision", "digest", "one line per entry", "2026-10-21T12:00:00Z", "main"),
		digestEntry("e-branch-1", "discover", "hooks", "Claude caps hook context at 10,000 characters", "2026-10-21T11:00:00Z", "main"),
	}}
	tasks := state.JournalContextTaskLayer{Available: true, Items: []state.JournalContextTask{}}
	return journalContextCLIResult{
		ProjectName: "loaf",
		Branch:      "main",
		Layers: journalContextLayersJSON{
			ProjectSynthesis:       &synthesis,
			ScopedCheckpoint:       &checkpoint,
			ActiveLineage:          &lineage,
			UnresolvedBlockers:     &blockers,
			DeferredIntent:         &deferrals,
			ExplorationCheckpoints: &explorations,
			ActiveChanges:          &active,
			BranchRecency:          &branch,
			TransitionalTasks:      &tasks,
		},
		Diagnostics: []state.JournalContextDiagnostic{{Code: "change-source-unavailable", Message: "Change source unavailable"}},
	}
}

func TestJournalContextDigestRendersCompactLayers(t *testing.T) {
	got := renderJournalContextDigest(digestFixture(), journalContextDigestBudget)
	want := strings.Join([]string{
		"loaf journal context: project loaf, branch main",
		"Entries are shortened to one line; read one in full with: loaf journal show <id>",
		"project-synthesis: showing 1 of 1",
		"  [2026-10-20 10:00] e-synth wrap(project): shipped the digest budget; next: mark stale items",
		"active-lineage: unavailable",
		"unresolved-blockers: showing 2 of 5 (more: loaf journal context --layer unresolved-blockers --limit 5; resolve one: loaf journal log \"unblock(<id>): how it was resolved\")",
		"  [2026-10-19 09:30] e-block-2 block(ship): waiting on review (branch: feature/x)",
		"  [2026-10-18 08:15] e-block-1 block(pr-12): rebase approval required",
		"deferred-intent: showing 1 of 1 (read one: loaf intent show <id>)",
		"  INTENT-1 (deferred): revisit prompt recall",
		"exploration-checkpoints: showing 1 of 1 (resume one: loaf exploration context <id>)",
		"  EXPL-1: Digest shape / next: compare budgets",
		"branch-recency: showing 2 of 2",
		"  [2026-10-21 12:00] e-branch-2 decision(digest): one line per entry",
		"  [2026-10-21 11:00] e-branch-1 discover(hooks): Claude caps hook context at 10,000 characters",
		"diagnostic change-source-unavailable: Change source unavailable",
	}, "\n")
	if got != want {
		t.Fatalf("digest mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if strings.Contains(got, "opaque-cursor") || strings.Contains(got, "--cursor") {
		t.Fatalf("default digest must carry no cursor tokens:\n%s", got)
	}
}

// bulkyDigestFixture fills four layers with ten long entries each, so any
// realistic budget forces the packer to leave items out.
func bulkyDigestFixture() journalContextCLIResult {
	result := digestFixture()
	long := strings.Repeat("detail ", 80)
	journalItems := func(prefix, entryType string) []state.JournalEntryRecord {
		items := []state.JournalEntryRecord{}
		for i := 0; i < 10; i++ {
			items = append(items, digestEntry(fmt.Sprintf("%s-%02d", prefix, i), entryType, prefix, fmt.Sprintf("item %02d %s", i, long), "2026-10-21T12:00:00Z", "main"))
		}
		return items
	}
	blockers := []state.JournalContextBlockerItem{}
	for _, entry := range journalItems("block", "block") {
		blockers = append(blockers, state.JournalContextBlockerItem{Key: entry.Scope, Block: entry})
	}
	result.Layers.UnresolvedBlockers = &state.JournalContextBlockerLayer{Available: true, AvailableCount: 19, ShownCount: 10, Truncated: true, Items: blockers}
	result.Layers.BranchRecency = &state.JournalContextJournalLayer{Available: true, AvailableCount: 1171, ShownCount: 10, Truncated: true, Items: journalItems("branch", "decision")}
	result.Layers.ActiveLineage = &state.JournalContextJournalLayer{Available: true, AvailableCount: 10, ShownCount: 10, Items: journalItems("lineage", "decision")}
	tasks := []state.JournalContextTask{}
	for i := 0; i < 10; i++ {
		tasks = append(tasks, state.JournalContextTask{Ref: fmt.Sprintf("TASK-%02d", i), Status: "todo", Title: long})
	}
	result.Layers.TransitionalTasks = &state.JournalContextTaskLayer{Available: true, AvailableCount: 10, ShownCount: 10, Items: tasks}
	return result
}

func digestLayerItemCounts(digest string) map[string]int {
	counts := map[string]int{}
	layer := ""
	for _, line := range strings.Split(digest, "\n") {
		if !strings.HasPrefix(line, " ") {
			if name, _, ok := strings.Cut(line, ": "); ok {
				layer = name
			}
			continue
		}
		counts[layer]++
	}
	return counts
}

func TestJournalContextDigestFitsBudgetAndKeepsEachLayersFirstItems(t *testing.T) {
	for _, budget := range []int{journalContextDigestBudget, 4000, 2000} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			got := renderJournalContextDigest(bulkyDigestFixture(), budget)
			if len(got) > budget {
				t.Fatalf("digest is %d bytes, budget %d", len(got), budget)
			}
			if !strings.Contains(got, "left out to stay within the injection budget") {
				t.Fatalf("digest that dropped items must say so:\n%s", got)
			}
			counts := digestLayerItemCounts(got)
			for _, layer := range []string{journalContextLayerActiveLineage, journalContextLayerBlockers, journalContextLayerBranchRecency, journalContextLayerTasks} {
				if counts[layer] == 0 {
					t.Fatalf("layer %s lost its newest item while others kept theirs: %v\n%s", layer, counts, got)
				}
			}
			// Equal-sized items trim evenly: no layer keeps two more than another.
			min, max := 10, 0
			for _, layer := range []string{journalContextLayerActiveLineage, journalContextLayerBlockers, journalContextLayerBranchRecency, journalContextLayerTasks} {
				if counts[layer] < min {
					min = counts[layer]
				}
				if counts[layer] > max {
					max = counts[layer]
				}
			}
			if max-min > 1 {
				t.Fatalf("uneven trimming %v\n%s", counts, got)
			}
			// Newest first: the kept blockers are exactly the leading ones.
			for i := 0; i < counts[journalContextLayerBlockers]; i++ {
				if !strings.Contains(got, fmt.Sprintf("block-%02d block(block): item %02d", i, i)) {
					t.Fatalf("blocker %d missing; kept items must be the newest:\n%s", i, got)
				}
			}
			if strings.Contains(got, "block-09 ") {
				t.Fatalf("oldest blocker survived a budget that trimmed the layer:\n%s", got)
			}
			if !strings.Contains(got, "branch-recency: showing ") || !strings.Contains(got, "(more: loaf journal context --layer branch-recency --limit 100 --branch 'main')") {
				t.Fatalf("trimmed layer must name the command that lists it:\n%s", got)
			}
		})
	}
}

func TestJournalContextDigestIsByteIdenticalForTheSameInput(t *testing.T) {
	first := renderJournalContextDigest(bulkyDigestFixture(), 3000)
	for i := 0; i < 5; i++ {
		if again := renderJournalContextDigest(bulkyDigestFixture(), 3000); again != first {
			t.Fatalf("render %d differs from the first render", i)
		}
	}
}

func TestJournalContextDigestShortensEntriesToOneValidUTF8Line(t *testing.T) {
	result := digestFixture()
	message := strings.Repeat("ação 🍞 ", 100) + "\n\nsecond paragraph"
	result.Layers.BranchRecency.Items = []state.JournalEntryRecord{digestEntry("e-long", "note", "", message, "2026-10-21T12:00:00Z", "main")}
	result.Layers.BranchRecency.AvailableCount, result.Layers.BranchRecency.ShownCount = 1, 1
	got := renderJournalContextDigest(result, journalContextDigestBudget)
	var line string
	for _, candidate := range strings.Split(got, "\n") {
		if strings.Contains(candidate, "e-long") {
			line = candidate
		}
	}
	if line == "" {
		t.Fatalf("long entry missing:\n%s", got)
	}
	if !utf8.ValidString(line) || !strings.HasSuffix(line, "…") || strings.Contains(line, "second paragraph") {
		t.Fatalf("entry line = %q, want one shortened valid UTF-8 line", line)
	}
	if runes := utf8.RuneCountInString(strings.SplitN(line, "note: ", 2)[1]); runes > journalContextDigestEntryRunes {
		t.Fatalf("message has %d runes, cap %d", runes, journalContextDigestEntryRunes)
	}
}

func TestJournalContextDigestHardCapsEvenItsFixedLines(t *testing.T) {
	result := digestFixture()
	result.Diagnostics = []state.JournalContextDiagnostic{{Code: "big", Message: strings.Repeat("é", 400)}}
	for _, budget := range []int{40, 120, 300} {
		got := renderJournalContextDigest(result, budget)
		if len(got) > budget || !utf8.ValidString(got) {
			t.Fatalf("budget %d: got %d bytes, valid UTF-8 %t", budget, len(got), utf8.ValidString(got))
		}
		if !strings.Contains(got, "[digest cut to fit its budget]") {
			t.Fatalf("budget %d: a cut digest must say so:\n%s", budget, got)
		}
	}
	// With room for the tail, the omission notice survives the cut too.
	got := renderJournalContextDigest(result, 300)
	if !strings.Contains(got, "left out to stay within the injection budget") {
		t.Fatalf("cut digest lost its omission notice:\n%s", got)
	}
}

// injectionLimitPayloadBytes is the largest UTF-8 payload a recorded limit
// admits. A character never takes fewer bytes than it counts, and Codex counts
// one token per four bytes. A target that documents no limit is held to the
// tightest documented one.
func injectionLimitPayloadBytes(t *testing.T, contract TargetCapabilityEvidenceContract, limit TargetInjectionLimitRecord) int {
	t.Helper()
	toBytes := func(limit TargetInjectionLimitRecord) int {
		switch limit.Unit {
		case "characters", "bytes":
			return limit.Limit
		case "approximate-tokens":
			return limit.Limit * 4
		}
		t.Fatalf("unknown injection limit unit %q", limit.Unit)
		return 0
	}
	if limit.Status == "documented" {
		return toBytes(limit)
	}
	tightest := 0
	for _, record := range contract.Records {
		if documented := record.Context.InjectionLimit; documented.Status == "documented" {
			if size := toBytes(documented); tightest == 0 || size < tightest {
				tightest = size
			}
		}
	}
	if tightest == 0 {
		t.Fatal("no documented injection limit to fall back on")
	}
	return tightest
}

func TestJournalContextHookPayloadsFitRecordedInjectionLimits(t *testing.T) {
	workingDir, stateHome := setupJournalHookRunner(t)
	// A real branch makes branch-recency available, the layer that grows fastest.
	runChangeContextGit(t, workingDir, "init", "-q", "-b", "main")
	runChangeContextGit(t, workingDir, "-c", "user.name=Loaf Test", "-c", "user.email=test@example.com", "commit", "-q", "--allow-empty", "-m", "init")
	// Messages are shortened but scopes are not, so long scopes push even
	// one-line entries past the budget.
	long := strings.Repeat("evidence that would overflow the harness ", 40)
	scope := strings.Repeat("long-scope-", 30)
	entries := []string{"wrap(project): " + long}
	for i := 0; i < 25; i++ {
		entries = append(entries, fmt.Sprintf("block(%s%02d): %s", scope, i, long), fmt.Sprintf("decision(%s%02d): %s", scope, i, long))
	}
	for _, entry := range entries {
		if err := (Runner{Stdout: &bytes.Buffer{}, WorkingDir: workingDir, StateHome: stateHome}).Run([]string{"journal", "log", entry}); err != nil {
			t.Fatal(err)
		}
	}
	run := func(input string, args ...string) string {
		t.Helper()
		var out bytes.Buffer
		if err := (Runner{Stdout: &out, WorkingDir: workingDir, StateHome: stateHome, Stdin: strings.NewReader(input)}).Run(append([]string{"journal", "context"}, args...)); err != nil {
			t.Fatalf("journal context %v error = %v", args, err)
		}
		return out.String()
	}
	hookContext := func(output string) string {
		t.Helper()
		var payload struct {
			HookSpecificOutput *struct {
				AdditionalContext string `json:"additionalContext"`
			} `json:"hookSpecificOutput"`
			AdditionalContext string `json:"additional_context"`
		}
		if err := json.Unmarshal([]byte(output), &payload); err != nil {
			t.Fatalf("hook output = %q: %v", output, err)
		}
		if payload.HookSpecificOutput != nil {
			return payload.HookSpecificOutput.AdditionalContext
		}
		return payload.AdditionalContext
	}
	// Each target's injected strings: session start, plus post-compaction where
	// the target delivers it through a separate command.
	payloads := map[string][]string{
		"claude-code": {hookContext(run(`{"hook_event_name":"SessionStart","source":"compact","session_id":"s1"}`, "--from-hook", "--claude-code"))},
		"codex":       {hookContext(run(codexSessionStartInput("compact"), "--from-hook", "--codex-hook"))},
		"cursor":      {hookContext(run(`{"hook_event_name":"sessionStart","composer_mode":"agent","is_background_agent":false,"cursor_version":"2026.05.09-0afadcc"}`, "--from-hook", "--cursor-hook"))},
		"opencode": {
			run(openCodeSessionStartPayload, "--from-hook", "--opencode-hook"),
			run(`{"target":"opencode","session_id":"ses_1","lifecycle_event":"session.compacting"}`, "for-resumption"),
		},
	}
	contract := loadTestTargetCapabilityEvidence(t)
	skipped := map[string]bool{}
	for _, record := range contract.Records {
		injected, ok := payloads[record.Target]
		if !ok {
			skipped[record.Target] = true
			continue
		}
		limit := injectionLimitPayloadBytes(t, contract, record.Context.InjectionLimit)
		for _, payload := range injected {
			if len(payload) > limit {
				t.Fatalf("%s/%s payload is %d bytes, recorded limit admits %d", record.Target, record.Surface, len(payload), limit)
			}
			if !strings.Contains(payload, "left out to stay within the injection budget") {
				t.Fatalf("%s/%s payload did not exercise the budget:\n%s", record.Target, record.Surface, payload)
			}
		}
	}
	// Only Amp, which delivers no automatic context, may go unchecked.
	if len(skipped) != 1 || !skipped["amp"] {
		t.Fatalf("unchecked targets = %v, want only amp", skipped)
	}
}

// The canonical layer order is the priority order: at a budget that fits only
// the first items of the leading layers, later layers keep just their counts
// and list commands.
func TestJournalContextDigestPriorityFollowsCanonicalLayerOrder(t *testing.T) {
	result := journalContextCLIResult{ProjectName: "loaf", Branch: "main"}
	result.Layers.ProjectSynthesis = &state.JournalContextJournalLayer{Available: true, AvailableCount: 1, ShownCount: 1, Items: []state.JournalEntryRecord{
		digestEntry("s1", "wrap", "project", "latest synthesis", "2026-10-21T12:00:00Z", "main"),
	}}
	blockers := []state.JournalContextBlockerItem{}
	recent := []state.JournalEntryRecord{}
	tasks := []state.JournalContextTask{}
	for i := 1; i <= 3; i++ {
		blockers = append(blockers, state.JournalContextBlockerItem{Key: "ship", Block: digestEntry(fmt.Sprintf("b%d", i), "block", "ship", fmt.Sprintf("blocker %d", i), "2026-10-21T12:00:00Z", "main")})
		recent = append(recent, digestEntry(fmt.Sprintf("r%d", i), "decision", "work", fmt.Sprintf("recent %d", i), "2026-10-21T12:00:00Z", "main"))
		tasks = append(tasks, state.JournalContextTask{Ref: fmt.Sprintf("T-%d", i), Status: "todo", Title: fmt.Sprintf("task %d", i)})
	}
	result.Layers.UnresolvedBlockers = &state.JournalContextBlockerLayer{Available: true, AvailableCount: 3, ShownCount: 3, Items: blockers}
	result.Layers.BranchRecency = &state.JournalContextJournalLayer{Available: true, AvailableCount: 3, ShownCount: 3, Items: recent}
	result.Layers.TransitionalTasks = &state.JournalContextTaskLayer{Available: true, AvailableCount: 3, ShownCount: 3, Items: tasks}
	want := strings.Join([]string{
		"loaf journal context: project loaf, branch main",
		"Entries are shortened to one line; read one in full with: loaf journal show <id>",
		"project-synthesis: showing 1 of 1",
		"  [2026-10-21 12:00] s1 wrap(project): latest synthesis",
		"unresolved-blockers: showing 1 of 3 (more: loaf journal context --layer unresolved-blockers --limit 3; resolve one: loaf journal log \"unblock(<id>): how it was resolved\")",
		"  [2026-10-21 12:00] b1 block(ship): blocker 1",
		"branch-recency: showing 0 of 3 (more: loaf journal context --layer branch-recency --limit 3 --branch 'main')",
		"transitional-tasks: showing 0 of 3 (more: loaf journal context --layer transitional-tasks --limit 3)",
		"8 entries left out to stay within the injection budget; each layer's more: command lists them",
	}, "\n")
	if got := renderJournalContextDigest(result, len(want)); got != want {
		t.Fatalf("digest mismatch at a tight budget\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// Exploration checkpoints page at five with no cursor, so the list command
// must name the layer's total rather than rely on the default page size.
func TestJournalContextDigestListCommandNamesTheLayerTotal(t *testing.T) {
	result := digestFixture()
	items := []state.JournalContextExplorationCheckpointItem{}
	for i := 1; i <= 5; i++ {
		items = append(items, state.JournalContextExplorationCheckpointItem{ExplorationID: fmt.Sprintf("expl-%d", i), Title: "Digest shape", NextAction: "compare budgets"})
	}
	result.Layers.ExplorationCheckpoints = &state.JournalContextCheckpointsLayer{Available: true, AvailableCount: 8, ShownCount: 5, Truncated: true, Items: items}
	got := renderJournalContextDigest(result, journalContextDigestBudget)
	want := "exploration-checkpoints: showing 5 of 8 (more: loaf journal context --layer exploration-checkpoints --limit 8; resume one: loaf exploration context <id>)"
	if !strings.Contains(got, want+"\n") {
		t.Fatalf("digest missing %q:\n%s", want, got)
	}
}

func TestJournalContextDigestGroupsRepeatedDiagnostics(t *testing.T) {
	result := digestFixture()
	result.Diagnostics = []state.JournalContextDiagnostic{}
	for i := 0; i < 200; i++ {
		result.Diagnostics = append(result.Diagnostics, state.JournalContextDiagnostic{Code: "journal-context-unmatched-unblock", Key: fmt.Sprintf("scope-%03d", i), Message: "unblock has no unresolved block for the exact scope key"})
	}
	result.Diagnostics = append(result.Diagnostics, state.JournalContextDiagnostic{Code: "change-source-unavailable", Message: "Change source unavailable"})
	got := renderJournalContextDigest(result, journalContextDigestBudget)
	lines := []string{}
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "diagnostic ") {
			lines = append(lines, line)
		}
	}
	want := []string{
		"diagnostic journal-context-unmatched-unblock: unblock has no unresolved block for the exact scope key (199 more with this code)",
		"diagnostic change-source-unavailable: Change source unavailable",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("diagnostic lines = %q, want %q", lines, want)
	}
	if !strings.Contains(got, "e-branch-2 decision(digest)") {
		t.Fatalf("repeated diagnostics crowded out entries:\n%s", got)
	}
}

func TestJournalContextDigestFoldsLabelsAndNamesLegacySparks(t *testing.T) {
	result := digestFixture()
	result.Layers.DeferredIntent = &state.JournalContextDeferralLayer{Available: true, AvailableCount: 1, ShownCount: 1, Items: []state.JournalContextDeferralItem{
		{OperationKey: "op-1", Spark: state.JournalContextDeferredSpark{ID: "spark-1", Text: "legacy\ndeferral"}},
	}}
	result.Layers.ExplorationCheckpoints.Items[0].Title = "Digest\n  shape"
	result.Layers.BranchRecency.Items[0].Scope = "multi\nline scope"
	got := renderJournalContextDigest(result, journalContextDigestBudget)
	for _, want := range []string{
		"deferred-intent: showing 1 of 1 (read a spark: loaf spark show <id>)\n  op-1 (spark spark-1): legacy deferral\n",
		"  EXPL-1: Digest shape / next: compare budgets\n",
		"e-branch-2 decision(multi line scope): one line per entry\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("digest missing %q:\n%s", want, got)
		}
	}
}
