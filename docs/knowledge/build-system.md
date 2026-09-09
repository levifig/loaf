---
topics:
  - build-system
  - targets
  - distribution
covers:
  - internal/cli/build*.go
  - config/targets.yaml
  - config/hooks.yaml
consumers:
  - implementer
  - reviewer
last_reviewed: '2026-09-04'
---

# Build System

Loaf packages shared skills, agents, and hooks for supported harnesses. [Skill Portability](../architecture/skill-portability.md) owns the shared-authoring model, rationale, and validation boundaries; this guide covers build mechanics.

## Key Rules

- **Shared intermediate.** Skills from `content/` compile to `dist/skills/`, with tracker-native Flow and provider sources from `vnext/content/` overlaid before each target reads the common content.
- **Targets are additive.** Each target gets what it supports — Claude Code gets everything, while Codex gets skills plus its policy template and current-schema SessionStart context hook.
- **Sidecars carry target-specific fields.** SKILL.md has standard fields only. `.claude-code.yaml`, `.opencode.yaml`, etc. carry extensions. Build merges them.
- **Shared templates distribute at build time.** `content/templates/` files (`journal.md`, `grilling.md`) are copied to specified skills via `shared-templates` in `targets.yaml`. The Flow overlay separately projects its shared templates into consuming packages. Architecture templates live in the Architecture skill itself.
- **No skill-prose substitution.** Target builders preserve common bodies and labeled harness sections. The shared intermediate may project template paths; executable integration artifacts may have install-time placeholders. Neither permits per-target rewriting of skill instructions.

## Build Flow

`make build` runs the native `cmd/loafdev` build workflow: compile the runtime, generate the CLI reference, build content, and verify artifacts. `loaf build` dispatches through `internal/cli/build.go`, loads `hooks.yaml` and `targets.yaml`, builds the shared skills intermediate in `dist/skills/`, and calls each target builder to produce `plugins/` or `dist/{target}/`.

## Targets

| Target | Output | Agents | Skills | Hooks | Runtime Plugin |
|--------|--------|:------:|:------:|:-----:|:--------------:|
| claude-code | `plugins/loaf/` | Yes | Yes | Yes | `plugin.json` + `hooks.json` |
| cursor | `dist/cursor/` | Yes | Yes | Yes | No |
| opencode | `dist/opencode/` | Yes | Yes | Yes | Yes (`hooks.ts`) |
| codex | `dist/codex/` | No | Yes | SessionStart context | No |
| amp | `dist/amp/` | Modes | Yes | No | Yes (`.amp/plugins/loaf.ts`, `.amp/plugins/loaf-modes.ts`) |

### Notes

- **Claude Code** ships content with bare PATH `loaf` hook commands, not a bundled binary or runtime-discovery shim. Hooks are registered in `hooks/hooks.json` because `plugin.json` silently drops non-matcher session events.
- **OpenCode and Amp** generate runtime plugins (`hooks.ts` / `.amp/plugins/loaf.ts`) that implement enforcement hooks via subprocess calls to `loaf check`. Amp also copies the authored mode plugin `.amp/plugins/loaf-modes.ts`, which registers `Loaf Medium`, `Loaf Ultra`, and the pinned delegation tools as a separate managed artifact. The orchestrator and oracle agents are GPT-6 Astra (medium/xhigh and high); review stays on GPT-5.6 Luna at max; the implementation agent is Grok 4.6 with Fast and no reasoning-effort pin.
- **Codex** generates a current-schema `.codex/hooks.json` SessionStart matcher group because Codex `0.144.1` rejects Loaf's legacy flat hook projection; generated command and `commandWindows` fields are PATH `loaf journal context --from-hook --codex-hook`. POSIX installs omit the Windows variant; Windows installs keep both fields as that PATH form. Isolated `CODEX_HOME` startup on `darwin-arm64` is model-visible smoke-proven; global-home installation, resume, clear, compact, Windows runtime behavior, and completion remain separately unproven. The separately opted-in basic command policy renders one PATH `loaf` prefix per explicitly classified leaf and does not grant a bare `loaf` namespace, while body/file-consuming leaves and path-taking `change check` remain operator-gated. Other harness adapters are not implied.
- **MCP servers** are not bundled. `loaf install` detects and recommends MCPs at install time; integration state stored in `.agents/loaf.json`.

### Hook Registration (Claude Code)

Claude Code has a split registration model: `plugin.json` handles the plugin manifest (skills, agents, metadata) while `hooks/hooks.json` handles all hook registrations. This split exists because `plugin.json` silently drops session events (SessionStart, PreCompact, PostCompact, TaskCompleted, etc.) that lack a `matcher` field. All hooks — enforcement, instruction, journal, and conversation — are registered in `hooks/hooks.json` for reliable dispatch.

## Fenced Sections

`loaf install` writes fenced Loaf sections into project instruction files (CLAUDE.md, .cursorrules, AGENTS.md, etc.) using markers to identify managed content. This is separate from the build — it runs during installation to inject project-level configuration.

`loaf config check` is the focused health gate for project Loaf configuration. It validates `.agents/loaf.json` and compares installed Loaf-managed target hook registrations against the current distribution. Use `loaf config check --fix` to add missing safe project-config defaults and refresh stale installed target artifacts when a release adds hooks such as `github-account`.

## Dev Tooling

| Script | Command | Purpose |
|--------|---------|---------|
| `cmd/loafdev` | `make build` / `make verify` | Builds the native runtime and generated content, then verifies artifacts. Use `LOAF_DEV_LINK=0` to keep the active runtime unchanged. |
| `cli/scripts/smoke-test.js` | `node cli/scripts/smoke-test.js` | Validates built hook artifacts across supported targets (structure, `if` conditions, `failClosed` flags). Run after build changes. |
| `cli/scripts/eval-skill-routing.mjs` | `node cli/scripts/eval-skill-routing.mjs` | Tests whether Claude routes prompts to correct skills. Requires `ANTHROPIC_API_KEY`. Use `--model` for cheaper runs, `--skill` to test one skill. |

**Smoke test** is a build output integration test for script-level artifact assertions that are not yet worth moving into native Go tests. **Routing eval** is a non-deterministic quality tool for tuning skill descriptions; test cases need updating when skills are added/removed/renamed.

## Cross-References

- [skill-architecture.md](skill-architecture.md) — how skills are structured
- [hook-system.md](hook-system.md) — how hooks are registered and distributed
