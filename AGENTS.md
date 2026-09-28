# Loaf Development Guidelines

Guidelines for maintaining and extending Loaf - An Opinionated Agentic Framework. See [README.md](README.md) for what Loaf is and how to install it.

> **New shared work is tracker-native.** The Loaf Flow is **pitch → shape → implement → ship → release**. `/pitch` discovers the problem. `/shape` creates or updates the canonical native tracker record through the selected `project-management/v1` provider skill, including body, definition of done, out-of-scope, and supported hierarchy. `/implement` reads that record, then uses repository-native Git mechanics. `/ship` is the sole quality gate and writes verified status through the provider after readback. Releases are retroactive. Historical local issue commands are frozen migration compatibility, never an ongoing synchronization path or second work record.

## Quick Start

```bash
make build                     # Binary + CLI reference + all content targets, then verify; does not activate PATH
bin/loaf build                 # Rebuild content with this checkout's CLI (--target <name> for one target)
bin/loaf upgrade --dry-run     # Preview changes to existing installations
make verify-local              # Full local gate: uncached tests, typecheck, vet, CGO-free build, adapter runners, generated content
make ci-check                  # Reproduce default remote spot and integrity checks locally
make ci-smoke                  # Only the bounded remote Go test selection
make test                      # go test -count=1 ./...  (also: make typecheck, make vet)
make build-cli                 # Compile the native executable only (make build-go is an alias)
make capability-tests          # Adapter runner tests; does not launch a live harness session
make vulncheck                 # Network-backed audit; before release or after dependency changes
make install                   # Full build + verification, then activate this checkout through ~/.local/bin/loaf
go run ./cmd/loafdev --help    # The build, release, packaging, and tag tooling behind the Makefile
```

- Use the Go toolchain declared in `go.mod`. There is no npm: `package.json` is only the distribution manifest, and Node 24+ is needed solely for the harness capability runners under `cli/scripts/` and JavaScript adapter syntax checks. npm and `tsc` are not build or install entry points.
- **Never change a user's runtime selection as a side effect of building or verifying.** `make`, `make build`, and `make verify-local` never activate the development launcher; only `make install` activates `bin/loaf` through `~/.local/bin/loaf` and Loaf's existing launcher pointer. `LOAF_DEV_LINK=0` remains a harmless no-op.
- **`loaf install` has no dry-run mode.** `bin/loaf install` changes live harness content, so apply it only as a separate, explicitly scoped action, and use isolated homes in tests.
- The generated command reference is the loaf-reference skill (`content/skills/loaf-reference/SKILL.md`); `loaf check` runs enforcement hooks manually.

### Dev Isolation

Real CLI commands can open or write the production SQLite DB (`XDG_DATA_HOME`, default `~/.local/share/loaf/loaf.sqlite`). Before dogfooding against throwaway state, set `LOAF_DB` to an absolute temporary path; a relative value is ignored:

```bash
export LOAF_DB="$(mktemp -d)/loaf.sqlite"
```

`LOAF_DB` isolates database state, not harness homes: install/upgrade testing also needs isolated target directories. Clear it before the unit suite (`env -u LOAF_DB make verify-local`) so it cannot redirect tests away from their fixture-owned databases; an isolated `XDG_DATA_HOME` is an acceptable fallback. Go unit tests use temporary directories and `t.Setenv`.

## Where Things Live

| Area | Location |
|------|----------|
| CLI entry and commands | `cmd/loaf/main.go` → `internal/cli/` (Runner dispatch in `cli.go`, one `runX` per command, per-target `build_{target}.go`) |
| Build, package, release tooling | `cmd/loafdev/`, `internal/devtool/` |
| SQLite continuity; project identity | `internal/state/`; `internal/project/` |
| Skills; agent profiles | `content/skills/{name}/`; `content/agents/{name}.md` |
| Shared templates | `content/templates/`, distributed via `shared-templates` in `config/targets.yaml` |
| Hooks | Maintained checks in `internal/cli/check*.go`; static guidance in `content/hooks/instructions/`; registry `config/hooks.yaml` |
| Tracker-native Flow overrides | `vnext/content/`, overlaid by the content build |
| Harness capability runners | `cli/scripts/` |
| Design and rationale; knowledge guides | `docs/architecture/`; `docs/knowledge/` |
| Build output (tracked) | `plugins/` (Claude Code), `dist/{target}/` (others); `bin/loaf` is generated and untracked |

## Before You Start

- **Skills, skill references or templates, agent profiles:** read [Skill Architecture](docs/knowledge/skill-architecture.md) first. It owns SKILL.md structure, frontmatter, descriptions, naming, sidecars, reference and template rules, harness-neutral authoring detail, and the skill authoring checklist. Skills are auto-discovered; do not register them in `hooks.yaml`. Profiles state intended tool boundaries; only the host enforces access.
- **Hooks:** read [Hook System](docs/knowledge/hook-system.md). Maintained deterministic behavior goes in the native CLI with regression tests, registered in `config/hooks.yaml` with `skill:` naming its owner. Never add new Bash/Python hook helpers.
- **Build targets or output:** read [Build System](docs/knowledge/build-system.md).
- **Journal or continuity behavior:** read [Private Continuity](docs/architecture/private-continuity.md) and the [journal template](content/templates/journal.md).
- **Architecture:** keep current design, constraints, and rationale in `docs/architecture/` topics. Git retains superseded text, so do not keep obsolete ADRs as a competing description of current design; use a separate decision record only for an exceptional, narrow commitment. The architecture skill owns this policy.

## Project Rules

**Workflow skills log themselves first.** A user-invocable workflow skill's first action is a journal entry with context (arguments, intent, or trigger), e.g. `loaf journal log "skill(implement): native hook rewrite for the selected issue"`. The branch and an opaque `harness_session_id` attach automatically; there is no start step. `/wrap` reads these entries to see whether periodic skills ran.

**The journal is the only session model.** Journal entries are project-scoped SQLite rows correlated by `harness_session_id`. There is no session entity, status, or lifecycle, so never add one or anything that opens, closes, or transitions a session. A `wrap` entry is an optional checkpoint for synthesis worth saving ("tried X, abandoned because Y, next is Z"), not a transition. Continuity context is derived at read time and never persisted, and it does not mirror tracker work. Rendered journal markdown is a projection: write with `loaf journal log`, never by editing markdown.

**Artifact names never cite their work unit.** Name an artifact for what it is (`research/target-capability-survey.md`, not `research/u8-target-capability-survey.md`). An issue points at its artifacts; an artifact never points back, because the containing directory already gives provenance and a work ID in the name goes stale. Record provenance in front matter (`source:`). Versions, timestamps, and numbered records inside their owning directory (`docs/decisions/ADR-007-slug.md`) are identity, not citation. `loaf check --hook artifact-names` enforces this on tracked paths at commit, grandfathering artifacts marked `final` or `archived`.

**Skill bodies are harness-neutral by authoring.** Every target receives the same bytes, with no build-time per-harness rewrite. Describe behavior and let the model choose its tools; put genuinely product-specific facts (literal tokens, spawn APIs, paths, invocation forms) in a labeled `### <Product>` section or a compact harness table, one product per fence. The test: the body reads correctly on every harness without substitution. Detail: [Harness-Neutral Authoring](docs/knowledge/skill-architecture.md#harness-neutral-authoring).

## Before Committing

- [ ] `env -u LOAF_DB make verify-local` passes without switching the user's runtime (Node 24+ must already be available)
- [ ] Before release or after dependency changes, `make vulncheck` passes; report unavailable network access as an unverified check, never a pass
- [ ] If tracked build artifacts in `dist/` or `plugins/` changed, commit them with the source changes that produced them
- [ ] Skill changes pass the [skill authoring checklist](docs/knowledge/skill-architecture.md#authoring-checklist)

Comprehensive verification belongs locally. Default remote CI runs `make ci-check`, a bounded spot selection plus generated-content and CGO-free build integrity; release automation runs delivery-specific spots and constructs/verifies artifacts. Neither is evidence of a full suite pass. Verify required capabilities, payloads, ownership, and rollback behavior locally. Do not require a new live-harness matrix merely because a harness version or build stamp changed. See [Runtime and Delivery](docs/architecture/runtime-and-delivery.md#compatibility-and-verification) for the boundary-specific policy.

## Versions and Releases

- The version lives in `package.json`, and the build injects it into output files. Use semantic versioning.
- Change the version only with explicit approval; keep manifest, generated content, release heading, and tag consistent.
- Curate `CHANGELOG.md` for users: aggregate landed behavior, explain compatibility changes, and cite public references rather than internal work IDs.
- `make release` and `make package` prepare and verify native archives but do not publish.
- Tagging, publishing GitHub releases, and updating the Homebrew tap require explicit authorization.

<!-- loaf:managed:start -->
<!-- Maintained by loaf install/upgrade; edits inside this section are overwritten. Put custom instructions outside it. -->
## Loaf Framework

**Journal Entry Types:**
- `decision(scope)`: Key decisions with rationale
- `discover(scope)`: Something learned
- `block(scope)` / `unblock(scope)`: Blockers and resolutions
- `spark(scope)`: Ideas to promote via `/idea`
- `todo(scope)`: Action items to file in the configured native tracker

**Tracker Authority:**
Shared work identity, definition, definition of done, status, hierarchy, assignment, and collaboration live only in the configured native tracker. Use the selected `project-management/v1` provider skill through a connection already exposed and authenticated by the harness.
Loaf never configures provider authentication, calls a provider API itself, proxies tracker traffic, or keeps an ongoing local-to-tracker mapping. Local-to-tracker synchronization does not exist. A legacy local project may be moved once through an explicit, agentic, verified migration; ongoing work then remains tracker-native.

**Before Implementation:**
Before substantive edits, including ordinary fixes and regression tests, use the configured project-management provider to search for and reuse or create a native issue, then verify its scope, completion criteria, and selected state. Honor an existing explicit implementation request without asking for selection again; issue creation or Backlog alone is not selection. Load the loaf-reference skill's Flow Semantics for the canonical rule and narrow exceptions.
Read-only discovery and purely incidental corrections with no behavior, meaning, requirement, or scope change may proceed first; a one-line bug or security fix is substantive. If the tracker or required readback is unavailable, report the blocker and continue only permitted discovery unless the user explicitly overrides this prerequisite. Never substitute a local shadow tracker or configure authentication.

**CLI Commands:**
- `loaf journal log/recent/search/context` - Project journal
- `loaf check` - Run enforcement hooks
- `loaf kb` - Local project knowledge
- `loaf issue` and its Linear push/pull/reconcile commands are frozen migration compatibility only; never use them as ongoing work authority or synchronization

**Journal Discipline:**
Before completing any response that includes edits, commits, or significant decisions, log journal entries using `loaf journal log "type(scope): description"`. Entry types: `decision`, `discover`, `wrap`. Do not defer journaling - log before responding.
In Codex Auto mode, when the user explicitly installed the managed basic-command policy, use the PATH `loaf` command for classified leaves, including `loaf journal log --execpolicy-safe` for journal writes. Do not substitute an absolute executable pin or a shell/environment wrapper. The policy authorizes only explicitly classified basic Loaf command leaves and does not grant unclassified/operator commands, a bare Loaf namespace, or general filesystem access. Global instructions remain user-owned; no Loaf block in `CODEX_HOME/AGENTS.md` is required. Other harness adapters are not implied.

See the Loaf `orchestration` skill for full details.
<!-- loaf:managed:end -->
