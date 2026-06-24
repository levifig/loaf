---
id: TASK-352
title: Enforce registered artifact body writes
spec: SPEC-043
status: todo
priority: P1
created: '2026-06-24T13:04:19Z'
updated: '2026-06-24T13:04:19Z'
completed_at: null
depends_on:
  - TASK-351
files:
  - internal/cli/check.go
  - internal/cli/build_*.go
  - config/hooks.yaml
  - content/hooks/
  - .agents/tasks/TASK-352-registered-artifact-body-write-enforcement.md
verify: >-
  go test ./internal/cli -run 'ArtifactBodyWrite|Check|Hook|BuildTarget' -count=1
  && npm run build
done: >-
  Harness-portable enforcement catches direct unregistered .agents artifact-body
  writes while allowing explicit generated/rendered exceptions.
---

# TASK-352: Enforce registered artifact body writes

## Description

Add SPEC-043 Track 3: a write-side registration guard so agents and hooks stop
free-handing body artifacts into `.agents` without registering SQLite nouns.

## Acceptance Criteria

- [ ] Direct writes to body-capable `.agents` artifact paths are detected when they bypass the SQLite body write path.
- [ ] Generated durable renders, templates, specs/tasks metadata updates, and non-artifact docs have explicit tested allow rules.
- [ ] The guard is exposed through the existing hook/check infrastructure and respects the five-target parity contract from SPEC-047.
- [ ] Failure output names the file path and the CLI command that should be used instead.
- [ ] The enforcement is non-breaking for existing Markdown fallback files in this spec.

## Verification

```bash
go test ./internal/cli -run 'ArtifactBodyWrite|Check|Hook|BuildTarget' -count=1
npm run build
```
