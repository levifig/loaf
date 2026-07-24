---
change: version-agnostic-managed-block
created: 2026-07-24
branch: version-agnostic-managed-block
---

# Version-Agnostic Managed Sections

## Problem

The managed AGENTS.md start marker embeds the installer's version — `<!-- loaf:managed:start v2.0.0-alpha.13 sha256=… -->` — and the up-to-date check in `internal/cli/install_fenced.go:74` requires both the version and the body fingerprint to match before skipping a rewrite. The body is a compiled-in constant that rarely changes, so every release forces a marker-only rewrite in every consumer repo: commit `04377171` in this repo changes exactly one line, `v2.0.0-alpha.11` → `v2.0.0-alpha.13`, with an identical sha256 (`ac6debb9…` — the body has not changed since alpha.11). `loaf doctor`'s `fenced-version` check amplifies the churn by warning on pure version-string drift and telling users to run `loaf install --upgrade`, which produces another content-free diff. Multiply by every Loaf-managed repo and every release, and the framework taxes its users with perpetual no-op chore commits.

## Hypothesis

Making the sha256 body fingerprint the sole identity of the managed section — marker `<!-- loaf:managed:start sha256=… -->`, rewrites decided purely by content mismatch — eliminates per-release churn without weakening any protection: the tamper guard (refuse to overwrite a hand-modified section) is already fingerprint-based, and older binaries encountering the new marker fail closed (their parser treats a version-less header as malformed and refuses to write), which is strictly better than today's silent backward rewrite — the exact regression class we hit when a brew alpha.12 binary reverted #129 content, version stamp notwithstanding.

## Scope

**In**

- Marker format: `generateFencedContent` emits `<!-- loaf:managed:start sha256=<hex> -->`; the version parameter disappears from the generated content (the body is a pure constant).
- Parser: `parseFencedStartHeader` accepts three forms — new `sha256=` only, legacy `v<version> sha256=<hex>`, legacy `v<version>` alone — and the up-to-date decision uses fingerprint equality plus new-form headers only, per the decision matrix below.
- One-way transition: a legacy-stamped section with matching content is rewritten once (the marker line loses its stamp), after which re-runs are no-ops until the body genuinely changes.
- Doctor: `fenced-version` is replaced by a content check (`fenced-content`) that compares the stored body fingerprint against the body this binary generates; version drift as a concept disappears from doctor output.
- Docs: `content/skills/loaf-reference/references/maintenance.md` stops describing "fenced-version drift"; rebuilt `dist/` and `plugins/` renders committed alongside.
- Regression-test observable swap: `cmd/loaf/installed_distribution_test.go:67-72` asserts the installed-vs-stale property via version-stamp presence in the fenced section; that observable ceases to exist and the test's anti-stale property must be re-expressed through the surfaces that still carry version identity (`loaf version` output, `.loaf-version` markers, installed dist content).
- This repo's own AGENTS.md transition (the last stamp-only diff it will ever see), committed with the implementation.

**Out** (deferred, not rejected)

- Converging `.loaf-version` markers for installed target content onto a fingerprint scheme — those markers genuinely carry version identity for content that varies by release; if churn ever shows up there, that is its own Change (spark filed via Open Questions).
- Cleaning up the unused `upgrade bool` parameter threading through `installFencedSection` — pre-existing, harmless, not this Change's business.

**Cut** (explicitly rejected)

- Option 1 from the shaping discussion: keeping the version in the marker but ignoring it in the up-to-date check. A permanently stale stamp is worse than no stamp — it asserts something false.
- Re-introducing version identity anywhere else in the managed section (warning comment, body text). The installed binary is the version authority (per `20260718-installed-distribution-authority`); the section carries content, not provenance.

## Observable Workflow

```
$ brew upgrade loaf                      # release N+1 ships, managed body unchanged
$ loaf install --upgrade
  ○ agents Loaf framework section already current (v2.0.0-alpha.14)
$ git status --short                     # AGENTS.md untouched — no chore commit
```

The first upgrade after this ships performs one final rewrite per repo — the marker line drops its stamp — and from then on the section changes only when a release actually changes the body. `loaf doctor` reports `Fenced section content matches installed loaf` (or warns that content drifted, with `loaf install --upgrade` as the remedy); it never again warns about a version string. An older binary pointed at a transitioned repo refuses with `managed Loaf section … has a malformed fingerprint; refusing to overwrite` instead of silently rewriting content backward.

## Rabbit Holes and No-Gos

- **No-go: touching the Codex rules blocks.** `install_codex_rules.go`'s `loaf:managed:codex-*` markers are already version-agnostic under a sha-manifest ownership model — they are the precedent, not a work item.
- **No-go: weakening the tamper guard.** A stored fingerprint that mismatches the actual body still refuses the overwrite, in every header form. No force flag is added.
- **Rabbit hole: a general marker-migration framework.** This is one dual-read, single-write transition; resist abstracting it.
- **Rabbit hole: doctor check taxonomy redesign.** Exactly one check repoints; the doctor surface otherwise stays put.
- **No-go: shimming compatibility for old binaries writing the new form.** Fail-closed refusal is the designed behavior; the remedy is upgrading the binary, documented in maintenance guidance.

## Decisions

Provenance: shaped in-session from the marker-churn discussion (2026-07-24); options presented with evidence, user selected option 2; interplay decisions resolved by code reconnaissance during shaping.

1. **The fingerprint is the section's only identity; the marker carries `sha256=` alone.** Forecloses per-release churn structurally — a rewrite can only ever be caused by a content change or a tamper refusal.
2. **The parser reads three header forms; the writer emits one.** New `sha256=`-only, legacy `v… sha256=…`, legacy `v…`-only all parse; only the new form is ever written. The transition is one-way with no downgrade path in this binary.
3. **Skip requires the new header form, not just fingerprint equality.** A legacy-stamped section with current content is rewritten once to strip the stamp rather than skipped; skipping would preserve a permanently stale version string (Cut item 1 by the back door). This is the single transition diff each repo pays.
4. **Legacy version-only headers (no sha) always rewrite.** Without a stored fingerprint the section can't be verified; current behavior is preserved exactly (they bypass the tamper guard and the skip check today too).
5. **Old-binary refusal is accepted and documented, not worked around.** A pre-change binary parses the new header as malformed and errors out rather than writing. That converts the silent-downgrade failure mode (the alpha.12/#129 regression) into a loud, safe one.
6. **Doctor's check becomes `fenced-content` and needs no version provenance.** It compares the file's body fingerprint against the body constant compiled into the running binary. Decision 3 of `20260718-installed-distribution-authority` (skip fenced comparison when provenance is unresolvable) becomes moot for this check — content comparison works regardless; the no-AGENTS.md and no-section skip/warn paths stay. The rename is honest labeling; no machine consumers of the old check name exist beyond loaf's own tests and docs.
7. **Install output keeps naming the acting binary's version.** `fencedInstallResult.Version` and the `(v…)` display strings stay — the version just isn't persisted into the file.

## Planning Contract

### Header decision matrix (normative)

| Stored header | Body vs stored sha | Body vs generated | Action |
|---|---|---|---|
| `sha256=` (new) | match | match | skipped |
| `sha256=` (new) | match | differ | updated (real content change) |
| any form with sha | differ | — | error: modified section, refuse |
| `v… sha256=…` (legacy) | match | match or differ | updated (transition strips stamp) |
| `v…` only (legacy, no sha) | unverifiable | — | updated (current behavior) |
| malformed / absent-with-section | — | — | error / append-or-create paths unchanged |

### Approach

All semantics live in `internal/cli/install_fenced.go`. `parseFencedStartHeader` gains the `sha256=`-first form (field count 1–2 where the first field may be either `v…` or `sha256=…`); `fencedSectionRange.version` is retained solely to detect legacy forms. The skip condition at `install_fenced.go:74` becomes: header is new-form ∧ stored fingerprint matches existing body ∧ matches generated body. `generateFencedContent` loses its `version` parameter (body is constant); `installFencedSection` keeps `version` only to populate the display result. Doctor's `checkFencedVersion` is replaced by `checkFencedContent` in `internal/cli/doctor.go` (`getDoctorFencedVersion` goes away or becomes a fingerprint reader); `cliVersion` gating drops out of the check.

### Interplay with installed-distribution-authority

`cmd/loaf/installed_distribution_test.go` currently proves "the installed binary, not the cwd checkout, stamps the fenced section" by asserting the version string. Since fenced bodies are version-invariant constants, the stamp was always a proxy; the property must be re-anchored to observables that actually differ between the stale checkout and the installed distribution in that fixture — `loaf version` output and installed dist content markers, which the test already exercises. The fenced-section assertions are removed or replaced by a new-form-marker assertion (section present, `sha256=`-only header, no `v` stamp).

### Sequencing

U1 is the semantic core and carries nearly all review risk; U2 depends on U1's parser helpers; U3 is mechanical fallout plus the dogfood transition. Single PR, three commit boundaries.

## Implementation Units

- **U1 — Fingerprint-only marker semantics.** `install_fenced.go` parser/emitter/skip-condition rewrite implementing the decision matrix, with unit tests covering every matrix row plus idempotence (second run byte-identical file) and the tamper refusal in both header forms. Touches `install_fenced_test.go`, `install_command_test.go`, `install_symlink_test.go` fixtures.
- **U2 — Doctor repoint and guidance.** `fenced-content` check replacing `fenced-version` in `doctor.go` + `doctor_test.go`; `maintenance.md` reference updated (diagnose step wording); `npm run build` regenerating `dist/` and `plugins/` renders.
- **U3 — Regression-observable swap and dogfood transition.** `installed_distribution_test.go` re-anchored per Planning Contract; this repo's `AGENTS.md` transitioned by running the built binary's `loaf install --upgrade` (the final stamp-strip diff), committed together with a `CHANGELOG.md` `[Unreleased]` entry.

## Verification Contract

- **V1.** `go test ./internal/cli/ -run 'Fenced'` passes with new cases named for each decision-matrix row; the transition case asserts exactly one rewrite (legacy → new form) followed by a skipped second run with a byte-identical file.
- **V2.** Falsification: reverting the skip condition to require a version match makes the no-churn test fail; restoring it passes. No temporary edits remain in the final diff.
- **V3.** `go test ./...` passes and `npm run build` succeeds with reproducible tracked artifacts (`git diff --exit-code -- dist plugins`).
- **V4.** Isolated smoke (temp project, `LOAF_DB` set): fresh `loaf install` writes a `sha256=`-only marker; `loaf install --upgrade` reports already-current and leaves the file hash unchanged; a seeded legacy `v2.0.0-alpha.11 sha256=…` file is rewritten once to the new form and is a no-op thereafter; a hand-edited body is refused.
- **V5.** `loaf doctor` on the smoke project passes `fenced-content` after install, warns after a simulated content drift (old body seeded), and no doctor output mentions fenced version drift.
- **H1.** Reviewer confirms no version string persists anywhere in the managed section (marker or body) in any fixture, golden, or generated artifact, while install/doctor display strings still name the acting binary's version.
- **H2.** Reviewer confirms the old-binary refusal behavior is documented in `maintenance.md` guidance (remedy: upgrade the binary) and that the transition path is one-way.

## Definition of Done

- All Verification Contract items pass with evidence (commands + output) recorded in the PR.
- Dogfood complete: this repo's `AGENTS.md` carries the new-form marker via the built binary, and a re-run of `loaf install --upgrade` is a no-op.
- `CHANGELOG.md` `[Unreleased]` describes the behavior change from the upgrading user's perspective, including the one-time transition rewrite and the old-binary refusal.

## Durable Outputs

- Updated `maintenance.md` reference (ships in-change — it is distributable content, not post-hoc documentation).
- No ADR: the decision record lives in this Change; the code change is small and self-describing.

## Open Questions

- [KU] Should `.loaf-version` markers for installed target content ever converge on fingerprint-only identity? → follow-up spark, out of scope here; only worth revisiting if version-only churn is observed on those surfaces.

## Source Inputs

- Conversation, 2026-07-24: marker-churn question ("why are we changing the managed AGENTS.md block with every upgrade?"), option 2 selected; evidence commit `04377171` (stamp-only diff, sha unchanged since alpha.11).
- Journal: `skill(shape)` entry logged 2026-07-24 for this shaping.
- Prior Change `20260718-installed-distribution-authority` — establishes the installed binary as version authority; its Decision 3 and regression test intersect this Change (see Planning Contract).
- Precedent: `install_codex_rules.go` Codex managed blocks — already version-agnostic, sha-manifest owned.
- Live regression context: brew alpha.12 binary rewriting #129 content backward (version stamp did not prevent it), this week's session history.

## Critique Gate

- **Bounded?** Yes — three files of production code (`install_fenced.go`, `doctor.go`, one test file in `cmd/loaf`), their tests, one docs reference, regenerated artifacts.
- **New ceremony?** None — no new commands, flags, or states; one doctor check renamed.
- **Status creep?** None — no frontmatter, no lifecycle; the marker loses metadata rather than gaining it.
- **CLI/skill boundary?** Untouched — behavior change is entirely CLI-internal; skill guidance only updates descriptive text.
- **Smaller and still true?** Option 1 (ignore version in check) is smaller but fails the Hypothesis — it leaves a lying stamp. This is the minimal honest version.
