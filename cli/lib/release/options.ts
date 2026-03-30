/**
 * Release Option Validation & Normalization
 *
 * Pure functions for validating and normalizing CLI options
 * used by the `loaf release` command. Extracted for testability.
 */

import { execFileSync } from "child_process";
import type { BumpType } from "./version.js";

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface ReleaseOptions {
  dryRun?: boolean;
  bump?: string;
  base?: string;
  tag?: boolean;
  gh?: boolean;
}

export interface NormalizedFlags {
  skipTag: boolean;
  skipGh: boolean;
}

// ─────────────────────────────────────────────────────────────────────────────
// Validation
// ─────────────────────────────────────────────────────────────────────────────

const VALID_BUMPS: BumpType[] = ["major", "minor", "patch", "prerelease", "release"];

/**
 * Validate a --bump value. Returns the validated BumpType or throws
 * with a descriptive message.
 */
export function validateBumpType(value: string): BumpType {
  if (!VALID_BUMPS.includes(value as BumpType)) {
    throw new Error(
      `Invalid bump type "${value}". Must be one of: ${VALID_BUMPS.join(", ")}`,
    );
  }
  return value as BumpType;
}

/**
 * Validate that a git ref exists and resolves to a commit.
 * Throws with a descriptive message if the ref is invalid.
 */
export function validateBaseRef(cwd: string, ref: string): void {
  try {
    execFileSync("git", ["rev-parse", "--verify", `${ref}^{commit}`], {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
    });
  } catch {
    throw new Error(
      `Base ref "${ref}" does not exist or is not reachable. ` +
      `If this is a remote branch, run: git fetch origin ${ref}`,
    );
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Normalization
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Normalize skip flags from CLI options.
 *
 * --no-tag implies --no-gh because `gh release create` auto-creates
 * missing tags on GitHub, which would defeat the purpose of --no-tag.
 */
export function normalizeSkipFlags(options: ReleaseOptions): NormalizedFlags {
  const skipTag = options.tag === false;
  const skipGh = options.gh === false || skipTag;
  return { skipTag, skipGh };
}
