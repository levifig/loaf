---
topics:
  - hooks
  - lifecycle
  - validation
  - enforcement
covers:
  - config/hooks.yaml
  - internal/cli/check*.go
  - internal/cli/github_account.go
  - internal/cli/journal_hook*.go
  - internal/cli/kb_nudge.go
  - internal/cli/build_*.go
  - content/hooks/**/*
consumers:
  - implementer
  - reviewer
last_reviewed: '2026-11-06'
---

# Hook System

## Contents

- Hook Model
- Adding a Hook
- Dispatch Types
- Hook Events
- Registration
- Enforcement Hooks
- Instruction Hooks
- Prompt Hooks
- Journal Capture Compatibility
- Conversation Hooks
- Knowledge Staleness Hook
- Current Hook Boundary
- Cross-References

Hooks enforce rules and inject guidance at supported tool and lifecycle boundaries. This guide describes the current registry and adapter behavior; a parser or historical event name alone does not prove installed hook delivery. Read it before adding or changing a hook.

## Hook Model

- **Enforcement hooks** are quality gates at supported harness tool boundaries; command matchers can target actions such as `git commit` or `git push`. Examples: secrets scanning and destructive-command checks. Blocking checks exit non-zero; advisory checks report without blocking. Run them manually with `loaf check`.
- **Instruction and context hooks** inject guidance when a tool is invoked or a lifecycle event fires (for example the pre-merge checklist or the continuity digest), registered in `hooks.yaml` with `matcher`, `if`, or `event`.

## Adding a Hook

Implement maintained deterministic behavior in the native CLI (`internal/cli/check*.go`) with regression tests, then register the hook in `config/hooks.yaml` with `skill:` pointing at its owner. Use static instruction files in `content/hooks/instructions/` for guidance. Do not add new Bash/Python helpers. Skills are auto-discovered, so a skill that ships no hooks needs no hook entry.

## Dispatch Types

Dispatch depends on the registered entry and the target adapter:

| Type | Field | Behavior | Example |
|------|-------|----------|---------|
| **Native check** | None of `script:`, `command:`, `instruction:`, or `prompt:` | Runs `loaf check --hook <id>` | `check-secrets` |
| **command** | `command:` | Runs a CLI command | `loaf journal context --from-hook` |
| **Instruction** | `instruction:` (with `type: command`) | Claude Code and OpenCode print the copied instruction file when the hook fires. Cursor reads the same copied file through its native JSON adapter. Codex and Amp do not project instruction hooks | `instructions/pre-merge.md` |
| **prompt** | `prompt:` | Requests an LLM decision where the target supports prompt hooks; not used by the current registry | Validation gates |
| **Legacy script** | `script:` | Compatibility only; do not add maintained Bash/Python implementations | — |

Instruction hooks are used for advisory checklists (pre-merge, pre-push, post-merge).

OpenCode pre-tool enforcement requires an executable command or script. Native checks, including `ephemeral-provenance` before `git push`, dispatch through `loaf check --hook <id>`. Exit 0 allows the tool; exit 2 rejects it and preserves the checker's refusal. A `failClosed: true` hook also rejects missing dispatch, a missing runtime, subprocess spawn errors, unexpected nonzero exits, signals, and timeouts. Advisory hooks warn on other failures and continue, while an explicit exit 2 still rejects the tool. Failure messages identify the hook or subprocess problem so the operator can restore the runtime or hook configuration and retry.

Cursor projects pre-merge and pre-push instructions to `beforeShellExecution`, whose command matcher filters shell command text and whose native response supports `permission: "allow"` with `agent_message`. Post-merge stays on `postToolUse` with a `Shell` matcher and returns `additional_context`. The native dispatcher checks executable arguments before reading guidance, so benign commands and quoted mentions receive no checklist. Missing or unreadable guidance produces valid advisory JSON, rather than a malformed-output block. These three entries carry no unsupported `if` field; other adapters keep their existing conditions. Historical `cat` entries and prior disabled intent remain recognized when the pre-tool instructions move to the native shell event. See [Cursor Hooks](https://cursor.com/docs/hooks) for the transport contract. This is source and protocol coverage, not a claim of live Cursor reproduction.

## Hook Events

The current registry uses these boundaries; target support and event names differ:

| Boundary | Timing | Loaf use |
|----------|--------|----------|
| PreToolUse | Before a supported tool action | Blocking checks and advisory pre-merge/pre-push instructions |
| PostToolUse | After a supported tool action | Advisory KB staleness and post-merge instructions |
| SessionStart | Conversation start or resumption where delivered | Emit the layered continuity digest |
| PreCompact | Before context compaction where delivered | Inject selective checkpoint guidance |
| PostCompact | After context compaction where delivered | Re-emit the continuity digest |
| UserPromptSubmit | Prompt submission where delivered | Inject the current implementation guidance |

The registry has no SessionEnd, Stop, or TaskCompleted journal hook. No hook opens, closes, or writes back to a session record. A target's event support must not be mistaken for a Loaf registration or a proven live integration.

### Hook JSON Context

Hook payloads are normalized by target adapters. Common and historical inputs include:

| Field | Present In | Purpose |
|-------|-----------|---------|
| `tool.name` / `tool_name` | pre-tool, post-tool | Which tool triggered the hook |
| `tool.input` / `tool_input` | pre-tool, post-tool | Tool arguments (command, file_path, content) |
| `tool_response` | post-tool payloads | Tool result evidence understood by compatibility journal capture |
| `session_id` | harness payloads that provide it | Opaque conversation correlation, not a Loaf session entity |
| `agent_id` | subagent payloads that provide it | Normalized subagent identity makes journal hook entry points exit silently |
| `hook_event_name` | lifecycle payloads | Event type identifier instead of a tool name |
| `task_subject`, `task_description` | historical TaskCompleted capture | Understood by compatibility parsing; no TaskCompleted hook is currently registered |

## Registration

Hooks are defined in `config/hooks.yaml` grouped under `pre-tool`, `post-tool`, or `session`. For Claude Code, hooks are registered in `hooks/hooks.json` inside the plugin directory — `plugin.json` silently drops non-matcher lifecycle events. See [build-system.md](build-system.md) for details on how hooks are distributed to targets.

```yaml
hooks:
  pre-tool:
    - id: check-secrets
      skill: security-compliance
      failClosed: true
      blocking: true
      matcher: "Edit|Write|Bash"
      timeout: 30000
    - id: workflow-pre-merge
      skill: git-workflow
      type: command
      instruction: instructions/pre-merge.md
      matcher: "Bash"
      if: "Bash(gh pr merge:*)"
      blocking: false
      timeout: 5000
```

### Fields

| Field | Required | Notes |
|-------|----------|-------|
| `id` | Yes | Unique hook identifier |
| `skill` | Yes | Owning skill name |
| `type` | No | Adapter dispatch type (`command`, `prompt`); omit it for maintained enforcement so the native check default applies |
| `matcher` | No | Tool name filter: `"Edit\|Write\|Bash"` |
| `if` | No | Conditional: `"Bash(git commit:*)"` — hook only fires when invocation matches |
| `blocking` | No | `true` if hook can block tool execution |
| `failClosed` | No | `true` to block on hook failure (enforcement hooks) |
| `timeout` | No | Timeout in milliseconds |
| `event` | No | Session event: `SessionStart`, `PreCompact`, `PostCompact`, `TaskCompleted`, `UserPromptSubmit` |

## Enforcement Hooks

Native enforcement and workflow checks run via `loaf check --hook <id>` in `internal/cli/check.go`:

Each pre-tool evaluation counts a content-free local observation when the project has initialized current-schema SQLite state. Counts are kept per UTC month for each hook ID, normalized tool name, allow/warn/block outcome, bounded reason class, and observed harness/version, so storage stays bounded however many tool calls run. They never include tool arguments, file content, or the check's diagnostic text. Recording uses the ordinary state open check and waits at most 100 ms for another writer, because hooks run on every tool call and a longer wait would stall it. `loaf telemetry status [--json]` reports counts by project, harness, outcome, and reason class for the current UTC month. Recording failure does not change the check's decision.

| Hook | Skill | Matcher | Blocking | What It Does |
|------|-------|---------|:--------:|-------------|
| `artifact-body-write` | orchestration | Edit\|Write\|Bash | Yes (failClosed) | Protects stored artifact bodies and projections while permitting authored reports and file-backed councils |
| `artifact-names` | foundations | Bash | Yes (failClosed) | Checks tracked artifact filenames before commit |
| `check-secrets` | security-compliance | Edit\|Write\|Bash | Yes (failClosed) | Scans file content and Bash commands for hardcoded secrets |
| `render-drift` | orchestration | Bash | Yes (failClosed) | Checks stamped durable renders before push |
| `ephemeral-provenance` | orchestration | Bash | Yes (failClosed) | Blocks tracked ephemeral Markdown and dangling references from retained specs before push |
| `security-audit` | security-compliance | Bash | Yes (failClosed) | Blocks dangerous shell patterns; eligible nontrivial commands run available scanners only for thorough validation or reviewer/implementer agent context |
| `github-account` | git-workflow | Bash | Yes (failClosed) | Switches the active GitHub CLI account to match `.agents/loaf.json` before `gh` commands (pass-with-warning), exempting `gh auth` administration; blocks when the configured identity cannot be established |
| `validate-push` | git-workflow | Bash | Advisory | Verifies version bump, CHANGELOG, build before push |
| `workflow-pre-pr` | git-workflow | Bash | Yes (failClosed), opted-in repos | Checks declared Issue coverage versus the review base and PR format; other repos receive generic advisory reminders |
| `validate-commit` | orchestration | Bash | Yes (failClosed) | Validates Conventional Commits format, blocks AI attribution |
| `validate-infra-safety` | infrastructure-management | Bash | Yes (failClosed) | Blocks destructive infrastructure commands and unsafe filesystem targets |
| `validate-sql-safety` | database-design | Bash | Yes (failClosed) | Blocks destructive SQL commands |

All enforcement entries above except `validate-push` use `failClosed: true`. `validate-push` remains advisory (`blocking: false`). The issue-changelog operator and dedicated PR check share the pre-PR coverage engine; Ship separately verifies meaning and scope completeness. A hosted check is advisory to mergeability until a native required-check rule is activated and read back.

Residual risk on `github-account`: convergence *writes* the shared global gh account pointer on every mismatched `gh` call -- including read-only ones like `gh pr view`, which the earlier pure-read check only observed -- so concurrent sessions on different identities collide on that pointer more frequently; the race window's shape is unchanged, its trigger rate is not. `gh auth` administration is exempt (identity management is the user's domain), passing through with no probe or switch.

## Instruction Hooks

Inject markdown file content at tool invocation via `type: command` with `instruction:` field:

| Hook | Event/Condition | Purpose |
|------|-----------------|---------|
| `workflow-pre-merge` | `Bash(gh pr merge:*)` | Ship-only merge path and merge-strategy checklist |
| `workflow-pre-push` | `Bash(git push:*)` | Pre-push advisory reminders |
| `workflow-post-merge` | `Bash(gh pr merge:*)` | Post-merge housekeeping checklist |

Instruction files live in `content/hooks/instructions/` and are rendered into hook output for targets that support this delivery.

Amp deliberately omits `workflow-pre-merge`, `workflow-pre-push`, and `workflow-post-merge` from both tool hook projections. Its documented [`tool.call` allow response](https://ampcode.com/docs/plugin-api) carries no model-context field, and this adapter discards successful hook stdout. `tool.result` can modify output after execution, which does not provide a pre-action checklist. Amp therefore has no automatic tool-boundary delivery for these advisory files; the shared Git Workflow and Ship skills remain available, and native enforcement hooks are unchanged. The registry and instruction sources remain canonical for targets that support them.

## Prompt Hooks

The current registry uses command hooks for context and advisory guidance, not prompt hooks. Do not change that dispatch type merely because the emitted content is a prompt to the agent.

The maintained PreCompact journal hook is a command hook owned by Foundations. It emits selective capture guidance, not a blocking prompt gate or automatic summary. Its delivery depends on the harness and mode; ordinary capture also lives directly in managed project instructions and does not require this event.

### Claude Code

Prompt hooks ask an LLM to return a structured decision: `ok: true` allows the action; `ok: false` is handled according to the event. A nonempty response is not itself a refusal. See the official [prompt-hook response schema](https://code.claude.com/docs/en/hooks#response-schema). Agent-hook capabilities and supported events are also product-specific; do not apply historical blanket tool or lifecycle limits across targets.

## Journal Capture Compatibility

`loaf journal log --from-hook` still parses supported git/gh/task event payloads, including result evidence, and silently skips normalized subagent invocations. This compatibility entry point does not mean automatic capture is installed.

The current registry installs no git-event, PR-event, task-completion, or Linear-magic journal auto-entry hook. Capture meaningful outcomes explicitly through ordinary journal logging. Foundations owns ordinary capture and selective synthesis; lack of a completion hook does not create a required end-of-session summary.

## Conversation Hooks

Registered under `session:` in `hooks.yaml` with an `event:` field. These command hooks emit context or guidance; none opens, closes, or mutates a session record:

| Hook | Event | Dispatch | Purpose |
|------|-------|----------|---------|
| `session-start-loaf` | SessionStart | command (`loaf journal context --from-hook`) | Emit the compact layered continuity digest, kept under every harness injection limit; silent for subagents |
| `session-context-inject` | UserPromptSubmit | command (`loaf journal context for-prompt`) | Inject implementation conventions on prompt submit |
| `pre-compact` | PreCompact | command (`loaf journal context for-compact`) | Reinforce ordinary capture and selective checkpoints; writes nothing itself |
| `post-compact` | PostCompact | command (`loaf journal context for-resumption`) | Re-emit the continuity digest for post-compaction resumption |

Target builders project only supported events and select their native context envelopes. Normalized subagent invocations exit silently. Codex currently projects SessionStart only; the [build guide](build-system.md#notes) distinguishes historical startup evidence from current-candidate and other live-mode limits.

The current `for-prompt` text still contains legacy task-creation and blanket-delegation instructions. That known runtime gap does not establish current shared-work authority; the [Flow semantics](../../content/skills/loaf-reference/references/flow-semantics.md) remain the work-selection contract.

## Knowledge Staleness Hook

The registered advisory edit nudge is:

| Hook | Condition | Purpose |
|------|-----------|---------|
| `kb-staleness-nudge` | Edit\|Write | Tracks covered edits and nudges when covered knowledge is stale |

It checks local coverage against committed changes since review and uses disposable notification-cache markers. It does not mark knowledge reviewed or regenerate a task board. See [Knowledge Base Operations](knowledge-management-design.md#advisory-edit-nudges) for scope and limitations.

## Current Hook Boundary

Language, infrastructure, design, and advisory quality guidance belongs in skill instructions. Deterministic enforcement belongs in `loaf check` hooks. Retired shell-hook categories include:

- Language hooks (python-\*, typescript-\*, rails-\*)
- Infrastructure hooks (k8s, dockerfile, terraform)
- Design hooks (a11y, tokens)
- Quality hooks (format-check, tdd-advisory, validate-changelog)

The former shared Bash helpers (`json-parser.sh`, `config-reader.sh`, `agent-detector.sh`, `timeout-manager.sh`) are no longer part of the active hook runtime.

### Claude Code

If adding a `Stop` hook, guard against repeated continuation: the event fires when Claude finishes responding, and a hook can keep the turn going. Check `stop_hook_active` or the transcript before requesting more work, and ensure the stopping condition can resolve. See [Stop hook input](https://code.claude.com/docs/en/hooks#stop-input).

## Cross-References

- [build-system.md](build-system.md) — how hooks get distributed to targets
- [skill-architecture.md](skill-architecture.md) — how skills own hooks via `skill:` field in hooks.yaml
