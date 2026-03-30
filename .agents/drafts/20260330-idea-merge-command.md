---
type: idea
created: '2026-03-30'
status: captured
---

# Idea: `/merge` Slash Command

## Nugget

A `/merge` command that sequences the full merge ritual — pre-flight checks, housekeeping verification, squash merge with clean body, and post-merge cleanup. Replaces the current spread of manual steps + hook enforcement with a single orchestrated flow.

## What It Would Do

1. **Pre-flight**: Run `loaf build`, `npm run typecheck`, `npm run test`
2. **Housekeeping check**: Verify spec status is `complete`, tasks are archived, changelog has `[Unreleased]` entry
3. **Version bump**: Bump `package.json` version, convert changelog `[Unreleased]` header to versioned entry with date, rebuild, commit on the feature branch — so the squash commit carries the new version
4. **Squash merge**: `gh pr merge --squash` with a prompted clean body (not the auto-generated commit dump)
5. **Post-merge**: Pull main, delete branch, suggest `/reflect` if session has extractable learnings

## Why

The `workflow-pre-merge` hook enforces the squash body format, but the surrounding steps (pre-flight, housekeeping, post-merge cleanup) are manual and easy to forget. A command makes the happy path the default path.

## Relationship to Existing Pieces

- `workflow-pre-merge` hook → absorbed into step 4 (or kept as a safety net if someone merges outside the command)
- `workflow-post-merge` hook → absorbed into step 5
- `validate-push` hook → already checks version bump on push, but fires too late for merge
- Implement skill's AFTER section → documents this flow but doesn't enforce it

## Key Insight: Version Bump Before Merge, Not After

The version bump must happen on the feature branch so it's included in the squash commit. Post-merge bumps mean main briefly has stale version metadata. The `/merge` command (or `workflow-pre-merge` hook) should verify the bump happened and block if not — or offer to do it interactively.

## Open Questions

- Should it auto-detect the PR number from the current branch, or require it as an argument?
- Should pre-flight failures block the merge or just warn?
- Is this a skill (loaded into context) or a CLI command (`loaf merge`)?
- Version bump strategy: auto-increment `dev.N`? Or prompt for bump type (patch/minor/major)?
