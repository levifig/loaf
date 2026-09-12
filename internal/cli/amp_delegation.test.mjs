import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtemp, mkdir, readFile, realpath, rm, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import test from 'node:test';
import { loafNativeModeContext, registerLoafDelegation } from './amp_delegation.ts';

test('generated plugin preserves normal hook dispatch when optional delegation registration fails', async t => {
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.ts');
  t.mock.method(console, 'warn', () => {});
  const handlers = new Map();
  const amp = {
    registerTool() { throw new Error('duplicate registration'); },
    on(event, handler) { handlers.set(event, handler); },
    helpers: { shellCommandFromToolCall: () => null },
  };
  assert.doesNotThrow(() => initialize(amp));
  assert.deepEqual([...handlers.keys()], ['agent.start', 'tool.call', 'tool.result']);
  const result = await handlers.get('tool.call')({ toolUseID: 'probe', tool: 'unmatched_probe', input: {}, thread: { id: 'T-parent' } });
  assert.deepEqual(result, { action: 'allow' });
  await handlers.get('tool.result')({ toolUseID: 'probe', tool: 'unmatched_probe', input: {}, thread: { id: 'T-parent' }, status: 'done' });
});

test('generated plugin injects native-mode policy through agent.start context', async t => {
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.ts');
  const handlers = new Map();
  initialize({
    registerTool() {},
    on(event, handler) { handlers.set(event, handler); },
    helpers: { shellCommandFromToolCall: () => null },
  });
  const prompt = 'Ship the bounded native contract.';
  const injected = await handlers.get('agent.start')(
    { thread: { id: 'T-main' }, message: prompt, id: 'm1' },
    { thread: { id: 'T-main', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'ultra' } }), parentThreadID: async () => null } },
  );
  assert.equal(injected.message.display, false);
  assert.match(injected.message.content, /loaf_delegate/);
  assert.equal(injected.message.content.includes(prompt), false);
});

test('generated startup hook uses workspace rather than plugin process cwd', async t => {
  const workspace = await mkdtemp(join(tmpdir(), 'loaf-amp-hook-ws-'));
  const binDir = await mkdtemp(join(tmpdir(), 'loaf-amp-hook-bin-'));
  const cwdFile = join(binDir, 'cwd');
  const previousPath = process.env.PATH;
  const previousCwdFile = process.env.LOAF_TEST_CWD_FILE;
  t.after(async () => {
    process.env.PATH = previousPath;
    if (previousCwdFile === undefined) delete process.env.LOAF_TEST_CWD_FILE;
    else process.env.LOAF_TEST_CWD_FILE = previousCwdFile;
    await rm(workspace, { recursive: true, force: true });
    await rm(binDir, { recursive: true, force: true });
  });
  await writeFile(
    join(binDir, 'loaf'),
    'printf \'%s\' "$PWD" > "$LOAF_TEST_CWD_FILE"; printf \'{"outcome":"current"}\\n\'\n',
    { mode: 0o755 },
  );
  process.env.PATH = `${binDir}:${previousPath ?? ''}`;
  process.env.LOAF_TEST_CWD_FILE = cwdFile;
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.ts');
  const handlers = new Map();
  initialize({
    registerTool() {},
    on(event, handler) { handlers.set(event, handler); },
    helpers: { filePathFromURI: uri => new URL(uri).pathname, shellCommandFromToolCall: () => null },
    system: { workspaceRoot: pathToFileURL(workspace) },
  });
  await handlers.get('agent.start')(
    { thread: { id: 'T-hook-workdir' }, message: 'probe', id: 'm-hook-workdir' },
    { thread: { id: 'T-hook-workdir', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'medium' } }), parentThreadID: async () => null } },
  );
  const loggedCwd = await readFile(cwdFile, 'utf8');
  assert.equal(loggedCwd, await realpath(workspace));
  assert.notEqual(loggedCwd, await realpath(process.cwd()));
});

test('generated plugin runs real pre-PR in helper shell dir and falls back to workspace', async t => {
  const warn = t.mock.method(console, 'warn', () => {});
  const previousPath = process.env.PATH;
  const previousHookLog = process.env.LOAF_TEST_HOOK_LOG;
  const workspace = await realpath(await mkdtemp(join(tmpdir(), 'loaf amp ws-')));
  const binDir = await mkdtemp(join(tmpdir(), 'loaf-amp-hook-log-'));
  const hookLog = join(binDir, 'hook-log');
  const shellDir = join(workspace, 'shell dir');
  const missingDir = join(workspace, 'missing changelog');
  await mkdir(shellDir);
  await mkdir(missingDir);
  t.after(async () => {
    if (previousPath === undefined) delete process.env.PATH;
    else process.env.PATH = previousPath;
    if (previousHookLog === undefined) delete process.env.LOAF_TEST_HOOK_LOG;
    else process.env.LOAF_TEST_HOOK_LOG = previousHookLog;
    await rm(workspace, { recursive: true, force: true });
    await rm(binDir, { recursive: true, force: true });
  });
  const changelog = '## [Unreleased]\n\n- Probe changelog entry\n';
  await writeFile(join(workspace, 'CHANGELOG.md'), changelog);
  await writeFile(join(shellDir, 'CHANGELOG.md'), changelog);
  const repoRoot = fileURLToPath(new URL('../..', import.meta.url));
  const realLoaf = join(binDir, 'real-loaf');
  execFileSync('go', ['build', '-o', realLoaf, './cmd/loaf'], { cwd: repoRoot, encoding: 'utf8' });
  await writeFile(
    join(binDir, 'loaf'),
    [
      '#!/bin/sh',
      'printf \'{"cwd":"%s","hook":"%s"}\\n\' "$PWD" "$LOAF_HOOK_ID" >> "$LOAF_TEST_HOOK_LOG"',
      'if [ "$LOAF_HOOK_ID" = "workflow-pre-pr" ]; then',
      `  exec ${JSON.stringify(realLoaf)} check --hook workflow-pre-pr`,
      'fi',
      'printf \'{"outcome":"current"}\\n\'',
      '',
    ].join('\n'),
    { mode: 0o755 },
  );
  await writeFile(
    join(binDir, 'cat'),
    [
      '#!/bin/sh',
      'printf \'{"cwd":"%s","hook":"%s"}\\n\' "$PWD" "$LOAF_HOOK_ID" >> "$LOAF_TEST_HOOK_LOG"',
      'exit 0',
      '',
    ].join('\n'),
    { mode: 0o755 },
  );
  process.env.PATH = `${binDir}:${previousPath ?? ''}`;
  process.env.LOAF_TEST_HOOK_LOG = hookLog;
  await writeFile(hookLog, '');
  const readLog = async () => {
    const body = await readFile(hookLog, 'utf8');
    return body.split('\n').filter(Boolean).map(line => JSON.parse(line));
  };
  const prePr = cwd => ({
    toolUseID: 'pre-pr',
    tool: 'shell_command',
    input: { command: "gh pr create --title 'fix: probe' --body 'probe'", cwd },
    thread: { id: 'T-parent' },
  });
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.ts');
  let helperDir;
  const handlers = new Map();
  initialize({
    registerTool() {},
    on(event, handler) { handlers.set(event, handler); },
    helpers: {
      filePathFromURI: uri => fileURLToPath(uri),
      shellCommandFromToolCall: event => ({ command: event.input.command, dir: helperDir }),
    },
    system: { workspaceRoot: pathToFileURL(workspace) },
  });

  helperDir = shellDir;
  assert.deepEqual(await handlers.get('tool.call')(prePr(missingDir)), { action: 'allow' });
  let log = await readLog();
  assert.ok(log.some(entry => entry.hook === 'workflow-pre-pr'));
  for (const entry of log) assert.equal(await realpath(entry.cwd), await realpath(shellDir));
  assert.notEqual(await realpath(shellDir), workspace);
  assert.notEqual(await realpath(shellDir), await realpath(process.cwd()));

  await writeFile(hookLog, '');
  helperDir = missingDir;
  const rejected = await handlers.get('tool.call')(prePr(shellDir));
  assert.equal(rejected.action, 'reject-and-continue');
  assert.match(rejected.message, /CHANGELOG\.md not found/);
  log = await readLog();
  assert.equal(await realpath(log.find(entry => entry.hook === 'workflow-pre-pr').cwd), await realpath(missingDir));

  await writeFile(hookLog, '');
  helperDir = 'shell dir';
  assert.deepEqual(await handlers.get('tool.call')(prePr(missingDir)), { action: 'allow' });
  log = await readLog();
  assert.equal(await realpath(log.find(entry => entry.hook === 'workflow-pre-pr').cwd), await realpath(shellDir));

  await writeFile(hookLog, '');
  helperDir = undefined;
  assert.deepEqual(await handlers.get('tool.call')(prePr(missingDir)), { action: 'allow' });
  log = await readLog();
  assert.equal(await realpath(log.find(entry => entry.hook === 'workflow-pre-pr').cwd), workspace);

  await writeFile(hookLog, '');
  await handlers.get('tool.result')({
    toolUseID: 'post-edit',
    tool: 'Edit',
    input: { path: join(workspace, 'CHANGELOG.md'), cwd: missingDir },
    thread: { id: 'T-parent' },
    status: 'done',
  });
  log = await readLog();
  assert.equal(log.length, 1);
  assert.equal(log[0].hook, 'kb-staleness-nudge');
  assert.equal(await realpath(log[0].cwd), workspace);

  await writeFile(hookLog, '');
  helperDir = shellDir;
  await handlers.get('tool.result')({
    toolUseID: 'post-merge',
    tool: 'shell_command',
    input: { command: 'gh pr merge', cwd: missingDir },
    thread: { id: 'T-parent' },
    status: 'done',
  });
  log = await readLog();
  assert.equal(log.length, 1);
  assert.equal(log[0].hook, 'workflow-post-merge');
  assert.equal(await realpath(log[0].cwd), await realpath(shellDir));

  await writeFile(hookLog, '');
  const missingHandlers = new Map();
  initialize({
    registerTool() {},
    on(event, handler) { missingHandlers.set(event, handler); },
    helpers: {
      filePathFromURI: uri => fileURLToPath(uri),
      shellCommandFromToolCall: event => ({ command: event.input.command, dir: helperDir }),
    },
  });
  const noWorkspacePre = await missingHandlers.get('tool.call')({
    toolUseID: 'edit-pre',
    tool: 'Edit',
    input: { path: join(workspace, 'CHANGELOG.md') },
    thread: { id: 'T-parent' },
  });
  assert.equal(noWorkspacePre.action, 'reject-and-continue');
  assert.match(noWorkspacePre.message, /Amp workspace is unavailable/);
  assert.equal((await readLog()).length, 0);

  await missingHandlers.get('agent.start')(
    { thread: { id: 'T-missing' }, message: 'probe', id: 'm-missing' },
    { thread: { id: 'T-missing', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'medium' } }), parentThreadID: async () => null } },
  );
  assert.equal((await readLog()).length, 0);
  assert.ok(warn.mock.calls.some(call => String(call.arguments[0]).includes('Managed-content reconcile skipped')));

  await missingHandlers.get('tool.result')({
    toolUseID: 'post-missing',
    tool: 'Edit',
    input: { path: join(workspace, 'CHANGELOG.md'), cwd: missingDir },
    thread: { id: 'T-parent' },
    status: 'done',
  });
  assert.equal((await readLog()).length, 0);
  assert.ok(warn.mock.calls.some(call => String(call.arguments[0]).includes('Post-hook kb-staleness-nudge skipped')));
});

const catalog = {
  builtinToolNames: ['Read', 'apply_patch'],
  models: [{ id: 'xai/grok-4.6', capabilities: { reasoning: true, efforts: [], tools: true, vision: true } }],
};

async function fixture(t, overrides = {}) {
  const root = await realpath(await mkdtemp(join(tmpdir(), 'loaf-amp-')));
  t.after(() => rm(root, { recursive: true, force: true }));
  let tool;
  let parentObserver;
  const calls = [];
  const child = {
    id: 'T-child',
    waitForResponse: async () => ({ role: 'assistant', content: [{ type: 'text', text: 'Evidence' }] }),
    appendUserMessage: async message => calls.push(['append', message]),
    cancel: async () => calls.push(['cancel']),
    state: { get: async () => 'idle' },
    ...overrides.child,
  };
  const amp = {
    system: { workspaceRoot: pathToFileURL(root), executor: { kind: 'local' } },
    helpers: {
      filePathFromURI: uri => new URL(uri.toString()).pathname,
      filesModifiedByToolCall: event => event.input.paths?.map(path => pathToFileURL(path)) ?? null,
    },
    registerTool: definition => { tool = definition; },
    createAgent: config => { calls.push(['agent', config]); return { createThread: async options => { calls.push(['thread', options]); return child; } }; },
    ...overrides.amp,
  };
  const guard = registerLoafDelegation(amp, overrides.command ?? (async (command, args, cwd) => {
    calls.push(['command', command, args, cwd]);
    return command === 'git' ? root : JSON.stringify(catalog);
  }));
  const ctx = {
    thread: {
      id: 'T-parent',
      agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'high' } }),
      parentThreadID: async () => null,
      state: { get: async () => 'running', subscribe: observer => { parentObserver = observer; return { unsubscribe: () => { calls.push(['unsubscribe']); parentObserver = undefined; } }; } },
      ...(overrides.ctx?.thread ?? {}),
    },
    ...Object.fromEntries(Object.entries(overrides.ctx ?? {}).filter(([key]) => key !== 'thread')),
  };
  const input = { role: 'implementation', native_ref: 'https://github.com/example/repo/issues/1', worktree: root, packet: 'Contract and task' };
  return { root, tool, guard, calls, child, ctx, input, amp, stopParent: state => parentObserver?.(state) };
}

test('native child is pinned Grok Fast with minimal tools and returns provenance, snapshot hash and turn evidence', async t => {
  const f = await fixture(t);
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(result.status, 'turn-complete');
  assert.equal(result.parent_thread_id, 'T-parent');
  assert.equal(result.child_thread_id, 'T-child');
  assert.match(result.packet_sha256, /^[a-f0-9]{64}$/);
  assert.equal(result.acceptance, 'main-agent-required');
  const agent = f.calls.find(([name]) => name === 'agent')[1];
  assert.deepEqual(agent.tools, ['Read', 'apply_patch']);
  assert.equal(agent.model, 'xai/grok-4.6');
  assert.deepEqual(agent.features, ['fast']);
  assert.equal(agent.reasoningEffort, undefined);
  assert.equal(agent.extends, undefined);
  assert.deepEqual(f.calls.find(([name]) => name === 'thread')[1], { parentThreadID: 'T-parent', executor: 'local', visibility: 'private' });
  assert.ok(await f.guard.check({ thread: { id: 'T-child' }, tool: 'Read', input: { path: join(f.root, 'source') } }));
});

test('review calls are rejected without spawning a child and point to native Oracle', async t => {
  const f = await fixture(t);
  assert.deepEqual(f.tool.inputSchema.properties.role.enum, ['implementation']);
  assert.equal(f.tool.inputSchema.properties.role.enum.includes('review'), false);
  const result = JSON.parse(await f.tool.execute({ ...f.input, role: 'review', packet: 'diff + exact source' }, f.ctx));
  assert.equal(result.status, 'incompatible');
  assert.match(result.message, /native Oracle/i);
  assert.equal(f.calls.some(([name]) => name === 'agent'), false);
  assert.equal(f.calls.some(([name]) => name === 'thread'), false);
  assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'turn-complete');
});

test('parent cancellation during preflight is interrupted rather than incompatible', async t => {
  let release;
  let started;
  const commandStarted = new Promise(resolve => { started = resolve; });
  const f = await fixture(t, { command: async (_command, _args, cwd) => {
    started();
    await new Promise(resolve => { release = resolve; });
    return cwd;
  } });
  const pending = f.tool.execute(f.input, f.ctx);
  await commandStarted;
  f.stopParent('idle');
  release();
  const result = JSON.parse(await pending);
  assert.equal(result.status, 'interrupted');
  assert.equal(result.child_thread_id, undefined);
  assert.equal(f.calls.some(([name]) => name === 'agent'), false);
});

test('invalid input and unsupported parent do not spawn', async t => {
  const f = await fixture(t);
  for (const input of [{ ...f.input, role: 'other' }, { ...f.input, packet: '' }, { ...f.input, worktree: '/' }]) {
    assert.equal(JSON.parse(await f.tool.execute(input, f.ctx)).status, 'incompatible');
  }
  const unsupported = await fixture(t, { ctx: { thread: { id: 'T-parent', agent: async () => ({ definition: { kind: 'agent-definition' } }) } } });
  assert.equal(JSON.parse(await unsupported.tool.execute(unsupported.input, unsupported.ctx)).status, 'incompatible');
  const deprecated = await fixture(t, { ctx: { thread: { id: 'T-parent', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'smart' } }) } } });
  assert.equal(JSON.parse(await deprecated.tool.execute(deprecated.input, deprecated.ctx)).status, 'incompatible');
  assert.equal(f.calls.some(([name]) => name === 'agent'), false);
  assert.equal(deprecated.calls.some(([name]) => name === 'agent'), false);
});

test('one writer and active guard reject shell, providers, unknown paths, symlink escapes, moves and Git metadata', async t => {
  let finish;
  let appended;
  const started = new Promise(resolve => { appended = resolve; });
  const f = await fixture(t, { child: {
    waitForResponse: () => new Promise(resolve => { finish = resolve; }),
    appendUserMessage: async () => appended(),
  } });
  const pending = f.tool.execute(f.input, f.ctx);
  await started;
  assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'incompatible');
  await symlink(tmpdir(), join(f.root, 'escape'));
  await symlink(join(tmpdir(), 'loaf-missing-target'), join(f.root, 'dangling'));
  await mkdir(join(f.root, '.git'));
  for (const event of [
    { tool: 'Bash', input: { command: 'echo bad' } },
    { tool: 'mcp_linear_create', input: {} },
    { tool: 'apply_patch', input: {} },
    { tool: 'Read', input: { path: join(f.root, 'escape', 'outside') } },
    { tool: 'Read', input: { path: join(f.root, '.git', 'config') } },
    { tool: 'Read', input: { path: join(f.root, 'dangling') } },
    { tool: 'Read', input: { path: 'relative.txt' } },
    { tool: 'Read', input: { path: pathToFileURL(join(f.root, 'okay')).toString() } },
    { tool: 'apply_patch', input: { paths: [join(f.root, 'okay'), '/outside'] } },
    { tool: 'apply_patch', input: { patchText: `*** Begin Patch\n*** Update File: /outside\n*** Move to: ${join(f.root, 'okay')}\n*** End Patch`, paths: [join(f.root, 'okay')] } },
    { tool: 'apply_patch', input: { patchText: `*** Begin Patch\n*** Update File: relative.txt\n*** Move to: ${join(f.root, 'okay')}\n*** End Patch`, paths: [join(f.root, 'okay')] } },
    { tool: 'apply_patch', input: { patchText: `*** Begin Patch\n*** Unknown: ${join(f.root, 'okay')}\n*** End Patch`, paths: [join(f.root, 'okay')] } },
  ]) assert.ok(await f.guard.check({ ...event, thread: { id: 'T-child' } }), JSON.stringify(event));
  await writeFile(join(f.root, 'okay'), 'source');
  assert.equal(await f.guard.check({ tool: 'Read', input: { path: join(f.root, 'okay') }, thread: { id: 'T-child' } }), undefined);
  assert.equal(await f.guard.check({ tool: 'apply_patch', input: { patchText: `*** Begin Patch\n*** Update File: ${join(f.root, 'okay')}\n@@\n-source\n+updated\n*** End Patch`, paths: [join(f.root, 'okay')] }, thread: { id: 'T-child' } }), undefined);
  assert.equal(await f.guard.check({ tool: 'other', input: {}, thread: { id: 'T-unrelated' } }), undefined);
  finish({ role: 'assistant', content: [{ type: 'text', text: 'done' }] });
  await pending;
});

test('wait failure cancels, reports uncertain state and retains writer lock', async t => {
  const f = await fixture(t, { child: { waitForResponse: async () => { throw new Error('timed out'); }, state: { get: async () => 'running' } } });
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(result.status, 'uncertain');
  assert.equal(result.child_thread_id, 'T-child');
  assert.ok(f.calls.some(([name]) => name === 'cancel'));
  assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'incompatible');
});

test('custom model routing, missing API, missing required tools and remote executors fail closed', async t => {
  for (const overrides of [
    { ctx: { thread: { id: 'T-parent', agent: async () => ({ definition: { kind: 'agent-definition', extends: 'high', model: 'operator/choice' } }) } } },
    { amp: { createAgent: undefined } },
    { amp: { system: { executor: { kind: 'orb' } } } },
    { command: async (command, args, cwd) => command === 'git' ? cwd : JSON.stringify({ builtinToolNames: ['Read'] }) },
    { command: async (command, args, cwd) => command === 'git' ? cwd : 'malformed' },
  ]) {
    const f = await fixture(t, overrides);
    assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'incompatible');
    assert.equal(f.calls.some(([name]) => name === 'agent'), false);
  }
});

test('sequential implementation calls spawn a fresh Grok child each time', async t => {
  let n = 0;
  const calls = [];
  const f = await fixture(t, { amp: { createAgent: config => {
    calls.push(['agent', config]);
    return { createThread: async options => {
      n += 1;
      const id = `T-child-${n}`;
      calls.push(['thread', options, id]);
      return {
        id,
        waitForResponse: async () => ({ role: 'assistant', content: [{ type: 'text', text: id }] }),
        appendUserMessage: async message => calls.push(['append', message]),
        cancel: async () => calls.push(['cancel', id]),
        state: { get: async () => 'idle' },
      };
    } };
  } } });
  const first = JSON.parse(await f.tool.execute(f.input, f.ctx));
  const second = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(first.status, 'turn-complete');
  assert.equal(second.status, 'turn-complete');
  assert.equal(first.child_thread_id, 'T-child-1');
  assert.equal(second.child_thread_id, 'T-child-2');
  assert.equal(calls.filter(([name]) => name === 'agent').length, 2);
});

test('unknown, custom, and nested callers cannot use the implementation route', async t => {
  const cases = [
    { ctx: { thread: { id: 'T-parent', agent: async () => ({ definition: { kind: 'agent-definition', model: 'xai/grok-4.6' } }), parentThreadID: async () => null } } },
    { ctx: { thread: { id: 'T-nested', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'high' } }), parentThreadID: async () => 'T-parent' } } },
    { ctx: { thread: { id: 'T-parent', agent: undefined, parentThreadID: async () => null } } },
    { ctx: { thread: { id: 'T-parent', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'high' } }), parentThreadID: undefined } } },
  ];
  for (const overrides of cases) {
    const f = await fixture(t, overrides);
    const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
    assert.equal(result.status, 'incompatible');
    assert.equal(f.calls.some(([name]) => name === 'agent'), false);
  }
});

test('Grok catalog mismatch and inherited-model substitution fail closed', async t => {
  for (const overrides of [
    { command: async (command, args, cwd) => command === 'git' ? cwd : JSON.stringify({ builtinToolNames: ['Read', 'apply_patch'], models: [{ id: 'openai/gpt-6-astra', capabilities: { reasoning: true, efforts: [], tools: true } }] }) },
    { command: async (command, args, cwd) => command === 'git' ? cwd : JSON.stringify({ builtinToolNames: ['Read', 'apply_patch'], models: [{ id: 'xai/grok-4.6', capabilities: { reasoning: true, efforts: [], tools: false } }] }) },
    { command: async (command, args, cwd) => command === 'git' ? cwd : JSON.stringify({ builtinToolNames: ['Read'], models: [{ id: 'xai/grok-4.6', capabilities: { reasoning: true, efforts: [], tools: true } }] }) },
  ]) {
    const f = await fixture(t, overrides);
    assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'incompatible');
    assert.equal(f.calls.some(([name]) => name === 'agent'), false);
  }
});

test('installed nested Grok capabilities catalog reaches createAgent', async t => {
  const installedCatalog = {
    builtinToolNames: ['Read', 'apply_patch', 'oracle', 'shell_command'],
    models: [{
      provider: 'xai',
      name: 'grok-4.6',
      id: 'xai/grok-4.6',
      displayName: 'Grok 4.6',
      contextWindow: 500000,
      maxOutputTokens: 32000,
      capabilities: { reasoning: true, efforts: [], tools: true, vision: true },
    }],
  };
  let createConfig;
  const f = await fixture(t, {
    command: async (command, _args, cwd) => command === 'git' ? cwd : JSON.stringify(installedCatalog),
    amp: {
      createAgent: config => {
        createConfig = config;
        throw new Error('PRECHECK_PASSED_NO_AGENT_STARTED');
      },
    },
  });
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(createConfig?.model, 'xai/grok-4.6');
  assert.deepEqual(createConfig?.features, ['fast']);
  assert.equal(createConfig?.reasoningEffort, undefined);
  assert.equal(result.status, 'incompatible');
  assert.match(result.message, /PRECHECK_PASSED_NO_AGENT_STARTED/);
  assert.equal(result.message.includes('Grok 4.6 Fast'), false);
});

test('advertised Grok efforts are accepted when no reasoningEffort is selected', async t => {
  const f = await fixture(t, {
    command: async (command, _args, cwd) => command === 'git' ? cwd : JSON.stringify({
      builtinToolNames: ['Read', 'apply_patch'],
      models: [{ id: 'xai/grok-4.6', capabilities: { reasoning: true, efforts: ['low', 'high'], tools: true } }],
    }),
  });
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(result.status, 'turn-complete');
  const agent = f.calls.find(([name]) => name === 'agent')[1];
  assert.equal(agent.model, 'xai/grok-4.6');
  assert.deepEqual(agent.features, ['fast']);
  assert.equal(agent.reasoningEffort, undefined);
});

test('policy injects into currently parentless builtin modes and never mutates the user prompt', async t => {
  const start = (event, ctx) => loafNativeModeContext(event, ctx);
  const prompt = 'Ship the bounded native contract.';
  const parentless = { id: 'T-main', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'medium' } }), parentThreadID: async () => null };
  const injected = await start({ thread: { id: 'T-main' }, message: prompt, id: 'm1' }, { thread: parentless });
  assert.equal(injected.message.display, false);
  assert.match(injected.message.content, /loaf_delegate/);
  assert.match(injected.message.content, /native Oracle/);
  assert.equal(injected.message.content.includes(prompt), false);
  for (const mode of ['low', 'medium', 'high', 'ultra']) {
    const result = await start({ thread: { id: 'T-main' }, message: prompt, id: mode }, {
      thread: { id: 'T-main', agent: async () => ({ definition: { kind: 'builtin-agent', mode } }), parentThreadID: async () => null },
    });
    assert.match(result.message.content, /loaf_delegate/);
  }
  const excluded = [
    { id: 'T-child', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'high' } }), parentThreadID: async () => 'T-parent' },
    { id: 'T-custom', agent: async () => ({ definition: { kind: 'agent-definition', model: 'xai/grok-4.6' } }), parentThreadID: async () => null },
    { id: 'T-unknown', agent: async () => { throw new Error('agent unavailable'); }, parentThreadID: async () => null },
    { id: 'T-missing-parent', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'high' } }) },
  ];
  for (const thread of excluded) {
    const result = await start({ thread: { id: thread.id }, message: prompt, id: thread.id }, { thread });
    assert.deepEqual(result, {});
  }
});

test('child instructions assign test authorship to Grok and command execution to main', async t => {
  const f = await fixture(t);
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(result.status, 'turn-complete');
  const agent = f.calls.find(([name]) => name === 'agent')[1];
  assert.deepEqual(agent.tools, ['Read', 'apply_patch']);
  assert.equal(result.acceptance, 'main-agent-required');
  assert.equal(f.calls.some(([name, command]) => name === 'command' && command !== 'git' && command !== 'amp'), false);
  // Instruction text records the author/run split. Matching it does not prove
  // the child followed those instructions.
  const instructions = agent.instructions;
  assert.match(instructions, /Author and update code plus necessary regression tests/);
  assert.match(instructions, /main agent executes test, build, and lint commands and evaluates results/);
  assert.match(instructions, /replace only inherited self-log, provider, and test-execution workflow/);
  assert.match(instructions, /do not license ignoring repository safety or style/i);
  assert.match(instructions, /Do not recursively delegate or spawn another agent/);
  assert.doesNotMatch(instructions, /owns the journal, tracker, and tests/);
});

test('own custom Grok with a null reported parent remains excluded and cannot recurse', async t => {
  const grokChild = {
    id: 'T-grok',
    agent: async () => ({ definition: { kind: 'agent-definition', name: 'loaf-implementation-agent', model: 'xai/grok-4.6' } }),
    parentThreadID: async () => null,
  };
  const policy = await loafNativeModeContext({ thread: { id: grokChild.id }, message: 'implement', id: 'm-grok' }, { thread: grokChild });
  assert.deepEqual(policy, {});
  const f = await fixture(t, { ctx: { thread: grokChild } });
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(result.status, 'incompatible');
  assert.match(result.message, /currently parentless builtin|custom/i);
  assert.equal(f.calls.some(([name]) => name === 'agent'), false);
  assert.equal(f.calls.some(([name]) => name === 'thread'), false);
});

test('null reported parent on a builtin follows observable API and is not ancestry attestation', async t => {
  const prompt = 'Ship the bounded native contract.';
  const orphanedBuiltin = {
    id: 'T-orphan-builtin',
    agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'high' } }),
    parentThreadID: async () => null,
  };
  // Observable API: currently parentless builtin. An orphaned builtin that
  // reports the same metadata would also qualify; this is not ancestry proof.
  const injected = await loafNativeModeContext({ thread: { id: orphanedBuiltin.id }, message: prompt, id: 'm-orphan' }, { thread: orphanedBuiltin });
  assert.match(injected.message.content, /loaf_delegate/);
  const f = await fixture(t, { ctx: { thread: orphanedBuiltin } });
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(result.status, 'turn-complete');
  assert.equal(result.child_thread_id, 'T-child');
  const agent = f.calls.find(([name]) => name === 'agent')[1];
  assert.deepEqual(agent.tools, ['Read', 'apply_patch']);
  assert.equal(agent.model, 'xai/grok-4.6');
});

test('waiter is established before submission, and append failure cannot escape guard sealing', async t => {
  let waiting = false;
  const f = await fixture(t, { child: {
    waitForResponse: async () => { waiting = true; throw new Error('wait failed'); },
    appendUserMessage: async () => { assert.equal(waiting, true); throw new Error('append failed'); },
  } });
  const result = JSON.parse(await f.tool.execute(f.input, f.ctx));
  assert.equal(result.status, 'interrupted');
  assert.match(result.message, /append failed/);
  assert.ok(await f.guard.check({ thread: { id: 'T-child' }, tool: 'Read', input: { path: join(f.root, 'file') } }));
});

test('parent cancellation during creation cancels child without appending work', async t => {
  let created;
  let creating;
  const creationStarted = new Promise(resolve => { creating = resolve; });
  const f = await fixture(t, { amp: { createAgent: () => ({ createThread: () => { creating(); return new Promise(resolve => { created = resolve; }); } }) } });
  const pending = f.tool.execute(f.input, f.ctx);
  await creationStarted;
  f.stopParent('idle');
  created(f.child);
  const result = JSON.parse(await pending);
  assert.equal(result.status, 'interrupted');
  assert.equal(f.calls.some(([name]) => name === 'append'), false);
  assert.ok(f.calls.some(([name]) => name === 'cancel'));
  assert.ok(f.calls.some(([name]) => name === 'unsubscribe'));
});

test('parent cancellation seals running child and cannot become turn-complete', async t => {
  let finish;
  let appended;
  const started = new Promise(resolve => { appended = resolve; });
  const f = await fixture(t, { child: {
    waitForResponse: () => new Promise(resolve => { finish = resolve; }),
    appendUserMessage: async () => appended(),
  } });
  const pending = f.tool.execute(f.input, f.ctx);
  await started;
  await new Promise(resolve => setImmediate(resolve));
  f.stopParent('idle');
  assert.ok(await f.guard.check({ thread: { id: 'T-child' }, tool: 'Read', input: { path: join(f.root, 'file') } }));
  finish({ role: 'assistant', content: [{ type: 'text', text: 'late reply' }] });
  assert.equal(JSON.parse(await pending).status, 'interrupted');
  assert.ok(f.calls.some(([name]) => name === 'cancel'));
});

test('failed parent-triggered cancellation retains uncertain writer lock', async t => {
  let appended;
  const started = new Promise(resolve => { appended = resolve; });
  const f = await fixture(t, { child: {
    waitForResponse: () => new Promise(() => {}),
    appendUserMessage: async () => appended(),
    cancel: async () => { throw new Error('cancel unavailable'); },
    state: { get: async () => 'running' },
  } });
  const pending = f.tool.execute(f.input, f.ctx);
  await started;
  f.stopParent('error');
  assert.equal(JSON.parse(await pending).status, 'uncertain');
  assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'incompatible');
});

test('pending submission remains owned and is cancelled again when it settles after parent stop', async t => {
  let settleSubmission;
  let submitting;
  let state = 'idle';
  let cancelCount = 0;
  const submissionStarted = new Promise(resolve => { submitting = resolve; });
  const f = await fixture(t, { child: {
    waitForResponse: () => new Promise(() => {}),
    appendUserMessage: () => { submitting(); return new Promise(resolve => { settleSubmission = () => { state = 'running'; resolve(); }; }); },
    cancel: async () => { cancelCount++; state = 'idle'; },
    state: { get: async () => state },
  } });
  const pending = f.tool.execute(f.input, f.ctx);
  await submissionStarted;
  f.stopParent('idle');
  const result = JSON.parse(await pending);
  assert.equal(result.status, 'uncertain');
  assert.equal(result.submission_pending, true);
  assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'incompatible');
  settleSubmission();
  await new Promise(resolve => setImmediate(resolve));
  assert.ok(cancelCount >= 2);
  assert.equal(state, 'idle');
  assert.equal(JSON.parse(await f.tool.execute(f.input, f.ctx)).status, 'incompatible');
});
