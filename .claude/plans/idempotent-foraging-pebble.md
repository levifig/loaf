# Plan: Rename Skills to Best Practices Naming Convention

## Summary

Rename 8 skills to domain-focused gerund form per Claude best practices. Keep 3 workflow skills as-is.

## Naming Changes

| Current | New | Change Type |
|---------|-----|-------------|
| foundations | foundations | *keep* |
| orchestration | orchestration | *keep* |
| research | research | *keep* |
| python | python-development | rename |
| typescript | typescript-development | rename |
| ruby | ruby-development | rename |
| go | go-development | rename |
| database | database-design | rename |
| infrastructure | infrastructure-management | rename |
| design | interface-design | rename |
| power-systems | power-systems-modeling | rename |

## Implementation Steps

For each skill being renamed:

1. **Update SKILL.md frontmatter** - Change `name:` field
2. **Update description** - Ensure third-person, starts with action verb
3. **Rename directory** - `src/skills/{old}` → `src/skills/{new}`
4. **Update hooks.yaml** - Update skill references in plugin-groups
5. **Update cross-references** - Any SKILL.md files that reference renamed skills

## Files to Modify

### SKILL.md files (name + description)
- `src/skills/python/SKILL.md`
- `src/skills/typescript/SKILL.md`
- `src/skills/ruby/SKILL.md`
- `src/skills/go/SKILL.md`
- `src/skills/database/SKILL.md`
- `src/skills/infrastructure/SKILL.md`
- `src/skills/design/SKILL.md`
- `src/skills/power-systems/SKILL.md`

### Description updates only (workflow skills)
- `src/skills/foundations/SKILL.md` - "Use for..." → "Establishes..."
- `src/skills/orchestration/SKILL.md` - "Use for..." → "Coordinates..."
- `src/skills/research/SKILL.md` - "Use for..." → "Conducts..."

### Configuration
- `src/config/hooks.yaml` - Update plugin-groups skill references

### Cross-references to update
- `src/skills/power-systems/SKILL.md` line 13 - references foundations
- `src/skills/research/SKILL.md` lines 286-288 - references orchestration, foundations
- Any other skills that cross-reference renamed skills

### Directory renames
```
src/skills/python → src/skills/python-development
src/skills/typescript → src/skills/typescript-development
src/skills/ruby → src/skills/ruby-development
src/skills/go → src/skills/go-development
src/skills/database → src/skills/database-design
src/skills/infrastructure → src/skills/infrastructure-management
src/skills/design → src/skills/interface-design
src/skills/power-systems → src/skills/power-systems-modeling
```

## Description Style Updates

Change from "Use for..." (prescriptive) to third-person action verbs:

| Skill | Current Start | New Start |
|-------|---------------|-----------|
| foundations | "Use for code quality..." | "Establishes code quality..." |
| orchestration | "Use for PM-style..." | "Coordinates multi-agent..." |
| research | "Use for project reflection..." | "Conducts project assessment..." |
| python | "Use for all Python..." | "Covers Python 3.12+..." |
| typescript | "TypeScript and JavaScript..." | "Covers TypeScript 5+..." |
| ruby | "Ruby development..." | "Covers Ruby and Rails 8+..." |
| go | "Use for all Go..." | "Covers idiomatic Go..." |
| database | "Use for database design..." | "Covers schema design..." |
| infrastructure | "Use for DevOps..." | "Covers Docker, Kubernetes..." |
| design | "Use for UI/UX..." | "Covers UI/UX design..." |
| power-systems | "Use for electrical..." | "Covers thermal rating models..." |

## Verification

1. `npm run build` succeeds
2. All plugin-groups in hooks.yaml reference correct skill names
3. No broken cross-references in any SKILL.md
4. Git shows clean directory renames (not delete+add)

## Commit

```
refactor: rename skills to follow best practices naming convention

- Language skills: python-development, typescript-development, etc.
- Domain skills: database-design, infrastructure-management, etc.
- Keep workflow skills: foundations, orchestration, research
- Update all descriptions to third-person action verbs
```
