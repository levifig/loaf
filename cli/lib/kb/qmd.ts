/**
 * QMD Soft Dependency
 *
 * Optional integration with QMD (https://github.com/tobi/qmd) for knowledge
 * retrieval. All functions gracefully handle QMD not being installed.
 * Uses execFileSync exclusively for subprocess calls.
 */

import { execFileSync } from "child_process";

/**
 * Check if QMD CLI is available on the system.
 */
export function isQmdAvailable(): boolean {
  try {
    execFileSync("which", ["qmd"], { stdio: "pipe" });
    return true;
  } catch {
    return false;
  }
}

/**
 * Register a QMD collection. Requires QMD to be installed.
 * @param name Collection name (e.g., "loaf-knowledge")
 * @param path Path to the directory
 */
export function registerCollection(name: string, path: string): void {
  try {
    execFileSync("qmd", ["collection", "add", name, path], {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`Failed to register QMD collection "${name}": ${message}`);
  }
}

/**
 * Remove a QMD collection.
 */
export function removeCollection(name: string): void {
  try {
    execFileSync("qmd", ["collection", "remove", name], {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`Failed to remove QMD collection "${name}": ${message}`);
  }
}

/**
 * List registered QMD collections. Returns collection names.
 */
export function listCollections(): string[] {
  try {
    const output = execFileSync("qmd", ["collection", "list"], {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return output.trim().split("\n").filter(Boolean);
  } catch {
    return [];
  }
}
