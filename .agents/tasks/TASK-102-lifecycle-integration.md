---
id: TASK-102
spec: SPEC-029
title: Lifecycle integration (wrap + housekeeping)
priority: P2
status: pending
blocked_by: [TASK-100]
---

# TASK-102: Lifecycle integration (wrap + housekeeping)

## Objective

Integrate `loaf session enrich` into the wrap and housekeeping lifecycle points so enrichment happens automatically at natural session boundaries.

## Acceptance Criteria

- [ ] Wrap skill calls `loaf session enrich` before generating wrap-up report
- [ ] Housekeeping skill calls `loaf session enrich` on active sessions with `claude_session_id`
- [ ] Both handle enrich failures gracefully (warn, don't block the primary workflow)
- [ ] Manual `loaf session enrich` still works independently

## Implementation Notes

- **Wrap skill** (`content/skills/wrap/SKILL.md`): Add a step before wrap-up generation: "Run `loaf session enrich` to fill in any missing journal entries from the conversation log."
- **Housekeeping skill** (`content/skills/housekeeping/SKILL.md`): Add an enrichment pass: "For each active session with a `claude_session_id`, run `loaf session enrich <file>` to catch up on unenriched entries."
- Both should treat enrich failure as non-fatal — log a warning and continue
- No hook changes needed — purely skill instruction updates

## Dependencies

- TASK-100 (enrich command must exist first)
