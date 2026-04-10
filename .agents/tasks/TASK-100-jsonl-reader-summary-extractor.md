---
id: TASK-100
spec: SPEC-029
title: JSONL extractor + enrich CLI command + hook isolation + librarian update
priority: P1
status: pending
---

# TASK-100: JSONL extractor + `loaf session enrich` + hook isolation + librarian update

## Objective

Build the full enrichment pipeline: JSONL extractor (deterministic, testable), CLI command (discovery + invocation + watermark), hook isolation (`LOAF_ENRICHMENT` env var), and librarian profile update.

## Acceptance Criteria

### JSONL extractor (`cli/lib/journal/extractor.ts`)
- [ ] Reads JSONL line-by-line, parses each as JSON
- [ ] Filters to `user` and `assistant` types only
- [ ] Skips: progress, agent_progress, hook_progress, attachment, queue-operation, all other types
- [ ] Applies `enriched_at` timestamp cutoff
- [ ] From `user` entries: extracts `message.content` text
- [ ] From `assistant` entries: extracts `text` blocks (full text) and `tool_use` blocks (name + key param)
- [ ] Skips `thinking` blocks in assistant content
- [ ] Discovers `<session_id>/subagents/agent-*.jsonl` and includes with `[Subagent: {description}]` markers
- [ ] Reads `agent-*.meta.json` for subagent description/type
- [ ] Enforces 100KB summary cap — oldest entries dropped first
- [ ] Tracks and returns latest JSONL timestamp from processed entries
- [ ] Handles malformed JSONL lines gracefully (skip + warning to stderr)
- [ ] Returns empty string when no entries match (enables no-op detection)
- [ ] Exports: `extractSummary(jsonlPath: string, projectDir: string, sessionId: string, since?: string): Promise<{summary: string, latestTimestamp: string | null}>`

### CLI command (`loaf session enrich`)
- [ ] `loaf session enrich` — enriches the active session
- [ ] `loaf session enrich <file>` — enriches a specific session file
- [ ] `--dry-run`: `--disallowedTools "Edit,Write"` + behavioral prompt, captures text output
- [ ] `--model <model>`: passed through to agent invocation
- [ ] Discovers JSONL path from `claude_session_id` + project directory derivation
- [ ] Falls back to scanning project directory if direct path construction fails
- [ ] Validates: `claude` binary available, `claude_session_id` in frontmatter, JSONL exists
- [ ] Calls extractor → checks if summary is empty (no-op, exit 0) → spawns agent
- [ ] Spawns: `LOAF_ENRICHMENT=1 claude --agent librarian -p --no-session-persistence --permission-mode acceptEdits --max-turns 10`
- [ ] Passes enrichment prompt with: session path + inline conversation summary + instructions
- [ ] On agent success: advances `enriched_at` to `latestTimestamp` from extractor (not current time)
- [ ] On agent failure: does NOT advance enriched_at, reports error to stderr
- [ ] `--agent librarian` not found → exit 1 with "Ensure Loaf is installed" message
- [ ] `enriched_at` added to `SessionFrontmatter` interface

### Hook isolation
- [ ] `LOAF_ENRICHMENT=1` check at top of `loaf session start` action → `process.exit(0)`
- [ ] `LOAF_ENRICHMENT=1` check at top of `loaf session end` action → `process.exit(0)`
- [ ] Enrichment does NOT create spurious session files

### Librarian profile update (`content/agents/librarian.md`)
- [ ] Add "Journal enrichment" to "What You Tend" section
- [ ] No read permission changes (summary is inline, librarian stays scoped to `.agents/`)
- [ ] Existing behavior unchanged

### Build + tests
- [ ] `loaf build` succeeds
- [ ] `npm run typecheck` passes
- [ ] Extractor unit tests: type filtering, timestamp cutoff, thinking skip, tool_use extraction, subagent discovery, 100KB cap, malformed lines, empty result, latest timestamp tracking
- [ ] CLI unit tests: JSONL discovery (path derivation, fallback), error handling (missing JSONL, missing claude, missing session_id)
- [ ] Hook isolation test: LOAF_ENRICHMENT=1 causes session start/end to exit early

## Dependencies

None — this is the foundation.
