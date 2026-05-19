---
id: TASK-168
title: >-
  Route user-content artifacts (kb/ideas/drafts/reports/councils) through
  findSharedAgentsDir
status: todo
priority: P2
created: '2026-05-18T23:58:56.055Z'
updated: '2026-05-18T23:58:56.055Z'
spec: SPEC-036
depends_on:
  - TASK-166
---

# TASK-168: Route user-content artifacts (kb/ideas/drafts/reports/councils) through findSharedAgentsDir

## Description

Migrate kb, ideas, drafts, reports, and councils call sites from `findLocalAgentsDir` to `findSharedAgentsDir`. The work pattern is mechanical (resolver swap per call site) and identical across the five subdirs — they're grouped here to avoid task ceremony, but verification asserts each subdir independently.

## Acceptance Criteria

- [ ] `cli/commands/kb.ts` and kb-related helpers use `findSharedAgentsDir`
- [ ] All ideas / drafts / reports / councils call sites use `findSharedAgentsDir`
- [ ] Per-subdir tests pass after the swap
- [ ] No regression in existing kb-listing, idea-capture, draft-listing, report-listing, council-listing flows
- [ ] Branch-local artifact paths (specs, tasks, plans) untouched

## Files

- `cli/commands/kb.ts`
- `cli/commands/report.ts`
- Any module reading/writing `ideas/`, `drafts/`, `councils/` (grep `findAgentsDir` and triage by subdir)

## Verification

```bash
npm run test -- kb
npm run test -- ideas
npm run test -- drafts
npm run test -- reports
npm run test -- councils
```

## Context

See SPEC-036. Depends on TASK-166.
