/**
 * Install / activation logic for souls.
 *
 * "Install" here means *copying* a catalog SOUL.md to `.agents/SOUL.md` and
 * recording the active soul name in `.agents/loaf.json`. This is the file
 * mechanic for `loaf soul use <name>` — divergence checking lives in
 * `divergence.ts` and is enforced by the command layer before calling
 * `installSoul`.
 *
 * Schema-level concerns (typed `LoafConfig`, `loaf install` integration)
 * belong to TASK-131; this module performs a minimal read-merge-write that
 * preserves any existing keys in `loaf.json`.
 */

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { dirname, join } from "path";

import { readSoul } from "./catalog.js";

/** Path of the project's `.agents/SOUL.md` given a project root. */
export function localSoulPath(projectRoot: string): string {
  return join(projectRoot, ".agents", "SOUL.md");
}

/** Path of the project's `.agents/loaf.json` given a project root. */
export function loafConfigPath(projectRoot: string): string {
  return join(projectRoot, ".agents", "loaf.json");
}

/**
 * Copy the catalog SOUL.md for `name` to `.agents/SOUL.md` inside
 * `projectRoot`. Overwrites unconditionally — divergence enforcement is the
 * caller's responsibility.
 *
 * Pass `loafRoot` to override the loaf catalog root (mainly for tests).
 */
export function copySoulToProject(
  name: string,
  projectRoot: string,
  loafRoot?: string,
): { written: string; bytes: number } {
  const content = readSoul(name, loafRoot);
  const dest = localSoulPath(projectRoot);
  mkdirSync(dirname(dest), { recursive: true });
  writeFileSync(dest, content, "utf-8");
  return { written: dest, bytes: Buffer.byteLength(content, "utf-8") };
}

/**
 * Read the `soul` field from `.agents/loaf.json`.
 *
 * Returns `null` when the file is missing, unreadable, or has no `soul:`
 * field. Callers (e.g. `loaf soul current`) are responsible for applying
 * the `none` default.
 */
export function readActiveSoul(projectRoot: string): string | null {
  const p = loafConfigPath(projectRoot);
  if (!existsSync(p)) return null;
  try {
    const raw = readFileSync(p, "utf-8");
    const json = JSON.parse(raw) as Record<string, unknown>;
    const value = json.soul;
    return typeof value === "string" && value.length > 0 ? value : null;
  } catch {
    return null;
  }
}

/**
 * Write `soul: <name>` into `.agents/loaf.json`, preserving any existing
 * keys. Creates the file (and the `.agents/` directory) if missing.
 *
 * Format: 2-space indent, trailing newline — matches the convention used by
 * `cli/lib/config/agents-config.ts`.
 */
export function writeActiveSoul(projectRoot: string, name: string): void {
  const agentsDir = join(projectRoot, ".agents");
  const p = loafConfigPath(projectRoot);

  let existing: Record<string, unknown> = {};
  if (existsSync(p)) {
    try {
      existing = JSON.parse(readFileSync(p, "utf-8")) as Record<string, unknown>;
    } catch {
      // Corrupt JSON — overwrite rather than crash. Same posture as
      // `readLoafConfig` in agents-config.ts.
      existing = {};
    }
  }

  const next = { ...existing, soul: name };

  if (!existsSync(agentsDir)) {
    mkdirSync(agentsDir, { recursive: true });
  }
  writeFileSync(p, `${JSON.stringify(next, null, 2)}\n`, "utf-8");
}
