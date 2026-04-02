---
id: TASK-086
title: Verify loaf binary accessibility from all hook environments
spec: SPEC-020
status: todo
priority: p1
dependencies: [TASK-082, TASK-083, TASK-084]
track: D
---

# TASK-086: Verify `loaf` binary accessibility from all hook environments

End-to-end verification of the full SPEC-020 implementation.

## Scope

### Binary accessibility
Verify `loaf` binary is callable from:
- Claude Code plugin hooks (`${CLAUDE_PLUGIN_ROOT}/bin/loaf`)
- Cursor hooks (PATH-based or plugin-relative)
- Codex hooks (PATH-based)
- OpenCode/Amp runtime plugins (subprocess)

### Cursor sandbox
Test Cursor sandbox PATH restrictions. If restricted, implement binary bundling in `.cursor/bin/loaf`.

### Install verification
`loaf install` warns if binary not on PATH for targets that need it.

### Full build + install cycle
Run complete build and install for all 6 targets. Verify hook execution end-to-end.

### Spec test conditions
Run through all test conditions from SPEC-020 across all 4 phases:
- Phase 1: shared intermediate exists, parity checks pass
- Phase 2: descriptions truncated, structural convention applied, hooks migrated
- Phase 3: `loaf check` works, `loaf session` works, direct commands in configs, no bash wrappers
- Phase 4: Amp target builds, Codex hooks installed, fenced sections work, binary accessible

## Verification

- [ ] `loaf check --hook check-secrets` callable from each target's hook environment
- [ ] `loaf session start` callable from each target's session hook
- [ ] `loaf install` warns if binary not on PATH
- [ ] Full `loaf build` for all 6 targets succeeds
- [ ] `npm run typecheck` and `npm run test` pass
- [ ] All SPEC-020 test conditions pass
