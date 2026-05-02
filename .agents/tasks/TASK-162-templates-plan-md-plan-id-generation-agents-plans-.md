---
id: TASK-162
title: templates/plan.md + plan ID generation + .agents/plans/ lazy creation
status: todo
priority: P2
created: '2026-05-02T01:25:49.154Z'
updated: '2026-05-02T01:25:49.154Z'
spec: SPEC-034
depends_on:
  - TASK-159
---

# TASK-162: templates/plan.md + plan ID generation + .agents/plans/ lazy creation

## Description

Author the minimal PLAN artifact template at `content/skills/refactor-deepen/templates/plan.md`. Document plan ID generation pattern (sequential, matches SPEC numbering) and lazy `.agents/plans/` directory creation. Include an explicit comment in skill prose noting that race conditions on concurrent plan creation are out of scope here — flagged for future work via the plan-lifecycle idea (`20260501-231922`).

## File Hints

- `content/skills/refactor-deepen/templates/plan.md` (new)
- Skill prose in TASK-159's SKILL.md will reference this template

## Acceptance Criteria

- [ ] Template includes minimal sections: candidate, dependency category (per `references/deepening.md`), proposed deepened module, what survives in tests, rejected alternatives
- [ ] Frontmatter schema: `id: PLAN-NNN`, `title`, `created` (ISO 8601 UTC), `status: drafting`, `spec: <SPEC-NNN if related, else null>`, optional `related:`
- [ ] Plan ID generation pattern documented: `ls .agents/plans/PLAN-*.md .agents/plans/archive/PLAN-*.md 2>/dev/null | grep -oE 'PLAN-[0-9]+' | sort -t- -k2 -n | tail -1 | awk -F- '{print $2 + 1}'`
- [ ] Lazy `.agents/plans/` creation rule documented: skill creates directory only on first plan write
- [ ] Comment in skill prose: "Plan ID race conditions on concurrent creation are out of scope; tracked for follow-up via idea 20260501-231922 (plan-lifecycle)."
- [ ] If template file >100 lines, include `## Contents` TOC
- [ ] `loaf build` distributes the template

## Verification

```bash
loaf build
ls plugins/loaf/skills/refactor-deepen/templates/plan.md
grep -q "race condition" content/skills/refactor-deepen/templates/plan.md
```
