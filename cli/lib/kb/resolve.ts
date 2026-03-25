/**
 * KB Config Resolution
 *
 * Locates the git root and loads knowledge base configuration from
 * .agents/loaf.json. Follows the same resolution pattern as
 * cli/lib/tasks/resolve.ts.
 */

import { execFileSync } from "child_process";
import { existsSync, readFileSync } from "fs";
import { join } from "path";

import type { KbConfig } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Defaults
// ─────────────────────────────────────────────────────────────────────────────

const DEFAULT_KB_CONFIG: KbConfig = {
  local: ["docs/knowledge", "docs/decisions"],
  staleness_threshold_days: 30,
  imports: [],
};

// ─────────────────────────────────────────────────────────────────────────────
// Git Root
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Find the git repository root by calling `git rev-parse --show-toplevel`.
 * Throws if not inside a git repository.
 */
export function findGitRoot(): string {
  try {
    const result = execFileSync("git", ["rev-parse", "--show-toplevel"], {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return result.trim();
  } catch {
    throw new Error("Not inside a git repository");
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Config Loading
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Load KB configuration from `.agents/loaf.json`.
 *
 * Returns the `knowledge` section with defaults applied for any missing
 * fields. If the file is missing or has no `knowledge` section, returns
 * the full default config.
 */
export function loadKbConfig(gitRoot: string): KbConfig {
  const configPath = join(gitRoot, ".agents", "loaf.json");

  if (!existsSync(configPath)) {
    return { ...DEFAULT_KB_CONFIG, local: [...DEFAULT_KB_CONFIG.local], imports: [...DEFAULT_KB_CONFIG.imports] };
  }

  try {
    const raw = readFileSync(configPath, "utf-8");
    const parsed = JSON.parse(raw);

    if (!parsed || typeof parsed !== "object" || !parsed.knowledge) {
      return { ...DEFAULT_KB_CONFIG, local: [...DEFAULT_KB_CONFIG.local], imports: [...DEFAULT_KB_CONFIG.imports] };
    }

    const kb = parsed.knowledge;

    return {
      local: Array.isArray(kb.local) ? kb.local : DEFAULT_KB_CONFIG.local,
      staleness_threshold_days:
        typeof kb.staleness_threshold_days === "number"
          ? kb.staleness_threshold_days
          : DEFAULT_KB_CONFIG.staleness_threshold_days,
      imports: Array.isArray(kb.imports) ? kb.imports : DEFAULT_KB_CONFIG.imports,
    };
  } catch {
    return { ...DEFAULT_KB_CONFIG, local: [...DEFAULT_KB_CONFIG.local], imports: [...DEFAULT_KB_CONFIG.imports] };
  }
}
