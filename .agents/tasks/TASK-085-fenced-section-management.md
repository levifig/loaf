---
id: TASK-085
title: Fenced-section management for CLAUDE.md/AGENTS.md
spec: SPEC-020
status: todo
priority: p2
dependencies: [TASK-075, TASK-084]
track: D
---

# TASK-085: Fenced-section management for CLAUDE.md/AGENTS.md

Install and upgrade Loaf framework conventions into user project instruction files.

## Scope

### Fenced-section format
```markdown
<!-- loaf:managed:start v2.1.0 -->
<!-- Maintained by loaf install/upgrade — do not edit manually -->
## Loaf Framework
[Compact framework essentials: session journal entry types, CLI commands,
 verification conventions, link to framework reference skill for full details]
<!-- loaf:managed:end -->
```

### `loaf install` behavior
- Creates fenced section at end of target file (or creates the file if missing)
- Content is ~20-30 lines, generated at build time from framework reference skill

### `loaf install --upgrade` behavior
- Finds `<!-- loaf:managed:start ... -->` / `<!-- loaf:managed:end -->` markers
- Replaces only content between markers
- User content outside fences preserved
- Version in start marker — skip refresh if already current
- If fences not found (user deleted them), append new fenced section

### Per-target file
| Target | File |
|---|---|
| Claude Code | `.claude/CLAUDE.md` |
| Cursor | `.cursor/rules/loaf.mdc` or `.agents/AGENTS.md` |
| Codex | `.agents/AGENTS.md` |
| OpenCode | `.agents/AGENTS.md` |
| Amp | `.agents/AGENTS.md` |

## Verification

- [ ] `loaf install` creates fenced section in target file
- [ ] `loaf install --upgrade` replaces only between fences
- [ ] User content outside fences untouched
- [ ] Version marker works (skip if current, refresh if outdated)
- [ ] Missing fences = append new section
- [ ] Fenced content is compact (~20-30 lines)
