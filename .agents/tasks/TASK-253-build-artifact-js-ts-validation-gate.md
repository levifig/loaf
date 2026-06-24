---
id: TASK-253
title: Add build artifact JS/TS validation gate
spec: SPEC-047
status: todo
priority: P1
created: '2026-06-24T12:03:41Z'
updated: '2026-06-24T12:03:41Z'
completed_at: null
depends_on: []
files:
  - internal/cli/build_test.go
  - internal/cli/build.go
  - internal/cli/build_amp.go
  - internal/cli/build_opencode.go
  - package.json
  - .agents/tasks/TASK-253-build-artifact-js-ts-validation-gate.md
verify: >-
  go test ./internal/cli -run 'TestRunnerBuild|NativeBuild.*Validation' -count=1
  && npm run typecheck
done: >-
  The native build/test gate validates emitted JavaScript and TypeScript
  artifacts with real tooling, removes the fake node shim, and preserves the
  explicit dependency-approval hold before adding any new npm dev dependency.
---

# TASK-253: Add build artifact JS/TS validation gate

## Description

SPEC-047 starts by making the build prove the artifacts it emits. Replace the
fake `node` test shim and the TypeScript-in-`.js` assertion with real validation
over generated JavaScript and TypeScript outputs.

If the implementation needs `typescript`, `@ampcode/plugin`, or another
third-party dev dependency to validate Amp/OpenCode TypeScript correctly, pause
and request explicit approval before editing `package.json`.

## Acceptance Criteria

- [ ] `setupFakeNodeForBuild` is removed or no longer used to prove native build
  correctness.
- [ ] Emitted JavaScript artifacts are checked with `node --check`.
- [ ] Emitted TypeScript artifacts have a deterministic validation command.
- [ ] CI hard-fails when required validation tooling is absent.
- [ ] Local TypeScript validation may warn-and-skip only outside CI, naming the
  skipped files.
- [ ] Tests prove malformed generated JS and TS artifacts fail validation.

## Verification

```bash
go test ./internal/cli -run 'TestRunnerBuild|NativeBuild.*Validation' -count=1
npm run typecheck
```
