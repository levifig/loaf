---
id: TASK-100
title: 'Track 1: CLI mechanical sync + hook swap'
status: todo
priority: P1
created: '2026-04-10T17:27:21.202Z'
updated: '2026-04-10T17:27:21.202Z'
spec: SPEC-029
---

# TASK-100: Track 1: CLI mechanical sync + hook swap

## Description

Build the JSONL parser, keyed cursor, and `loaf session sync` command. Implement Claude Code adapter for conversation log discovery and event extraction. Remove 4 mechanical PostToolUse hooks (journal-post-commit, journal-post-pr, journal-post-merge/journal-gh-events, detect-linear-magic). Add Stop command hook calling `loaf session sync`. Implement dedup logic. Update smoke tests and build target whitelists.

**File hints:**
- NEW: `cli/lib/journal/sync.ts` — harness-agnostic sync engine (cursor tracking, dedup, entry writing)
- NEW: `cli/lib/journal/adapters/claude-code.ts` — JSONL parser, keyed cursor seek, event extraction
- MODIFY: `cli/commands/session.ts` — add `sync` subcommand with `--final` flag
- MODIFY: `config/hooks.yaml` — remove 4 hooks, add Stop command hook
- MODIFY: `cli/scripts/smoke-test.js` — update assertions for removed hooks
- MODIFY: test files for hook artifacts and runtime logic

## Acceptance Criteria

- [ ] `loaf session sync` parses JSONL and writes commit/PR/merge entries to journal
- [ ] Keyed cursor tracks position in session frontmatter (`log_cursor: {session_id, offset}`)
- [ ] Cursor resets on session_id mismatch (handles `/clear`)
- [ ] Dedup prevents duplicate entries from CLI + skill self-logging
- [ ] 4 mechanical PostToolUse hooks removed from hooks.yaml
- [ ] Stop command hook calls `loaf session sync` on every Stop event
- [ ] Graceful no-op when JSONL is missing (warning to stderr, no crash)
- [ ] Malformed JSONL lines skipped with warning
- [ ] `--final` flag processes remaining entries without advancing cursor
- [ ] Smoke tests pass with updated assertions
- [ ] `npm run typecheck` and `npm run test` pass
- [ ] `loaf build` succeeds for all targets

## Verification

```bash
npm run typecheck && npm run test && loaf build
```
