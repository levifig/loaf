package cli

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/levifig/loaf/internal/state"
)

// journalContextDigestBudget bounds the default continuity digest in UTF-8
// bytes, including any drift nudge an adapter appends. Harnesses inject hook
// context only up to a limit and replace anything larger with a short preview
// (Claude Code: 10,000 characters; Codex: about 2,500 tokens, counted as
// bytes ÷ 4). A byte count bounds both. Each target's recorded limit lives in
// config/target-capabilities.json, and a test holds every adapter's payload
// under it.
const journalContextDigestBudget = 8000

const (
	journalContextDigestEntryRunes      = 160
	journalContextDigestSynthesisRunes  = 500
	journalContextDigestDiagnosticRunes = 300
	journalContextDigestListLimitMax    = 100
	journalContextDigestCutMarker       = "\n[digest cut to fit its budget]"
)

// journalContextDigestLayer is one layer prepared for packing: every item is
// already a single line in the layer's own order, and shown counts how many
// of them survive.
type journalContextDigestLayer struct {
	name        string
	unavailable bool
	total       int
	hints       []string
	// listedHints appear only once packing lists at least one item.
	listedHints []string
	listCommand string
	items       []string
	shown       int
	full        bool
}

// renderJournalContextDigest renders the default continuity digest every
// harness and the plain CLI receive. Empty layers are left out and every item
// is one shortened line.
//
// The canonical layer order is also the priority order: the latest deliberate
// synthesis first (a stale one moves to the end), then live obligations (lineage, blockers, deferred intent,
// explorations, active changes), then branch recency and transitional tasks,
// which the layer commands and `loaf journal recent` recover cheaply. When the
// budget is tight, items are admitted one round at a time across layers in
// that order, so every layer keeps its first items before any layer gets more.
// The output depends only on its input, so the same facts and budget produce
// the same bytes.
func renderJournalContextDigest(result journalContextCLIResult, budget int) string {
	header := []string{
		journalContextDigestTitle(result),
		"Entries are shortened to one line; read one in full with: loaf journal show <id>",
	}
	layers := journalContextDigestLayers(result)
	diagnostics := groupJournalContextDiagnostics(result.Diagnostics)
	for admitted := true; admitted; {
		admitted = false
		for _, layer := range layers {
			if layer.full || layer.shown == len(layer.items) {
				continue
			}
			layer.shown++
			body, notice := assembleJournalContextDigest(header, layers, diagnostics)
			if len(joinJournalContextDigest(body, notice)) <= budget {
				admitted = true
				continue
			}
			layer.shown--
			layer.full = true
		}
	}
	body, notice := assembleJournalContextDigest(header, layers, diagnostics)
	return capJournalContextDigest(body, notice, budget)
}

func journalContextDigestTitle(result journalContextCLIResult) string {
	project := result.ProjectName
	if project == "" {
		project = result.ProjectID
	}
	branch := "branch " + foldJournalContextText(result.Branch)
	if result.Branch == "" {
		branch = "branch detached or unavailable"
	}
	return "loaf journal context: project " + project + ", " + branch
}

// assembleJournalContextDigest returns the digest body and, separately, the
// notice saying how many items the budget left out, so a hard cut can keep it.
func assembleJournalContextDigest(header []string, layers []*journalContextDigestLayer, diagnostics []string) (string, string) {
	lines := append([]string(nil), header...)
	omitted := 0
	for _, layer := range layers {
		if layer.unavailable {
			lines = append(lines, layer.name+": unavailable")
			continue
		}
		line := fmt.Sprintf("%s: showing %d of %d", layer.name, layer.shown, layer.total)
		notes := []string{}
		if layer.shown < layer.total {
			notes = append(notes, "more: "+layer.listCommand)
		}
		notes = append(notes, layer.hints...)
		if layer.shown > 0 {
			notes = append(notes, layer.listedHints...)
		}
		if len(notes) > 0 {
			line += " (" + strings.Join(notes, "; ") + ")"
		}
		lines = append(lines, line)
		for _, item := range layer.items[:layer.shown] {
			lines = append(lines, "  "+item)
		}
		omitted += len(layer.items) - layer.shown
	}
	lines = append(lines, diagnostics...)
	notice := ""
	if omitted > 0 {
		noun := "entries"
		if omitted == 1 {
			noun = "entry"
		}
		notice = fmt.Sprintf("%d %s left out to stay within the injection budget; each layer's more: command lists them", omitted, noun)
	}
	return strings.Join(lines, "\n"), notice
}

func joinJournalContextDigest(body, notice string) string {
	if notice == "" {
		return body
	}
	return body + "\n" + notice
}

// capJournalContextDigest is the last guarantee: if the fixed lines alone
// overflow the budget, cut the body at a rune boundary, say so, and keep the
// omission notice when it fits.
func capJournalContextDigest(body, notice string, budget int) string {
	full := joinJournalContextDigest(body, notice)
	if len(full) <= budget {
		return full
	}
	tail := ""
	for _, candidate := range []string{joinJournalContextDigest(journalContextDigestCutMarker, notice), journalContextDigestCutMarker} {
		if len(candidate) <= budget {
			tail = candidate
			break
		}
	}
	cut := budget - len(tail)
	if cut > len(body) {
		cut = len(body)
	}
	for cut > 0 && cut < len(body) && !utf8.RuneStart(body[cut]) {
		cut--
	}
	return body[:cut] + tail
}

// groupJournalContextDiagnostics renders one line per diagnostic code, in the
// order codes first appear, so repeated diagnostics cannot crowd out entries.
func groupJournalContextDiagnostics(diagnostics []state.JournalContextDiagnostic) []string {
	codes := []string{}
	counts := map[string]int{}
	first := map[string]string{}
	for _, diagnostic := range diagnostics {
		if _, seen := counts[diagnostic.Code]; !seen {
			codes = append(codes, diagnostic.Code)
			first[diagnostic.Code] = diagnostic.Message
		}
		counts[diagnostic.Code]++
	}
	lines := make([]string, 0, len(codes))
	for _, code := range codes {
		line := "diagnostic " + foldJournalContextText(code) + ": " + shortenJournalContextText(first[code], journalContextDigestDiagnosticRunes)
		if more := counts[code] - 1; more > 0 {
			line += fmt.Sprintf(" (%d more with this code)", more)
		}
		lines = append(lines, line)
	}
	return lines
}

func journalContextDigestLayers(result journalContextCLIResult) []*journalContextDigestLayer {
	branch := result.Branch
	layers := []*journalContextDigestLayer{}
	add := func(name string, available bool, total int, items []string, hints ...string) {
		if available && total == 0 {
			return
		}
		layers = append(layers, &journalContextDigestLayer{
			name:        name,
			unavailable: !available,
			total:       total,
			hints:       hints,
			listCommand: journalContextDigestListCommand(name, total, branch),
			items:       items,
		})
	}
	// A stale project synthesis no longer leads: it keeps its label and moves
	// to the end, so under a tight budget it is the last layer to get its
	// first item.
	var staleSynthesis *journalContextDigestLayer
	if layer := result.Layers.ProjectSynthesis; layer != nil {
		items := []string{}
		for _, entry := range layer.Items {
			items = append(items, formatJournalContextDigestEntry(entry, branch, journalContextDigestSynthesisRunes))
		}
		if !layer.Stale {
			add(journalContextLayerProjectSynthesis, layer.Available, layer.AvailableCount, items)
		} else {
			add(journalContextLayerProjectSynthesis, layer.Available, layer.AvailableCount, items, journalContextStaleSynthesisHint(layer.EntriesSince))
			if last := len(layers) - 1; last >= 0 && layers[last].name == journalContextLayerProjectSynthesis {
				staleSynthesis, layers = layers[last], layers[:last]
			}
		}
	}
	if layer := result.Layers.ScopedCheckpoint; layer != nil {
		items := []string{}
		for _, item := range layer.Items {
			items = append(items, foldJournalContextText(item.Label)+": "+formatJournalContextDigestEntry(item.Entry, branch, journalContextDigestSynthesisRunes))
		}
		add(journalContextLayerScopedCheckpoint, layer.Available, layer.AvailableCount, items)
	}
	if layer := result.Layers.ActiveLineage; layer != nil {
		add(journalContextLayerActiveLineage, layer.Available, layer.AvailableCount, formatJournalContextDigestEntries(layer.Items, branch))
	}
	if layer := result.Layers.UnresolvedBlockers; layer != nil {
		// Blockers older than the age limit are counted, not listed; the
		// layer's list command still reads every one.
		items := []string{}
		for _, item := range layer.Items {
			if !item.Stale {
				items = append(items, formatJournalContextDigestEntry(item.Block, branch, journalContextDigestEntryRunes))
			}
		}
		hints := []string{}
		if layer.StaleCount > 0 {
			hints = append(hints, fmt.Sprintf("%d older than %d days not listed", layer.StaleCount, state.JournalContextBlockerStaleDays))
		}
		add(journalContextLayerBlockers, layer.Available, layer.AvailableCount, items, hints...)
		if last := len(layers) - 1; last >= 0 && layers[last].name == journalContextLayerBlockers {
			layers[last].listedHints = []string{journalContextBlockerResolveHint}
		}
	}
	if layer := result.Layers.DeferredIntent; layer != nil {
		items := []string{}
		canonical, legacy := false, false
		for _, item := range layer.Items {
			if item.Intent == nil {
				legacy = true
				items = append(items, fmt.Sprintf("%s (spark %s): %s", foldJournalContextText(item.OperationKey), firstNonEmpty(item.Spark.Alias, item.Spark.ID), shortenJournalContextText(item.Spark.Text, journalContextDigestEntryRunes)))
				continue
			}
			canonical = true
			items = append(items, fmt.Sprintf("%s (%s): %s", firstNonEmpty(item.Intent.Alias, item.Intent.ID), item.Intent.Disposition, shortenJournalContextText(item.Intent.Body, journalContextDigestEntryRunes)))
		}
		hints := []string{}
		if canonical {
			hints = append(hints, "read one: loaf intent show <id>")
		}
		if legacy {
			hints = append(hints, "read a spark: loaf spark show <id>")
		}
		add(journalContextLayerDeferredIntent, layer.Available, layer.AvailableCount, items, hints...)
	}
	if layer := result.Layers.ExplorationCheckpoints; layer != nil {
		items := []string{}
		for _, item := range layer.Items {
			items = append(items, fmt.Sprintf("%s: %s / next: %s", firstNonEmpty(item.Alias, item.ExplorationID), foldJournalContextText(item.Title), shortenJournalContextText(item.NextAction, journalContextDigestEntryRunes)))
		}
		add(journalContextLayerExplorationCheckpoints, layer.Available, layer.AvailableCount, items, "resume one: loaf exploration context <id>")
	}
	if layer := result.Layers.ActiveChanges; layer != nil {
		items := []string{}
		for _, item := range layer.Items {
			items = append(items, formatActiveChangeDigestItem(item))
		}
		add(journalContextLayerActiveChanges, layer.Available, layer.AvailableCount, items)
	}
	if layer := result.Layers.BranchRecency; layer != nil {
		add(journalContextLayerBranchRecency, layer.Available, layer.AvailableCount, formatJournalContextDigestEntries(layer.Items, branch))
	}
	if layer := result.Layers.TransitionalTasks; layer != nil {
		items := []string{}
		for _, task := range layer.Items {
			items = append(items, fmt.Sprintf("%s (%s): %s", task.Ref, task.Status, shortenJournalContextText(task.Title, journalContextDigestEntryRunes)))
		}
		add(journalContextLayerTasks, layer.Available, layer.AvailableCount, items)
	}
	if staleSynthesis != nil {
		layers = append(layers, staleSynthesis)
	}
	return layers
}

func journalContextStaleSynthesisHint(entriesSince int) string {
	return fmt.Sprintf("stale, %d entries since; write a fresh wrap(project) when you finish substantial work", entriesSince)
}

// journalContextDigestListCommand names the cursor-free read that lists the
// whole layer. Layers page at different sizes, so the limit is always explicit.
// Only branch-recency is branch-scoped, so only it names a branch.
func journalContextDigestListCommand(layer string, total int, branch string) string {
	limit := total
	if limit > journalContextDigestListLimitMax {
		limit = journalContextDigestListLimitMax
	}
	command := "loaf journal context --layer " + layer + " --limit " + strconv.Itoa(limit)
	if layer == journalContextLayerBranchRecency && branch != "" {
		command += " --branch " + journalContextShellQuote(branch)
	}
	return command
}

func formatJournalContextDigestEntries(entries []state.JournalEntryRecord, branch string) []string {
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		items = append(items, formatJournalContextDigestEntry(entry, branch, journalContextDigestEntryRunes))
	}
	return items
}

// formatJournalContextDigestEntry renders one entry as a single line: date and
// age, id, label, and the shortened message. An entry recorded on another branch names
// it, since the digest header already names the current one.
func formatJournalContextDigestEntry(entry state.JournalEntryRecord, branch string, maxRunes int) string {
	label := foldJournalContextText(entry.EntryType)
	if scope := foldJournalContextText(entry.Scope); scope != "" {
		label += "(" + scope + ")"
	}
	line := fmt.Sprintf("%s %s %s: %s", formatJournalEntryStamp(entry), entry.ID, label, shortenJournalContextText(entry.Message, maxRunes))
	if entry.ObservedBranch != "" && entry.ObservedBranch != branch {
		line += " (branch: " + foldJournalContextText(entry.ObservedBranch) + ")"
	}
	return line
}

func formatActiveChangeDigestItem(item activeChangeItem) string {
	line := fmt.Sprintf("%s (%s): worktree %s; active because %s", foldJournalContextText(firstNonEmpty(item.Slug, "(unparseable slug)")), foldJournalContextText(item.ChangePath), item.WorktreeState, strings.Join(item.ActiveReasons, ", "))
	if len(item.Findings) > 0 {
		line += fmt.Sprintf("; %d findings", len(item.Findings))
	}
	if len(item.Gaps) > 0 {
		line += fmt.Sprintf("; %d gaps", len(item.Gaps))
	}
	return line
}

// foldJournalContextText puts text on one line by collapsing every run of
// whitespace, newlines included, to a single space.
func foldJournalContextText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// shortenJournalContextText folds text onto one line and cuts it to at most
// maxRunes runes, ending a cut with an ellipsis.
func shortenJournalContextText(text string, maxRunes int) string {
	folded := foldJournalContextText(text)
	if utf8.RuneCountInString(folded) <= maxRunes {
		return folded
	}
	runes := []rune(folded)
	return strings.TrimRight(string(runes[:maxRunes-1]), " ") + "…"
}
