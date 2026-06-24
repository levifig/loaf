---
id: TASK-258
title: Align git-workflow commit-format guidance with enforcement
spec: SPEC-047
status: todo
priority: P2
created: '2026-06-24T12:03:41Z'
updated: '2026-06-24T12:03:41Z'
completed_at: null
depends_on:
  - TASK-257
files:
  - content/skills/git-workflow/SKILL.md
  - content/skills/git-workflow/references/commits.md
  - dist/
  - plugins/
  - .agents/tasks/TASK-258-git-workflow-unscoped-commit-format.md
verify: >-
  ! rg -n 'type\(scope\): description' content/skills/git-workflow dist plugins
  && npm run build
done: >-
  git-workflow source and generated copies teach the unscoped Conventional
  Commit format enforced by the native hook.
---

# TASK-258: Align git-workflow commit-format guidance with enforcement

## Description

The git-workflow skill documents scoped Conventional Commit examples that the
native enforcement hook rejects. Update source guidance and generated copies to
teach unscoped `type: description`.

## Acceptance Criteria

- [ ] `content/skills/git-workflow/SKILL.md` teaches unscoped commit messages.
- [ ] `content/skills/git-workflow/references/commits.md` does not teach scoped
  commits as the default accepted format.
- [ ] Generated `dist/` and `plugins/` copies match source after build.
- [ ] A sample commit message matching the documented format passes the native
  validator.

## Verification

```bash
! rg -n 'type\(scope\): description' content/skills/git-workflow dist plugins
npm run build
```
