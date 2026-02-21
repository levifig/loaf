# Plan: Documentation audit and fixes

## Context

After the signal-density refactoring (templates extraction, skill trimming, build system changes), checking whether project docs reflect the current state. The user's standard: strictly essential, succinct, respecting the reader's time.

## Assessment

**Two documentation files exist:**
- `README.md` (223 lines) — user-facing: what Loaf is, how to install, what it does
- `.agents/AGENTS.md` (318 lines) — developer-facing: how to maintain and extend Loaf
- `src/SETUP.md` (117 lines) — post-install LSP/MCP server setup

**Overall: docs are 95%+ accurate.** The signal-density refactoring already updated AGENTS.md with templates architecture. No major rewrites needed.

## Issues found

### 1. AGENTS.md: Cursor target note is wrong (factual error)

Line 260 says "Skills and agents only" — but Cursor also gets hooks. README correctly says "Agents, skills, hooks."

**Fix:** Change "Skills and agents only" to "Skills, agents, and hooks" in the Build System targets table.

### 2. AGENTS.md: Common Tasks missing "Add template" entry

The Common Tasks section has entries for skill, agent, hook, target — but not for adding a template. Since templates are now a first-class concept with their own build pipeline and shared-templates config, this deserves a one-liner.

**Fix:** Add: `**Add template:** Create in src/skills/{name}/templates/ (skill-specific) or src/templates/ + register in shared-templates in targets.yaml`

### 3. AGENTS.md: "Before Committing" checklist incomplete

Missing template-related checks after the refactoring.

**Fix:** Add: `- [ ] Template links resolve (no broken templates/ paths)`

### 4. README.md: no changes needed

README is user-facing. Templates are a build-system internal. Users don't need to know about template directories — they interact through commands (`/implement`, `/shape`, etc.). The README accurately describes what Loaf is, how to install it, and what commands/agents/skills exist. No stale content found.

## Files to modify

- `.agents/AGENTS.md` — 3 small edits (lines 46, 260, 271)

## Verification

- Confirm AGENTS.md targets table matches README targets table
- Confirm all Common Tasks map to actual project workflows
- `npm run build` still passes (no content changes, just docs)
