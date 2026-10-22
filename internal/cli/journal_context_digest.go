package cli

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/levifig/loaf/internal/state"
)

// journalContextDigestBudget bounds the default continuity digest in UTF-8
// bytes. Harnesses inject hook context only up to a limit and replace anything
// larger with a short preview (Claude Code: 10,000 characters; Codex: about
// 2,500 tokens, counted as bytes ÷ 4). A byte count bounds both, and the
// headroom covers the drift nudge some adapters append. Each target's recorded
// limit lives in config/target-capabilities.json, and a test holds every
// adapter's payload under it.
const journalContextDigestBudget = 8000

const (
	journalContextDigestEntryRunes      = 160
	journalContextDigestSynthesisRunes  = 500
	journalContextDigestDiagnosticRunes = 300
	journalContextDigestListLimitMax    = 100
	journalContextDigestCutMarker       = "\n[digest cut to fit its budget]"
)

// journalContextDigestLayer is one layer prepared for packing: every item is
// already a single line, newest first, and shown counts how many survive.
type journalContextDigestLayer struct {
	name        string
	unavailable bool
	total       int
	hint        string
	listCommand string
	items       []string
	shown       int
	full        bool
}

// renderJournalContextDigest renders the default continuity digest every
// harness and the plain CLI receive. Layers keep their canonical order, empty
// layers are left out, and every item is one shortened line. When the budget is
// tight, items are admitted one round at a time across layers in that order, so
// each layer gives up its oldest items before any layer loses its newest. The
// output depends only on its input, so the same facts and budget produce the
// same bytes.
func renderJournalContextDigest(result journalContextCLIResult, budget int) string {
	header := []string{
		journalContextDigestTitle(result),
		"Entries are shortened to one line; read one in full with: loaf journal show <id>",
	}
	layers := journalContextDigestLayers(result)
	diagnostics := []string{}
	for _, diagnostic := range result.Diagnostics {
		diagnostics = append(diagnostics, "diagnostic "+diagnostic.Code+": "+shortenJournalContextText(diagnostic.Message, journalContextDigestDiagnosticRunes))
	}
	render := func() string {
		return assembleJournalContextDigest(header, layers, diagnostics, budget)
	}
	for admitted := true; admitted; {
		admitted = false
		for _, layer := range layers {
			if layer.full || layer.shown == len(layer.items) {
				continue
			}
			layer.shown++
			if len(render()) <= budget {
				admitted = true
				continue
			}
			layer.shown--
			layer.full = true
		}
	}
	return capJournalContextDigest(render(), budget)
}

func journalContextDigestTitle(result journalContextCLIResult) string {
	project := result.ProjectName
	if project == "" {
		project = result.ProjectID
	}
	branch := "branch " + result.Branch
	if result.Branch == "" {
		branch = "branch detached or unavailable"
	}
	return "loaf journal context: project " + project + ", " + branch
}

func assembleJournalContextDigest(header []string, layers []*journalContextDigestLayer, diagnostics []string, budget int) string {
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
		if layer.hint != "" {
			notes = append(notes, layer.hint)
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
	if omitted > 0 {
		noun := "entries"
		if omitted == 1 {
			noun = "entry"
		}
		lines = append(lines, fmt.Sprintf("%d %s left out to fit the %d-byte budget; each layer's more: command lists them", omitted, noun, budget))
	}
	return strings.Join(lines, "\n")
}

// capJournalContextDigest is the last guarantee: if the fixed lines alone
// overflow the budget, cut at a rune boundary and say so.
func capJournalContextDigest(digest string, budget int) string {
	if len(digest) <= budget {
		return digest
	}
	marker := journalContextDigestCutMarker
	if len(marker) > budget {
		marker = ""
	}
	cut := budget - len(marker)
	for cut > 0 && !utf8.RuneStart(digest[cut]) {
		cut--
	}
	return digest[:cut] + marker
}

func journalContextDigestLayers(result journalContextCLIResult) []*journalContextDigestLayer {
	branch := result.Branch
	layers := []*journalContextDigestLayer{}
	add := func(name string, available bool, total int, items []string, hint string) {
		if available && total == 0 {
			return
		}
		layers = append(layers, &journalContextDigestLayer{
			name:        name,
			unavailable: !available,
			total:       total,
			hint:        hint,
			listCommand: journalContextDigestListCommand(name, total, branch),
			items:       items,
		})
	}
	if layer := result.Layers.ProjectSynthesis; layer != nil {
		items := []string{}
		for _, entry := range layer.Items {
			items = append(items, formatJournalContextDigestEntry(entry, branch, journalContextDigestSynthesisRunes))
		}
		add(journalContextLayerProjectSynthesis, layer.Available, layer.AvailableCount, items, "")
	}
	if layer := result.Layers.ScopedCheckpoint; layer != nil {
		items := []string{}
		for _, item := range layer.Items {
			items = append(items, item.Label+": "+formatJournalContextDigestEntry(item.Entry, branch, journalContextDigestSynthesisRunes))
		}
		add(journalContextLayerScopedCheckpoint, layer.Available, layer.AvailableCount, items, "")
	}
	if layer := result.Layers.ActiveLineage; layer != nil {
		add(journalContextLayerActiveLineage, layer.Available, layer.AvailableCount, formatJournalContextDigestEntries(layer.Items, branch), "")
	}
	if layer := result.Layers.UnresolvedBlockers; layer != nil {
		items := []string{}
		for _, item := range layer.Items {
			items = append(items, formatJournalContextDigestEntry(item.Block, branch, journalContextDigestEntryRunes))
		}
		add(journalContextLayerBlockers, layer.Available, layer.AvailableCount, items, "")
	}
	if layer := result.Layers.DeferredIntent; layer != nil {
		items := []string{}
		hint := ""
		for _, item := range layer.Items {
			if item.Intent == nil {
				items = append(items, item.OperationKey+": "+shortenJournalContextText(item.Spark.Text, journalContextDigestEntryRunes))
				continue
			}
			hint = "read one: loaf intent show <id>"
			items = append(items, fmt.Sprintf("%s (%s): %s", firstNonEmpty(item.Intent.Alias, item.Intent.ID), item.Intent.Disposition, shortenJournalContextText(item.Intent.Body, journalContextDigestEntryRunes)))
		}
		add(journalContextLayerDeferredIntent, layer.Available, layer.AvailableCount, items, hint)
	}
	if layer := result.Layers.ExplorationCheckpoints; layer != nil {
		items := []string{}
		for _, item := range layer.Items {
			items = append(items, fmt.Sprintf("%s: %s / next: %s", firstNonEmpty(item.Alias, item.ExplorationID), item.Title, shortenJournalContextText(item.NextAction, journalContextDigestEntryRunes)))
		}
		add(journalContextLayerExplorationCheckpoints, layer.Available, layer.AvailableCount, items, "resume one: loaf exploration context <id>")
	}
	if layer := result.Layers.ActiveChanges; layer != nil {
		items := []string{}
		for _, item := range layer.Items {
			items = append(items, formatActiveChangeDigestItem(item))
		}
		add(journalContextLayerActiveChanges, layer.Available, layer.AvailableCount, items, "")
	}
	if layer := result.Layers.BranchRecency; layer != nil {
		add(journalContextLayerBranchRecency, layer.Available, layer.AvailableCount, formatJournalContextDigestEntries(layer.Items, branch), "")
	}
	if layer := result.Layers.TransitionalTasks; layer != nil {
		items := []string{}
		for _, task := range layer.Items {
			items = append(items, fmt.Sprintf("%s (%s): %s", task.Ref, task.Status, shortenJournalContextText(task.Title, journalContextDigestEntryRunes)))
		}
		add(journalContextLayerTasks, layer.Available, layer.AvailableCount, items, "")
	}
	return layers
}

// journalContextDigestListCommand names the cursor-free read that lists the
// whole layer. Only branch-recency is branch-scoped, so only it names a branch.
func journalContextDigestListCommand(layer string, total int, branch string) string {
	command := "loaf journal context --layer " + layer
	if total > defaultCLIJournalContextLimit {
		limit := total
		if limit > journalContextDigestListLimitMax {
			limit = journalContextDigestListLimitMax
		}
		command += " --limit " + strconv.Itoa(limit)
	}
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

// formatJournalContextDigestEntry renders one entry as a single line: date, id,
// label, and the shortened message. An entry recorded on another branch names
// it, since the digest header already names the current one.
func formatJournalContextDigestEntry(entry state.JournalEntryRecord, branch string, maxRunes int) string {
	label := entry.EntryType
	if entry.Scope != "" {
		label = entry.EntryType + "(" + entry.Scope + ")"
	}
	line := fmt.Sprintf("[%s] %s %s: %s", formatJournalTimestamp(entry.CreatedAt), entry.ID, label, shortenJournalContextText(entry.Message, maxRunes))
	if entry.ObservedBranch != "" && entry.ObservedBranch != branch {
		line += " (branch: " + entry.ObservedBranch + ")"
	}
	return line
}

func formatActiveChangeDigestItem(item activeChangeItem) string {
	line := fmt.Sprintf("%s (%s): worktree %s; active because %s", firstNonEmpty(item.Slug, "(unparseable slug)"), item.ChangePath, item.WorktreeState, strings.Join(item.ActiveReasons, ", "))
	if len(item.Findings) > 0 {
		line += fmt.Sprintf("; %d findings", len(item.Findings))
	}
	if len(item.Gaps) > 0 {
		line += fmt.Sprintf("; %d gaps", len(item.Gaps))
	}
	return line
}

// shortenJournalContextText folds text onto one line and cuts it to at most
// maxRunes runes, ending a cut with an ellipsis.
func shortenJournalContextText(text string, maxRunes int) string {
	folded := strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(folded) <= maxRunes {
		return folded
	}
	runes := []rune(folded)
	return strings.TrimRight(string(runes[:maxRunes-1]), " ") + "…"
}
