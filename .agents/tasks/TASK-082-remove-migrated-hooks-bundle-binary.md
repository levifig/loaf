---
id: TASK-082
title: Remove migrated hooks, shell libs, session skills + bundle binary
spec: SPEC-020
status: todo
priority: p1
dependencies: [TASK-079, TASK-080, TASK-081]
track: C
---

# TASK-082: Remove migrated hooks, shell libs, session skills + bundle binary

Cleanup task: remove artifacts that have been replaced by `loaf check`, skill instructions, and `loaf session`.

## Scope

### Remove ~20 hook scripts
Delete migrated hook scripts from `content/hooks/pre-tool/` and `content/hooks/post-tool/`:
- All language hooks (python-*, typescript-*, rails-*)
- All infra hooks (k8s, dockerfile, terraform)
- All design hooks (a11y, tokens)
- format-check, tdd-advisory, validate-changelog, changelog-reminder

### Remove 4 shared shell libraries
Delete from `content/hooks/lib/`:
- `json-parser.sh`
- `config-reader.sh`
- `agent-detector.sh`
- `timeout-manager.sh`

### Remove session skills
Delete skill directories:
- `content/skills/resume-session/` (absorbed by `loaf session start`)
- `content/skills/reference-session/` (absorbed by archived journal reads)
Update `config/targets.yaml` shared-templates references.

### Audit remaining hooks
Before removing libraries, verify remaining hooks (side-effect, CLI-wrapping) have no dependencies on deleted libraries.

### Bundle CLI binary
Copy compiled CLI binary into plugin output directories:
- `plugins/loaf/bin/loaf` (Claude Code)
- Add binary copy step to `loaf build`

## Verification

- [ ] Deleted hook scripts no longer exist
- [ ] Shared shell libraries deleted
- [ ] `resume-session` and `reference-session` directories deleted
- [ ] `targets.yaml` updated (no references to deleted skills)
- [ ] Remaining hooks functional (no broken library deps)
- [ ] `plugins/loaf/bin/loaf` exists after build
- [ ] `loaf build` succeeds for all targets
- [ ] `npm run typecheck` and `npm run test` pass
