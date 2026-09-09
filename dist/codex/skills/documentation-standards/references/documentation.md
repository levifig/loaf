# Documentation Standards

## Contents
- Core Principle
- Document Hierarchy
- Architecture Documentation
- API Documentation
- Project Changelog
- Micro-Changelog
- Critical Rules

Project documentation presentation and API doc rules. Architecture policy belongs to the architecture skill.

## Core Principle

**API documentation follows implementation.** API specifications describe implemented and released behavior. Accepted architecture may precede implementation when the record labels that gap honestly; shared work and delivery status belong in the selected native tracker.

## Document Hierarchy

Discover the repository's canonical documentation paths. The following is an example shape, not a required root layout:

```
docs/
├── VISION.md             # Product purpose and scope
├── STRATEGY.md           # Current bets and learning sought
├── ARCHITECTURE.md       # System map and topic navigation
├── QUICK_REFERENCE.md    # One-page command reference
├── api/                  # API docs (implemented features only)
│   ├── openapi.yaml
│   └── endpoints/
└── architecture/         # Living topics, owned by the architecture skill
    ├── authority-boundaries.md
    └── persistence.md
```

| Document | Purpose | Updates When |
|----------|---------|--------------|
| **VISION.md** | Product purpose and scope | Vision changes |
| **STRATEGY.md** | Bets and learning sought | Strategy changes |
| **Architecture overview and topics** | Current model and rationale | Architecture or supporting evidence changes |
| **API docs** | Implemented endpoints | Features ship |

## Architecture Documentation

The [architecture skill](../../architecture/SKILL.md) owns the convention, topic template, rationale preservation, affected-surface review, and migration. Use its guidance rather than inventing a second architecture format here. This reference owns presentation and documentation quality only.

Keep headings descriptive and links navigable. Do not impose ADR sections, numbering, lifecycle metadata, or a micro-changelog on architecture topics. Architecture owns exceptional ADR selection and decision history as well as legacy-format guidance; follow the repository's actual format for a deliberately selected record.

## API Documentation

**The implemented-only rule:** `Feature Request -> Implementation -> Tests Pass -> Release -> Update API Docs`

Deprecation markers in OpenAPI:
```yaml
paths:
  /api/v1/legacy-endpoint:
    get:
      deprecated: true
      description: |
        **Deprecated since**: v1.5.0
        **Removal planned**: v2.0.0
```

## Project Changelog

`CHANGELOG.md` is release communication for humans. Loaf follows
[Common Changelog](https://common-changelog.org/) as a stricter profile of Keep
a Changelog, with one local workflow allowance: `## [Unreleased]` is a staging
section for PR and release preparation.

### File Shape

- Start with `# Changelog`.
- Keep `## [Unreleased]` at the top for unreleased, curated entries.
- Use release headings shaped as `## [VERSION] - YYYY-MM-DD`.
- Sort release sections semver-latest first.
- Match each released `VERSION` to the git tag, allowing the tag to use a `v`
  prefix.
- Copy the final release section into the GitHub Release body; do not use
  generated GitHub notes as the source of truth.

### Categories

Use Common Changelog categories in this order:

1. `Changed` for changes in existing functionality
2. `Added` for new functionality
3. `Removed` for removed functionality
4. `Fixed` for bug fixes

Avoid `Other`, `Internal`, `Migration`, and catch-all headings in curated
release notes. Express migrations or upgrade requirements as a one-sentence
notice under the release heading, or as a `Changed` / `Removed` entry with a
`**Breaking:**` prefix when the software behavior changed.

### Entries

Each entry is one list item and should:

- Start with an imperative, present-tense verb: `Add`, `Fix`, `Remove`,
  `Document`, `Refactor`, `Bump`.
- Describe the user, operator, or integrator impact without depending on the
  category heading.
- Stay on one line whenever possible.
- Include the best public reference in parentheses when available, such as a
  PR, issue, ADR, release, or commit link.
- Merge related commits into one meaningful release note.
- Sort breaking changes first, then by importance, then latest-first.

Examples:

```markdown
### Changed

- **Breaking:** migrate private continuity across linked worktrees ([Private Continuity](docs/architecture/private-continuity.md))
- Document release guardrails for post-merge tagging ([#50](https://github.com/levifig/loaf/pull/50))

### Added

- Add `loaf task list --status <status>` for lifecycle filtering
```

### Curation

Generated changelog text is a draft. Before release:

- Remove maintenance noise that has no release impact.
- Rewrite commit subjects into human release notes.
- Drop internal spec, task, session, branch, and review-gate identifiers.
- Keep public commands, config keys, documented file names, public ADR IDs, and
  user-visible hook names when they help the reader understand the impact.
- Skip no-op changes that were reverted before the release.

## Micro-Changelog

When the repository explicitly requires document-local changelogs, place them at the **bottom** of the relevant documents. Do not introduce them by default; architecture topics follow the architecture skill's history convention.

```markdown
---

## Changelog

- 2025-11-14 - Added section on agent instructions
- 2025-11-12 - Updated architecture overview
```

**Format:** `- YYYY-MM-DD - Short description`, reverse chronological. Log section additions, significant updates, restructuring, corrections. Skip typos and formatting.

## Critical Rules

**Always:** Follow the repository's established date and changelog conventions; keep docs minimal and move substantial conditional detail into linked references.

**Never:** Document APIs before they ship, include `.agents/` links outside `.agents/` artifacts, use lengthy code samples, add planning details to docs.
