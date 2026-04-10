---
id: TASK-100
spec: SPEC-029
title: loaf session enrich CLI command + librarian profile update
priority: P1
status: pending
---

# TASK-100: `loaf session enrich` CLI command + librarian profile update

## Objective

Add the `loaf session enrich` subcommand and update the librarian agent profile to support enrichment. The CLI handles JSONL discovery and agent invocation; the librarian does the actual reading and journal writing.

## Acceptance Criteria

### CLI command
- [ ] `loaf session enrich` — enriches the active session
- [ ] `loaf session enrich <file>` — enriches a specific session file
- [ ] `--dry-run` flag shows what would be added without writing
- [ ] `--model <model>` flag overrides the model for the librarian call
- [ ] Discovers JSONL path from `claude_session_id` + project directory derivation
- [ ] Falls back to scanning project directory if direct path construction fails
- [ ] Validates JSONL exists before spawning agent
- [ ] Validates `claude` binary is available
- [ ] Validates `claude_session_id` exists in session frontmatter
- [ ] Spawns `claude --agent librarian -p --no-session-persistence --max-turns 10`
- [ ] Passes enrichment prompt with: session path, JSONL path, enriched_at timestamp
- [ ] For `--dry-run`: modifies prompt to ask for output-only, captures and prints agent text
- [ ] Exit 0 on success, exit 1 on error
- [ ] Errors to stderr with actionable messages
- [ ] `enriched_at` added to `SessionFrontmatter` interface
- [ ] Registered as subcommand in session command group

### Librarian profile update
- [ ] Update constraint: "Scope **write** operations to `.agents/` paths. Read JSONL conversation logs from `${CLAUDE_CONFIG_DIR}/projects/` when performing enrichment."
- [ ] Add enrichment to "What You Tend" section
- [ ] Existing librarian behavior unchanged

### Build + tests
- [ ] `loaf build` succeeds
- [ ] `npm run typecheck` passes
- [ ] Tests for JSONL discovery logic (path derivation, fallback scan)
- [ ] Tests for error handling (missing JSONL, missing claude, missing session_id)

## Implementation Notes

- Add `enriched_at?: string` to `SessionFrontmatter` interface in `session.ts`
- JSONL project directory derivation: leading dash + path separators become dashes
  Example: `/Users/levifig/Code/loaf` → `-Users-levifig-Code-loaf`
- `claude` invocation via `child_process.spawn` or `execSync`
- Use `which claude` or `command -v claude` to check binary availability
- The enrichment prompt is constructed by the CLI, not stored as a template file
- For `--dry-run`: append "Output entries only, DO NOT edit files" + use `--tools "Read,Glob,Grep"` (strip Edit)
- Existing helpers: `findActiveSessionForBranch()`, `readSessionFile()`, `getCurrentBranch()`

## Dependencies

None — this is the foundation.
