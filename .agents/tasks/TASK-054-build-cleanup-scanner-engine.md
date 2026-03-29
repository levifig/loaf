---
id: TASK-054
title: Build cleanup scanner engine
spec: SPEC-012
status: todo
priority: P1
created: '2026-03-28T23:36:09.760Z'
updated: '2026-03-28T23:36:09.760Z'
---

# TASK-054: Build cleanup scanner engine

## Description

Build the pure-logic scanner that walks `.agents/` directories, applies the existing cleanup rules, and produces a typed list of recommendations. No I/O, no prompts — just analysis.

## What to do

1. Create `cli/lib/cleanup/types.ts`:
   - `ArtifactType` enum: session, task, spec, plan, draft, council, report
   - `RecommendedAction`: archive, delete, flag, skip
   - `CleanupRecommendation`: `{ type, path, action, reason, metadata? }`
   - `ScanResult`: `{ recommendations: CleanupRecommendation[], summary: ScanSummary }`
   - `ScanOptions`: `{ filter?: ArtifactType[], agentsDir: string }`

2. Create `cli/lib/cleanup/scanner.ts`:
   - `scanArtifacts(options: ScanOptions): ScanResult`
   - Per-artifact scanner functions implementing existing cleanup skill rules:
     - Sessions: completed → archive, >7 days inactive → flag, cancelled → archive
     - Tasks: done → archive (reuse `findOrphans()` for orphan detection), orphaned ref → flag
     - Specs: complete → archive, stale drafting → flag
     - Plans: orphaned/linked-to-archived-session → delete, >14 days stale → delete
     - Drafts: >30 days → flag, promoted → archive-or-delete
     - Councils: orphaned → flag, summary captured → archive
     - Reports: processed + session archived → archive
   - Reuse existing helpers: `getOrBuildIndex()`, `findOrphans()`, `collectFiles()`
   - Handle missing optional directories (skip silently)
   - Warn on missing required directories (sessions, specs, tasks)
   - Detect `/crystallize` candidates (sessions with key decisions/lessons)

3. Tests (`scanner.test.ts`):
   - Create temp `.agents/` with fixtures for each artifact state
   - Verify correct recommendation per rule
   - Verify missing directories handled correctly
   - Verify filter option works

## Acceptance Criteria

- [ ] Scanner returns typed recommendations for all 7 artifact types
- [ ] Missing optional directories are skipped silently
- [ ] Missing required directories produce a warning in results
- [ ] Sessions with extractable learnings flagged for `/crystallize`
- [ ] Done tasks detected via TASKS.json status
- [ ] Orphaned tasks detected as "referenced spec doesn't exist" (not "spec is null")
- [ ] Plans linked to archived/missing sessions recommended for deletion
- [ ] Drafts >30 days old flagged for review
- [ ] Filter option restricts scan to specified artifact types
- [ ] Tests cover each artifact type's rules

## Verification

```bash
npm run typecheck && npm run test
```
