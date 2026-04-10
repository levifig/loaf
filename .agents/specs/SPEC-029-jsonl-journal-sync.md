---
id: SPEC-029
title: JSONL-Driven Journal Enrichment
source: spark — session 20260409-110153
created: 2026-04-09T12:30:00.000Z
status: approved
branch: feat/jsonl-journal-sync
session: 20260410-183845-session.md
---

# SPEC-029: JSONL-Driven Journal Enrichment

## Problem Statement

The session journal relies on the model manually calling `loaf session log` throughout a conversation. This is lossy — the model forgets, skips entries, or logs imprecisely. After context compaction, the model may have forgotten half the session's work entirely. The journal-nudge PostToolUse hook helps but is best-effort.

Meanwhile, Claude Code's JSONL conversation logs capture **everything** — every tool call, every response, every timestamp. This is the session's black box recorder.

Rather than replacing the existing journaling mechanism (which works well enough for mechanical events via PostToolUse hooks), this spec adds a **complementary enrichment step** that reviews the JSONL after the fact and fills in what the model missed — decisions, discoveries, and context that didn't get logged in the moment.

## Strategic Alignment

- **Vision:** Advances "Autonomous Execution" — journal becomes more complete and reliable without increasing the model's bookkeeping burden during work.
- **Personas:** Benefits all Loaf users — enrichment runs at lifecycle points, invisible during active work.
- **Architecture:** Leverages the existing librarian agent profile via `claude --agent librarian -p`. The CLI command handles discovery and invocation; the librarian reads the JSONL and writes journal entries using its standard Read/Edit tools.

## Solution Direction

Add a `loaf session enrich` command that delegates to the librarian agent for JSONL review and journal enrichment.

### Architecture — CLI orchestrates, librarian enriches

The enrichment splits cleanly between two concerns:

**CLI command (`loaf session enrich`)** — discovery and invocation:
1. Find the active session file (or accept a specific file argument)
2. Read `claude_session_id` and `enriched_at` from session frontmatter
3. Discover the JSONL path via project directory + session ID
4. Validate the JSONL exists and is readable
5. Spawn `claude --agent librarian -p` with the enrichment prompt
6. Verify exit code and report result

**Librarian agent** — reads JSONL, identifies gaps, writes entries:
1. Read the JSONL conversation log (the CLI passes the path)
2. Read the existing session journal
3. Identify decisions, discoveries, and context not already captured
4. Append missing entries to the session journal via Edit
5. Update `enriched_at:` in session frontmatter

### Agent invocation

```bash
claude --agent librarian -p \
  --no-session-persistence \
  --max-turns 10 \
  "Enrich the session journal at <session-path>.
   JSONL conversation log: <jsonl-path>
   Last enriched: <enriched_at or 'never'>
   
   Read the JSONL log (entries after the enriched_at timestamp).
   Read the existing journal entries.
   Identify semantic events not already captured:
   - Decisions with rationale (decision)
   - Discoveries or things learned (discover)  
   - Blockers encountered or resolved (block/unblock)
   
   Skip: commits, PRs, skill invocations (already logged by hooks).
   Skip: routine tool calls, file reads, thinking blocks.
   
   Append missing entries to the journal section.
   Update enriched_at in frontmatter to the current time."
```

Key flags:
- `--agent librarian` — uses the Loaf librarian profile (session/journal expertise)
- `-p` — non-interactive, exit when done
- `--no-session-persistence` — don't save the enrichment session to disk
- `--max-turns 10` — bounded execution (read JSONL + read journal + edit = ~3-5 turns typical)

### What stays the same

- **Mechanical events** (commits, PRs, merges) — existing PostToolUse hooks handle these. No changes.
- **Skill self-logging** — Skills continue to log their own invocation. No changes.
- **`loaf session log`** — Remains available for manual logging. No changes.
- **Journal format** — Still `[YYYY-MM-DD HH:MM] type(scope): description`. No changes.

### JSONL conversation log discovery

**Verified:** Claude Code names JSONL files as `<session_id>.jsonl` in the project config directory. The `session_id` is available from the session file's `claude_session_id` frontmatter field.

Discovery strategy:
1. Read `claude_session_id` from session frontmatter
2. Derive project directory: `${CLAUDE_CONFIG_DIR}/projects/` + path-to-dashed-name of cwd
3. Construct path: `<project-dir>/<session_id>.jsonl`
4. Fallback: scan project directory for `<session_id>.jsonl` if path construction fails

The project hash directory name is derived from the working directory path (leading dash, path separators become dashes). Example: `/Users/levifig/Code/levifig/projects/loaf` → `-Users-levifig-Code-levifig-projects-loaf`. Do NOT hardcode it — discover at runtime.

### JSONL structure (for librarian reference)

The JSONL contains several entry types:

| JSONL `type` | Relevant | What's in it |
|--------------|----------|-------------|
| `user` | Yes | `message.content` — user requests, decisions, direction |
| `assistant` | Yes | `message.content` — text blocks (reasoning), tool_use blocks (actions) |
| `attachment` | No | System/config noise — deferred tools, MCP instructions |
| `queue-operation` | No | Internal bookkeeping |

Within assistant messages, `content` is an array of blocks:
- `{type: "text", text: "..."}` — model's words, decisions, explanations
- `{type: "tool_use", name: "...", input: {...}}` — tool invocations
- `{type: "thinking", ...}` — internal reasoning (skip)

Each JSONL line has a `timestamp` field (ISO 8601) for incremental processing.

### Large JSONL handling

For long sessions, the JSONL can be tens of megabytes. The librarian uses the Read tool, which supports `offset` and `limit` parameters for sliced reading. Strategy:

1. If the file is small (<2000 lines), read it all
2. If large, the CLI can pass the file size in the prompt so the librarian reads from the tail
3. The `enriched_at:` timestamp tells the librarian which entries to focus on — it can skip to recent content

For v1, rely on the Read tool's natural pagination. Optimize if real-world sessions hit limits.

### `enriched_at:` frontmatter

Session frontmatter gains an `enriched_at` field — ISO 8601 timestamp of the last successful enrichment:

```yaml
---
branch: feat/some-feature
status: active
enriched_at: '2026-04-10T14:30:00.000Z'
---
```

This serves two purposes:
1. **Incremental processing** — Only JSONL entries after this timestamp need review
2. **Idempotency** — Re-running enrich on an already-enriched session with no new entries produces no changes

### `--dry-run` implementation

When `--dry-run` is passed, the CLI modifies the agent prompt to say: "Output the entries you would add, but do NOT edit any files." The CLI captures the agent's text output and prints it to stdout.

### Lifecycle integration

| Trigger | How | Purpose |
|---------|-----|---------|
| **Wrap skill** | Calls `loaf session enrich` before generating wrap-up | Ensures journal is complete before summary |
| **Housekeeping skill** | Calls `loaf session enrich` on active sessions | Catches up sessions that haven't been enriched recently |
| **Manual** | User runs `loaf session enrich` directly | On-demand enrichment |

No Stop hook — enrichment involves spawning a separate agent, too expensive for every turn.

### CLI interface

```bash
loaf session enrich              # Enrich the active session
loaf session enrich <file>       # Enrich a specific session file
loaf session enrich --dry-run    # Show what would be added without writing
loaf session enrich --model <m>  # Override model for the librarian call
```

**Exit codes:**
- `0` — Success (entries added or already up-to-date)
- `1` — Error (JSONL not found, `claude` binary not available, agent failed)
- Errors go to stderr with actionable messages

### Librarian profile update

The librarian agent profile (`content/agents/librarian.md`) needs a minor update to its constraints:

**Current:** "Scope all file operations to `.agents/` paths."
**Updated:** "Scope **write** operations to `.agents/` paths. Read JSONL conversation logs from `${CLAUDE_CONFIG_DIR}/projects/` when performing enrichment."

This allows the librarian to read the JSONL (external to `.agents/`) while keeping all writes scoped to session files.

## Scope

### In Scope
- `loaf session enrich` CLI command with `--dry-run` and `--model` flags
- JSONL discovery logic (session ID → file path)
- Agent invocation via `claude --agent librarian -p`
- `enriched_at:` frontmatter field in `SessionFrontmatter` interface
- Librarian profile update: read access for JSONL files
- Wrap skill integration: call enrich before wrap-up
- Housekeeping skill integration: call enrich on active sessions
- Tests: CLI discovery logic, exit code handling

### Out of Scope
- Custom JSONL parser module (the librarian reads JSONL directly via Read tool)
- Changing existing PostToolUse hooks (commits, PRs, merges — they work fine)
- Adapters for other harnesses (Cursor, OpenCode, etc.)
- Changing the journal entry format
- Stop hook for enrichment (too expensive per-turn)
- Contract migration (no docs changes needed — enrichment is additive)

### Rabbit Holes
- **Building a JSONL parser module** — Tempting to pre-process in TypeScript. Don't — the librarian reads the file directly. The Read tool handles file slicing. Keep the CLI thin.
- **Running on every Stop** — Tempting for completeness. Don't — spawning a separate agent per turn is too expensive.
- **Abstracting a harness adapter interface** — Build for Claude Code directly. Extract an interface when a second harness is needed.
- **Replacing PostToolUse hooks** — The hooks work. Enrichment is complementary, not a replacement.
- **Parsing thinking blocks** — Internal reasoning, not journal events. Skip.

### No-Gos
- Don't write to the JSONL — it's read-only, owned by Claude Code
- Don't hardcode the project hash path — discover at runtime
- Don't swallow errors — non-zero exit from the agent must surface
- Don't run enrichment as a Stop hook — it spawns a separate agent, too heavy per-turn

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| JSONL format changes between Claude Code versions | Medium | Medium | Librarian reads defensively. If format changes, the prompt instructions adapt without code changes. |
| JSONL naming convention changes | Low | High | Filesystem scan fallback in discovery logic. |
| `claude` binary not available | Low | High | Check before spawning. Clear error message. Manual `loaf session log` remains as fallback. |
| Librarian produces low-quality entries | Medium | Low | `--dry-run` lets user preview. Entries can be manually edited. Librarian profile/prompt can be iterated. |
| Large JSONL exceeds Read tool limits | Low | Medium | Read tool supports offset/limit for pagination. CLI can pass file size hint in prompt. |
| Token cost of agent invocation | Medium | Low | Only runs at lifecycle points. `--model` flag allows cost control (haiku for routine). |
| `--agent` flag doesn't discover librarian in plugin-only install | Low | Medium | Agent must be accessible. If plugin caching is an issue, the spark about leaving the plugin system applies. Alternatively, use `--append-system-prompt-file` with the librarian profile content as fallback. |

## Open Questions

- [x] What model should enrichment use? → Default to whatever `claude` defaults to. `--model` flag for override. Haiku is likely sufficient.
- [x] Should `--dry-run` output go to stdout or stderr? → stdout — it's the primary output.
- [ ] Should the CLI pre-compute JSONL line count / byte size and pass it in the prompt to help the librarian paginate? (Probably yes for v2, not needed for v1)

## Test Conditions

- [ ] `loaf session enrich` on a session with a JSONL log adds semantic entries not already in the journal
- [ ] `--dry-run` shows entries without writing them
- [ ] Re-running enrich on an already-enriched session with no new JSONL entries is a no-op
- [ ] `enriched_at:` timestamp advances after successful enrichment
- [ ] Missing JSONL file produces a clear error (exit 1) and does not crash
- [ ] Missing `claude` binary produces a clear error
- [ ] Missing `claude_session_id` in session frontmatter produces a clear error
- [ ] Wrap skill calls enrich before generating wrap-up report
- [ ] Housekeeping skill calls enrich on active sessions
- [ ] `loaf build` succeeds with new command registered
- [ ] Librarian profile update doesn't break existing behavior

## Priority Order

1. **CLI command + librarian update** — `loaf session enrich`, JSONL discovery, agent invocation, `enriched_at:` field, librarian profile update. Go/no-go: enrichment adds entries to journal via librarian.
2. **Lifecycle integration** — Wrap and housekeeping skill updates. Go/no-go: enrichment runs automatically at session boundaries.
