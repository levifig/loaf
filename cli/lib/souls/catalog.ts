/**
 * Souls catalog reader.
 *
 * The catalog lives at `content/souls/{name}/SOUL.md`. Each entry is a single
 * markdown file — there is no frontmatter, no metadata sidecar.
 *
 * For `loaf soul list` we need a one-line description per soul. SOUL.md files
 * may include a tagline as a blockquote line directly under the H1, e.g.:
 *
 *   # The Orchestrator
 *
 *   > A neutral, function-only soul — describes the team by role, not by character.
 *
 * The `fellowship` SOUL.md does not include a tagline (it is byte-identical to
 * `content/templates/soul.md`). When no `> tagline` line is present, we fall
 * back to the H1 line itself as the description.
 */

import { existsSync, readFileSync, readdirSync, statSync } from "fs";
import { join } from "path";

import { getCatalogDir } from "./paths.js";

export interface SoulEntry {
  /** Catalog name (directory name inside content/souls/) */
  name: string;
  /** One-line description for `loaf soul list` */
  description: string;
  /** Absolute path to the SOUL.md file */
  path: string;
}

/**
 * List all souls in the catalog. Each soul is a subdirectory of
 * `content/souls/` containing a `SOUL.md` file. Directories without a
 * `SOUL.md` are skipped silently.
 *
 * Pass `loafRoot` to override the loaf package root (mainly for tests).
 */
export function listSouls(loafRoot?: string): SoulEntry[] {
  const catalogDir = getCatalogDir(loafRoot);
  if (!existsSync(catalogDir)) return [];

  const entries: SoulEntry[] = [];
  for (const name of readdirSync(catalogDir).sort()) {
    const dir = join(catalogDir, name);
    let isDir = false;
    try {
      isDir = statSync(dir).isDirectory();
    } catch {
      continue;
    }
    if (!isDir) continue;

    const soulPath = join(dir, "SOUL.md");
    if (!existsSync(soulPath)) continue;

    const content = readFileSync(soulPath, "utf-8");
    entries.push({
      name,
      description: extractDescription(content),
      path: soulPath,
    });
  }
  return entries;
}

/**
 * Read a soul's SOUL.md content by catalog name. Throws if the soul does not
 * exist in the catalog.
 */
export function readSoul(name: string, loafRoot?: string): string {
  const soulPath = soulPathFor(name, loafRoot);
  if (!existsSync(soulPath)) {
    throw new Error(`Unknown soul: ${name}`);
  }
  return readFileSync(soulPath, "utf-8");
}

/**
 * Absolute path to a soul's SOUL.md file. Does not check existence.
 */
export function soulPathFor(name: string, loafRoot?: string): string {
  return join(getCatalogDir(loafRoot), name, "SOUL.md");
}

/**
 * Extract a one-line description from SOUL.md content.
 *
 * Preference order:
 *   1. First non-empty line beginning with `>` after the H1 (tagline blockquote).
 *   2. The H1 text itself (e.g. `# The Warden` -> `The Warden`).
 *   3. Empty string when no H1 is present.
 *
 * Only inspects the first ~20 lines so a stray blockquote deep in the prose
 * (e.g. a quoted line under "Orchestration Principles") cannot masquerade as
 * the tagline.
 */
export function extractDescription(content: string): string {
  const lines = content.split(/\r?\n/);

  // Find the first H1.
  let headingIdx = -1;
  let headingText = "";
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const m = line.match(/^#\s+(.+?)\s*$/);
    if (m) {
      headingIdx = i;
      headingText = m[1].trim();
      break;
    }
  }

  if (headingIdx === -1) return "";

  // Look for a tagline blockquote in the next handful of lines, allowing only
  // blank lines between the heading and the blockquote. Stop as soon as we
  // hit non-blank, non-blockquote prose (the soul body has begun).
  for (let i = headingIdx + 1; i < Math.min(lines.length, headingIdx + 20); i++) {
    const trimmed = lines[i].trim();
    if (trimmed === "") continue;
    if (trimmed.startsWith(">")) {
      return trimmed.replace(/^>\s*/, "").trim();
    }
    break;
  }

  return headingText;
}
