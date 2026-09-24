import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtemp, mkdir, readFile, realpath, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import test from 'node:test';

test('generated plugin preserves ordinary hook dispatch when registerTool is unavailable', async t => {
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.js');
  t.mock.method(console, 'warn', () => {});
  const handlers = new Map();
  const amp = {
    registerTool() { throw new Error('duplicate registration'); },
    on(event, handler) { handlers.set(event, handler); },
    helpers: { filePathFromURI: uri => fileURLToPath(uri), shellCommandFromToolCall: () => null },
    system: { workspaceRoot: pathToFileURL(process.cwd()) },
  };
  assert.doesNotThrow(() => initialize(amp));
  assert.deepEqual([...handlers.keys()], ['agent.start', 'tool.call', 'tool.result']);
  const result = await handlers.get('tool.call')({ toolUseID: 'probe', tool: 'unmatched_probe', input: {}, thread: { id: 'T-parent' } });
  assert.deepEqual(result, { action: 'allow' });
  await handlers.get('tool.result')({ toolUseID: 'probe', tool: 'unmatched_probe', input: {}, thread: { id: 'T-parent' }, status: 'done' });
});

test('generated plugin registers no delegate and agent.start injects no policy even in builtin mode', async () => {
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.js');
  const handlers = new Map();
  const tools = [];
  initialize({
    registerTool(definition) { tools.push(definition); },
    on(event, handler) { handlers.set(event, handler); },
    helpers: { shellCommandFromToolCall: () => null },
  });
  assert.deepEqual([...handlers.keys()], ['agent.start', 'tool.call', 'tool.result']);
  assert.equal(tools.length, 0);
  assert.equal(tools.some(tool => tool?.name === 'loaf_delegate' || tool?.name === 'delegate_implementation'), false);
  const prompt = 'Ship the bounded native contract.';
  const started = await handlers.get('agent.start')(
    { thread: { id: 'T-main' }, message: prompt, id: 'm1' },
    { thread: { id: 'T-main', agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'ultra' } }), parentThreadID: async () => null } },
  );
  assert.deepEqual(started, {});
  assert.equal(started?.message, undefined);
  const serialized = JSON.stringify(started ?? {});
  assert.equal(serialized.includes('loaf_delegate'), false);
  assert.equal(serialized.includes(prompt), false);
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
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.js');
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
  const { default: initialize } = await import('../../dist/amp/.amp/plugins/loaf.js');
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
  assert.deepEqual(await handlers.get('tool.call')({
    toolUseID: 'pre-push',
    tool: 'shell_command',
    input: { command: 'git push origin HEAD' },
    thread: { id: 'T-parent' },
  }), { action: 'allow' });
  log = await readLog();
  assert.ok(log.some(entry => entry.hook === 'ephemeral-provenance'));
  assert.equal(await realpath(log.find(entry => entry.hook === 'ephemeral-provenance').cwd), await realpath(shellDir));

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
