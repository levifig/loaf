# Loaf Architecture

## Current Architecture (v2.0)

```
cli/                            # CLI tool (TypeScript, bundled by tsup)
├── index.ts                    # Commander.js entry point
├── commands/
│   ├── build.ts                # loaf build
│   ├── check.ts                # loaf check (enforcement backend)
│   ├── doctor.ts               # loaf doctor (alignment diagnostics, --fix)
│   ├── install.ts              # loaf install
│   ├── session.ts              # loaf session (start/end/log/enrich/list/archive)
│   └── spec.ts                 # loaf spec (list/archive)
└── lib/
    ├── build/
    │   ├── types.ts            # Shared types (BuildContext, HooksConfig, etc.)
    │   ├── targets/            # Target transformers (claude-code, cursor, opencode, codex, gemini, amp)
    │   └── lib/                # Utilities (version, sidecar, shared-templates, etc.)
    ├── detect/                 # AI tool + MCP detection
    ├── install/                # Target installers, fenced-section management, symlink helper
    │   ├── fenced-section.ts   # Managed-section writer (realpath dedup)
    │   └── symlinks.ts         # ensureProjectSymlinks, 4-state ensureSymlink + content migration
    ├── tasks/                  # Task/spec types, parser, migration, archival
    ├── release/                # Version bump, changelog generation
    ├── housekeeping/           # Artifact scanning, stale detection
    ├── journal/                # JSONL extraction for session enrichment
    ├── kb/                     # Knowledge base loader, staleness, resolution
    └── session/                # Session routing helpers (store, find, resolve)

content/                        # Distributable content (separated from tooling)
├── skills/{name}/SKILL.md      # Domain knowledge (Agent Skills standard)
├── agents/{name}.md            # Functional profiles (tool boundaries + behavioral contracts)
├── hooks/session/              # Session lifecycle scripts (compact.sh)
└── templates/                  # Shared templates (distributed at build time)

config/
├── hooks.yaml                  # Hook definitions (enforcement, instruction, session lifecycle)
└── targets.yaml                # Target defaults + shared-templates mapping

Output:
├── dist-cli/                   # Bundled CLI (single JS file via tsup)
├── plugins/loaf/               # Claude Code plugin (hooks, skills, agents, binary)
└── dist/{target}/              # Other targets (cursor, opencode, codex, gemini, amp)
```

### Build Flow

```
cli/index.ts → tsup → dist-cli/index.js (bundled CLI)
content/ + config/ → loaf build → dist/ + plugins/
```

Each target transformer reads content (skills/agents/hooks) and config, then produces target-specific output. Skills get sidecar files merged. Hooks get registered in plugin manifests. Shared templates get distributed to specified skills.

All TypeScript, bundled into a single file by tsup. No dynamic imports. The `loaf` binary in `plugins/loaf/bin/` is a self-contained copy of the bundled CLI with all npm dependencies inlined.

### Targets

| Target | Output | Agents | Skills | Hooks | Runtime Plugin |
|--------|--------|:------:|:------:|:-----:|:--------------:|
| claude-code | plugins/loaf/ | Yes | Yes | Yes | plugin.json |
| cursor | dist/cursor/ | Yes | Yes | Yes | hooks.json |
| opencode | dist/opencode/ | Yes | Yes | Yes | hooks.ts |
| codex | dist/codex/ | No | Yes | Yes | hooks.json |
| gemini | dist/gemini/ | No | Yes | No | — |
| amp | dist/amp/ | No | Yes | No | loaf.js |

### Prompt Overlay Consolidation (ADR-010)

The managed fenced section is written once to a canonical file (`.agents/AGENTS.md`). Per-harness paths are symlinks to it.

```
.agents/AGENTS.md                         # Canonical (source of truth, committed)
.claude/CLAUDE.md        → symlink →      .agents/AGENTS.md
./AGENTS.md              → symlink →      .agents/AGENTS.md  (agents.md spec)
```

**Write path (`loaf install`):** `installFencedSectionsForTargets` resolves each target's destination via `realpath` and groups writes by canonical path. Five of six targets share `.agents/AGENTS.md`, so they produce a single write. Before writing, `ensureProjectSymlinks` runs the 4-state machine per link:

| State | Action |
|-------|--------|
| nothing at linkPath | Create symlink |
| correct symlink | No-op (silent) |
| wrong symlink | Prompt to relink (auto-yes under `--yes`) |
| regular file | Prompt to merge content into canonical under `## Migrated from <path>`, back up as `.bak`, then symlink |

Fresh installs pre-create an empty canonical shell so symlinks are never dangling. `--yes` flag and non-TTY auto-detection allow the flow to run under CI/skills without interactive prompts.

**Drift detection (`loaf doctor`):** Six checks — canonical presence, per-harness symlink target, stale `.cursor/rules/loaf.mdc`, fenced-section version match, duplicate-resolved writes, and target coverage. `loaf doctor --fix` applies safe repairs non-interactively.

This extends the "CLI is the correct protocol layer" principle to filesystem convention enforcement: the CLI owns the on-disk overlay state, not the skills or the user. When ADR-010 shipped, five harnesses went from "each writes its own file" to "each resolves to the same file" without any skill edits.

### Mode-Aware Skills (Linear-Native Mode, ADR-011)

Skills that orchestrate specs and tasks (`/breakdown`, `/implement`, `/housekeeping`, `/shape`, `/council`) branch on `integrations.linear.enabled` in `.agents/loaf.json`:

- **Local-tasks mode** (default): specs in `.agents/specs/`, tasks in `.agents/tasks/`, `TASKS.json` as the programmatic index.
- **Linear-native mode**: specs stay in `.agents/specs/` (canonical, deliberation layer); tasks move to Linear as sub-issues under a `spec`-labeled parent rollup issue (execution layer).

The split reflects an architectural principle from ADR-010's consolidation pattern extended to the spec/task artifact model:

- **Deliberation artifacts belong with code.** Specs, ADRs, councils. Need git history, code-adjacent visibility, travel with the branch, survive the tracker being down or switched.
- **Execution artifacts belong in the tracker.** Tasks, blockers, comments, assignees. Need real-time state, dashboards, blocking graphs, notifications.

In Linear-native mode, the parent Linear issue is a **canonical-elsewhere rollup** — summary + link to the local spec file, not a re-host. Parallels the `.agents/AGENTS.md` → per-harness symlink pattern from ADR-010: the canonical artifact exists in exactly one place; the other surface is a thin pointer.

Skills detect the mode and branch accordingly. No skill edits are required to switch modes — same skill content, different backend. This sets up SPEC-023's backend abstraction as a narrower refactor than originally scoped: the contract is already mode-aware; SPEC-023 just extracts the Linear MCP calls into a shared `tracker` CLI subcommand with pluggable implementations.

### Pre-Flight Dependency Gate

`/implement`'s Linear-native routing enforces `blockedBy` as a **hard pre-flight gate**: before moving a sub-issue to `in_progress` or creating a session, every issue in its `blockedBy` field must be in a `completed`-type state. If not, the skill refuses to start — no session created, no issue moved.

This is different from advisory dependency ordering: the dependency graph becomes a runtime invariant, not a suggestion. The orchestrator cannot implement through open blockers even by accident.

The local-tasks equivalent (dependencies in `TASKS.json`) is still advisory. Pre-flight gating requires a system of record that can be queried reliably for all blockers' current state; Linear provides that, local files do not without a separate polling layer. This asymmetry is intentional: teams that care about strict dependency enforcement get Linear's gate for free; solo developers on local tasks trade enforcement for simplicity.

Parent-issue completion follows from the same contract: when the last sub-issue flips to `completed`, `/implement` auto-closes the parent. A parent with any open sub-issue (including `blocked`) stays open — the spec is not done until its execution tree is.

### Agent Model: Functional Profiles

Loaf uses **functional profiles** defined by tool access boundaries, not role-based agents defined by domain identity. Skills provide all domain knowledge; profiles provide the tool sandbox; the active soul provides the orchestrator's identity and team vocabulary.

**The Orchestrator:**

The main session is the **orchestrator** — a persistent coordinator identity defined in `.agents/SOUL.md`. The orchestrator coordinates, plans, and delegates but does not directly implement, review, research, or curate session state. The character and vocabulary of the orchestrator and its team are configurable via the soul catalog (see [Soul Catalog](#soul-catalog) below); `SOUL.md` is the single source of that identity. A SessionStart hook validates its presence and restores it from the configured soul (`content/souls/<name>/SOUL.md`) if missing.

**4 Functional Profiles:**

| Profile | Tool Access | Purpose |
|---------|-------------|---------|
| implementer | Full write | Writes code, tests, config, docs. Speciality via skills at spawn time. |
| reviewer | Read-only | Audits and verifies. Cannot modify what it reviews — independence is structural. |
| researcher | Read + Web | Investigates options, compares approaches, returns structured reports. No write or execute. |
| librarian | Read + Edit (.agents/) | Tends session lifecycle, state, wrap summaries. Does not implement or research. |

Each profile is defined in `content/agents/{implementer,reviewer,researcher,librarian}.md` — a minimal behavioral contract and tool boundary, not domain knowledge. A spawned implementer becomes a backend engineer, DBA, or devops engineer depending entirely on the skills loaded at spawn time. Profile prompts are soul-neutral: each agent reads `.agents/SOUL.md` at spawn time to internalize its character and instance-naming convention. Profiles operate even if `SOUL.md` is missing — agents lose personality, not capability.

**1 System Agent:**

| Agent | Purpose |
|-------|---------|
| background-runner | Async non-blocking tasks (haiku model) |

**Council Composition:**

Councils convene implementers and researchers for deliberation; reviewers join only after, to verify the outcome. The orchestrator runs the council but never votes — the team decides, the orchestrator integrates.

### Soul Catalog

The orchestrator's identity is decoupled from its mechanics. Profile prompts and skill prose are written in functional terms (implementer/reviewer/researcher/librarian/orchestrator); the **soul catalog** at [`content/souls/`](../content/souls/) carries character, naming convention, and team vocabulary as drop-in alternatives.

Two souls ship with v1:

| Soul | Description |
|------|-------------|
| `none` | Minimal, function-only. Roles named by what they do. No characters, no metaphor. The default for fresh installs. |
| `fellowship` | Tolkien-flavoured. The `fellowship` soul names them: the orchestrator is **the Warden** (Wizard); implementers are **Smiths** (Dwarves); reviewers are **Sentinels** (Elves); researchers are **Rangers** (Humans); librarians are **Librarians** (Ents). Race-appropriate instance names (e.g., "Borin — auth API"). Used by legacy installs and projects that want the lore. |

The active soul is recorded in `.agents/loaf.json` as `soul: <name>` and copied to `.agents/SOUL.md` (real file, not a symlink) at install time. The `loaf soul` CLI manages the catalog: `loaf soul list` enumerates available souls; `loaf soul current` reads the active name; `loaf soul show <name>` prints a catalog SOUL.md without writing; `loaf soul use <name>` activates a soul, with divergence detection that requires `--force` to overwrite a locally-edited `.agents/SOUL.md`.

Install behavior: fresh projects default to `none`; existing installs that already have an `.agents/SOUL.md` and no `soul:` field migrate to `soul: fellowship` automatically (preserving the legacy default). The SessionStart hook restores `.agents/SOUL.md` from the configured soul when missing, so deletes self-heal on the next session.

**Skills as Universal Knowledge Layer:**

Skills are the only knowledge mechanism that works across all targets (Claude Code, Cursor, Codex, Gemini, Amp). Profiles are Claude Code infrastructure — other targets activate knowledge through skills alone. This makes skills the primary investment surface: better skill descriptions and organization improve all targets simultaneously.

## Execution Model

The execution model is a three-artifact pipeline. No separate "plan" artifact — the journal serves as both execution trace and resumption protocol.

```
/shape → SPEC file → /breakdown → TASK files → /implement → Session/Journal → Done
```

### Task System

```
.agents/specs/SPEC-XXX.md       # Bounded work definitions (scope, test conditions, priority order)
.agents/tasks/TASK-XXX-slug.md  # Individual work items (criteria, file hints, verification)
.agents/tasks/TASKS.json        # Programmatic index (CLI readable)
.agents/sessions/               # Active session journals
.agents/sessions/archive/       # Completed sessions
```

**Specs** define *what* to build — problem, solution direction, boundaries, test conditions. Multi-part specs use priority ordering with go/no-go gates between tracks (ship in order, drop from end). Sized by complexity (small/medium/large), not time.

**Tasks** define *what to do* — one concern per task, file hints, verification command, observable done condition. Created by `/breakdown`, worked by `/implement`.

**Sessions** capture *what happened* — the journal records decisions, discoveries, commits, and progress as structured entries. The `## Current State` section provides handoff-ready context for compaction recovery and cross-conversation resumption.

### Session Lifecycle

Sessions are keyed by `claude_session_id` (the JSONL identity), **not** by branch. A session file's `branch:` frontmatter is a property recorded at start, not its identifier. One Claude conversation = one session file, regardless of how many branches that conversation visits; multiple Claude conversations on the same branch produce multiple session files. `loaf session start` routes on `claude_session_id`, consolidating splits when the same id appears across branch contexts.

`loaf session start` (SessionStart hook) and `loaf session end` (SessionEnd hook) manage the lifecycle programmatically.

**Subagent detection:** Hook JSON from Claude Code includes `agent_id` only for subagents. `loaf session start` checks for this and exits silently — subagents are session-unaware, preventing the session churn that occurs when Task tool spawns trigger SessionStart.

**Cross-conversation continuity:** `session_id` from hook JSON is stored as `claude_session_id` in session frontmatter. On SessionStart, if the incoming session_id differs from the stored one, the session knows it's a new conversation and writes resume entries. `loaf session end` writes the `--- PAUSE ---` separator with the correct timestamp.

**Session enrichment:** `loaf session enrich` reviews JSONL conversation logs and fills in missing journal entries via the librarian agent. The CLI extracts a deterministic summary (filtering noise types, applying timestamp cutoffs, discovering subagent transcripts), writes it to `.agents/tmp/`, and spawns `claude --agent librarian -p` with `LOAF_ENRICHMENT=1` for hook isolation. The librarian reads the summary and session file, identifies gaps, and appends entries. `enriched_at` in session frontmatter tracks the watermark.

**Compaction resilience:** The session journal is external memory that survives context compaction. PreCompact requires flushing unrecorded entries and writing a state summary to `## Current State`. PostCompact nudges the model to re-read the session file for resumption context. No separate snapshot mechanism needed.

**Session routing (SPEC-032, v2.0.0-dev.31):** User-facing session-mutating commands (`loaf session log`, `archive`, `enrich`, `end --wrap`) resolve their target via `resolveCurrentSession` in `cli/lib/session/resolve.ts` — a 3-tier priority chain: `--session-id <id>` flag → hook stdin payload (`--from-hook` opt-in only) → branch-fallback. Tier 3 emits a stderr WARN naming the branch and the silencing flag, so misroutes are visible in real time instead of corrupting state silently.

Hook-aware code paths (Stop event handlers, SessionStart resumption, PreCompact context, internal create-lock re-checks) keep the older inline pattern (`findSessionByClaudeId(...) || findActiveSessionForBranch(...)`) and exit silently on no-match — they fire frequently and silent failure is correct hook behavior. The asymmetry is documented in a block comment near the helpers in `cli/commands/session.ts`. Modules: `cli/lib/session/store.ts` (persistence primitives), `find.ts` (the two finders), `resolve.ts` (the chain helper), `index.ts` (public re-exports).

### Journal Entry Sources

Session journals receive entries from multiple layered sources:

| Source | Mechanism | When |
|--------|-----------|------|
| Skills | `loaf session log` in skill Critical Rules | Self-logging on invocation |
| Git events | PostToolUse command hooks | Commits, PRs, merges (automatic) |
| Task events | TaskCompleted session hook | Task completed/cancelled (automatic) |
| Context | UserPromptSubmit command hook | Every user prompt |
| Compaction | PreCompact prompt hook | Emergency journal flush |
| Enrichment | `loaf session enrich` → librarian agent | Lifecycle points (wrap, housekeeping) |

Skills self-log as their first action. Git and task events are captured automatically by hooks. The UserPromptSubmit hook injects session context and orchestration conventions on every prompt.

**Session management policy:**

| Scenario | Action |
|----------|--------|
| Same scope, continuing work | Compact (journal survives) |
| Different scope entirely | New conversation (new session) |
| Finished and archived a spec | New conversation |
| Context full mid-task | Auto-compact |
| Quick unrelated question | New conversation |

## Hook Architecture

Hooks are defined in `config/hooks.yaml` and distributed to target-specific formats at build time. For Claude Code, the canonical hook registration file is `hooks/hooks.json` (inside the plugin output directory). `plugin.json` silently drops non-matcher session lifecycle events — all hooks should be registered in `hooks.json`.

### Dispatch Types

| Type | Field | Behavior |
|------|-------|----------|
| script | `script:` | Runs a shell script |
| command | `command:` | Runs a CLI command (e.g., `loaf check --hook <id>`) |
| prompt | `prompt:` | Injects text directly to the AI model |

### Hook Type Behavioral Constraints

Hard-won constraints validated during SPEC-030 implementation:

- **`type: prompt`** — Binary gate. Any non-empty LLM response is treated as rejection (`ok: false`). Cannot express "this looks fine, proceed" — the response itself blocks. Unusable for advisory hooks or hooks requiring LLM judgment. Use only for validation that returns empty on success.
- **`type: agent`** — Read-only tool access (Read, Grep, Glob, WebFetch, WebSearch). No Edit, Write, or Bash. Max 50 turns. Useful for observation, not mutation.
- **`type: command`** — Correct primitive for context injection and side effects. Exit 0 with stdout for context injection. Exit 1 for non-blocking warning. Exit 2 to block the action.
- **Stop event circularity** — Writing to session files from a Stop hook can re-trigger the hook chain. State writes must be idempotent or guarded against re-entry.
- **PreCompact prompt hooks** — Not supported outside REPL sessions. Use `type: command` for PreCompact context injection.
- **`plugin.json` drops non-matcher events** — Session lifecycle events (SessionStart, SessionEnd, TaskCompleted) must be registered in `hooks/hooks.json`, not `plugin.json`.
- **UserPromptSubmit has no matcher** — Fires on every user message, cannot be filtered by tool name or input.
- **Session events use different JSON shape** — `hook_event_name` field instead of `tool_name`. TaskCompleted passes `task_subject` and `task_description`.
- **Plugin caching** — Cached plugin versions serve stale hook handlers during development. Marketplace remove/re-add is the reliable cache-busting path.
- **CLI-spawned agents need hook isolation** — When the CLI spawns `claude --agent <name> -p`, the child process triggers SessionStart/SessionEnd hooks. Set `LOAF_ENRICHMENT=1` (or similar) in the child env to suppress Loaf hooks. Do NOT use `--bare` — it breaks OAuth for subscription users.
- **`--bare` skips OAuth** — `--bare` mode requires API key auth (`ANTHROPIC_API_KEY`). Subscription users on OAuth cannot use `--bare`. Use env var isolation instead.

### Hook Categories

**Enforcement hooks** — quality gates that block bad actions. Run by `loaf check` as a unified TypeScript backend. Exit non-zero to block. `failClosed: true` means failures block the action. `validate-push` (pre-push) restricts direct pushes to the default branch to `.agents/` and `docs/` files only. Code changes require a feature branch and pull request.

**Instruction hooks** — context injection at tool invocation. Triggered by `matcher` patterns (tool name) and optionally filtered by `if` conditions (tool input). Inject relevant skill instructions or nudges.

**Session lifecycle hooks** — tied to events (`SessionStart`, `SessionEnd`, `PreCompact`, `PostCompact`, `Stop`). Manage session journals, compaction, and SOUL.md validation.

### Hook JSON Data Model

Claude Code passes JSON to hooks via stdin. Key fields for post-tool hooks:

| Field | Description |
|-------|-------------|
| `session_id` | Current Claude conversation ID |
| `agent_id` | Present only for subagents — the discriminator for session-aware hooks |
| `tool_name` | Name of the tool invoked (e.g., `"Bash"`) |
| `tool_input` | Arguments sent to the tool |
| `tool_response` | Result/output returned by the tool (post-tool only) |
| `cwd` | Working directory |

`loaf session log --from-hook` uses `tool_input.command` to detect commit/PR/merge patterns and `tool_response` to extract PR numbers from output.

## Knowledge Management

```
docs/knowledge/          # Knowledge files with frontmatter (covers:, topics:, etc.)
docs/decisions/          # ADRs (immutable decision records)
.agents/loaf.json        # Project config (local KB dirs, imports, integration toggles)
```

Knowledge files are managed by `loaf kb` — staleness detection compares file modification time against configurable thresholds. SessionStart surfaces stale file counts. The `/housekeeping` skill flags stale files for review.

## Config

```
.agents/loaf.json               # Project-level (knowledge dirs, integration toggles, settings)
~/.local/state/loaf/            # User-level (registered KBs, default settings)
~/.config/loaf/                 # User preferences
```

Integration toggles in `loaf.json` gate runtime features (Linear magic-word detection, MCP recommendations) without rebuilding.

## Test Fixture Hygiene

Any test that spawns a CLI subprocess must use OS-tmp isolation for its fixtures:

```ts
import { mkdtempSync, realpathSync } from "fs";
import { tmpdir } from "os";
import { join } from "path";

let TEST_ROOT: string;

beforeEach(() => {
  TEST_ROOT = realpathSync(mkdtempSync(join(tmpdir(), "loaf-<suite>-")));
  // ...
});
```

CWD-relative fixtures (`join(process.cwd(), ".test-..."`)) are forbidden for subprocess tests. Under vitest's default file parallelism, workers share the filesystem and cwd — CWD-relative paths race against other test files' subprocesses. The failure mode is silent under per-file runs (where pollution cannot occur) and non-deterministic under full-suite runs. One such leak in `cli/commands/check.test.ts` silently failed `cli/commands/report.test.ts` for 17+ commits before v2.0.0-dev.28 bisected it.

`realpathSync` is required on macOS because the system tmpdir (`/var/folders/...`) is reached through a `/private/var/folders/...` symlink; without realpath, subprocess cwd comparisons can fail.

Until every test file is migrated, `vitest.config.ts` sets `fileParallelism: false` as a defensive default — ~20% slower but deterministic. The plan is to migrate remaining cwd-relative fixtures and re-enable parallelism once the pattern is enforced throughout.

## Cross-Cutting Patterns

Patterns that apply across multiple subsystems and emerged from specific post-release followups. Captured here so they inform future work rather than being re-discovered.

### Single-Source Runtime Versioning via Build-Time Injection

The CLI version must report the same value in every runtime mode: dev (`tsx`), source-built (`npm run`), and bundled-binary (`dist-cli/index.js` on PATH). `cli/lib/version.ts` exposes `LOAF_VERSION`, sourced from tsup's `define: { __LOAF_VERSION__: JSON.stringify(pkg.version) }`, with a `package.json` walk-up fallback for dev/test contexts where the define is not applied. Consumers: `cli/index.ts` (Commander `.version()`), `cli/lib/install/fenced-section.ts` (fenced-section version marker), `cli/commands/doctor.ts` (`fenced-version` check).

Before PR #35 (v2.0.0-dev.30), three separate runtime `package.json` walkers in those same files diverged and all fell back to `"0.0.0"` from the bundled binary — a false-positive factory for every version-comparison check in the CLI. The lesson generalizes: any value that must be identical across runtime modes should be injected at build time, not resolved at runtime.

### Generated Runtime Plugin Artifacts Parsed From Emitted Output

Files the build emits for downstream runtimes to execute — OpenCode `hooks.ts`, Amp `loaf.js`, and any future per-target runtime plugin — must have a test that parses the **actual emitted file** via a real parser (TypeScript compiler API, Acorn), not just the generator's input string.

Template-literal escape bugs are invisible at the string level: `cli/lib/build/lib/hooks/runtime-plugin.ts` emitted invalid regex (`/*/g`) into `dist/opencode/plugins/hooks.ts` for multiple versions because the broken code path was unreachable at runtime. The syntactic breakage was dormant until OpenCode's plugin loader tightened its validation and rejected the file on load. `cli/lib/build/targets/runtime-logic.test.ts` now parses both OpenCode's and Amp's emitted output via the TypeScript compiler API as a regression fence for the entire class of escape/interpolation bugs.

### Visible-Degraded Fallback with Stderr WARN

When strict invariant enforcement would break existing callers but silent fallback corrupts data, emit a stderr WARN naming the missing signal and the silencing flag. The action proceeds (preserving compatibility) but the WARN makes the misroute visible in real time and provides a regression-testable surface for the eventual cutover.

SPEC-032 used this pattern for branch-fallback session routing in `loaf session log`, `archive`, `enrich`, and `end --wrap`. The 3-tier resolution chain emits `WARN: no session_id signal — falling back to branch routing for branch '<branch>'. Pass --session-id <id> to silence.` only when neither the `--session-id` flag nor the hook stdin payload provided a session id. Skill self-logging trips this WARN today; a future skill refactor will source `session_id` per-process and remove the branch tier, with the WARN serving as the cutover's regression gate.

The pattern generalizes: any compatibility carve-out that violates a stated invariant should be observable in real time, not invisible. The cost (one extra stderr line for legacy callers) is paid once per invocation; the benefit (every misroute surfaces immediately) is paid forward to whoever next opens an issue saying "my entry didn't land where I expected."
