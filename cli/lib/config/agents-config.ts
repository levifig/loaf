/**
 * Read/write `.agents/loaf.json` for project configuration and integration toggles.
 *
 * This is the single typed surface for `loaf.json`. Format conventions
 * (2-space indent, trailing newline, key preservation) live here so every
 * writer agrees.
 *
 * Worktree behavior (SPEC-036, A3):
 *   When invoked from a linked git worktree, `.agents/loaf.json` is resolved
 *   to the **main worktree's** copy. Without this routing, callers that pass a
 *   linked worktree's root would read/write a stray `loaf.json` next to the
 *   `.moved-to` back-pointer — invisible to `loaf release` and friends, which
 *   resolve the main `.agents/` via `findAgentsDir`. See SPEC-042 Track A.
 */

import { existsSync, mkdirSync, readFileSync, statSync, writeFileSync } from "fs";
import { join } from "path";

import { findMainWorktreeRoot } from "../tasks/resolve.js";

export interface LoafConfig {
  knowledge?: {
    local?: string[];
    staleness_threshold_days?: number;
    imports?: string[];
  };
  integrations?: Record<string, { enabled: boolean }>;
  release?: {
    /** Repo-relative paths to version files. Replaces root auto-detection when set. */
    versionFiles?: string[];
  };
  [key: string]: unknown;
}

/**
 * Resolve the effective directory that hosts `.agents/loaf.json` for a given
 * `projectRoot`. In a linked worktree this returns the main worktree's root;
 * everywhere else it returns `projectRoot` unchanged.
 *
 * Defensive fallback: if `findMainWorktreeRoot` returns a path that does not
 * exist as a directory (e.g., the main worktree was deleted), we fall back to
 * `projectRoot`. The migrate command surfaces this state through a separate
 * dedicated error; we do not want a config helper to crash here.
 */
function resolveEffectiveRoot(projectRoot: string): string {
  const mainRoot = findMainWorktreeRoot(projectRoot);
  if (!mainRoot) return projectRoot;
  try {
    if (statSync(mainRoot).isDirectory()) {
      return mainRoot;
    }
  } catch {
    // mainRoot doesn't exist or is unreadable — fall through to projectRoot.
  }
  return projectRoot;
}

/**
 * Resolve the absolute path to the effective `.agents/loaf.json` for the given
 * `projectRoot`. Routes through the main worktree under SPEC-036.
 */
function resolveEffectiveConfigPath(projectRoot: string): string {
  return join(resolveEffectiveRoot(projectRoot), ".agents", "loaf.json");
}

/**
 * Absolute path to `.agents/loaf.json` for a project root.
 *
 * In a linked git worktree this resolves to the **main worktree's** copy so
 * every caller — whether they reach via this helper or via `findAgentsDir` —
 * agrees on a single source of truth (SPEC-036).
 */
export function loafConfigPath(projectRoot: string): string {
  return resolveEffectiveConfigPath(projectRoot);
}

export function readLoafConfig(projectRoot: string): LoafConfig {
  const p = resolveEffectiveConfigPath(projectRoot);
  if (!existsSync(p)) return {};
  try {
    const raw = readFileSync(p, "utf-8");
    return JSON.parse(raw) as LoafConfig;
  } catch {
    return {};
  }
}

/**
 * Write `loaf.json`, ensuring the `.agents/` directory exists. Format:
 * 2-space indent + trailing newline. Single source of truth for the file
 * format — every writer in the codebase delegates here.
 *
 * Routes through `resolveEffectiveRoot` so a write from a linked worktree
 * lands in the main worktree's `.agents/`, never next to the `.moved-to`
 * back-pointer.
 */
function writeLoafConfigRaw(
  projectRoot: string,
  next: Record<string, unknown>,
): void {
  const effectiveRoot = resolveEffectiveRoot(projectRoot);
  const agentsDir = join(effectiveRoot, ".agents");
  if (!existsSync(agentsDir)) {
    mkdirSync(agentsDir, { recursive: true });
  }
  writeFileSync(
    join(agentsDir, "loaf.json"),
    `${JSON.stringify(next, null, 2)}\n`,
    "utf-8",
  );
}

export function mergeLoafConfigIntegrations(
  projectRoot: string,
  updates: Partial<{ linear: { enabled: boolean }; serena: { enabled: boolean } }>,
): void {
  const existing = readLoafConfig(projectRoot);
  const integrations = {
    ...existing.integrations,
  };
  if (updates.linear !== undefined) {
    integrations.linear = updates.linear;
  }
  if (updates.serena !== undefined) {
    integrations.serena = updates.serena;
  }
  const next: LoafConfig = {
    ...existing,
    integrations,
  };
  writeLoafConfigRaw(projectRoot, next as Record<string, unknown>);
}
