/**
 * Install / activation logic for souls.
 *
 * "Install" here means *copying* a catalog SOUL.md to `.agents/SOUL.md` and
 * recording the active soul name in `.agents/loaf.json`. This is the file
 * mechanic for `loaf soul use <name>` — divergence checking lives in
 * `divergence.ts` and is enforced by the command layer before calling
 * `copySoulToProject`.
 *
 * `loaf.json` reads/writes delegate to the typed config layer
 * (`cli/lib/config/agents-config.ts`) so format conventions and the schema
 * shape live in one place. The souls library does not open `loaf.json`
 * directly.
 */

import { mkdirSync, writeFileSync } from "fs";
import { dirname, join } from "path";

import {
  getActiveSoul,
  loafConfigPath as configLoafConfigPath,
  setActiveSoul,
} from "../config/agents-config.js";
import { readSoul } from "./catalog.js";

/** Path of the project's `.agents/SOUL.md` given a project root. */
export function localSoulPath(projectRoot: string): string {
  return join(projectRoot, ".agents", "SOUL.md");
}

/**
 * Path of the project's `.agents/loaf.json` given a project root.
 *
 * Re-exported from the typed config layer so existing souls-library
 * consumers keep importing it from `cli/lib/souls/index.js`.
 */
export function loafConfigPath(projectRoot: string): string {
  return configLoafConfigPath(projectRoot);
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
 * Thin pass-through to `getActiveSoul` in the typed config layer. Returns
 * `null` when the file is missing, unreadable, or has no `soul:` field.
 * Callers (e.g. `loaf soul current`) are responsible for applying the
 * `none` default.
 */
export function readActiveSoul(projectRoot: string): string | null {
  return getActiveSoul(projectRoot);
}

/**
 * Write `soul: <name>` into `.agents/loaf.json`, preserving any existing
 * keys. Thin pass-through to `setActiveSoul` in the typed config layer.
 */
export function writeActiveSoul(projectRoot: string, name: string): void {
  setActiveSoul(projectRoot, name);
}
