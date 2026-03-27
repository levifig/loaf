## Before creating this PR, complete these steps:

### 1. CHANGELOG entry

Add an entry under `## [Unreleased]` in `CHANGELOG.md`, categorized by change type:

| Commit type | Category |
|-------------|----------|
| `feat` | Added |
| `fix` | Fixed |
| `refactor`, `perf` | Changed |
| `docs` (user-facing) | Changed |
| `chore`, `ci`, `test` | Skip unless notable |

One line per meaningful change. Include spec reference where applicable.

```markdown
### Added
- /bootstrap skill for 0-to-1 project setup (SPEC-013)
```

If `CHANGELOG.md` does not exist, create it first:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Your entry here
```

### 2. PR title

Conventional commit format, under 70 characters:

```
feat: add thermal rating calculation
fix: prevent divide by zero in sag calculation
```

No scope prefixes. No SPEC/TASK IDs in the title.

### 3. PR body

```markdown
## Summary
- Key changes (2-4 bullets)

## Test plan
- [ ] Tests added/updated
- [ ] Manual testing performed
```

### 4. Merge strategy

Squash merge. Write a clean extended description (2-4 lines summarizing the branch). Never use the auto-generated squash description.

---

Complete these steps, then re-run `gh pr create`.
