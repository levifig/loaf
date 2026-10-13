import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { copyFile, mkdtemp, mkdir, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { delimiter, join } from 'node:path';
import { pathToFileURL } from 'node:url';
import test from 'node:test';

const { default: initialize } = await import(pathToFileURL(process.env.LOAF_OPENCODE_PLUGIN_PATH));
const { default: initializeProbes } = await import(pathToFileURL(process.env.LOAF_OPENCODE_PROBE_PATH));

function preserveEnv(t, keys) {
  const previous = Object.fromEntries(keys.map(key => [key, process.env[key]]));
  t.after(() => {
    for (const key of keys) {
      if (previous[key] === undefined) delete process.env[key];
      else process.env[key] = previous[key];
    }
  });
  return previous;
}

test('generated OpenCode plugin enforces provenance and infrastructure failures', async t => {
  const root = await mkdtemp(join(tmpdir(), 'loaf-opencode-provenance-'));
  const bin = join(root, 'bin');
  const calls = join(root, 'calls');
  await mkdir(bin);
  await writeFile(join(bin, 'loaf'), [
    '#!/bin/sh',
    'printf "%s\\n" "$*" >> "$LOAF_TEST_CALLS"',
    'if [ "$*" = "check --hook ephemeral-provenance" ]; then',
    '  payload="$(cat)"',
    '  case "$payload" in *"git push origin coverage"*) ;; *) echo "missing push payload" >&2; exit 3 ;; esac',
    '  echo "untrack ephemeral reports before pushing" >&2',
    '  exit "${LOAF_TEST_PROVENANCE_EXIT:-0}"',
    'fi',
    'exit 0',
    '',
  ].join('\n'), { mode: 0o755 });
  const previous = preserveEnv(t, ['PATH', 'LOAF_TEST_CALLS', 'LOAF_TEST_PROVENANCE_EXIT']);
  const warnings = [];
  t.mock.method(console, 'warn', (...args) => warnings.push(args.join(' ')));
  process.env.PATH = bin + delimiter + previous.PATH;
  process.env.LOAF_TEST_CALLS = calls;
  delete process.env.LOAF_TEST_PROVENANCE_EXIT;
  const handlers = await initialize({ client: {} });
  const probes = await initializeProbes({ client: {} });
  const before = (tool = 'bash', command = 'git push origin coverage', plugin = handlers) =>
    plugin['tool.execute.before']({ tool }, { args: { command } });

  await t.test('production git-push hook dispatches native command and allows exit 0', async () => {
    await before();
    assert.equal((await readFile(calls, 'utf8')).trim().split('\n').filter(call => call === 'check --hook ephemeral-provenance').length, 1);
  });
  await t.test('production git-push hook preserves exit 2 rejection', async () => {
    process.env.LOAF_TEST_PROVENANCE_EXIT = '2';
    await assert.rejects(before(), /untrack ephemeral reports before pushing/);
  });
  await t.test('production hook blocks unexpected checker failure', async () => {
    process.env.LOAF_TEST_PROVENANCE_EXIT = '3';
    await assert.rejects(before(), /fail-closed hook ephemeral-provenance.*exit 3/i);
  });
  await t.test('non-push tools do not dispatch provenance', async () => {
    const previousCalls = await readFile(calls, 'utf8');
    process.env.LOAF_TEST_PROVENANCE_EXIT = '0';
    await before('bash', 'git status');
    assert.equal((await readFile(calls, 'utf8')).slice(previousCalls.length).includes('ephemeral-provenance'), false);
  });
  await t.test('missing dispatch blocks with actionable diagnostic', async () => {
    await assert.rejects(before('MissingDispatch', '', probes), /fail-closed hook missing-dispatch.*No command or script/i);
  });
  await t.test('signal, timeout, and unexpected exit block', async () => {
    for (const tool of ['Signal', 'Timeout', 'Unexpected']) {
      await assert.rejects(before(tool, '', probes), /fail-closed hook .*could not prove/i);
    }
  });
  await t.test('timeout blocks even when the shell catches SIGTERM and exits 0', async () => {
    await assert.rejects(before('TimeoutZeroExit', '', probes), /fail-closed hook timeout-zero-exit.*could not prove/i);
  });
  await t.test('missing runtime and spawn error block', async () => {
    process.env.PATH = '/usr/bin:/bin';
    await assert.rejects(before(), /not found|could not prove/i);
    process.env.PATH = root;
    await assert.rejects(before('SpawnError', '', probes), /ENOENT/);
    process.env.PATH = bin + delimiter + previous.PATH;
  });
  await t.test('advisory failure warns and allows; explicit rejection still blocks', async () => {
    warnings.length = 0;
    await before('Advisory', '', probes);
    assert.equal(warnings.some(warning => warning.includes('advisory') && warning.includes('exit 3')), true);
    await assert.rejects(before('AdvisoryReject', '', probes), /policy rejection/);
  });
});

test('production plugin and real native checker allow clean push and refuse tracked ephemeral Markdown', async t => {
  const root = await mkdtemp(join(tmpdir(), 'loaf-opencode-native-'));
  const bin = join(root, 'bin');
  const repo = join(root, 'repo');
  const home = join(root, 'home');
  await Promise.all([mkdir(bin), mkdir(repo), mkdir(home)]);
  await copyFile(process.env.LOAF_OPENCODE_NATIVE_BINARY, join(bin, 'loaf'));
  const previous = preserveEnv(t, ['PATH', 'HOME', 'XDG_CONFIG_HOME', 'XDG_DATA_HOME', 'LOAF_DB', 'GIT_CONFIG_GLOBAL', 'GIT_CONFIG_NOSYSTEM']);
  process.env.PATH = bin + delimiter + previous.PATH;
  process.env.HOME = home;
  process.env.XDG_CONFIG_HOME = join(home, 'config');
  process.env.XDG_DATA_HOME = join(home, 'data');
  process.env.LOAF_DB = join(root, 'state.sqlite');
  process.env.GIT_CONFIG_GLOBAL = '/dev/null';
  process.env.GIT_CONFIG_NOSYSTEM = '1';
  execFileSync('git', ['init', '-q'], { cwd: repo });
  const previousCwd = process.cwd();
  process.chdir(repo);
  t.after(() => process.chdir(previousCwd));
  const handlers = await initialize({ client: {} });
  const before = () => handlers['tool.execute.before']({ tool: 'bash' }, { args: { command: 'git push origin coverage' } });
  await before();
  await mkdir(join(repo, '.agents', 'tasks'), { recursive: true });
  await writeFile(join(repo, '.agents', 'tasks', 'TASK-001-example.md'), '# Ephemeral example\n');
  execFileSync('git', ['add', '.agents/tasks/TASK-001-example.md'], { cwd: repo });
  await assert.rejects(before(), /git rm --cached \.agents\/tasks\/TASK-001-example\.md/);
});
