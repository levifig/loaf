# Plan: Migrate Commands to Skills (v3.1)

Supersedes v2.1 at `.agents/plans/20260217-212637-migrate-commands-to-skills-v2-1.md`.

## Context

Claude Code has merged custom slash commands into skills — a skill at `.claude/skills/review/SKILL.md` creates the same `/review` invocation as a command at `.claude/commands/review.md`, but skills are a superset (supporting files, dynamic context injection, lifecycle hooks, invocation control). Loaf currently maintains both `src/commands/` (13 workflow files) and `src/skills/` (11 knowledge directories). This migration eliminates `src/commands/` entirely, making skills the single source of truth for all capabilities.

**Key difference from v2.1**: No "canonical wrapper" special cases. `/shape` and `/implement` become skills like everything else. One source format, one migration path.

## Preflight Note

`src/commands/implement.md` frontmatter is malformed today; normalize it during migration so the defect is not copied into `src/skills/implement/SKILL.md`.

## Locked Decisions

1. **All commands become skills** — `src/commands/` is deleted entirely
2. **At cutover, all former commands become `user-invocable: true`** (default) — users still type `/research`, `/shape`, etc.
3. **Each command becomes its own skill directory** — standalone, not folded into orchestration
4. **OpenCode commands generated from `SKILL.opencode.yaml`** — only skills with this sidecar emit commands
5. **OpenCode sidecar carries routing metadata** — `agent`, `subtask`, `model` fields; SKILL.md provides body content
6. **Template placement** follows CLAUDE.md convention: quick templates inline in SKILL.md, detailed in `references/`
7. **Shared template deduplication** (orchestration templates used by multiple skills) deferred to follow-up
8. **Cursor, Codex, Gemini** get skills only, no command output

## Command-to-Skill Mapping

| Command | Skill | Notes |
|---|---|---|
| `/research` | `research` (exists) | Merge command workflow into SKILL.md; existing reference content stays |
| `/shape` | `shape` (new) | |
| `/implement` | `implement` (new) | Includes `/orchestrate` batch mechanics (already done) |
| `/architecture` | `architecture` (new) | |
| `/brainstorm` | `brainstorm` (new) | |
| `/breakdown` | `breakdown` (new) | |
| `/council-session` | `council-session` (new) | |
| `/idea` | `idea` (new) | |
| `/reflect` | `reflect` (new) | |
| `/resume` | `resume` (new) | |
| `/review-sessions` | `review-sessions` (new) | Carry over hooks only if a valid existing script path is confirmed; otherwise omit hooks |
| `/reference-session` | `reference-session` (new) | |
| `/strategy` | `strategy` (new) | |

Result: 12 new skill directories + 1 merge = 23 total skills.

## Phase 1: Create Skill Directories

Guarded coexistence phase — existing command surfaces keep working, but migrated command-skills stay hidden until cutover.

### Coexistence guardrails

- For each newly migrated command-skill, set `user-invocable: false` in `SKILL.claude-code.yaml` during coexistence
- Keep this guard only while legacy command output still exists
- In Phase 2 cutover (before/with legacy command-output removal), flip migrated command-skills to user-invocable (`true` or remove explicit `false`) so slash-command continuity is preserved

### Template placement

Follow existing CLAUDE.md convention:
- **Quick templates** (1-2 most common output formats): embed directly in SKILL.md
- **Detailed templates** (full session file format, spec structure, etc.): move to `references/` files
- No new `templates/` directory convention — use `references/` for all supporting material

### For each of the 12 new skills, create:

**`src/skills/{name}/SKILL.md`**
- Move command `.md` body content into SKILL.md
- Replace simple command frontmatter (`description: ...`) with full skill frontmatter (`name`, `description`)
- Rewrite `description` per CLAUDE.md guidelines: third-person, action-verb start, user-intent phrases, negative routing
- Keep `$ARGUMENTS`, `{{AGENT:...}}`, `{{IMPLEMENT_CMD}}`, `{{RESUME_CMD}}` placeholders — build system resolves these
- Keep slash-command references (`/brainstorm`, `/shape`) — build system scopes them for Claude Code
- Add `## Contents` TOC for files over 100 lines
- For `implement` (~865 lines): split into SKILL.md (core workflow) + `references/` for batch orchestration, session management, delegation model

**`src/skills/{name}/SKILL.claude-code.yaml`**
- `argument-hint` from existing command sidecar
- During coexistence, set `user-invocable: false` for all migrated command-skills
- For `review-sessions`: include `hooks:` only when the referenced script path exists; do not carry invalid/missing paths

**`src/skills/{name}/SKILL.opencode.yaml`** (all 13 former commands)
- `agent`, `subtask`, `model` fields from existing command `.opencode.yaml` sidecars where they exist
- Default for skills without existing OpenCode sidecar: `agent: "{{AGENT:pm}}"`, `subtask: false`

### Research merge (special case)

Rewrite `src/skills/research/SKILL.md`:
- Command workflow instructions become the primary content
- Existing skill reference knowledge remains (or moves to `references/` if SKILL.md exceeds ~500 lines)
- Add `SKILL.claude-code.yaml` with `argument-hint: "[topic]"`
- Add `SKILL.opencode.yaml` with routing metadata

### Verify Phase 1
- `npm run build` succeeds (old commands + new skills coexist)
- All 23 skill directories exist under `src/skills/`
- Each has valid frontmatter (`name` matches directory, `description` starts with action verb)
- All migrated command-skills are `user-invocable: false` during coexistence
- No skill sidecar references a non-existent hook script

## Phase 2: Build System Changes

### 2A. `build/targets/claude-code.js`

**Cutover sequencing (explicit):**

- Before or alongside removing legacy command output, flip migrated command-skills from `user-invocable: false` to user-invocable (`true` or no explicit override)
- After this flip, `knownCommands` should include those migrated command-skills (they are now visible slash commands)

**Replace command discovery with user-invocable skill discovery:**

In `buildUnifiedPlugin()` (line 241), post-cutover:
- Remove `const allCommands = discoverCommands(srcDir);` (line 247)
- Add: derive `knownCommands` from `allSkills` by checking which skills are user-invocable (those whose `SKILL.claude-code.yaml` does NOT have `user-invocable: false`, or have no sidecar at all — since default is `true`)
- Reuse existing `loadSkillExtensions()` from `build/lib/sidecar.js` to check each skill's sidecar

```javascript
// Replace discoverCommands with user-invocable skill names for command scoping
const knownCommands = allSkills.filter(skill => {
  const extensions = loadSkillExtensions(join(srcDir, "skills", skill));
  return extensions["user-invocable"] !== false;
});
```

**Remove command output:**
- Remove `copyCommands()` call (line 263)
- Remove `copyCommands()` function (line 557+)
- Remove `commands` parameter from `createPluginJson()` (line 254, 324)
- Remove `commands: commands.map(...)` from `plugin.json` output (line 332)

**Update scoping:**
- Pass `knownCommands` (user-invocable skill names) to `copySkills()` and `copyReferencesWithSubstitution()` — same mechanism as today, different source
- Keep `substituteCommands()` function as-is (it handles `{{IMPLEMENT_CMD}}`, `{{RESUME_CMD}}`, `{{ORCHESTRATE_CMD}}` placeholders + auto-scoping)

**Cleanup:**
- Remove `loadCommandSidecar` import (line 40)
- Update file header comment to remove `commands/` from output structure

### 2B. `build/targets/opencode.js`

**Replace `copyCommands()` with `generateCommandsFromSkills()`:**

The new function iterates over skills that have `SKILL.opencode.yaml`:
- Load `SKILL.opencode.yaml` for routing metadata (`agent`, `subtask`, `model`)
- Load `SKILL.md` for body content and standard frontmatter (`description`)
- Merge: skill frontmatter + sidecar routing + version → command frontmatter
- Resolve `{{AGENT:...}}` placeholders in `agent` field
- Apply `substituteCommands()` and `substituteAgentNames()` to body
- Write to `dist/opencode/command/{skill-name}.md`

This mirrors the existing `copyCommands()` pattern (lines 220-262) but reads from skills instead of `src/commands/`.

**In `build()` function (line 115):**
- Replace `copyCommands(srcDir, distDir)` with `generateCommandsFromSkills(srcDir, distDir)`

**Cleanup:**
- Remove old `copyCommands()` function
- Remove `loadCommandSidecar` import (line 34)

### 2C. `build/targets/cursor.js`

**Remove command output:**
- Remove `copyCommands()` call (line 137)
- Remove `copyCommands()` function (line 295+)
- Remove `commandsDir` from directory creation
- Update file header comment

### 2D. `build/lib/sidecar.js`

- Remove `loadCommandSidecar()` function (unused after Phase 2A-2C)
- Add `loadSkillOpencodeSidecar()` if needed (or reuse existing pattern in the `generateCommandsFromSkills` function)

### 2E. `build/build.js`

- Update header comment to remove `src/commands/` source assumptions while preserving OpenCode generated `dist/opencode/command/` semantics

### Verify Phase 2
- `npm run build` succeeds
- `plugins/loaf/commands/` does not exist
- `plugins/loaf/.claude-plugin/plugin.json` has no `commands` array
- `plugins/loaf/skills/` contains all 23 skills with merged frontmatter
- `/research` in skill content is scoped to `/loaf:research` in Claude Code output
- `dist/opencode/command/` has command files for all 13 former-command skills
- `dist/opencode/command/` has NO command for reference-only skills (python-development, etc.)
- `dist/cursor/commands/` does not exist
- `{{AGENT:pm}}` resolved in all outputs

## Phase 3: Config + Docs Cleanup

### `src/config/hooks.yaml`

The `plugin-groups` section is documented as "NOT used by the build system" — it's documentation only. Update:
- Move all entries from `commands:` lists into `skills:` lists
- Remove `commands:` keys from all groups
- Update comment about "commands" scoping

### Docs cleanup (`AGENTS.md`, `CLAUDE.md`, `.claude/CLAUDE.md`)

- Remove `commands/{name}.md` from Project Structure tree
- Remove Commands row from Quick Reference table
- Remove "Add command" from Common Tasks
- Remove or replace "Command Development" section
- Update Before Committing checklist
- Remove remaining `src/commands/` references and align language to skills-first architecture

### Verify Phase 3
- Build still succeeds
- No references to `src/commands/` in documentation

## Phase 4: Delete `src/commands/`

Only after Phases 1-3 verified.

- Update `install.sh` cleanup logic so Cursor command removal also deletes stale legacy files in `~/.cursor/commands` from previous installs/upgrades.

Delete all files in `src/commands/`:
- 13 command `.md` files
- All `.claude-code.yaml` sidecars
- All `.opencode.yaml` sidecars
- Orphan sidecars: `start-session.claude-code.yaml`, `start-session.opencode.yaml`, `resume-session.claude-code.yaml`, `resume-session.opencode.yaml`
- The directory itself

### Verify Phase 4
- `npm run build` succeeds
- `src/commands/` does not exist
- Migrated former-command skills are user-invocable in Claude output
- Outputs remain behaviorally consistent with Phase 3 (Phase 4 introduces no new behavior changes)
- Install/upgrade removes stale Cursor command files from `~/.cursor/commands`

## Critical Files

| File | Changes |
|---|---|
| `build/targets/claude-code.js` | Remove command discovery/copy, update scoping source, update plugin.json |
| `build/targets/opencode.js` | Replace `copyCommands()` with `generateCommandsFromSkills()` |
| `build/targets/cursor.js` | Remove command copy |
| `build/lib/sidecar.js` | Remove `loadCommandSidecar()` |
| `build/build.js` | Update comment |
| `install.sh` | Remove stale Cursor command files from legacy installs |
| `src/config/hooks.yaml` | Move `commands:` to `skills:` in plugin-groups |
| `src/skills/research/SKILL.md` | Merge with command workflow content |
| `AGENTS.md` + `CLAUDE.md` + `.claude/CLAUDE.md` | Remove command references, align to skills-first architecture |

## Risks

1. **Command scoping regression** — After migration, `substituteCommands()` gets `knownCommands` from user-invocable skills instead of `discoverCommands()`. If any skill name doesn't match the old command name, scoping breaks. Mitigation: the command-to-skill mapping preserves all names exactly.

2. **OpenCode command fidelity** — Generated commands have skill-style frontmatter (richer than old command frontmatter). Functionally equivalent but structurally different. Verify by diffing `dist/opencode/command/` before/after.

3. **`implement` skill size** — At ~865 lines, exceeds the 500-line guideline. Must split into SKILL.md + references/. This is the highest-effort single migration.

4. **`review-sessions` hooks ambiguity** — Carrying stale/missing hook paths into skill sidecars can break runtime behavior. Mitigation: include `hooks:` only when path existence is verified; otherwise omit.

## Verification (End-to-End)

```bash
npm run build
```

Then verify:
- [ ] Build succeeds for all 5 targets
- [ ] `plugins/loaf/skills/` has 23 skill directories
- [ ] `plugins/loaf/commands/` does not exist
- [ ] `plugin.json` has no `commands` key
- [ ] `dist/opencode/command/` has exactly 13 command files (one per former command)
- [ ] `dist/opencode/command/` has no `orchestrate.md`
- [ ] `dist/cursor/` has `skills/` but no `commands/`
- [ ] `dist/codex/skills/` and `dist/gemini/skills/` have 23 skill directories
- [ ] User-invocable filtering is correct (`user-invocable: false` excluded; default-true included)
- [ ] No skill sidecar references a non-existent hook script
- [ ] Command scoping is correct across all migrated names and already-scoped forms are not double-scoped (`/loaf:loaf:*` absent)
- [ ] OpenCode generated command frontmatter preserves `description`, routing (`agent`, `subtask`, `model`), and `version`
- [ ] No unresolved placeholders remain (`{{AGENT:`, `{{IMPLEMENT_CMD}}`, `{{RESUME_CMD}}`, `{{ORCHESTRATE_CMD}}`)
- [ ] `src/commands/` does not exist
