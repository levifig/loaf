---
id: SPEC-012
title: "Cleanup Refinement — differentiated processes + CLI"
source: direct
created: 2026-03-16T23:50:30Z
status: drafting
appetite: "Medium (3-5 days)"
---

# SPEC-012: Cleanup Refinement

## Problem Statement

The `/cleanup` skill (formerly `/review-sessions`) reviews agent artifacts and recommends actions, but it treats all artifact states the same way and relies entirely on the agent following skill instructions. There's no CLI surface for humans to run cleanup directly, no differentiated handling for completed vs stale vs cancelled work, and no guardrails against accidental data loss.

## Strategic Alignment

- **Vision:** Supports the "knowledge management" pillar by ensuring agent artifacts don't accumulate as noise.
- **Personas:** Framework user gets a `loaf cleanup` command for project hygiene. Agents get clearer rules for artifact lifecycle.
- **Architecture:** Builds on SPEC-010's task management CLI pattern (CLI mediates operations).

## Solution Direction

### Part 1: Differentiated Artifact Handling

Refine the cleanup skill with explicit processes per artifact state:

| Artifact | State | Process | Preservation |
|----------|-------|---------|-------------|
| **Sessions** | Completed | Archive (move to archive/) | Full — set `archived_at`, `archived_by` |
| **Sessions** | Stale (>7 days inactive) | Review → user decides | Never auto-delete |
| **Sessions** | Cancelled/abandoned | Archive with `status: cancelled` | Preserved with reason |
| **Specs** | Complete | Archive | Full — move to archive/ |
| **Specs** | Stale drafting | Flag for review | Never auto-delete |
| **Plans** | Session archived | Delete | Ephemeral — decisions in session |
| **Plans** | Orphaned (no session) | Delete | Ephemeral — no value without session |
| **Drafts** | Promoted to spec | Archive or delete (user choice) | Ask first |
| **Drafts** | Stale (>30 days) | Flag for review | Never auto-delete |
| **Councils** | Session summary captured | Archive | Full preservation |
| **Reports** | Processed + session archived | Archive | Full preservation |

**Core principle:** Archive, don't delete — except for truly ephemeral artifacts (plans) whose value lives entirely in session files and git history.

### Part 2: `loaf cleanup` CLI Command

A new CLI command that automates the scanning and reporting:

```
loaf cleanup              # Scan all artifacts, show summary + recommendations
loaf cleanup --dry-run    # Same but don't prompt for actions
loaf cleanup --sessions   # Only review sessions
loaf cleanup --specs      # Only review specs
loaf cleanup --plans      # Only review plans
loaf cleanup --drafts     # Only review drafts
```

The command:
1. Scans `.agents/` directories for each artifact type
2. Applies the differentiated rules above
3. Shows a formatted summary table (like `loaf task status`)
4. For each actionable item, prompts: Archive / Delete / Keep / Skip
5. Performs the chosen action (move, delete, update status)
6. Suggests `/crystallize` for sessions with extractable learnings

### Part 3: Skill Update

Update the `/cleanup` skill to reference the CLI command and the differentiated rules. The skill becomes guidance for agents on when/how to invoke `loaf cleanup`, while the CLI does the actual work.

## Scope

### In Scope
- Differentiated handling rules per artifact type and state
- `loaf cleanup` CLI command with scanning, reporting, and interactive actions
- `--dry-run` and artifact-type filters
- Archive operations (move + set metadata)
- Delete operations for ephemeral artifacts (plans)
- Suggest `/crystallize` for sessions with extractable learnings
- Update `/cleanup` skill to reference CLI
- Register command in `cli/index.ts`

### Out of Scope
- Automatic cleanup (always interactive or dry-run)
- Linear integration (existing, not changing)
- Knowledge staleness detection (SPEC-009)
- The `/crystallize` skill itself (SPEC-011)

### Rabbit Holes
- Building a TUI for cleanup — stick with sequential prompts like `loaf release`
- Smart staleness detection beyond file dates — keep it simple (mtime, frontmatter dates)
- Recursive archive scanning for cleanup — `.agents/` subdirectories are one level deep

### No-Gos
- Don't auto-delete anything without user confirmation
- Don't archive sessions without checking for extractable learnings first
- Don't touch files outside `.agents/`
- Don't require Linear for any functionality

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Users accidentally delete drafts with unique content | Low | High | Always show content preview before delete; default to "Keep" |
| Cleanup removes plans that active sessions reference | Low | Medium | Check session status before deleting linked plans |
| Edge cases in frontmatter parsing | Medium | Low | Reuse SPEC-010's gray-matter parser; graceful fallbacks |

## Open Questions

- [x] Appetite → Medium (3-5 days)
- [x] CLI or skill-only → CLI command + skill update

## Test Conditions

- [ ] `loaf cleanup` scans all artifact types and shows formatted summary
- [ ] `loaf cleanup --dry-run` shows recommendations without prompting for actions
- [ ] `loaf cleanup --sessions` filters to sessions only
- [ ] Completed sessions are offered for archive with correct metadata
- [ ] Stale sessions (>7 days) are flagged but not auto-actioned
- [ ] Orphaned plans (no linked session) are offered for deletion
- [ ] Plans linked to archived sessions are offered for deletion
- [ ] Drafts >30 days old are flagged for review
- [ ] Archive operations set `archived_at`, `archived_by`, and move to archive/
- [ ] Delete operations require confirmation before removing files
- [ ] Suggests `/crystallize` for sessions with key decisions or lessons

## Circuit Breaker

At 50%: Drop artifact-type filters (`--sessions`, `--specs`, etc.). Ship `loaf cleanup` with full scan only.

At 75%: Drop interactive actions. Ship scanning + dry-run reporting only. User manually archives/deletes based on the report.
