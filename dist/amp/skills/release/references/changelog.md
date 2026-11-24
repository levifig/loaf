# Changelog

## Contents
- When an Entry Is Required
- File Shape
- Categories
- Entries
- Curation
- Drafted Notes
- Version Protection
- Micro-Changelog

These are the only changelog rules Loaf ships. Implement, ship, and other skills point here instead of restating them. Follow the repository's own changelog convention where it differs; these rules are Loaf's defaults.

`CHANGELOG.md` is release communication for humans. Loaf follows [Common Changelog](https://common-changelog.org/), a stricter profile of Keep a Changelog, with one local allowance: `## [Unreleased]` is a staging section for PR and release preparation.

## When an Entry Is Required

- Every implemented canonical work record gets a new or substantively updated Unreleased entry, whatever its size or category: fixes, tests, docs, maintenance, and features. Link each record using the repository's reference convention. One accurate entry may cover several records; refining an existing entry is better than adding a duplicate.
- Discussion, inspection, and unimplemented ideas get no entry.
- An unchanged, already released, unrelated, or formatting-only entry does not cover new implementation.
- Declare the complete implemented work set through the repository's explicit review input, and run its coverage gate against the actual review base and candidate. Deterministic parsing cannot attest canonical identity or completeness; the main agent verifies both through the tracker.
- A release-only change is the one exception: it must prove it only moves already-landed entries and adds no new implementation. A release title, tag, or skip label alone is not proof.

## File Shape

- Start with `# Changelog`.
- Keep `## [Unreleased]` at the top for unreleased, curated entries.
- Use release headings shaped as `## [VERSION] - YYYY-MM-DD`.
- Sort release sections semver-latest first.
- Match each released `VERSION` to the git tag, allowing the tag a `v` prefix.
- Copy the final release section into the hosted release body; do not use generated hosting notes as the source of truth.

## Categories

Use Common Changelog categories in this order:

1. `Changed` for changes in existing functionality
2. `Added` for new functionality
3. `Removed` for removed functionality
4. `Fixed` for bug fixes

Avoid `Other`, `Internal`, `Migration`, and catch-all headings. Express migrations or upgrade requirements as a one-sentence notice under the release heading, or as a `Changed` or `Removed` entry with a `**Breaking:**` prefix when behavior changed. List breaking entries before non-breaking entries in the same category.

## Entries

Each entry is one list item that:

- Starts with an imperative, present-tense verb: `Add`, `Fix`, `Remove`, `Document`, `Refactor`, `Bump`.
- Describes the user, operator, or integrator impact without depending on the category heading.
- Stays on one line whenever possible.
- Links the best public reference in parentheses when available: a PR, issue, release, ADR, or commit.
- Uses backticks for code references such as commands, config keys, file names, and hook names.
- Merges related commits into one meaningful note.
- Is sorted breaking first, then by importance, then latest first.

Keep what the reader works with: commands, behaviors, config keys, documented file paths, user-visible hook names, compatibility and migration notes, and public IDs such as ADR numbers.

Drop what has no meaning outside the team's working context:

- Internal work-unit numbering that is not the issue ID, and session, sprint, or branch references
- Internal skill or document terminology outside the user's mental model, such as gate-question numbers or hook IDs that are not user-facing
- "How the work got done" framing: interview, decomposition, and review-gate steps
- File lists and implementation details with no user benefit
- Verbatim commit or PR-title dumps

```markdown
### Changed

- **Breaking:** migrate private continuity across linked worktrees ([Private Continuity](https://github.com/levifig/loaf/blob/main/docs/architecture/private-continuity.md))
- Document release guardrails for post-merge tagging ([#50](https://github.com/levifig/loaf/pull/50))

### Added

- Add `loaf release suggest` and `loaf release cut` for retroactive releases
```

## Curation

Curate `[Unreleased]` as PRs land, so the release reads as user-facing prose rather than a worklog. Before release:

- Compare `[Unreleased]` against the actual release range.
- Remove maintenance noise with no release impact, and changes reverted before the release.
- Rewrite commit subjects into human release notes.
- Remove scaffolding language from specs, reviews, tasks, or triage. An entry that only explains why the work was discovered or how it was organized does not belong.

## Drafted Notes

`loaf release suggest` drafts notes from landed issues, and `loaf release cut` prepends them into `CHANGELOG.md`. Treat drafted text as a draft: rewrite internal terms before cutting.

## Version Protection

Never add or change a version heading (`## [X.Y.Z]`) without explicit user approval. A new version heading needs the user's confirmation, and it must stay consistent with the manifest version and the tag.

## Micro-Changelog

When a repository explicitly requires document-local changelogs, place them at the bottom of the relevant document. Do not introduce them by default; architecture topics keep their history in Git.

```markdown
---

## Changelog

- 2025-11-14 - Added section on agent instructions
- 2025-11-12 - Updated architecture overview
```

Use `- YYYY-MM-DD - Short description`, reverse chronological. Log section additions, significant updates, restructuring, and corrections; skip typos and formatting. `loaf check changelog <file.md>` validates this schema.
