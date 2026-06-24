---
id: TASK-255
title: Remove Gemini from in-repo build and install surfaces
spec: SPEC-047
status: todo
priority: P1
created: '2026-06-24T12:03:41Z'
updated: '2026-06-24T12:03:41Z'
completed_at: null
depends_on:
  - TASK-254
files:
  - config/targets.yaml
  - internal/cli/build.go
  - internal/cli/build_test.go
  - internal/cli/install_fenced.go
  - internal/cli/install_mcp.go
  - dist/gemini/
  - .agents/tasks/TASK-255-remove-gemini-in-repo-build-install-surfaces.md
verify: >-
  ! rg -n 'gemini' config internal/cli dist plugins package.json README.md docs
  content && npm run build && npm run test
done: >-
  Gemini is absent from source build/install wiring and generated in-repo output
  while user-side cleanup remains deferred to SPEC-053.
---

# TASK-255: Remove Gemini from in-repo build and install surfaces

## Description

Shrink the first-class harness matrix from six targets to five by removing Gemini
from repo build output, build wiring, install wiring, generated artifacts, and
tests.

This task must not remove already-installed user Gemini files. User-side cleanup
is a breaking migration action and remains gated on SPEC-053.

## Acceptance Criteria

- [ ] `config/targets.yaml` no longer declares a Gemini target.
- [ ] `defaultBuildTargets`, target validation, and target dispatch enumerate
  exactly Claude Code, OpenCode, Cursor, Codex, and Amp.
- [ ] `dist/gemini/` is no longer produced or tracked.
- [ ] Install fencing/MCP logic no longer advertises or installs Gemini.
- [ ] Tests and fixtures no longer include Gemini expectations.
- [ ] User-side orphan cleanup is not implemented in this task.

## Verification

```bash
! rg -n 'gemini' config internal/cli dist plugins package.json README.md docs content
npm run build
npm run test
```
