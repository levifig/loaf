/**
 * SPEC-036 / TASK-170 — Worktree storage migration
 *
 * Under SPEC-036 (A3), `.agents/` is project-scoped, not branch-scoped: every
 * worktree resolves to the **main worktree's** `.agents/` directory. Existing
 * checkouts that already populated a linked worktree's local `.agents/` need
 * a one-shot migration that moves everything into the main worktree's store.
 *
 * This module implements the migration as a pure function (`runMigration`)
 * plus a tiny commander adapter (`cli/commands/migrate.ts`). The function is
 * deliberately decoupled from process.exit / Commander so tests can call it
 * directly against tmp fixtures.
 *
 * Behavior summary:
 *   - Dry-run by default; `--apply` mutates.
 *   - Conflict policy: newest mtime wins, with `--force-from-worktree` /
 *     `--force-from-main` overrides.
 *   - After a successful `--apply`, writes a `.moved-to` back-pointer in the
 *     worktree-local `.agents/` so the refusal nudge can detect post-A3 state.
 *   - Idempotent: re-running on a migrated worktree is a no-op.
 *   - Run from the main checkout: clean no-op exit (nothing to migrate).
 *   - Outside a git context: error (the command is meaningless).
 */

import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  renameSync,
  rmSync,
  statSync,
  writeFileSync,
} from "fs";
import { dirname, join, relative } from "path";

import { DEBUG_RESOLVE_ENV, findMainWorktreeRoot } from "../tasks/resolve.js";

// SYMLINK BEHAVIOR (current, intentional but not fully specified):
//   - Same-FS moves preserve the link itself (renameSync moves the symlink)
//   - EXDEV fallback dereferences (cpSync follows symlinks unless verbatimSymlinks is set)
//   - Conflict policy uses statSync, which follows symlinks for mtime comparison
// This is a known asymmetry. Pinning the behavior requires a design decision —
// tracked in .agents/ideas/ as a follow-up. Do not extend symlink-touching logic
// without consulting that idea first.

// ─────────────────────────────────────────────────────────────────────────────
// Centralized refusal message — referenced by cli/index.ts and tests.
// ─────────────────────────────────────────────────────────────────────────────

/**
 * The single source of truth for the pre-A3 refusal nudge.
 *
 * Properties this string MUST preserve (tests assert on them):
 *   - Mentions `loaf migrate worktree-storage` exactly.
 *   - References SPEC-036 by name (so users can grep the docs).
 *   - Mentions the LOAF_DEBUG_RESOLVE env knob.
 */
export const PRE_A3_REFUSAL_MESSAGE = [
  "This worktree has unmigrated agentic state under .agents/.",
  "SPEC-036 centralizes .agents/ to the main worktree, so this command is refused",
  "until you run:",
  "",
  "    loaf migrate worktree-storage         # dry-run preview",
  "    loaf migrate worktree-storage --apply # perform the migration",
  "",
  `(Tip: set ${DEBUG_RESOLVE_ENV}=1 to see git probe diagnostics if the refusal seems unexpected.)`,
].join("\n");

// ─────────────────────────────────────────────────────────────────────────────
// Detection: is this worktree in a pre-A3 state?
// ─────────────────────────────────────────────────────────────────────────────

/** Filename of the post-migration back-pointer dropped in worktree-local .agents/. */
export const BACK_POINTER_FILE = ".moved-to";

/**
 * Read the worktree-local `.moved-to` back-pointer, if any.
 * Returns the trimmed absolute path it points to, or null if absent/empty.
 */
export function readBackPointer(worktreeAgentsDir: string): string | null {
  const path = join(worktreeAgentsDir, BACK_POINTER_FILE);
  if (!existsSync(path)) return null;
  try {
    const content = readFileSync(path, "utf-8").trim();
    return content.length > 0 ? content : null;
  } catch {
    return null;
  }
}

/**
 * True if `worktreeAgentsDir` contains any *file* (recursively) other than
 * the back-pointer. Empty leftover subdirectories (the residue of a
 * successful migration — sessions/, kb/, etc.) do NOT count as content.
 *
 * Used to short-circuit refusal-nudge detection for already-cleaned (or
 * never-populated) worktrees and to recognise the already-migrated state
 * inside `runMigration`.
 */
export function worktreeAgentsHasContent(worktreeAgentsDir: string): boolean {
  if (!existsSync(worktreeAgentsDir)) return false;
  return enumerateFiles(worktreeAgentsDir).length > 0;
}

/**
 * Pre-A3 detection signal used by the top-level CLI dispatcher.
 *
 * Returns true iff ALL of the following hold:
 *   1. `startDir` is inside a linked git worktree (findMainWorktreeRoot is non-null).
 *   2. The worktree-local `.agents/` exists AND contains content other than
 *      the back-pointer.
 *   3. EITHER the back-pointer is absent, OR it points somewhere other than
 *      the current main worktree root.
 *
 * Main checkouts and single-worktree repos always return false, so the
 * dispatcher pays near-zero cost in the common case (existsSync short-circuits
 * before the git probe even fires for repos without a local `.agents/`).
 */
export function detectPreA3State(startDir: string = process.cwd()): boolean {
  // Cheapest possible short-circuit: walk up looking for a worktree-local
  // `.agents/`. If we never find one before we hit a `.git` (worktree root)
  // OR the filesystem root, there's nothing to migrate — and the git probe
  // never fires. This keeps the dispatcher near-zero-cost in the common case
  // (running loaf from any tree without a linked-worktree's local .agents/).
  const wtRoot = findWorktreeRootForDetection(startDir);
  if (!wtRoot) return false;

  const localAgents = join(wtRoot, ".agents");
  if (!existsSync(localAgents)) return false;

  // Run the cheap git probe BEFORE the recursive content scan. On a main
  // checkout with a populated `.agents/`, this collapses the common-case
  // cost from O(.agents/ size) to a single git probe — every loaf invocation
  // would otherwise pay the walk just to be told "not in a linked worktree".
  const mainRoot = findMainWorktreeRoot(wtRoot);
  if (!mainRoot) return false; // main checkout or non-git context

  if (!worktreeAgentsHasContent(localAgents)) return false;

  const pointer = readBackPointer(localAgents);
  if (!pointer) return true; // populated + no pointer = pre-A3

  // Back-pointer present: ensure it actually points at the current main
  // worktree root AND that root still exists. If either is false, treat as
  // pre-A3 (re-migration required).
  if (!existsSync(pointer)) return true;
  return pointer !== mainRoot;
}

/**
 * Walk up from `startDir` to the nearest directory containing a `.git`
 * entry. Mirrors `findWorktreeRoot` below but exported through
 * `detectPreA3State` only — kept private to avoid encouraging callers to
 * reinvent it.
 */
function findWorktreeRootForDetection(startDir: string): string | null {
  let current = startDir;
  while (true) {
    if (existsSync(join(current, ".git"))) return current;
    const parent = dirname(current);
    if (parent === current) return null;
    current = parent;
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Migration plan
// ─────────────────────────────────────────────────────────────────────────────

export type ConflictResolution = "worktree" | "main" | "newer";

export interface MigrationOptions {
  /** Working directory the user invoked `loaf migrate worktree-storage` from. */
  cwd: string;
  /** When true, mutate the filesystem; otherwise emit a dry-run plan. */
  apply: boolean;
  /** Conflict policy. Defaults to "newer". */
  conflictPolicy: ConflictResolution;
}

export interface PlannedMove {
  /** Absolute source path inside the worktree-local `.agents/`. */
  from: string;
  /** Absolute destination path inside the main worktree's `.agents/`. */
  to: string;
  /** Relative path from `.agents/` (for logging). */
  rel: string;
  /** True if a file already exists at `to`. */
  conflict: boolean;
  /** When `conflict` is true, the resolution this run will apply. */
  resolution?: "keep-worktree" | "keep-main";
  /** When `conflict` is true, a one-line reason for the resolution. */
  resolutionReason?: string;
}

export interface MigrationPlan {
  kind: "plan";
  worktreeAgents: string;
  mainAgents: string;
  mainRoot: string;
  moves: PlannedMove[];
  backPointerPath: string;
}

export type MigrationStatus =
  | "not-in-worktree"
  | "not-in-git"
  | "no-local-agents"
  | "already-migrated"
  | "planned"
  | "applied";

export interface MigrationResult {
  status: MigrationStatus;
  message: string;
  plan?: MigrationPlan;
}

// ─────────────────────────────────────────────────────────────────────────────
// Plan computation
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Recursively enumerate every file under `dir`, returning paths relative to
 * `dir`. Empty directories are not emitted (move-by-file semantics mean
 * empty dirs vanish naturally). The back-pointer file is excluded.
 */
function enumerateFiles(dir: string): string[] {
  const out: string[] = [];
  const walk = (current: string, rel: string) => {
    let entries: import("fs").Dirent[];
    try {
      entries = readdirSync(current, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      const absPath = join(current, entry.name);
      const relPath = rel ? join(rel, entry.name) : entry.name;
      // Skip the back-pointer at the root — it's a control file, not content.
      if (rel === "" && entry.name === BACK_POINTER_FILE) continue;
      if (entry.isDirectory()) {
        walk(absPath, relPath);
      } else if (entry.isFile() || entry.isSymbolicLink()) {
        out.push(relPath);
      }
    }
  };
  walk(dir, "");
  return out;
}

function mtime(path: string): number {
  try {
    return statSync(path).mtimeMs;
  } catch {
    return 0;
  }
}

function planMoves(
  worktreeAgents: string,
  mainAgents: string,
  policy: ConflictResolution,
): PlannedMove[] {
  const relFiles = enumerateFiles(worktreeAgents);
  const moves: PlannedMove[] = [];
  for (const rel of relFiles) {
    const from = join(worktreeAgents, rel);
    const to = join(mainAgents, rel);
    const conflict = existsSync(to);

    let resolution: "keep-worktree" | "keep-main" | undefined;
    let resolutionReason: string | undefined;

    if (conflict) {
      switch (policy) {
        case "worktree":
          resolution = "keep-worktree";
          resolutionReason = "forced by --force-from-worktree";
          break;
        case "main":
          resolution = "keep-main";
          resolutionReason = "forced by --force-from-main";
          break;
        case "newer": {
          // Ties resolve to "keep-main" (intentional: prefer the canonical store on no-signal).
          const fromMtime = mtime(from);
          const toMtime = mtime(to);
          if (fromMtime > toMtime) {
            resolution = "keep-worktree";
            resolutionReason = `worktree mtime ${new Date(fromMtime).toISOString()} > main mtime ${new Date(toMtime).toISOString()}`;
          } else {
            resolution = "keep-main";
            resolutionReason = `main mtime ${new Date(toMtime).toISOString()} >= worktree mtime ${new Date(fromMtime).toISOString()}`;
          }
          break;
        }
      }
    }

    moves.push({ from, to, rel, conflict, resolution, resolutionReason });
  }
  return moves;
}

// ─────────────────────────────────────────────────────────────────────────────
// Apply
// ─────────────────────────────────────────────────────────────────────────────

function applyMove(move: PlannedMove): void {
  if (move.conflict) {
    if (move.resolution === "keep-main") {
      // Source loses — delete the worktree-local copy.
      try {
        rmSync(move.from, { force: true });
      } catch {
        // best-effort
      }
      return;
    }
    // keep-worktree: overwrite main with source. We need to remove the
    // destination first because renameSync semantics over an existing file
    // differ across platforms.
    try {
      rmSync(move.to, { force: true });
    } catch {
      // ignore
    }
  }

  mkdirSync(dirname(move.to), { recursive: true });
  try {
    renameSync(move.from, move.to);
  } catch (err) {
    // EXDEV: crossing filesystem boundaries. Fall back to cpSync (preserves
    // mode bits, mtimes; handles directories) + rmSync. We previously used
    // readFileSync/writeFileSync, which lost mode bits (executable scripts),
    // mtimes (matters for future re-migrations), and dereferenced symlinks.
    const nodeErr = err as NodeJS.ErrnoException;
    if (nodeErr.code === "EXDEV") {
      cpSync(move.from, move.to, { recursive: true, preserveTimestamps: true });
      rmSync(move.from, { recursive: true, force: true });
    } else {
      throw err;
    }
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Top-level entry
// ─────────────────────────────────────────────────────────────────────────────

export function runMigration(options: MigrationOptions): MigrationResult {
  const { cwd, apply, conflictPolicy } = options;

  // 1. Must be inside a git repository.
  //    findMainWorktreeRoot returns null both for main-checkouts and for
  //    non-git contexts; we must distinguish them. We do that by attempting
  //    a cheap git probe — if even that fails, we surface a clear error.
  const mainRoot = findMainWorktreeRoot(cwd);

  if (!mainRoot) {
    // Either: (a) main checkout (clean no-op) or (b) outside git (error).
    // Distinguish by probing for any `.git` along the parent chain.
    if (!isInGitContext(cwd)) {
      return {
        status: "not-in-git",
        message:
          "loaf migrate worktree-storage: not in a git repository — this command is only meaningful inside a linked git worktree.",
      };
    }
    return {
      status: "not-in-worktree",
      message: "Nothing to migrate — already in the main worktree.",
    };
  }

  // We need the worktree root, not just `cwd`. The user may have invoked the
  // command from a subdirectory of the worktree (e.g., `src/foo/`); the
  // worktree-local `.agents/` lives at the worktree root.
  const wtRoot = findWorktreeRoot(cwd) ?? cwd;
  const localAgents = join(wtRoot, ".agents");
  const mainAgents = join(mainRoot, ".agents");

  // 2. Already migrated?
  const pointer = readBackPointer(localAgents);
  if (pointer === mainRoot && !worktreeAgentsHasContent(localAgents)) {
    return {
      status: "already-migrated",
      message: "Nothing to do — already migrated.",
    };
  }

  // 3. No local `.agents/` at all? Edge case — treat as nothing to migrate.
  if (!existsSync(localAgents) || !worktreeAgentsHasContent(localAgents)) {
    // If the back-pointer is set but stale, fall through to a clean plan.
    if (pointer && pointer === mainRoot) {
      return {
        status: "already-migrated",
        message: "Nothing to do — already migrated.",
      };
    }
    return {
      status: "no-local-agents",
      message: "Nothing to migrate — worktree has no local .agents/ content.",
    };
  }

  // 4. Build the plan.
  const moves = planMoves(localAgents, mainAgents, conflictPolicy);
  const plan: MigrationPlan = {
    kind: "plan",
    worktreeAgents: localAgents,
    mainAgents,
    mainRoot,
    moves,
    backPointerPath: join(localAgents, BACK_POINTER_FILE),
  };

  // 5. Dry run? Emit the plan and stop.
  if (!apply) {
    return {
      status: "planned",
      message: "Dry run — re-run with --apply to perform the migration.",
      plan,
    };
  }

  // 6. Apply.
  mkdirSync(mainAgents, { recursive: true });
  for (const move of plan.moves) {
    applyMove(move);
  }
  // Write the back-pointer (overwrites if it existed and was stale).
  writeFileSync(plan.backPointerPath, `${mainRoot}\n`, "utf-8");

  return {
    status: "applied",
    message: `Migrated ${plan.moves.length} file(s) to ${mainAgents}.`,
    plan,
  };
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers — git context probes (light, no exec)
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Walk up from `startDir` looking for a `.git` entry (file or directory).
 * Used to distinguish "main checkout" (return null) from "outside git
 * context" (return null) without re-running `git rev-parse`.
 */
function isInGitContext(startDir: string): boolean {
  let current = startDir;
  while (true) {
    if (existsSync(join(current, ".git"))) return true;
    const parent = dirname(current);
    if (parent === current) return false;
    current = parent;
  }
}

/**
 * Walk up from `startDir` looking for a directory that contains a `.git`
 * entry — i.e., the working-tree root of the current worktree. We do this
 * client-side (rather than asking git) so the function is safe to call
 * even when `git` is unavailable.
 */
function findWorktreeRoot(startDir: string): string | null {
  let current = startDir;
  while (true) {
    if (existsSync(join(current, ".git"))) return current;
    const parent = dirname(current);
    if (parent === current) return null;
    current = parent;
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Pretty printer for CLI output
// ─────────────────────────────────────────────────────────────────────────────

const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;
const cyan = (s: string) => `\x1b[36m${s}\x1b[0m`;

export function formatResult(
  result: MigrationResult,
  opts: { apply: boolean },
): string {
  const out: string[] = [];

  if (result.plan) {
    const plan = result.plan;
    out.push(`${bold("loaf migrate worktree-storage")} ${opts.apply ? cyan("--apply") : gray("(dry-run)")}\n`);
    out.push(`  ${gray("from")} ${plan.worktreeAgents}`);
    out.push(`  ${gray("to  ")} ${plan.mainAgents}`);
    out.push("");

    if (plan.moves.length === 0) {
      out.push(`  ${green("✓")} Nothing to move.`);
    } else {
      for (const move of plan.moves) {
        const arrow = move.conflict
          ? move.resolution === "keep-main"
            ? yellow("[conflict→main]  ")
            : yellow("[conflict→worktree]")
          : cyan("→");
        const path = relative(plan.worktreeAgents, move.from);
        if (move.conflict) {
          if (move.resolution === "keep-main") {
            out.push(`  ${arrow} ${path}`);
            out.push(`      ${gray(`keep main; discard worktree (${move.resolutionReason})`)}`);
          } else {
            out.push(`  ${arrow} ${path}`);
            out.push(`      ${gray(`keep worktree; overwrite main (${move.resolutionReason})`)}`);
          }
        } else {
          out.push(`  ${arrow} ${path}`);
        }
      }
    }

    out.push("");
    out.push(`  ${gray("back-pointer:")} ${plan.backPointerPath}`);
    out.push("");
  }

  out.push(`  ${result.status === "applied" ? green("✓") : cyan("→")} ${result.message}`);
  return out.join("\n");
}
