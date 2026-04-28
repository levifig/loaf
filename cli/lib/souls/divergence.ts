/**
 * Divergence detection for `.agents/SOUL.md`.
 *
 * A local SOUL.md is "diverged" when its sha256 does not match any catalog
 * soul's sha256. If it matches at least one, it is considered untouched
 * (or matches a known soul) and `loaf soul use <name>` may overwrite it
 * without prompting.
 */

import { createHash } from "crypto";
import { existsSync, readFileSync } from "fs";

import { listSouls, type SoulEntry } from "./catalog.js";

/** sha256 hex digest of a string. */
export function sha256(content: string): string {
  return createHash("sha256").update(content, "utf-8").digest("hex");
}

/** sha256 of a file's bytes, or null if the file does not exist. */
export function hashFile(path: string): string | null {
  if (!existsSync(path)) return null;
  return sha256(readFileSync(path, "utf-8"));
}

/**
 * Build the set of sha256 hashes for every soul in the catalog.
 */
export function catalogHashes(loafRoot?: string): Set<string> {
  const hashes = new Set<string>();
  for (const soul of listSouls(loafRoot)) {
    hashes.add(sha256(readFileSync(soul.path, "utf-8")));
  }
  return hashes;
}

export interface DivergenceResult {
  /** True when a local SOUL.md exists and matches no catalog soul. */
  diverged: boolean;
  /** Local file hash, or null when the file does not exist. */
  localHash: string | null;
  /** Catalog soul whose hash matches the local file, if any. */
  matchedSoul: SoulEntry | null;
}

/**
 * Compare a local SOUL.md against every soul in the catalog.
 *
 * - No local file → `diverged: false`, `localHash: null`. (Fresh install case
 *   — `loaf soul use` is free to write the file.)
 * - Local file matches some catalog soul → `diverged: false`, `matchedSoul`
 *   set. (User has the canonical copy of a known soul.)
 * - Local file matches no catalog soul → `diverged: true`. (User-modified;
 *   `--force` required to overwrite.)
 */
export function checkDivergence(
  localPath: string,
  loafRoot?: string,
): DivergenceResult {
  const localHash = hashFile(localPath);
  if (localHash === null) {
    return { diverged: false, localHash: null, matchedSoul: null };
  }

  for (const soul of listSouls(loafRoot)) {
    const soulHash = sha256(readFileSync(soul.path, "utf-8"));
    if (soulHash === localHash) {
      return { diverged: false, localHash, matchedSoul: soul };
    }
  }

  return { diverged: true, localHash, matchedSoul: null };
}
