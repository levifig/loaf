/**
 * Path resolution for the souls catalog.
 *
 * The CLI is bundled by tsup into `dist-cli/`; at runtime `__dirname` points
 * inside the bundle, so we walk up looking for `package.json` with `name`
 * `loaf` to locate the loaf source root (mirrors the pattern in
 * `cli/commands/{build,install,version,setup}.ts`).
 *
 * In installed contexts the bundled CLI may live somewhere with no nearby
 * `package.json` (e.g. `~/.local/bin/loaf`). For those cases we fall back to
 * a list of installed-tool candidates that ship the catalog alongside the
 * tool's per-target config (mirrors the legacy `templates/soul.md` lookup
 * chain).
 */

import { existsSync, readFileSync, statSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

/**
 * Walk up from `__dirname` looking for the `loaf` package root.
 * Throws if not found within 10 levels.
 */
export function findLoafRoot(): string {
  let dir = __dirname;
  for (let i = 0; i < 10; i++) {
    const pkgPath = join(dir, "package.json");
    try {
      const pkg = JSON.parse(readFileSync(pkgPath, "utf-8")) as { name?: string };
      if (pkg.name === "loaf") return dir;
    } catch {
      // not found, walk up
    }
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error("Could not find loaf root directory (no package.json with name 'loaf')");
}

/**
 * Absolute path to the souls catalog (`content/souls/`) inside the loaf
 * package root. Optional `loafRoot` lets callers (mainly tests) override the
 * root resolution.
 */
export function getCatalogDir(loafRoot?: string): string {
  return join(loafRoot ?? findLoafRoot(), "content", "souls");
}

/**
 * Candidate catalog directories in resolution order:
 *
 *   1. `<loaf-root>/content/souls` — dev tree and Claude Code plugin
 *      (`plugins/loaf/content/souls/`), located via `findLoafRoot()`.
 *   2. `$CLAUDE_PLUGIN_ROOT/{content/,}souls` — explicit plugin override.
 *   3. Per-tool install dirs (`~/.cursor/souls`, `$XDG_CONFIG_HOME/opencode/souls`, …).
 *
 * Used by the SessionStart restoration path where the bundled CLI may run
 * outside the loaf source tree. The first candidate that exists wins.
 */
export function findCatalogCandidates(): string[] {
  const candidates: string[] = [];

  try {
    candidates.push(join(findLoafRoot(), "content", "souls"));
  } catch {
    // Bundled CLI with no nearby package.json — fall through to installed
    // candidates below.
  }

  if (process.env.CLAUDE_PLUGIN_ROOT) {
    candidates.push(join(process.env.CLAUDE_PLUGIN_ROOT, "content", "souls"));
    candidates.push(join(process.env.CLAUDE_PLUGIN_ROOT, "souls"));
  }

  const homeDir = process.env.HOME || process.env.USERPROFILE || "";
  const configHome = process.env.XDG_CONFIG_HOME || join(homeDir, ".config");

  candidates.push(join(configHome, "opencode", "souls"));
  candidates.push(join(homeDir, ".cursor", "souls"));
  candidates.push(join(process.env.CODEX_HOME || join(homeDir, ".codex"), "souls"));
  candidates.push(join(homeDir, ".amp", "souls"));

  return candidates.filter(Boolean);
}

/**
 * First existing catalog directory from `findCatalogCandidates()`, or `null`
 * when no candidate is reachable. `loafRoot` short-circuits the chain (used
 * by tests).
 */
export function resolveCatalogDir(loafRoot?: string): string | null {
  if (loafRoot) {
    return join(loafRoot, "content", "souls");
  }

  for (const candidate of findCatalogCandidates()) {
    try {
      if (existsSync(candidate) && statSync(candidate).isDirectory()) {
        return candidate;
      }
    } catch {
      continue;
    }
  }
  return null;
}
