/**
 * Project Root Resolution
 *
 * Locates the .agents/ directory by walking up from the current working
 * directory. Shared by task and spec CLI commands.
 */

import { existsSync } from "fs";
import { join, dirname } from "path";

import { loadIndex, buildIndexFromFiles, saveIndex } from "./migrate.js";
import type { TaskIndex } from "./types.js";

/**
 * Walk up from `startDir` looking for a `.agents/` directory.
 * Returns the absolute path to `.agents/` or null if not found.
 */
export function findAgentsDir(startDir: string = process.cwd()): string | null {
  let current = startDir;

  while (true) {
    const candidate = join(current, ".agents");
    if (existsSync(candidate)) {
      return candidate;
    }

    const parent = dirname(current);
    if (parent === current) {
      // Reached filesystem root
      return null;
    }
    current = parent;
  }
}

/**
 * Load TASKS.json from the agents directory. If it doesn't exist, build
 * the index from .md files and persist it. Returns null only if the
 * index file exists but has an invalid shape.
 */
export function getOrBuildIndex(agentsDir: string): TaskIndex {
  const indexPath = join(agentsDir, "TASKS.json");

  if (existsSync(indexPath)) {
    const index = loadIndex(indexPath);
    if (index) return index;

    // Invalid shape — rebuild from files
  }

  const index = buildIndexFromFiles(agentsDir);
  saveIndex(indexPath, index);
  return index;
}
