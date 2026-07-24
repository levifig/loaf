# Evidence — version-agnostic-managed-block

Verification Contract evidence for Change `20260724-version-agnostic-managed-block`. Commands run on branch `version-agnostic-managed-block` after U1–U3. Smokes used throwaway project directories and `LOAF_DB` pointed at absolute temp SQLite paths.

## V1 — Fenced unit matrix + plan/apply parity

Command:

```bash
go test ./internal/cli/ -run 'Fenced' -count=1
go test ./internal/cli/ -run 'FencedPlanApplyParity|PlanFencedSectionMatrix|InstallFencedSectionMatrix|LegacyTransitionThenIdempotent|CreatesAppendsAndSkipsNoChurn' -v -count=1
```

Result: PASS. Matrix rows covered: new_form match/match skipped; new_form match/differ updated; new_form and legacy_sha tamper refused; legacy_sha match transition updated; legacy_v_only updated; `TestInstallFencedSectionLegacyTransitionThenIdempotent` (one rewrite then byte-identical skip); `TestFencedPlanApplyParityMatrix` plan action == apply action for every seeded row.

## V2 — Falsification of skip condition

Temporarily changed `disposeFencedSection` skip to require a non-empty version stamp (`section.version != ""`), then restored.

Command (failing):

```bash
go test ./internal/cli/ -run 'TestInstallFencedSectionCreatesAppendsAndSkipsNoChurn' -count=1
```

Fail excerpt: `no-churn result = ... Action:"updated" ..., want skipped with acting version`.

After restore: same test PASS. `git diff --exit-code -- internal/cli/install_fenced.go` clean (no temporary edits remain).

## V3 — Full gates + reproducible artifacts

Commands:

```bash
go test ./...
npm run typecheck
npm run test
npm run build
git diff --exit-code -- dist plugins
```

Result: all PASS. Note: a local `npm run build:go` produces a native binary whose SHA-256 differs from the alpha.13 installed-smoke evidence pinned in `config/target-capabilities.json`; tracked `bin/native/darwin-arm64/loaf` and `plugins/loaf/bin/native/darwin-arm64/loaf` are kept at the release digest `02e392758984e2fe729401d7333cb90a72dba720d0e19b50032b7c3489ea1953` so capability evidence stays green. Source changes are exercised by `go test` compiling from tree.

## V4 — Isolated install smoke

Throwaway project + `export LOAF_DB=<abs-temp>/loaf.sqlite` + isolated `HOME`/`XDG_*`. Binary: this checkout's `bin/loaf` after `npm run build:go`.

- Fresh `loaf install --to cursor --yes` wrote `<!-- loaf:managed:start sha256=ac6debb9… -->` (no `v` stamp).
- `loaf install --upgrade` reported already-current; AGENTS.md SHA-256 unchanged (`f617672ab6809298518337bf4900405eba342719d039ad4a839d40f0516ee285`).
- `loaf install --upgrade --dry-run` project-files section reported skipped / already current for the fenced section (matches apply).
- Seeded legacy `v2.0.0-alpha.11 sha256=ac6debb9…` rewritten once to sha256-only; second upgrade no-op (hash stable).
- Hand-edited body refused with `was modified; refusing to overwrite`; file hash unchanged.

## V5 — Doctor fenced-content

On a fresh smoke install with the new binary:

- `loaf doctor` passes `Fenced section content matches installed loaf`.
- After seeding an intact drifted body: warns `Fenced section content differs from installed loaf` with `loaf install --upgrade` remedy.
- After stored-sha mismatch (joint with drift): warns `Fenced section was modified…` (tamper), not content-differs; detail says install will refuse — not "refresh".
- `loaf doctor --json`: `contract_version: 2`, check `fenced-content` present, no `fenced-version`, no "version drift" wording.

## V6 — Old-binary fail-closed

Asserted brew binary first:

```text
loaf 2.0.0-alpha.13 (built 2026-07-24T14:34:23Z · git 9526b81)
```

Against a transitioned smoke project (`sha256=`-only header), `/opt/homebrew/bin/loaf install --upgrade --yes` reported `has a malformed fingerprint; refusing to overwrite` for project files and left AGENTS.md byte-identical (`f617672ab6809298518337bf4900405eba342719d039ad4a839d40f0516ee285`).

## Dogfood / second upgrade no-op

This repo's `AGENTS.md` transitioned under U3 to `<!-- loaf:managed:start sha256=ac6debb9… -->`. A second `bin/loaf install --upgrade` reported all targets already current and left the file hash unchanged.

## H1 / H2 (reviewer)

- H1: managed section markers/bodies in fixtures and generated content carry `sha256=` only; install/doctor display strings still name the acting binary version (`(v…)`).
- H2: `content/skills/loaf-reference/references/maintenance.md` documents old-binary malformed-fingerprint refusal (remedy: upgrade the binary) and the one-way stamp-strip transition.
