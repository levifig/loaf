import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const source = join(dirname(fileURLToPath(import.meta.url)), '../../content/amp/plugins/loaf-modes.ts');

test('loaf-modes compatibility plugin is inert and points routing at loaf.ts', async () => {
  const body = await readFile(source, 'utf8');
  assert.match(body, /Managed Amp routing now lives in loaf\.ts/);
  assert.match(body, /export const description = '/);
  assert.match(body, /export default function \(\) \{\}/);
  for (const token of ['registerAgentMode', 'registerTool', 'createAgent', 'amp.on', '@amp-agent-mode', 'delegate_implementation', 'delegate_review', 'consult_oracle', 'loaf-medium', 'loaf-ultra']) {
    assert.equal(body.includes(token), false, token);
  }
});

test('inert loaf-modes plugin default export is a no-op', async () => {
  const { default: initialize, description } = await import('../../content/amp/plugins/loaf-modes.ts');
  assert.match(description, /loaf\.ts/);
  const amp = {
    registerTool() { throw new Error('must not register'); },
    registerAgentMode() { throw new Error('must not register'); },
    createAgent() { throw new Error('must not register'); },
    on() { throw new Error('must not register'); },
  };
  assert.doesNotThrow(() => initialize(amp));
});
