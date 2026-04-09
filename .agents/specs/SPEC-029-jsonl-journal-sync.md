---
id: SPEC-029
title: "JSONL-Driven Journal Sync"
source: "spark — session 20260409-110153"
created: 2026-04-09T12:30:00Z
status: approved
---

# SPEC-029: JSONL-Driven Journal Sync

## Problem Statement

The session journal relies on the model manually calling `loaf session log` throughout a conversation. This is lossy — the model forgets, skips entries, or logs imprecisely. After context compaction, the model may have forgotten half the session's work entirely. The journal-nudge PostToolUse hook helps but is best-effort.

Meanwhile, Claude Code's JSONL conversation logs in `${CLAUDE_CONFIG_DIR}/projects/` capture **everything** — every tool call, every response, every timestamp. This is the session's black box recorder. The journal should be derived from it, not maintained by hand.

**Note on prior art:** The journal-nudge hook was previously moved FROM Stop TO PostToolUse because Stop-level semantic filtering degraded to commit-only logging. This spec returns semantic filtering to Stop but with a different mechanism: the model reviews a **bounded diff summary** (only new entries since last sync), not the entire conversation. This is the mitigation for the known degradation risk.

## Strategic Alignment

- **Vision:** Advances "Autonomous Execution" — journal becomes reliable ground-truth rather than best-effort model memory. Sessions are accurately recorded regardless of compaction or model forgetfulness.
- **Personas:** Benefits all Loaf users — removes cognitive burden of manual logging. The model focuses on work, not bookkeeping.
- **Architecture:** Adds `cli/lib/journal/` with two layers: harness-agnostic sync logic (cursor tracking, dedup, entry writing) and a harness-specific adapter (Claude Code JSONL parser). Other harness adapters plug into the same sync interface. Hooks orchestrate.

## Solution Direction

Replace the manual `loaf session log` workflow with automated journal sync driven by conversation log parsing.

### Architecture — two layers

- **Sync engine** (`cli/lib/journal/sync.ts`) — harness-agnostic. Accepts a list of extracted events (typed, see mapping table below). Handles cursor tracking, dedup against existing journal entries, and atomic writes to the session file.
- **Harness adapter** (e.g., `cli/lib/journal/adapters/claude-code.ts`) — reads the harness's conversation log, extracts events, returns them to the sync engine. Each harness gets its own adapter. This spec implements Claude Code only.

### Event type mapping

Extracted events map to journal entry types:

| Extracted event | Journal entry | Example |
|-----------------|---------------|---------|
| commit | `commit(hash): message` | `commit(abc1234): feat: add routing hook` |
| pr-create | `pr(#N): title` | `pr(#28): feat: release redesign` |
| merge | `merge(#N): title` | `merge(#28): feat: release redesign` |
| branch-change | `branch(name): description` | `branch(feat/release): switched to feature branch` |
| linear-ref | `linear(PROJ-123): description` | `linear(LOAF-42): referenced in commit` |

File edit counts and other mechanical signals are NOT written as individual entries — they inform the model filter's diff summary but don't become journal entries on their own.

### Three-layer extraction, run on every Stop

1. **CLI mechanical extraction** — The adapter reads the conversation log since the last cursor position. Extracts structured events (commits, PRs, merges, branch changes, Linear refs). The sync engine writes them as journal entries.

2. **Model semantic filtering + state update** — A single Stop prompt hook presents the CLI's extraction summary (new entries since last sync) and asks the model to: (a) add decisions, discoveries, and context that the CLI can't infer, and (b) update the session's `## Current State` section. This merges the current `session-state-update` prompt hook with the new semantic filter — one Stop prompt, not two.

3. **Skills self-log** — Skills continue to log their own invocation (e.g., `skill(shape): ...`). These are deduplicated against what the sync engine already wrote.

### Cursor tracking

Session frontmatter stores a keyed cursor that tracks both the conversation identity and the read position:

```yaml
log_cursor:
  session_id: "abc123"
  offset: 45678
```

On each Stop, the adapter checks if the current `claude_session_id` matches the cursor's `session_id`:
- **Match** → seek to offset, read new lines, advance offset
- **Mismatch** (post-`/clear`) → reset cursor to `{session_id: <new_id>, offset: 0}` and start fresh on the new log

**`/clear` gap:** `/clear` is an uncontrolled event — the agent cannot act during it, so there's no opportunity to sync the old log. Entries from the tail of the previous conversation may not be synced. Under normal operation, the Stop hook syncs after every turn, so the gap is one turn. However, if the sync command fails silently for multiple turns (timeout, JSONL locked), the gap widens — `/clear` then loses the entire unsynced range. Mitigation: sync errors must be visible (non-zero exit code logged to stderr), not swallowed.

### Conversation log discovery

**Verified assumption:** Claude Code names JSONL files as `<session_id>.jsonl` in the project directory. Empirically confirmed: session `207bd9ba-0e21-4f22-bcfb-b0a192fdd152` maps to file `207bd9ba-0e21-4f22-bcfb-b0a192fdd152.jsonl`. This is an internal implementation detail that may change.

Discovery strategy:
1. **`session_id`** from `HookInput` → construct candidate filename `<session_id>.jsonl`
2. **Filesystem scan** of `${CLAUDE_CONFIG_DIR}/projects/` for matching file as fallback
3. **`transcript_path`** from `HookInput` — declared in the interface but currently unused by Claude Code. Do not depend on it. If Claude Code starts providing it, prefer it over filesystem scan.

Do NOT hardcode the project hash path construction — it's fragile.

### PreCompact safety net

Before context compaction, a **blocking** forced sync runs to ensure nothing is lost. The existing PreCompact command hook already has `timeout: 60000` (60s), which is more than sufficient for sync. The PreCompact prompt hook is updated to tell the model to review the journal (now complete from sync) rather than manually flush entries.

### Wrap integration

The wrap skill triggers a final forced sync (`loaf session sync --final`) before generating the wrap-up report. After the sync, the model still reviews the journal for missing semantic events — the forced sync handles mechanical events, but the model may notice decisions or discoveries that neither the CLI nor previous Stop filters caught. Step 1 becomes: "(a) trigger forced sync, then (b) review journal for missing semantic events."

### Hook changes

This spec modifies 7 hooks total:

| Hook | Action |
|------|--------|
| `journal-post-commit` | **Remove** — replaced by CLI sync |
| `journal-post-pr` | **Remove** — replaced by CLI sync |
| `journal-post-merge` | **Remove** — replaced by CLI sync |
| `detect-linear-magic` | **Remove** — replaced by CLI sync |
| `journal-nudge` | **Remove** — replaced by model filter |
| `session-state-update` | **Merge** into model filter prompt (one Stop prompt, not two) |
| `session-pre-compact-nudge` | **Update** — reference forced sync instead of manual flush |

### Journaling contract migration

The current contract ("model manually calls `loaf session log` for everything") is replaced:

| Responsibility | Old model | New model |
|----------------|-----------|-----------|
| Mechanical events (commits, PRs, merges) | Model + PostToolUse hooks | CLI adapter (automatic) |
| Semantic events (decisions, discoveries) | Model memory + journal-nudge | Stop prompt hook (model reviews diff) |
| Session state | Separate Stop prompt hook | Merged into model filter Stop prompt |
| Skill invocation | Skills self-log | Skills self-log (unchanged) |
| PreCompact flush | Model reviews conversation | Blocking sync + model reviews journal |
| `loaf session log` | Primary mechanism | Supplementary — manual override for edge cases |

**`loaf session log` remains available** as a command but is no longer the primary journaling mechanism. The following must be updated:

- **CLAUDE.md**: Remove "Journal Discipline" paragraph requiring manual logging before every response
- **AGENTS.md**: Update session journal instructions to reflect automated sync
- **Skill Critical Rules**: Keep `skill(name): context` self-logging (it's useful metadata). Remove instructions to manually log other event types. See Appendix A for per-skill changes.
- **hooks.yaml**: Remove 5 hooks, merge 1, update 1 (see Hook changes table above)
- **Wrap skill**: Step 1 becomes forced sync + model review (not just forced sync)

## Scope

### In Scope
- **Journal sync engine** (`cli/lib/journal/sync.ts`) — harness-agnostic: cursor tracking, dedup, entry writing. Accepts extracted events from any adapter
- **Claude Code adapter** (`cli/lib/journal/adapters/claude-code.ts`) — JSONL parser, keyed cursor seek, event extraction
- **Conversation log discovery** — `session_id`-based filename matching with filesystem scan fallback
- CLI command: `loaf session sync [--final]` — detects active harness, calls adapter, writes journal entries
- Keyed cursor in session frontmatter (`log_cursor: {session_id, offset}`) for incremental reads across `/clear` boundaries
- Stop hook: one command hook (CLI sync) + one prompt hook (model filter + state update, merging `session-state-update`)
- Removal of 5 existing hooks, merge of 1, update of 1 (see Hook changes)
- **Journaling contract migration**: update CLAUDE.md, AGENTS.md, skill Critical Rules (see Appendix A), hooks.yaml, wrap skill, PreCompact prompt
- **Test and build artifact updates**: smoke-test.js (~6 assertions), hooks-artifacts.test.ts (~7 assertions), runtime-logic.test.ts (~7 assertions), cursor.ts and claude-code.ts hook whitelists, installer.ts legacy signatures
- Wrap skill update: forced sync + model review before wrap-up report
- PreCompact update: blocking forced sync + updated prompt
- Dedup logic: entries from CLI, model, and skills don't create duplicates

### Out of Scope
- Adapters for other harnesses (Cursor, OpenCode, Codex, Gemini, Amp) — the interface is ready, but research into their log formats is a separate effort
- Changing the journal entry format itself (still `[YYYY-MM-DD HH:MM] type(scope): description`)
- Session file structural changes beyond adding `log_cursor` to frontmatter

### Rabbit Holes
- **Parsing thinking blocks for decisions** — Tempting to scan `thinking` content blocks for decision language. Don't — thinking blocks are internal reasoning, not journal events. Let the model filter handle semantic extraction from its own context.
- **Real-time streaming** — Tempting to watch the JSONL file for changes. Unnecessary — Stop hook batches are sufficient.
- **Cross-session JSONL correlation** — Tempting to link parent/child sessions via `parentUuid`. Out of scope — each session syncs its own JSONL independently.
- **Abstracting the adapter interface prematurely** — Build the Claude Code adapter directly. Extract the interface when the second adapter arrives, not before.
- **`transcript_path` dependency** — Tempting to design around this HookInput field. It's declared but currently unused by Claude Code. Don't depend on it.

### No-Gos
- Don't depend on JSONL field ordering or optional fields — parse defensively
- Don't write to the JSONL — it's read-only, owned by Claude Code
- Don't parse assistant message content beyond tool_use blocks for mechanical extraction — natural language parsing is the model filter's job
- Don't hardcode the project hash path construction — use session_id filename matching or filesystem scan
- Don't swallow sync errors — non-zero exit must be visible so the model (and user) know entries may be missing

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| JSONL format changes between Claude Code versions | Medium | Medium | Direct parsing with defensive checks. Parser tests against sample JSONL fixtures. Graceful degradation on unknown fields. |
| JSONL naming convention changes | Low | High | Verified empirically but it's an internal detail. Filesystem scan fallback handles renamed files. |
| Stop hook adds latency to every turn | Low | Low | CLI processes only the diff (keyed cursor). Typical diff is a few KB. |
| Model filter adds token cost to every turn | Medium | Low | Diff summary is bounded by entries-since-last-sync. Trivial turns produce empty diffs — model skips silently. |
| Semantic filter degrades to commit-only logging (prior art) | Medium | Medium | Bounded diff summary (not full conversation) prevents the degradation that caused the original journal-nudge move to PostToolUse. |
| JSONL file missing or unreadable | Low | Medium | Graceful no-op. Warning to stderr. `loaf session log` remains available as manual fallback. |
| Sync command fails silently for multiple turns, then `/clear` | Low | Medium | Sync errors must be visible (stderr). Gap is bounded by turns-since-last-successful-sync, not just one turn. |
| Dedup false negatives (duplicate entries) | Low | Low | Dedup by matching type + scope + description substring. Duplicates are annoying but not harmful. |
| Journaling contract migration incomplete | Medium | High | Track 3 explicitly scopes all files (see Appendix A). Smoke tests verify old hooks are gone. |

## Open Questions

- [ ] Do other harnesses (Cursor, OpenCode) have equivalent conversation logs? (Research for future adapters, not blocking)
- [ ] How should the model filter's prompt be structured to minimize token waste on trivial turns? (Prototype in Track 2)

## Test Conditions

- [ ] After a session with 5+ commits, all commits appear in the journal without manual `loaf session log` calls
- [ ] Decisions logged by the model filter match what a human would consider journal-worthy
- [ ] Keyed cursor advances correctly — re-running sync produces no duplicate entries
- [ ] `/clear` scenario: cursor resets to new JSONL, sync continues from offset 0 on new conversation
- [ ] PreCompact triggers blocking forced sync — journal is complete before compaction
- [ ] Wrap skill's forced sync + model review captures all remaining events before generating report
- [ ] Graceful degradation: if JSONL is missing, session works normally (no crash, warning to stderr)
- [ ] Sync failure visibility: non-zero exit code produces visible warning, cursor does not advance
- [ ] The removed hooks produce no regressions — journal quality is equal or better
- [ ] Skills self-log entries are not duplicated by the CLI mechanical extraction
- [ ] All smoke tests pass after hook changes (updated assertions)
- [ ] Build artifacts for all targets generate correctly with new hook configuration
- [ ] CLAUDE.md, AGENTS.md, and skill files reflect the new journaling contract
- [ ] Cursor atomicity: crash between read and cursor write does not corrupt state (re-sync produces duplicates, not data loss — dedup handles it)
- [ ] Malformed JSONL lines are skipped with a warning, not a crash
- [ ] Large JSONL files (10MB+): sync completes within Stop hook timeout
- [ ] `--final` flag processes remaining entries and does NOT advance cursor (wrap may run multiple times)

## Priority Order

Tracks ship in this order. If scope needs cutting, drop from the end.

1. **CLI mechanical sync + hook swap** — JSONL parser, keyed cursor, conversation log discovery, `loaf session sync` command, Stop command hook, removal of 4 mechanical PostToolUse hooks (`journal-post-commit`, `journal-post-pr`, `journal-post-merge`, `detect-linear-magic`), update to corresponding smoke tests and build target whitelists. Go/no-go: commits and PRs appear in journal automatically; no duplicate entries from old+new hooks.
2. **Model semantic filter** — Single Stop prompt hook replacing both `journal-nudge` and `session-state-update`. Model reviews bounded diff for decisions/discoveries and updates Current State. Go/no-go: model adds contextual entries that CLI can't detect; Current State still updates correctly.
3. **Contract migration** — Update CLAUDE.md, AGENTS.md, all affected skill files (see Appendix A), wrap skill, PreCompact prompt, remaining installer signatures. Go/no-go: full suite passes, no journal regressions, all documentation reflects new model.

## Appendix A: Skill files requiring journal instruction updates

Skills that reference `loaf session log` for non-self-log purposes (i.e., instructions telling the model to manually log events that the sync now handles):

| Skill | Current instruction | Action |
|-------|--------------------|--------|
| orchestration | Critical Rules: "Log important decisions to the session journal" | Remove — model filter handles |
| release | Critical Rule: `loaf session log "decision(release): vX.Y.Z shipped"` | Keep as self-log (skill invocation result) |
| shape | Critical Rule: `loaf session log "decision(shape): SPEC-NNN created"` | Keep as self-log |
| wrap | Step 1: "Log each via `loaf session log` before proceeding" | Replace with forced sync + model review |
| implement | References to manual logging in session management | Remove — sync handles |
| research | Session journal logging instructions | Remove — sync handles |
| reflect | Decision logging instructions | Keep as self-log (reflection outcome) |
| brainstorm | Spark logging instructions | Keep as self-log |
| council | Decision logging instructions | Keep as self-log (council outcome) |
| architecture | ADR decision logging | Keep as self-log |
| idea | Idea capture logging | Keep as self-log |
| breakdown | Task creation logging | Keep as self-log |
| housekeeping | Cleanup logging | Keep as self-log |
| bootstrap | Session logging instructions | Remove — sync handles |
| strategy | Decision logging | Keep as self-log |

**Rule of thumb:** If the `loaf session log` call records the **skill's own output** (a decision it made, a spec it created, an idea it captured), keep it. If it instructs the model to log **ambient events** (commits, discoveries during work), remove it — the sync handles those.
