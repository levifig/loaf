# Runtime and Delivery

## Current Model

Go implements Loaf-owned deterministic operations: command dispatch, project identity, persistent state, migrations, diagnostics, filesystem safety, content builds, and recovery. Skills guide judgment; the runtime does not own shared tracker work or provider credentials. [Authority Boundaries](authority-boundaries.md) defines those responsibilities.

[The Go runtime decision](../decisions/ADR-014-go-for-stateful-runtime.md) preserves the language choice and its TypeScript and Rust alternatives. A single native command gives hooks and users the same implementation. This retains the useful direction of the earlier unified-CLI proposal without adopting its local-task authority, TUI, GUI, or speculative product scope. Fragmented shell and language-specific scripts are not independent product entrypoints; future interfaces require separate evidence and design.

The public runtime, development launcher, build, release, and packaging tools are Go. The [Makefile](../../Makefile) delegates to [`cmd/loafdev`](../../cmd/loafdev/) and [`internal/devtool`](../../internal/devtool/). `package.json` remains distribution metadata, not an npm execution surface. Node remains for harness capability runners and emitted TypeScript checks, not for running or installing Loaf.

SQLite uses the [ncruces `database/sql` driver](../../internal/state/store.go), and [native builds](../../internal/devtool/buildgo.go) set `CGO_ENABLED=0` without requiring a platform SQLite library. The original driver evaluation accepted a larger native binary to avoid cgo cross-compilation friction and the alternative modernc driver's dependency graph and libc-version coupling.

## Native Delivery Direction and Gaps

The agreed destination is native binaries delivered through GitHub Release artifacts, a verified installer, Homebrew, and harness integrations. Users should not need Node or a Go toolchain merely to run Loaf. Installation must preserve artifact provenance and establish one effective upgrade owner, preventing package managers, installers, and plugins from overwriting one another ambiguously.

Native archive/checksum packaging, a curl installer, Homebrew tooling, and Claude plugin onboarding are implemented. Their presence does not prove every platform's live install, signature verification, plugin/runtime compatibility, package-manager ownership, update, rollback, or recovery journey. Native cross-platform builds, signing, publication, installation, and safe updates remain maintenance obligations, not consequences supplied by choosing Go.

The [release workflow](../../.github/workflows/release.yml) builds and uploads release assets and can update a separate Homebrew tap; [native development tooling](../../internal/devtool/) creates archives and checksums. Native binaries are no longer tracked; generated content is. The Claude plugin ships content and bare PATH `loaf` hook invocations, with no binary bundle, runtime-discovery shim, `LOAF_BIN` override, or fixed-location fallback.

Harness invocations must use `loaf` from user-managed PATH. The user chooses and upgrades the runtime; content installation must not pin, deploy, or silently select a private binary. Missing or incapable runtimes should produce actionable install/upgrade guidance. Runtime compatibility must be checked against required capabilities rather than pressuring users to upgrade an older but capable installation.

Continuing npm delivery was rejected because it kept Node as the product entrypoint after the native runtime migration. Building from source during installation would avoid publishing a platform matrix but require a Go toolchain on every user machine and weaken repeatable installation. Neither replaces native delivery.

## Compatibility and Verification

The agreed delivery model is Plugin + Skills + CLI: plugins provide native packaging where available, shared skills carry methods, and the PATH CLI implements deterministic behavior. Keep harness-specific code limited to boundaries that genuinely differ, such as discovery, hook payloads, and permission configuration. These boundaries need verification; shared behavior does not need recertification across every harness version.

Amp keeps two digest-bound plugin artifacts, `loaf.ts` and `loaf-modes.ts`, so install and upgrade continue to own the same destinations. Managed native-mode routing lives in `loaf.ts`. The `loaf-modes.ts` filename remains an inert compatibility plugin with no mode, agent, tool, or hook registrations.

Compatibility depends on required behavior, not exact version identity. Check the commands, flags, payload contracts, and safety capabilities an operation needs. An unfamiliar version alone must not block use or trigger an upgrade demand. Missing required behavior or an unproven safety boundary still requires a refusal or an explicit limitation; removing version gates does not establish compatibility by assumption.

Use deterministic regression and package tests by default. Scope live smokes to materially changed adapters or protocols, newly supported boundaries, or suspected regressions. A routine harness update, unrelated CLI change, or new build stamp does not by itself require a full live-harness matrix. Versions and binary hashes remain useful historical provenance, not an evergreen support whitelist. This does not relax archive integrity, signature, ownership-digest, or rollback checks.

Verification is local first. Before review and tagging, `make verify-local` runs uncached full Go tests, compile and vet checks, a CGO-free build, every deterministic adapter runner test, and a complete generated-content build with TypeScript validation and durable render checks. Node and `tsc` are existing prerequisites; the gate fails if `tsc` is unavailable. Ordinary builds do not activate the development launcher. Review regenerated content and include it with the source that produced it; `make ci-check` additionally requires generated files to match the Git index, as they must in a clean CI checkout. Tests own their database, harness home, and executable fixtures and must work before any build or installation; trusted executable fixtures must respect production path checks, including the rejection of disposable locations. Run tests in a writable checkout outside forbidden OS temporary roots: policy fixtures create and clean up a trusted runtime directory inside that checkout. `make vulncheck` retains the network-backed vulnerability audit locally for release preparation and dependency changes. A network failure leaves that audit unverified.

The exact remote selection lives in the [Makefile](../../Makefile), shared with local reproduction:

| Command | Evidence and boundary |
|---------|-----------------------|
| `make ci-smoke` | Native public command dispatch, SQLite backup/recovery metadata, scoped Codex policy convergence/rollback/late edits, clean-machine fixture isolation and rejected disposable executables, release tag classification, archive/checksum contents, Homebrew checksum completeness, and plugin/runtime artifact separation |
| `make ci-check` | The smoke selection, CGO-free compilation, all generated content and emitted JavaScript/TypeScript validation, durable render integrity, and tracked distribution/CLI reference drift |
| `make release-smoke` | Only tag, packaging/checksum, formula, and artifact-separation spots; release automation checks the package/tag version, builds every platform, verifies archive checksums, and publishes |

The selected tests use exact names, uncached results, and a two-minute timeout per Go package. A measured local `make ci-smoke` run took about 12 seconds with a warm compilation cache; that is evidence for the selection, not a promise for cold runners. Default remote CI has a ten-minute total limit including runner setup and compilation. Full Go suites, static analysis, adapter runner suites, and vulnerability auditing remain local responsibilities; a green remote job does not certify them. Release construction and publication are separate from test coverage and retain their version, target, checksum, and packaging safeguards. No workflow repairs reviewed source or activates a developer's PATH runtime.

[Scoped PATH preflight](../../internal/cli/upgrade_path_loaf.go) checks required command capabilities. Optional [smoke runners](../../cli/scripts/capability-runner-utils.mjs) require `--client` and `--receipt`, not `--expected-version`; they record the observed version or `unknown` without using it to qualify compatibility. Their invocation, isolation, cleanup, and proof requirements remain independent of version discovery. The [passive evidence validator](../../internal/cli/target_capability_contract.go) checks retained receipt structure, historical identity, artifact-path and digest format, and boundary-specific observations without reading or matching today's build artifacts. A record must still agree with its cited receipt's version to prevent historical mislabeling; neither is consulted by install or upgrade as a runtime allowlist. No replacement registry is introduced.

## Reviewed Source and Release Evidence

The [verify-only CI decision](../decisions/ADR-012-ci-verify-only-build-artifacts.md) preserves the reason CI must not repair or push changes to reviewed source branches. Tracked generated content stays with the source that produced it until a replacement delivery path is proven. This does not require tracking release binaries: automation may build, sign, and publish artifacts from reviewed source without modifying that source branch. A separate tap update is distribution metadata, not a repair commit to Loaf's reviewed source.

Verification evidence must identify what it vouches for. A receipt remains a historical observation of those bytes; it is not proof of an untested replacement. Assess changes to the behavior or boundary under test before requiring new compatibility evidence, rather than treating every version or binary digest change as a reason for full recertification. Branch ancestry alone is insufficient: merges and rebases can preserve or change content independently of topology. Retired cohort receipts and their exact digest schema do not become a universal evidence database or another release authority. Current build, review, and ship checks own executable freshness rules.

## Version Identity

Loaf uses major-zero Semantic Versioning (`0.y.z`) until it deliberately declares a stable `1.0.0` public contract. A release identifies a coherent set of already-landed work; it does not create or advance a planning cohort. Minor-versus-patch judgment during major zero and the `1.0.0` readiness bar remain release decisions, with clear compatibility notes needed for users.

Arc-boundary releases were rejected because they exposed retired planning machinery. Build provenance is distinct from the product version: `go build -buildvcs=true` embeds the source revision and dirty state, and the [native entrypoint](../../cmd/loaf/main.go) reads that stamp rather than trusting a mutable sidecar. Development versions may display `+g<short-sha>.dirty`; release metadata and tracked generated content retain the [package version](../../package.json). Missing or malformed provenance is reported as absent rather than invented. Executable timestamps and sidecar identities were rejected because copying a binary or moving HEAD could mislabel its source.

Development builds stage the requested binaries before publication. Ordinary `make`/`make build` work never retargets Loaf's user-local launcher pointer, even if `LOAF_DEV_LINK=1` is inherited; `LOAF_DEV_LINK=0` remains a harmless leftover. Explicit `make install` activates this checkout through `$XDG_DATA_HOME/loaf/current-dev-launcher` and creates `~/.local/bin/loaf` only when that name is absent. It never replaces an unrelated PATH entry, Homebrew install, or other symlink, and it fails rather than reporting a successful install if activation is unsupported or conflicted. Rebuilding an already-linked checkout still replaces `bin/loaf` in place. That developer facility is separate from harness content installation.

## Implementation References

- [Public command entrypoint and tests](../../cmd/loaf/) and [Go command implementation](../../internal/cli/) carry the native runtime.
- [Build verification workflow](../../.github/workflows/build.yml) checks tracked generated drift; [Skill Portability](skill-portability.md) owns common content and target adaptation.
- [Release tooling](../../internal/devtool/) owns archive construction, tag classification, provenance injection, and artifact checks.
- [Tracker-native release method](../../vnext/content/skills/release/SKILL.md) owns release operation; the [strategy](../STRATEGY.md) names unresolved delivery judgments without making them shipped behavior.
