---
id: TASK-102
title: 'Track 3: Contract migration'
status: todo
priority: P2
created: '2026-04-10T17:27:28.241Z'
updated: '2026-04-10T17:27:28.241Z'
spec: SPEC-029
---

# TASK-102: Track 3: Contract migration

## Description

Update all documentation and skill files to reflect the new journaling contract. CLAUDE.md: remove "Journal Discipline" manual logging requirement. AGENTS.md: update session journal instructions. Skill files: per Appendix A in SPEC-029, remove ambient-event logging instructions (keep skill self-logs). Wrap skill: forced sync + model review before wrap-up. PreCompact: blocking sync + updated prompt.

**File hints:**
- MODIFY: `.claude/CLAUDE.md` — remove Journal Discipline paragraph
- MODIFY: `.agents/AGENTS.md` — update session journal instructions
- MODIFY: `content/skills/orchestration/SKILL.md` — remove manual logging instructions
- MODIFY: `content/skills/wrap/SKILL.md` — forced sync + model review in Step 1
- MODIFY: `content/skills/implement/SKILL.md` — remove manual logging references
- MODIFY: `content/skills/research/SKILL.md` — remove manual logging references
- MODIFY: `content/skills/bootstrap/SKILL.md` — remove manual logging references
- MODIFY: `config/hooks.yaml` — update PreCompact prompt

## Acceptance Criteria

- [ ] CLAUDE.md no longer requires manual `loaf session log` before every response
- [ ] AGENTS.md reflects automated sync model
- [ ] Skill files updated per Appendix A (ambient logging removed, self-logs kept)
- [ ] Wrap skill Step 1: forced sync (`loaf session sync --final`) + model review
- [ ] PreCompact: blocking forced sync + prompt references journal (not manual flush)
- [ ] `loaf build` succeeds
- [ ] Full test suite passes
- [ ] No regressions in journal quality (equal or better)

## Verification

```bash
npm run typecheck && npm run test && loaf build
```
