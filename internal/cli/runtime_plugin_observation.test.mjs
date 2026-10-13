import assert from 'node:assert/strict';
import { mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL, fileURLToPath } from 'node:url';
import test from 'node:test';

for (const harness of ['amp', 'opencode']) {
  test(`${harness} hook subprocess identifies its active adapter despite inherited markers`, async t => {
    const dir = await mkdtemp(join(tmpdir(), 'loaf-adapter-origin-'));
    const log = join(dir, 'origin.jsonl');
    await writeFile(join(dir, 'loaf'), '#!/bin/sh\nprintf \'{"harness":"%s","phase":"%s"}\\n\' "$LOAF_HOOK_HARNESS" "$LOAF_HOOK_TYPE" >> "$LOAF_TEST_ORIGIN_LOG"\n', { mode: 0o755 });
    for (const [key, value] of Object.entries({ PATH: `${dir}:${process.env.PATH}`, LOAF_TEST_ORIGIN_LOG: log, CODEX_THREAD_ID: 'parent', CLAUDECODE: '1' })) {
      const old = process.env[key];
      process.env[key] = value;
      t.after(() => { if (old === undefined) delete process.env[key]; else process.env[key] = old; });
    }
    const { default: initialize } = await import(pathToFileURL(join(process.env.LOAF_TEST_PLUGIN_DIR, harness + '.mjs')));
    if (harness === 'amp') {
      const handlers = new Map();
      initialize({ on: (event, fn) => handlers.set(event, fn), helpers: { filePathFromURI: fileURLToPath, shellCommandFromToolCall: () => null }, system: { workspaceRoot: pathToFileURL(dir) } });
      assert.deepEqual(await handlers.get('tool.call')({ tool: 'shell_command', input: { command: 'pwd' } }), { action: 'allow' });
    } else {
      const hooks = await initialize({ client: {}, $: () => {} });
      await hooks['tool.execute.before']({ tool: 'bash' }, { args: { command: 'pwd' } });
    }
    assert.deepEqual(JSON.parse((await readFile(log, 'utf8')).trim()), { harness, phase: 'pre-tool' });
  });
}
