---
id: TASK-163
title: Sentinel vocabulary test (Track B go/no-go)
status: done
priority: P1
created: '2026-05-02T01:25:49.208Z'
updated: '2026-05-02T01:25:49.208Z'
completed_at: '2026-05-02T03:35:00.000Z'
spec: SPEC-034
depends_on:
  - TASK-159
  - TASK-160
  - TASK-161
  - TASK-162
---

# TASK-163: Sentinel vocabulary test (Track B go/no-go)

## Description

**Track B go/no-go gate.** Manual end-to-end validation of vocabulary discipline. Pick one demonstrably shallow Loaf module (suggested candidate: `cli/lib/install/` — multiple small files passing through to file-system primitives). Invoke `/refactor-deepen` on it. Manually grade the output: count occurrences of correct terms (Module, Interface, Seam, Adapter, Depth, Leverage, Locality) vs drifted terms (boundary, service, component, layer, API, signature). **Pass:** zero drifted-term occurrences in any candidate description. **Fail:** iterate `references/language.md` and SKILL.md Critical Rules in TASK-159 until pass. Track B is not done until this test passes.

## File Hints

- `.agents/plans/PLAN-NNN-*.md` (the artifact produced by the test invocation — commit it)
- `.agents/sessions/<file>` (journal capturing the test result)
- Iteration target: `content/skills/refactor-deepen/SKILL.md` Critical Rules

## Acceptance Criteria

- [ ] One full `/refactor-deepen` invocation completed against a real Loaf module
- [ ] PLAN file produced and committed at `.agents/plans/PLAN-NNN-*.md`
- [ ] Manual vocabulary grading of the PLAN file + skill output: count drifted-term occurrences
- [ ] **Pass condition:** zero drifted-term occurrences (no "boundary/service/component/layer/API/signature" where source taxonomy has a precise term)
- [ ] If grading fails, TASK-159 SKILL.md Critical Rules iterated; this task re-run until pass
- [ ] Journal entry: `validate(refactor-deepen): vocabulary sentinel <pass|fail> for PLAN-NNN`
- [ ] On final pass: `validate(track-b): vocabulary discipline holds`

## Verification

```bash
# Manual: invoke /refactor-deepen on cli/lib/install/, review output for drifted terms
grep -ciE "\b(boundary|service|component|layer|API|signature)\b" .agents/plans/PLAN-NNN-*.md   # should be 0 in deepening contexts
loaf session log "validate(refactor-deepen): sentinel <pass|fail> for PLAN-NNN"
```
