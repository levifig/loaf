package state

import (
	"regexp"
	"time"
)

// JournalContextBlockerStaleDays is the age, in whole UTC calendar days, past
// which an unresolved block stops being listed one by one in the default
// digest. Thirty days matches the project knowledge staleness default; a block
// that old has usually been resolved without anyone logging the unblock.
const JournalContextBlockerStaleDays = 30

// journalEntryIDPattern matches the shapes journal entry ids take: minted
// UUIDv7 ids and the prefixed ids of imported or migrated rows. An unblock key
// in one of these shapes names a block by id rather than by scope.
var journalEntryIDPattern = regexp.MustCompile(`^(?:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|journal[a-z-]*:[0-9a-f]{24})$`)

func looksLikeJournalEntryID(key string) bool {
	return journalEntryIDPattern.MatchString(key)
}

// journalOpenBlock is a block entry that no later unblock has resolved.
type journalOpenBlock struct {
	Candidate         journalContextJournalCandidate
	PreviousUnblockID string
}

// journalBlockResolution is the outcome of replaying every block and unblock
// entry in journal order. It is derived on read and never stored.
type journalBlockResolution struct {
	// Open lists unresolved blocks oldest first.
	Open []journalOpenBlock
	// ResolvedBy maps a resolved block id to the unblock that resolved it.
	ResolvedBy map[string]string
	// BlockScope maps every block id to its scope.
	BlockScope map[string]string
	// Latest maps a scope to the newest block or unblock entry that affected it,
	// including unblocks that named one of its blocks by id.
	Latest map[string]journalContextJournalCandidate
	// Unmatched lists unblocks that resolved nothing, oldest first.
	Unmatched []journalContextJournalCandidate
}

// resolveJournalBlocks replays block and unblock entries, given oldest first.
// An unblock whose key is the id of an earlier block resolves exactly that
// block. Any other unblock resolves every open block with exactly its scope.
func resolveJournalBlocks(events []journalContextJournalCandidate) journalBlockResolution {
	resolution := journalBlockResolution{
		ResolvedBy: map[string]string{},
		BlockScope: map[string]string{},
		Latest:     map[string]journalContextJournalCandidate{},
	}
	open := map[string]journalOpenBlock{}
	order := []string{}
	lastResolution := map[string]string{}
	resolve := func(blockID, unblockID string) {
		delete(open, blockID)
		resolution.ResolvedBy[blockID] = unblockID
		lastResolution[resolution.BlockScope[blockID]] = unblockID
	}
	for _, event := range events {
		entry := event.Entry
		switch entry.EntryType {
		case "block":
			resolution.BlockScope[entry.ID] = entry.Scope
			resolution.Latest[entry.Scope] = event
			open[entry.ID] = journalOpenBlock{Candidate: event, PreviousUnblockID: lastResolution[entry.Scope]}
			order = append(order, entry.ID)
		case "unblock":
			if scope, byID := resolution.BlockScope[entry.Scope]; byID {
				resolution.Latest[scope] = event
				if _, isOpen := open[entry.Scope]; isOpen {
					resolve(entry.Scope, entry.ID)
				} else {
					resolution.Unmatched = append(resolution.Unmatched, event)
				}
				continue
			}
			resolution.Latest[entry.Scope] = event
			matched := false
			for _, blockID := range order {
				if block, isOpen := open[blockID]; isOpen && block.Candidate.Entry.Scope == entry.Scope {
					resolve(blockID, entry.ID)
					matched = true
				}
			}
			if !matched {
				resolution.Unmatched = append(resolution.Unmatched, event)
			}
		}
	}
	for _, blockID := range order {
		if block, isOpen := open[blockID]; isOpen {
			resolution.Open = append(resolution.Open, block)
		}
	}
	return resolution
}

// journalContextAgeDays is the number of whole UTC calendar days between an
// entry's creation and now. It reports false for an unparseable timestamp.
func journalContextAgeDays(createdAt string, now time.Time) (int, bool) {
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return 0, false
	}
	return int(utcDate(now).Sub(utcDate(created)).Hours() / 24), true
}

func utcDate(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
