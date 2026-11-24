---
description: >-
  Releases already-landed work through repository-native publishing surfaces and
  owns Loaf's changelog rules. Use when writing or reviewing a changelog entry,
  or when shipped tracker records and Git history are ready for a versioned
  publication. Produces curated release notes and a verified release outcome
  without a local release ledger.
version: 0.6.0-rc.3
---

# Release

Release is retroactive: select already-landed work from Git and live native tracker records, publish through the repository's configured surfaces, then record observed evidence on the same native work.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules

- Read current tags, versions, published releases, repository rules, and candidate native records before proposing a cohort.
- Read the project's release-tag convention from its instructions and apply the [Version Identity policy](https://github.com/levifig/loaf/blob/main/docs/architecture/runtime-and-delivery.md#version-identity): before 1.0, reserve the major transition for 1.0.0, use minor versions for significant or breaking work, and patch versions for other work worth releasing. After 1.0, keep patch, minor, and next-major planning buckets, with distant ones broad and created only when needed. Preserve project-specific tooling constraints.
- When GitHub is selected, look for an open milestone whose title exactly matches the proposed release tag through the selected connection. Show its still-open Issues and ask whether to close that milestone and roll the remaining work forward, naming a destination only when rollover is chosen. Keep publication authorization separate; perform approved closure/reassignment only after publication is confirmed, with native readbacks. A milestone is planned scope, not a second Status or proof that its Issues landed. Other providers keep their existing release path.
- Without a matching milestone, follow the ordinary release journey unchanged. Unplanned intermediary releases are allowed; do not create a milestone merely to release already-landed work. Follow [version planning](../loaf-reference/references/flow-semantics.md#version-planning) for horizon and naming.
- Include only work whose repository landing and ship verdict are observable. Tracker status alone does not prove landing.
- Derive human release notes from the actual landed change and live work definition; do not copy internal execution logs.
- Write and curate changelog entries by [Changelog](references/changelog.md), the single owner of the changelog rules: when an entry is required, its format, curation, and version-heading protection. Other skills point here rather than restating them.
- Describe progress by the larger rideable journey now possible, not by horizontal layers completed. If a release contains foundation work only, label it honestly and name the immediate journey intended to consume it; this is release-note truth, not a new refusal gate.
- Ask for or verify explicit authorization immediately before version changes, tags, pushes, or release publication when governing instructions require it.
- Use repository-native Git and hosting capabilities. Do not create a Loaf-owned release record or provider bridge.
- After publication, read the authoritative tag and release state before claiming success.
- Add a [tracker update](templates/tracker-update.md) or supported native release metadata only after publication is confirmed. Write it as [Record Writing](../project-management/references/record-writing.md) describes: one to three plain lines naming the release and what it means for the reader, with agent detail collapsed. Comments remain evidence, not workflow substitutes.
- Re-read every changed native record and report partial or indeterminate provider outcomes independently from publication success.

## Verification

- The release cohort resolves to exact landed commits and native work references.
- Version, tag, artifact, and release-note state are mutually consistent.
- Required build, package, signing, or publication checks were actually run and read.
- The authoritative hosting surface confirms the published release and artifacts.
- Tracker updates name the observed release identity and were confirmed by readback.
- The released changelog section follows the Changelog rules, matches the tag, and was curated from the landed range rather than copied from commits.
- Release notes state what a real rider can now do, what dogfood proved, and which breadth remains explicitly deferred.
- A matching milestone's open work and closure/rollover choice are explicit; approved post-publication mutations have readbacks. No-match releases gain no new milestone requirement.

## Quick Reference

| Phase | Evidence |
|-------|----------|
| Suggest | Landed commit range plus live native work |
| Prepare | Version and release notes consistent with repository policy |
| Publish | Explicitly authorized native Git/hosting mutation |
| Verify | Authoritative tag, release, and artifact reads |
| Record | Confirmed native tracker update with release identity |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Changelog | [references/changelog.md](references/changelog.md) | Writing, reviewing, or curating a changelog entry, or preparing a release section |

Load repository-specific packaging, signing, and release guidance before publication.
