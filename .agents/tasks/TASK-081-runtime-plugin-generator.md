---
id: TASK-081
title: Create shared runtime plugin generator (OpenCode + Amp)
spec: SPEC-020
status: todo
priority: p1
dependencies: [TASK-076, TASK-077]
track: C
---

# TASK-081: Create shared runtime plugin generator (OpenCode + Amp)

Unified runtime plugin generator for TypeScript-plugin harnesses.

## Scope

Create `cli/lib/build/lib/hooks/runtime-plugin.ts`:

```typescript
interface RuntimePlatform {
  platform: 'opencode' | 'amp';
  header: string;
  events: { preTool: string; postTool: string; sessionStart: string; sessionEnd?: string; };
  toolNameAccessor: string;
  wrapExport: (body: string) => string;
  rejectPattern: (msgVar: string) => string;
  extraEvents?: (config: HooksConfig, srcDir: string) => string;
}
```

**Shared core:** `runHook()` (calls `loaf check`/`loaf session` as subprocess), `matchesTool()`, hook grouping by matcher, exit code interpretation (0=allow, 2=block, 1=error), `failClosed` handling.

**OpenCode adapter:** Maps `tool.execute.before/after`, `session.created/ended`. `extraEvents` handles `session.compacting` -> `loaf session log "compact: context compaction triggered"`.

**Rewrite OpenCode target:** Replace `generateHooks()`/`generateHooksJs()` in `opencode.ts` with call to shared `generateRuntimePlugin()`. Output becomes `hooks.ts` (TypeScript).

## Constraints

- Amp adapter created here but Amp target wiring is TASK-083
- OpenCode `hooks.js` -> `hooks.ts` filename change (document in release notes)
- `extraEvents` callback for platform-specific events beyond shared core
- Each runtime plugin calls `loaf check`/`loaf session` as subprocess, not inline logic

## Verification

- [ ] `loaf build --target opencode` produces valid `dist/opencode/plugins/hooks.ts`
- [ ] Plugin calls `loaf check` as subprocess
- [ ] Plugin handles `session.compacting` via `extraEvents`
- [ ] Generated TypeScript is valid
- [ ] `failClosed` handling in subprocess error path
- [ ] `npm run typecheck` passes
