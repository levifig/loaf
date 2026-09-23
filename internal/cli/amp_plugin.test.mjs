import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { delimiter, join } from 'node:path';
import { pathToFileURL } from 'node:url';
import test from 'node:test';

const pluginPath = process.env.LOAF_AMP_PLUGIN_PATH;
if (!pluginPath) throw new Error('LOAF_AMP_PLUGIN_PATH is required');
const { default: initialize } = await import(pathToFileURL(pluginPath));

const agentContext = {
  thread: {
    id: 'T-main',
    agent: async () => ({ definition: { kind: 'builtin-agent', mode: 'medium' } }),
    parentThreadID: async () => null,
  },
};

function initializePlugin(workspace) {
  const handlers = new Map();
  initialize({
    registerTool() {},
    on(event, handler) { handlers.set(event, handler); },
    helpers: {
      filePathFromURI: uri => new URL(uri).pathname,
      shellCommandFromToolCall: () => null,
    },
    system: { workspaceRoot: pathToFileURL(workspace) },
  });
  return handlers;
}

function toolEvent(tool = 'Unmatched') {
  return { toolUseID: `tool-${tool}`, tool, input: {}, thread: { id: 'T-main' } };
}

async function writePinnedConsumer(workspace, withBootstrap = true) {
  const agents = join(workspace, '.agents');
  await mkdir(agents, { recursive: true });
  await writeFile(join(agents, 'loaf-orb.pin'), 'schema=1\n');
  if (withBootstrap) {
    await writeFile(
      join(agents, 'loaf-orb-bootstrap.sh'),
      [
        '#!/bin/sh',
        'if [ "${LOAF_TEST_AGENT_CHECK:-ready}" = "missing" ]; then',
        '  exec loaf-command-that-does-not-exist agent-check',
        'fi',
        'exec loaf agent-check',
        '',
      ].join('\n'),
      { mode: 0o755 },
    );
  }
}

test('generated Amp plugin gates pinned consumers and applies fail-closed hook semantics', async t => {
  const fixtureRoot = await mkdtemp(join(tmpdir(), 'loaf-amp-plugin-'));
  const binDir = join(fixtureRoot, 'bin');
  const callLog = join(fixtureRoot, 'calls');
  await mkdir(binDir);
  await writeFile(
    join(binDir, 'loaf'),
    [
      '#!/bin/sh',
      'printf "%s\\n" "$*" >> "$LOAF_TEST_CALL_LOG"',
      'case "$1" in',
      '  agent-check)',
      '    if [ "${LOAF_TEST_AGENT_CHECK:-ready}" = "not-ready" ]; then echo "installed Loaf does not match pin" >&2; exit 2; fi',
      '    exit 0',
      '    ;;',
      '  harness)',
      '    printf \'{"outcome":"current"}\\n\'',
      '    exit 0',
      '    ;;',
      '  fail-closed-2)',
      '    echo "policy says no" >&2',
      '    exit 2',
      '    ;;',
      '  fail-closed-3)',
      '    echo "unexpected hook failure" >&2',
      '    exit 3',
      '    ;;',
      '  fail-closed-signal)',
      '    kill -TERM $$',
      '    ;;',
      '  fail-closed-timeout)',
      '    exec sleep 5',
      '    ;;',
      '  advisory)',
      '    echo "advisory failure" >&2',
      '    exit 3',
      '    ;;',
      'esac',
      'exit 0',
      '',
    ].join('\n'),
    { mode: 0o755 },
  );

  const previous = {
    path: process.env.PATH,
    mode: process.env.LOAF_TEST_AGENT_CHECK,
    log: process.env.LOAF_TEST_CALL_LOG,
  };
  const warnings = [];
  const originalWarn = console.warn;
  process.env.PATH = `${binDir}${delimiter}${previous.path ?? ''}`;
  process.env.LOAF_TEST_CALL_LOG = callLog;
  console.warn = (...args) => warnings.push(args.map(String).join(' '));
  t.after(async () => {
    console.warn = originalWarn;
    if (previous.path === undefined) delete process.env.PATH;
    else process.env.PATH = previous.path;
    if (previous.mode === undefined) delete process.env.LOAF_TEST_AGENT_CHECK;
    else process.env.LOAF_TEST_AGENT_CHECK = previous.mode;
    if (previous.log === undefined) delete process.env.LOAF_TEST_CALL_LOG;
    else process.env.LOAF_TEST_CALL_LOG = previous.log;
    await rm(fixtureRoot, { recursive: true, force: true });
  });

  await t.test('agent-check exit 127 blocks the first tool call', async () => {
    const workspace = join(fixtureRoot, 'missing-loaf');
    await writePinnedConsumer(workspace);
    process.env.LOAF_TEST_AGENT_CHECK = 'missing';
    const handlers = initializePlugin(workspace);
    await handlers.get('agent.start')({ thread: { id: 'T-main' }, message: 'probe', id: 'm1' }, agentContext);
    const result = await handlers.get('tool.call')(toolEvent());
    assert.equal(result.action, 'reject-and-continue');
    assert.match(result.message, /not ready/i);
    assert.match(result.message, /not found|exited? 127/i);
  });

  await t.test('not-ready state blocks the first tool and retries agent-check', async () => {
    const workspace = join(fixtureRoot, 'not-ready');
    await writePinnedConsumer(workspace);
    await writeFile(callLog, '');
    process.env.LOAF_TEST_AGENT_CHECK = 'not-ready';
    const handlers = initializePlugin(workspace);
    await handlers.get('agent.start')({ thread: { id: 'T-main' }, message: 'probe', id: 'm2' }, agentContext);
    const result = await handlers.get('tool.call')(toolEvent());
    assert.equal(result.action, 'reject-and-continue');
    assert.match(result.message, /agent-check exited 2/);
    assert.match(result.message, /setup or resume lifecycle/);
    assert.equal(result.message.includes(fixtureRoot), false);
    assert.equal(result.message.includes('installed Loaf does not match pin'), false);
    const calls = (await readFile(callLog, 'utf8')).trim().split('\n');
    assert.deepEqual(calls, ['agent-check', 'agent-check']);
  });

  await t.test('ready state is cached for the session and allows tools', async () => {
    const workspace = join(fixtureRoot, 'ready');
    await writePinnedConsumer(workspace);
    await writeFile(callLog, '');
    process.env.LOAF_TEST_AGENT_CHECK = 'ready';
    const handlers = initializePlugin(workspace);
    await handlers.get('agent.start')({ thread: { id: 'T-main' }, message: 'probe', id: 'm3' }, agentContext);
    assert.deepEqual(await handlers.get('tool.call')(toolEvent()), { action: 'allow' });
    const calls = (await readFile(callLog, 'utf8')).trim().split('\n');
    assert.equal(calls.filter(call => call === 'agent-check').length, 1);
  });

  await t.test('pin without bootstrap blocks', async () => {
    const workspace = join(fixtureRoot, 'missing-bootstrap');
    await writePinnedConsumer(workspace, false);
    process.env.LOAF_TEST_AGENT_CHECK = 'ready';
    const handlers = initializePlugin(workspace);
    const result = await handlers.get('tool.call')(toolEvent());
    assert.equal(result.action, 'reject-and-continue');
    assert.match(result.message, /missing \.agents\/loaf-orb-bootstrap\.sh/);
  });

  await t.test('unpinned projects retain existing allow behavior', async () => {
    const workspace = join(fixtureRoot, 'unpinned');
    await mkdir(workspace);
    const handlers = initializePlugin(workspace);
    assert.deepEqual(await handlers.get('tool.call')(toolEvent()), { action: 'allow' });
  });

  await t.test('exit 2 preserves the fail-closed hook message', async () => {
    const workspace = join(fixtureRoot, 'hook-exit-2');
    await mkdir(workspace);
    const result = await initializePlugin(workspace).get('tool.call')(toolEvent('FailClosed2'));
    assert.deepEqual(result, { action: 'reject-and-continue', message: 'policy says no' });
  });

  await t.test('exit 2 remains a policy rejection for advisory hooks', async () => {
    const workspace = join(fixtureRoot, 'hook-advisory-exit-2');
    await mkdir(workspace);
    const result = await initializePlugin(workspace).get('tool.call')(toolEvent('Advisory2'));
    assert.deepEqual(result, { action: 'reject-and-continue', message: 'policy says no' });
  });

  await t.test('unexpected nonzero rejects for fail-closed hooks', async () => {
    const workspace = join(fixtureRoot, 'hook-exit-3');
    await mkdir(workspace);
    const result = await initializePlugin(workspace).get('tool.call')(toolEvent('FailClosed3'));
    assert.equal(result.action, 'reject-and-continue');
    assert.match(result.message, /fail-closed hook test-fail-closed-3 could not prove/i);
    assert.match(result.message, /unexpected hook failure/);
  });

  await t.test('signal and timeout reject for fail-closed hooks', async () => {
    for (const [tool, hook] of [['FailClosedSignal', 'test-fail-closed-signal'], ['FailClosedTimeout', 'test-fail-closed-timeout']]) {
      const workspace = join(fixtureRoot, tool);
      await mkdir(workspace);
      const result = await initializePlugin(workspace).get('tool.call')(toolEvent(tool));
      assert.equal(result.action, 'reject-and-continue');
      assert.match(result.message, new RegExp(`fail-closed hook ${hook} could not prove`, 'i'));
    }
  });

  await t.test('non-fail-closed hook warns and allows', async () => {
    const workspace = join(fixtureRoot, 'hook-advisory');
    await mkdir(workspace);
    warnings.length = 0;
    const result = await initializePlugin(workspace).get('tool.call')(toolEvent('Advisory'));
    assert.deepEqual(result, { action: 'allow' });
    assert.ok(warnings.some(message => message.includes('Advisory hook test-advisory') && message.includes('advisory failure')));
  });
});
