---
type: idea
created: '2026-03-30'
status: captured
---

# Idea: Version-Aware Release Behavior

## Nugget

Make `loaf release` tag/release decisions automatic based on version semantics. Pre-release versions skip tags; stable promotions re-evaluate the target version from accumulated commits.

## What It Would Do

### Conditional tagging (no more `--no-tag`)

The resulting version determines whether to tag:
- `2.0.0-dev.7` (pre-release) → no tag, no GitHub release
- `2.0.0` (stable) → tag `v2.0.0`, GitHub release draft

`--no-tag` stays as an explicit override but the default becomes version-aware. The `/release` skill drops `--no-tag --no-gh` from its invocation entirely.

### Target version validation at promotion

When promoting from pre-release to stable (`--bump release`), analyze all commits since the last stable tag to validate the target:
- `2.0.0-dev.7` with only `fix:` and `feat:` commits → promote to `2.0.0` (correct)
- `2.0.0-dev.7` with `feat!:` or `BREAKING CHANGE:` commits → warn that target should be `3.0.0`, not `2.0.0`

### Pre-release bumps only touch the build counter

`--bump prerelease` increments `dev.N` and never modifies the target version. This is already the behavior — documenting it as the intentional contract.

## Why

Discovered during SPEC-019 merge ritual — the `/release` skill had to explicitly pass `--no-tag --no-gh` because the CLI doesn't distinguish dev bumps from stable releases. The version number already encodes this; the CLI should read it.

The promotion validation catches a real risk: shipping breaking changes across a dev cycle without bumping the major version.

## Scope

- Modify tag/gh steps in `release.ts` to check `isPrerelease` on the new version
- Add promotion validation: `suggestBump` on commits since last stable tag vs target version
- Update `/release` skill to drop explicit `--no-tag --no-gh`
- Add tests for conditional behavior
