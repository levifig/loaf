/**
 * Session resolution chain
 *
 * Implements SPEC-032's 3-tier routing:
 *
 *   1. `--session-id` CLI flag        → findSessionByClaudeId
 *   2. hook stdin (when opt-in)       → parseHookSessionId → findSessionByClaudeId
 *   3. branch lookup                  → findActiveSessionForBranch (emits stderr WARN)
 *
 * The chain is the single source of truth for "which session does this command
 * belong to?" — every subcommand that previously called
 * `findActiveSessionForBranch(...)` directly should migrate to this helper
 * (see TASK-117 / TASK-118).
 */

import { readFileSync } from "fs";

import {
  findActiveSessionForBranch,
  findSessionByClaudeId,
} from "./find.js";
import type { SessionFrontmatter } from "./store.js";

/**
 * Read stdin synchronously via fd 0 and parse it as hook JSON, returning the
 * `session_id` field if present and non-empty.
 *
 * Returns `undefined` for: missing field, malformed JSON, empty stdin,
 * non-string session_id, or any IO error.
 *
 * **Internal helper** — not part of the public session API. Callers must use
 * `resolveCurrentSession({ parseStdin: true })` instead. See SPEC-032 A5.
 *
 * Underscore prefix marks this as module-private; the only legitimate caller
 * is `resolveCurrentSession` in this same file. The function remains
 * exported (rather than truly private) so unit tests in `resolve.test.ts`
 * can reach it without surface area leaking to other modules.
 */
export function _parseHookSessionId(): string | undefined {
  // Read stdin synchronously from fd 0. Synchronous reads are fine here
  // because hook contexts always pipe a small JSON payload and exit; no
  // long-running IO or interactive consumers compete for stdin.
  let raw: string;
  try {
    raw = readFileSync(0, "utf-8");
  } catch {
    return undefined;
  }

  if (!raw || !raw.trim()) return undefined;

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return undefined;
  }

  if (!parsed || typeof parsed !== "object") return undefined;
  const value = (parsed as { session_id?: unknown }).session_id;
  if (typeof value !== "string" || value.length === 0) return undefined;
  return value;
}

/** Options for `resolveCurrentSession`. */
export interface ResolveCurrentSessionOptions {
  /** Explicit session id (e.g., from a `--session-id` CLI flag). Tier 1. */
  sessionIdFlag?: string;
  /**
   * When `true`, parse stdin via `parseHookSessionId`. Caller must opt in
   * (e.g., `--from-hook`) — auto-detection is rejected per SPEC-032 A5.
   */
  parseStdin?: boolean;
}

export interface ResolvedSession {
  filePath: string;
  data: SessionFrontmatter;
  content: string;
}

/** Exact text emitted to stderr when the chain falls through to branch routing. */
function emitBranchFallbackWarning(branch: string): void {
  // The literal text is asserted by tests — do not reformat.
  process.stderr.write(
    `WARN: no session_id signal — falling back to branch routing for branch '${branch}'. Pass --session-id <id> to silence.\n`
  );
}

/**
 * Resolve the current session via SPEC-032's 3-tier chain.
 *
 * - Tier 1 (`opts.sessionIdFlag`) → `findSessionByClaudeId`. On null, fall through.
 * - Tier 2 (`opts.parseStdin === true`) → `parseHookSessionId` → `findSessionByClaudeId`.
 *   On null, fall through.
 * - Tier 3 → `findActiveSessionForBranch`. Always emits stderr WARN (whether
 *   it returns a session or `null`).
 *
 * Returns the resolved session (active or stopped — caller decides what to do
 * with `data.status`) or `null` if all three tiers fail.
 */
export async function resolveCurrentSession(
  agentsDir: string,
  branch: string,
  opts: ResolveCurrentSessionOptions = {}
): Promise<ResolvedSession | null> {
  // Tier 1: explicit flag wins
  if (opts.sessionIdFlag) {
    const hit = findSessionByClaudeId(agentsDir, opts.sessionIdFlag, branch);
    if (hit) return hit;
    // Fall through on null — do NOT exit early.
  }

  // Tier 2: hook stdin (only when caller opts in)
  if (opts.parseStdin === true) {
    const stdinId = _parseHookSessionId();
    if (stdinId) {
      const hit = findSessionByClaudeId(agentsDir, stdinId, branch);
      if (hit) return hit;
      // Fall through on null.
    }
  }

  // Tier 3: branch routing (degraded path — always WARN)
  emitBranchFallbackWarning(branch);
  return findActiveSessionForBranch(agentsDir, branch);
}
