---
topics:
  - hooks
  - lifecycle
  - validation
  - enforcement
covers:
  - config/hooks.yaml
  - internal/cli/check.go
  - content/hooks/**/*
consumers:
  - implementer
  - reviewer
last_reviewed: '2026-10-12'
---

# Hook System

Hooks run at lifecycle events to enforce rules, inject context, and capture journal entries. Read this before adding or changing a hook.

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
| **prompt** | `prompt:` | Injects inline text into model context | Validation gates |
| **Legacy script** | `script:` | Compatibility only; do not add maintained Bash/Python implementations | — |

Instruction hooks are used for advisory checklists (pre-merge, pre-push, post-merge).

OpenCode pre-tool enforcement requires an executable command or script. Native checks, including `ephemeral-provenance` before `git push`, dispatch through `loaf check --hook <id>`. Exit 0 allows the tool; exit 2 rejects it and preserves the checker's refusal. A `failClosed: true` hook also rejects missing dispatch, a missing runtime, subprocess spawn errors, unexpected nonzero exits, signals, and timeouts. Advisory hooks warn on other failures and continue, while an explicit exit 2 still rejects the tool. Failure messages identify the hook or subprocess problem so the operator can restore the runtime or hook configuration and retry.

Cursor projects pre-merge and pre-push instructions to `beforeShellExecution`, whose command matcher filters shell command text and whose native response supports `permission: "allow"` with `agent_message`. Post-merge stays on `postToolUse` with a `Shell` matcher and returns `additional_context`. The native dispatcher checks executable arguments before reading guidance, so benign commands and quoted mentions receive no checklist. Missing or unreadable guidance produces valid advisory JSON, rather than a malformed-output block. These three entries carry no unsupported `if` field; other adapters keep their existing conditions. Historical `cat` entries and prior disabled intent remain recognized when the pre-tool instructions move to the native shell event. See [Cursor Hooks](https://cursor.com/docs/hooks) for the transport contract. This is source and protocol coverage, not a claim of live Cursor reproduction.

## Hook Events

| Event | Timing | Can Block | Use Case |
|-------|--------|:---------:|----------|
| PreToolUse | Before Edit/Write/Bash | Yes | Secrets check, security audit, commit validation |
| PostToolUse | After Edit/Write/Bash | No | Task board refresh, KB staleness, journal auto-entries |
| SessionStart | Conversation begins | No | Emit the layered continuity digest |
| PreCompact | Before context archival | No | Inject journal-flush guidance |
| PostCompact | After context compaction | No | Re-emit the continuity digest |
| UserPromptSubmit | Every user message | No | Context injection, orchestration conventions |
| TaskCompleted | Task marked complete | No | Journal auto-entry for task events |

There is no SessionEnd or Stop journal obligation: the SessionEnd hook is absent, and no hook writes back to a session record. The general Stop-circularity caution still governs any future stateful hook; see [Hook Type Behavioral Constraints](#hook-type-behavioral-constraints).

### Hook JSON Context

Harnesses pass JSON on stdin to hooks. Key fields:

| Field | Present In | Purpose |
|-------|-----------|---------|
| `tool.name` / `tool_name` | pre-tool, post-tool | Which tool triggered the hook |
| `tool.input` / `tool_input` | pre-tool, post-tool | Tool arguments (command, file_path, content) |
| `tool_response` | post-tool only | Tool output (stdout/stderr); used by `--from-hook` to extract PR URLs |
| `session_id` | session hooks | Claude conversation identifier; recorded as the journal entry's `harness_session_id` |
| `agent_id` | session hooks | Present only for subagents; journal `--from-hook` exits silently when set (subagents write nothing) |
| `hook_event_name` | session events (TaskCompleted, etc.) | Event type identifier — session events use this instead of `tool_name` |
| `task_subject`, `task_description` | TaskCompleted | Task details for journal logging |

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

Each pre-tool evaluation records a content-free local observation when the project has initialized current-schema SQLite state. The observation stores the hook ID, a normalized tool name, allow/warn/block, a bounded reason class, and observed harness/version. It never stores tool arguments, file content, or the check's diagnostic text. `loaf telemetry status [--json]` reports counts by project, harness, outcome, and reason class since the start of the current UTC month. Recording failure does not change the check's decision.

| Hook | Skill | Matcher | Blocking | What It Does |
|------|-------|---------|:--------:|-------------|
| `check-secrets` | security-compliance | Edit\|Write\|Bash | Yes (failClosed) | Scans file content and Bash commands for hardcoded secrets |
| `security-audit` | security-compliance | Bash | Yes (failClosed) | Blocks dangerous shell patterns; runs Trivy/Semgrep/npm-audit when available |
| `github-account` | git-workflow | Bash | Yes (failClosed) | Switches the active GitHub CLI account to match `.agents/loaf.json` before `gh` commands (pass-with-warning), exempting `gh auth` administration; blocks only when the switch fails |
| `validate-push` | git-workflow | Bash | Advisory | Verifies version bump, CHANGELOG, build before push |
| `workflow-pre-pr` | git-workflow | Bash | Yes (failClosed), opted-in repos | Checks declared Issue coverage versus the review base and PR format; other repos receive generic advisory reminders |
| `validate-commit` | orchestration | Bash | Yes (failClosed) | Validates Conventional Commits format, blocks AI attribution |
| `validate-infra-safety` | infrastructure-management | Bash | Yes (failClosed) | Blocks destructive infrastructure commands and unsafe filesystem targets |
| `validate-sql-safety` | database-design | Bash | Yes (failClosed) | Blocks destructive SQL commands |

Security hooks (`check-secrets`, `security-audit`, `validate-infra-safety`, `validate-sql-safety`), `github-account`, `validate-commit`, and `workflow-pre-pr` use `failClosed: true`. `validate-push` remains advisory (`blocking: false`). The issue-changelog operator and dedicated PR check share the pre-PR coverage engine; Ship separately verifies meaning and scope completeness. A hosted check is advisory to mergeability until a native required-check rule is activated and read back.

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

Inject inline text into the model's context. Prompt hooks are binary gates — any non-empty LLM response blocks the action. Use only for validation that returns empty on success, never for advisory nudges. See [Hook Type Behavioral Constraints](#hook-type-behavioral-constraints) for the full list.

The maintained PreCompact journal hook is a command hook owned by Foundations. It emits selective capture guidance, not a blocking prompt gate or automatic summary. Its delivery depends on the harness and mode; ordinary capture also lives directly in managed project instructions and does not require this event.

## Journal Auto-Entry Hooks

Command hooks that auto-log to the project journal via `loaf journal log --from-hook`, which parses the hook JSON (including `tool_response` for PR URLs) to determine entry type. `--from-hook` reads the harness payload on stdin and exits silently for subagent invocations:

| Hook | Condition | Logs |
|------|-----------|------|
| `journal-git-events` | `Bash(git commit:*)` | Commits and merges — `commit(SHA): message` |
| `journal-gh-events` | `Bash(gh pr:*)` | PR creation and merges — `pr(create): title (#N)`, `pr(merge): #N merged` |
| `journal-task-completed` | TaskCompleted event | Task completions — `task(complete): subject` |

The `detect-linear-magic` hook runs as a pre-tool command (`loaf journal log --detect-linear`) and only fires when Linear integration is enabled in `.agents/loaf.json`.

## Conversation Hooks

Registered under `session:` in hooks.yaml with an `event:` field. These hooks emit the derived continuity digest or auto-log events; none opens, closes, or mutates a session record:

| Hook | Event | Dispatch | Purpose |
|------|-------|----------|---------|
| `session-start-loaf` | SessionStart | command (`loaf journal context --from-hook`) | Emit the compact layered continuity digest, kept under every harness injection limit; silent for subagents |
| `session-context-inject` | UserPromptSubmit | command (`loaf journal context for-prompt`) | Inject implementation conventions on prompt submit |
| `pre-compact` | PreCompact | command (`loaf journal context for-compact`) | Reinforce ordinary capture and selective checkpoints; writes nothing itself |
| `post-compact` | PostCompact | command (`loaf journal context for-resumption`) | Re-emit the continuity digest for post-compaction resumption |
| `journal-task-completed` | TaskCompleted | command (`loaf journal log --from-hook`) | Auto-logs task completions to the journal |

`loaf journal context --from-hook` and `loaf journal log --from-hook` use `agent_id` from hook JSON to detect subagents — subagent invocations exit silently and write nothing.

## Side-Effect Hooks

Post-tool hooks that perform background maintenance:

| Hook | Condition | Purpose |
|------|-----------|---------|
| `generate-task-board` | Edit\|Write | Regenerates TASKS.md when task files change (`loaf task refresh`) |
| `kb-staleness-nudge` | Edit\|Write | Tracks covered edits and nudges when covered knowledge is stale |

## Current Hook Boundary

Language, infrastructure, design, and advisory quality guidance belongs in skill instructions. Deterministic enforcement belongs in `loaf check` hooks. Retired shell-hook categories include:
- Language hooks (python-\*, typescript-\*, rails-\*)
- Infrastructure hooks (k8s, dockerfile, terraform)
- Design hooks (a11y, tokens)
- Quality hooks (format-check, tdd-advisory, validate-changelog)

The former shared Bash helpers (`json-parser.sh`, `config-reader.sh`, `agent-detector.sh`, `timeout-manager.sh`) are no longer part of the active hook runtime.

## Hook Type Behavioral Constraints

Hook types have hard behavioral limits. Key constraints:

- **Prompt hooks** are binary gates — any non-empty LLM response blocks. Never use for advisory guidance.
- **Agent hooks** are read-only (no Edit/Write/Bash). Max 50 turns.
- **Command hooks** are the correct primitive for context injection and side effects.
- **Stop hooks** risk circular re-triggers when writing to monitored files.
- **PreCompact prompt hooks** don't work outside REPL sessions.

## Cross-References

- [build-system.md](build-system.md) — how hooks get distributed to targets
- [skill-architecture.md](skill-architecture.md) — how skills own hooks via `skill:` field in hooks.yaml
