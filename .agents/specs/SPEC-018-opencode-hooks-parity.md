---
id: SPEC-018
title: OpenCode Hooks Parity — stdin piping, blocking, config, and docs
source: code-review
created: '2026-03-30T10:50:00.000Z'
status: drafting
appetite: Small (1–2 sessions)
---

# SPEC-018: OpenCode Hooks Parity

## Problem Statement

The OpenCode build target generates hooks that look correct but don't function at runtime. Four pre-existing gaps were surfaced during SPEC-014 review:

1. **Hook input mismatch**: `runHook()` in the generated `hooks.js` passes tool context via environment variables (`TOOL_NAME`, `TOOL_INPUT`), but every hook script reads from stdin (`INPUT=$(cat)`). Result: hooks execute but receive empty input and can't inspect the tool call. All pre-tool validation (secrets check, format check, type checking, linting) is effectively a no-op.

2. **No blocking capability**: Pre-tool hooks return exit codes (e.g., `exit 2` from `check-secrets.sh` to block), but the `tool.execute.before` handler ignores the result. Dangerous operations like committing secrets cannot be stopped. Claude Code's hook system supports blocking via exit codes; OpenCode's generated plugin does not.

3. **No plugin registration config**: There is no `opencode.jsonc` with a `plugins` section pointing to `hooks.js`. Users must manually discover and register the plugin after `loaf install`.

4. **No setup documentation**: No `OPENCODE.md` or equivalent guides users through installation, plugin registration, or known limitations.

## Solution Direction

### Fix 1: Pipe JSON to stdin

In `cli/lib/build/targets/opencode.ts`, update the generated `runHook()` function to pass tool input via stdin (matching what the shell scripts expect) while keeping env vars as a fallback:

```javascript
const result = execFileSync(interpreter, [scriptPath], {
  cwd: process.cwd(),
  input: JSON.stringify({ tool_name: toolName, tool_input: toolInput }),
  env: {
    ...process.env,
    TOOL_NAME: toolName || '',
    TOOL_INPUT: JSON.stringify(toolInput || {}),
  },
  encoding: 'utf-8',
  timeout,
});
```

Adding the `input` option to `execFileSync` pipes the JSON to the child process's stdin. The env vars stay for scripts that may prefer them.

### Fix 2: Blocking pre-tool hooks

Update the `tool.execute.before` handler to check hook exit codes. The `execFileSync` call already throws on non-zero exit — catch exit code 2 specifically (the convention used by `check-secrets.sh`) and propagate the block:

```javascript
'tool.execute.before': async (input, output) => {
  // ... matcher logic ...
  const result = runHook(hook.script, toolName, toolInput, hook.timeout);
  if (!result.success && result.blocked) {
    return { blocked: true, message: result.output };
  }
}
```

**Caveat**: This depends on OpenCode's plugin API supporting a return value from `tool.execute.before` that blocks execution. Need to verify against OpenCode's plugin docs. If the API doesn't support blocking, document the limitation and make hooks advisory-only with clear warnings.

### Fix 3: Generate opencode.jsonc

Add a step in the OpenCode build target (or install command) that generates or updates `opencode.jsonc` with the plugins entry:

```json
{
  "plugins": [
    { "path": "~/.config/opencode/plugins/hooks.js" }
  ]
}
```

**Caveat**: Need to verify OpenCode's current config format and plugin registration mechanism. This may have changed since the target was originally written.

### Fix 4: Create OPENCODE.md

Add `dist/opencode/OPENCODE.md` with:
- Prerequisites (OpenCode version, Node.js)
- Installation steps (`loaf install --to opencode`)
- Plugin registration (manual step if needed)
- Known limitations vs Claude Code (blocking, session hooks, etc.)
- Troubleshooting

## Scope

### In Scope

- Fix `runHook()` to pipe JSON via stdin
- Investigate and implement blocking if OpenCode supports it
- Generate or document plugin registration config
- Create setup documentation for OpenCode target
- Verify all pre-tool hooks receive valid input after the fix

### Out of Scope

- Rewriting hook scripts to support env var input (scripts should stay Claude-agnostic)
- Changes to Cursor, Codex, or Gemini targets
- New hooks or hook functionality
- OpenCode-specific hooks that don't exist for Claude Code

### Rabbit Holes

- Don't build a full abstraction layer between hook invocation styles — just pipe stdin
- Don't try to make OpenCode hooks feature-identical to Claude Code — document the gaps

## Dependencies

| Dependency | Type | Notes |
|---|---|---|
| SPEC-014 | Soft (upstream) | Profile model ships first; this fixes pre-existing OpenCode gaps |

## Open Questions

- [ ] Does OpenCode's `tool.execute.before` support blocking via return value? If not, what's the alternative?
- [ ] What is the current OpenCode plugin registration mechanism? Has `opencode.jsonc` format changed?
- [ ] Should `loaf install --to opencode` handle plugin registration automatically, or is manual config acceptable?

## Test Conditions

- [ ] A pre-tool hook (e.g., check-secrets) receives valid JSON on stdin when triggered via OpenCode
- [ ] A blocking hook (exit 2) prevents tool execution, or the limitation is documented
- [ ] `loaf install --to opencode` produces a working setup (hooks fire on tool use)
- [ ] OPENCODE.md exists in dist output with accurate setup instructions
