import type { PluginAPI as DelegationAPI, PluginToolContext, PluginToolDefinition, PluginThread, Subscription, ToolCallEvent } from '@ampcode/plugin';
import { execFile as delegationExecFile } from 'node:child_process';
import { createHash } from 'node:crypto';
import { lstat, realpath } from 'node:fs/promises';
import { dirname as delegationDirname, isAbsolute, relative, resolve, sep } from 'node:path';
import { promisify as delegationPromisify } from 'node:util';

const delegationExec = delegationPromisify(delegationExecFile);
type DelegationCommand = (command: string, args: string[], cwd?: string) => Promise<string>;
type DelegationGuard = { root: string; sealed: boolean };
type DelegationToolCall = Pick<ToolCallEvent, 'toolUseID' | 'tool' | 'input'> & { thread: { id: string } };
const nativeBuiltinModes = ['low', 'medium', 'high', 'ultra'] as const;
const grokModel = 'xai/grok-4.6';
const implementationTools = ['Read', 'apply_patch'] as const;
const implementationInstructions = 'These child instructions replace only inherited self-log, provider, and test-execution workflow that this child cannot run. They do not license ignoring repository safety or style. Respect repository constraints. Author and update code plus necessary regression tests. The main agent executes test, build, and lint commands and evaluates results. Do not recursively delegate or spawn another agent. Implement only the supplied bounded native tracker contract in the current worktree. You have only Read and apply_patch. Use absolute paths for Read and every apply_patch file and Move header. Do not seek shell, provider, or delegation tools. Report changed files and the checks main must run. The main agent owns the journal, tracker, Git, command execution, and acceptance.';
const nativeModePolicy = 'Stay in this native Amp mode. Investigate, plan, assess, coordinate, use the tracker, and run tests here. Delegate all implementation, including fixes and test writing, through loaf_delegate to a fresh Grok 4.6 Fast child. Do not implement locally if loaf_delegate is missing or fails. Use native Oracle for read-only review and advice. The Review button remains unchanged. Review-only requests do not authorize fixes; send authorized findings to a fresh loaf_delegate child, not the reviewer. Repeated Ship should reuse valid contract, diff, test, and review evidence and rerun only stale or missing checks. Do not infer commit, push, or merge authority. Native Low, Medium, High, Ultra, Oracle, and Subagents Auto settings stay untouched.';

async function delegationCommand(command: string, args: string[], cwd?: string): Promise<string> {
  const result = await delegationExec(command, args, { cwd, timeout: 10000, maxBuffer: 1024 * 1024 });
  return result.stdout.trim();
}

async function delegationCanonicalPath(path: string): Promise<string> {
  try {
    return await realpath(path);
  } catch (error) {
    if (!(error instanceof Error) || !('code' in error) || error.code !== 'ENOENT') throw error;
    // A dangling symlink is not a new file: resolving its parent would conceal
    // its target and could authorize creating a file outside the worktree.
    try {
      if ((await lstat(path)).isSymbolicLink()) throw new Error('Dangling symlink path cannot be verified.');
    } catch (statError) {
      if (!(statError instanceof Error) || !('code' in statError) || statError.code !== 'ENOENT') throw statError;
    }
    const parent = delegationDirname(path);
    if (parent === path) throw error;
    return resolve(await delegationCanonicalPath(parent), relative(parent, path));
  }
}

function delegationContained(root: string, path: string): boolean {
  const suffix = relative(root, path);
  return suffix !== '' && suffix !== '..' && !suffix.startsWith('..' + sep) && !isAbsolute(suffix) && !suffix.split(sep).includes('.git');
}

type NativeModeThread = {
  agent?: () => Promise<{ definition?: { kind?: string; mode?: string } }>;
  parentThreadID?: () => Promise<string | null>;
};

function isNativeBuiltinMode(mode: string | undefined): mode is typeof nativeBuiltinModes[number] {
  return mode === 'low' || mode === 'medium' || mode === 'high' || mode === 'ultra';
}

export async function loafNativeModeContext(_event: unknown, ctx: { thread?: NativeModeThread }): Promise<{ message?: { content: string; display: false } } | Record<string, never>> {
  try {
    if (typeof ctx.thread?.agent !== 'function' || typeof ctx.thread.parentThreadID !== 'function') return {};
    const [agent, parent] = await Promise.all([ctx.thread.agent(), ctx.thread.parentThreadID()]);
    // Current reported parent plus builtin definition only. Amp may return null
    // after a parent is deleted, so an orphaned builtin indistinguishable from a
    // parentless builtin can receive this policy. This is not lifetime ancestry
    // or an OS sandbox.
    if (parent !== null) return {};
    if (agent.definition?.kind !== 'builtin-agent' || !isNativeBuiltinMode(agent.definition.mode)) return {};
    return { message: { content: nativeModePolicy, display: false } };
  } catch {
    return {};
  }
}

function grokCatalogReady(catalog: unknown): boolean {
  if (!catalog || typeof catalog !== 'object') return false;
  const names = 'builtinToolNames' in catalog ? catalog.builtinToolNames : undefined;
  const models = 'models' in catalog ? catalog.models : undefined;
  if (!Array.isArray(names) || names.some(name => typeof name !== 'string') || implementationTools.some(tool => !names.includes(tool))) return false;
  if (!Array.isArray(models)) return false;
  return models.some(model => {
    if (!model || typeof model !== 'object' || !('id' in model) || model.id !== grokModel) return false;
    const capabilities = 'capabilities' in model ? model.capabilities : undefined;
    return !!capabilities && typeof capabilities === 'object' && 'tools' in capabilities && capabilities.tools === true;
  });
}

// This is a trusted-host, single-plugin-runtime boundary, not an OS sandbox or a
// cross-process lock. Keep guards after completion so a resumed child cannot write.
export function registerLoafDelegation(amp: DelegationAPI, command: DelegationCommand = delegationCommand): {
  owns: (id: string) => boolean;
  check: (event: DelegationToolCall) => Promise<string | undefined>;
} {
  const children = new Map<string, DelegationGuard>();
  let writer = false;
  const check = async (event: DelegationToolCall): Promise<string | undefined> => {
    const guard = children.get(event.thread.id);
    if (!guard) return undefined;
    if (guard.sealed) return 'Loaf child has no tool authority.';
    if (event.tool !== 'Read' && event.tool !== 'apply_patch') return 'Loaf implementation child allows only Read and apply_patch.';
    try {
      let paths: string[];
      if (event.tool === 'Read') {
        if (typeof event.input.path !== 'string' || !isAbsolute(event.input.path)) return 'Loaf requires an absolute Read path.';
        paths = [event.input.path];
      } else {
        // Amp's helper can omit relative paths and Move sources. This bounded
        // adapter accepts only explicit absolute patch headers and checks both.
        const patch = event.input.patchText;
        if (typeof patch !== 'string') return 'Loaf requires apply_patch.patchText with absolute file paths.';
        const headers = patch.split('\n').filter(line => line.startsWith('*** '));
        const declared: string[] = [];
        for (const header of headers) {
          if (['*** Begin Patch', '*** End Patch', '*** End of File'].includes(header)) continue;
          const match = /^\*\*\* (?:Add File|Update File|Delete File|Move to): (.+)$/.exec(header);
          if (!match || match[1] !== match[1].trim() || !isAbsolute(match[1])) return 'Loaf requires known patch headers and absolute file paths.';
          declared.push(match[1]);
        }
        if (!declared.length) return 'Loaf cannot verify every patch path.';
        const modified = amp.helpers.filesModifiedByToolCall(event);
        if (!modified?.length) return 'Loaf cannot verify every patch path.';
        paths = [...declared, ...modified.map(uri => amp.helpers.filePathFromURI(uri))];
      }
      for (const path of paths) {
        const absolute = resolve(guard.root, path);
        if (!delegationContained(guard.root, absolute) || !delegationContained(guard.root, await delegationCanonicalPath(absolute))) {
          return 'Loaf child path is outside its worktree or targets Git metadata.';
        }
      }
      return guard.sealed ? 'Loaf child has no tool authority.' : undefined;
    } catch {
      return 'Loaf could not verify the child tool boundary.';
    }
  };

  if (typeof amp.registerTool !== 'function') {
    console.warn('[loaf] Native delegation unavailable: installed Amp has no tool registration API.');
    return { owns: id => children.has(id), check };
  }
  const tool: PluginToolDefinition = {
    name: 'loaf_delegate',
    description: 'Delegate one bounded native tracker implementation in the current local Git worktree to a fresh Grok 4.6 Fast child. Implementation has Read/apply_patch only with absolute paths. Review is not a loaf_delegate role; use native Oracle. The main agent owns shell/tests, providers, acceptance, and exclusion of parent or unrelated writers. Prevents overlapping delegated implementers in this plugin runtime only; not an OS sandbox. Returns child identity, packet digest and turn evidence, never acceptance.',
    inputSchema: {
      type: 'object', additionalProperties: false,
      properties: {
        role: { type: 'string', enum: ['implementation'] },
        native_ref: { type: 'string', description: 'Canonical tracker record reference already read by the main agent.' },
        worktree: { type: 'string', description: 'Absolute path of the existing current Amp Git worktree.' },
        packet: { type: 'string', description: 'Live contract and bounded implementation task. Do not use this tool for review.' },
      },
      required: ['role', 'native_ref', 'worktree', 'packet'],
    },
    execute: async (input: Record<string, unknown>, ctx: PluginToolContext): Promise<string> => {
      let child: PluginThread | undefined;
      let guard: DelegationGuard | undefined;
      let acquired = false;
      let stopped = false;
      let parentStopped = false;
      let submissionPending = false;
      let parentSubscription: Subscription | undefined;
      let cancellation: Promise<void> | undefined;
      let interrupt: (error: Error) => void = () => {};
      const interrupted = new Promise<never>((_resolve, reject) => { interrupt = reject; });
      // Cancellation can arrive during preflight, before the promise is raced.
      void interrupted.catch(() => {});
      const cancelChild = (again = false): Promise<void> => {
        if (guard) guard.sealed = true;
        if ((!cancellation || again) && child) {
          try { cancellation = child.cancel().catch(() => {}); } catch { cancellation = Promise.resolve(); }
        }
        return cancellation ?? Promise.resolve();
      };
      const receipt: Record<string, unknown> = { acceptance: 'main-agent-required', parent_thread_id: ctx.thread.id };
      try {
        if (input.role === 'review') throw new Error('loaf_delegate is implementation-only; use native Oracle for read-only review.');
        if (input.role !== 'implementation') throw new Error('role must be implementation');
        for (const field of ['native_ref', 'worktree', 'packet']) {
          if (typeof input[field] !== 'string' || !(input[field] as string).trim()) throw new Error(`${field} must be nonempty`);
        }
        const worktree = input.worktree as string;
        const packet = input.packet as string;
        if (!isAbsolute(worktree)) throw new Error('worktree must be absolute');
        if (writer) throw new Error('An implementation child is active or its stop is uncertain; inspect it before continuing.');
        writer = true;
        acquired = true;
        if (typeof amp.createAgent !== 'function' || typeof ctx.thread.agent !== 'function' || typeof ctx.thread.parentThreadID !== 'function' ||
            typeof ctx.thread.state?.subscribe !== 'function' || typeof ctx.thread.state?.get !== 'function' ||
            typeof amp.helpers?.filePathFromURI !== 'function' || typeof amp.helpers?.filesModifiedByToolCall !== 'function') {
          throw new Error('Installed Amp lacks required stable delegation APIs.');
        }
        const observeParent = (state: string): void => {
          if (state !== 'idle' && state !== 'error') return;
          parentStopped = true;
          void cancelChild();
          interrupt(new Error('Parent turn stopped; child work was interrupted.'));
        };
        parentSubscription = ctx.thread.state.subscribe(observeParent);
        observeParent(await ctx.thread.state.get());
        if (amp.system?.executor?.kind !== 'local' || !amp.system.workspaceRoot) throw new Error('Delegation requires the current local workspace executor.');
        const root = await realpath(worktree);
        const workspace = await realpath(amp.helpers.filePathFromURI(amp.system.workspaceRoot));
        if (root !== workspace || root !== await realpath(await command('git', ['rev-parse', '--show-toplevel'], root))) {
          throw new Error('Requested worktree must be the canonical current Amp Git worktree root.');
        }
        if (parentStopped) throw new Error('Parent turn stopped before child creation.');
        const agent = await ctx.thread.agent();
        const mode = agent.definition.kind === 'builtin-agent' ? agent.definition.mode : undefined;
        // Current reported parent plus builtin definition only. Amp may return
        // null after a parent is deleted, so an orphaned builtin that looks
        // parentless can qualify. This is not lifetime ancestry or an OS sandbox.
        // Our Grok child remains custom with Read/apply_patch only, including
        // after parent deletion.
        if (!isNativeBuiltinMode(mode) || await ctx.thread.parentThreadID() !== null) {
          throw new Error('loaf_delegate is for a currently parentless builtin Amp mode; currently parented, custom, and unknown callers cannot implement.');
        }
        const catalog: unknown = JSON.parse(await command('amp', ['plugins', 'show-agent-options', '--json']));
        if (!grokCatalogReady(catalog)) throw new Error('Installed Amp does not expose xai/grok-4.6 with tools plus the required Read and apply_patch catalog.');
        if (parentStopped) throw new Error('Parent turn stopped before child creation.');
        Object.assign(receipt, { native_ref: input.native_ref, role: 'implementation', worktree: root, native_mode: mode, requested_model: grokModel, requested_features: ['fast'], requested_tools: implementationTools, packet_sha256: createHash('sha256').update(packet).digest('hex') });
        const childAgent = amp.createAgent({
          name: 'loaf-implementation-agent', model: grokModel, features: ['fast'], tools: [...implementationTools], instructions: implementationInstructions, display: { label: 'implementer' },
        });
        if (typeof childAgent.createThread !== 'function') throw new Error('Installed Amp cannot create a native child.');
        child = await childAgent.createThread({ parentThreadID: ctx.thread.id, executor: 'local', visibility: 'private' });
        receipt.child_thread_id = child.id;
        guard = { root, sealed: false };
        children.set(child.id, guard);
        if (parentStopped) throw new Error('Parent turn stopped during child creation.');
        if (typeof child.waitForResponse !== 'function' || typeof child.appendUserMessage !== 'function' || typeof child.cancel !== 'function' || typeof child.state?.get !== 'function') {
          throw new Error('Installed Amp lacks required child response or cancellation APIs.');
        }
        // Subscribe before append: waitForResponse observes the next running -> idle
        // transition. Capture rejection immediately while append is in flight.
        const response = child.waitForResponse({ timeoutMs: 600000 }).then(reply => ({ reply }), error => ({ error }));
        submissionPending = true;
        const submittingChild = child;
        const submission = Promise.resolve().then(() => {
          if (parentStopped) throw new Error('Parent turn stopped before child submission.');
          return submittingChild.appendUserMessage({ type: 'user-message', content: `Native record: ${input.native_ref}\nWorktree: ${root}\nPacket SHA-256: ${receipt.packet_sha256}\n\n${packet}` });
        }).finally(() => {
          submissionPending = false;
          // A previously idle child may start only after append settles. Cancel
          // again, but never silently release ownership after an uncertain result.
          if (parentStopped || guard?.sealed) void cancelChild(true);
        });
        await Promise.race([submission, interrupted]);
        const result = await Promise.race([response, interrupted]);
        if ('error' in result) throw result.error;
        if (parentStopped) throw new Error('Parent turn stopped before child acceptance.');
        stopped = true;
        const text = result.reply.content.filter(block => block.type === 'text').map(block => block.text).join('\n');
        return JSON.stringify({ ...receipt, status: 'turn-complete', signal: 'waitForResponse running-to-idle', text });
      } catch (error) {
        let state: string | undefined;
        const unresolvedSubmission = submissionPending;
        if (child) {
          await cancelChild();
          try { state = await child.state.get(); stopped = !unresolvedSubmission && !submissionPending && (state === 'idle' || state === 'error'); } catch { stopped = false; }
        }
        return JSON.stringify({ ...receipt, status: child ? (stopped ? 'interrupted' : 'uncertain') : (parentStopped ? 'interrupted' : 'incompatible'), child_state: state,
          submission_pending: submissionPending,
          message: error instanceof Error ? error.message : 'Delegation failed; inspect child state.' });
      } finally {
        parentSubscription?.unsubscribe();
        if (guard) guard.sealed = true;
        if (acquired && (!child || stopped)) writer = false;
      }
    },
  };
  try {
    amp.registerTool(tool);
  } catch (error) {
    console.warn('[loaf] Native delegation unavailable; existing hooks remain active:', error instanceof Error ? error.message : 'tool registration failed');
  }
  return { owns: id => children.has(id), check };
}
