# Plan: Add ID-Free Naming Conventions

## Summary

Add conventions to avoid using SPEC/TASK IDs in commit messages and session names. IDs belong in frontmatter/footers, not in human-readable names.

## Changes

### 1. Update `src/skills/foundations/references/commits.md`

Add to the "Rules to Follow" section:

```markdown
### ID References

- **Never put SPEC or TASK IDs in commit subject** - Use human-readable names
  - Bad: `feat: implement SPEC-002 invisible sessions`
  - Good: `feat: implement invisible sessions and task board`
- **Linear issue IDs go in footer only** (e.g., `Closes BACK-123`)
- Commit message should be understandable without looking up IDs
```

### 2. Update `src/skills/orchestration/references/sessions.md`

Update the naming examples in the "Session Filename Format" section:

```markdown
### Naming Guidelines

- **Don't include IDs in slugs** - Use descriptive names
  - Bad: `20260124-143000-orchestration-spec-002.md`
  - Good: `20260124-143000-invisible-sessions-task-board.md`
- **Don't prefix with session type** - Type is in frontmatter
  - Bad: `20260124-143000-orchestration-auth-feature.md`
  - Good: `20260124-143000-auth-feature.md`
- IDs (TASK-XXX, SPEC-XXX) belong in frontmatter fields, not filenames
```

### 3. Update `src/commands/orchestrate.md`

Change the session filename format from:
```
YYYYMMDD-HHMMSS-orchestration-{identifier}.md
```

To:
```
YYYYMMDD-HHMMSS-{spec-or-task-description}.md
```

Where the description comes from the spec/task title, not the ID.

## Files to Modify

| File | Change |
|------|--------|
| `src/skills/foundations/references/commits.md` | Add ID reference rules |
| `src/skills/orchestration/references/sessions.md` | Add naming guidelines |
| `src/commands/orchestrate.md` | Update session filename format |

## Verification

1. `npm run build` succeeds
2. Review output in `plugins/loaf/` for updated docs
3. Manual review: conventions are clear and actionable
