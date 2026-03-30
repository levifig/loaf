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
 * Returns the resolved ref name — which may differ from the input
 * if the literal ref doesn't exist but `origin/<ref>` does.
 *
 * This handles the common case where a clone has `origin/main` but
 * no local `main` branch.
 */
export function validateBaseRef(cwd: string, ref: string): string {
  // Try the literal ref first
  if (refResolves(cwd, ref)) return ref;

  // Fall back to origin/<ref> if the ref doesn't contain a slash
  // (i.e., don't try origin/origin/main)
  if (!ref.includes("/")) {
    const remoteRef = `origin/${ref}`;
    if (refResolves(cwd, remoteRef)) return remoteRef;
  }

  throw new Error(
    `Base ref "${ref}" does not exist or is not reachable. ` +
    `Tried "${ref}" and "origin/${ref}". ` +
    `If this is a remote branch, run: git fetch origin ${ref}`,
  );
}

function refResolves(cwd: string, ref: string): boolean {
  try {
    execFileSync("git", ["rev-parse", "--verify", `${ref}^{commit}`], {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
    });
    return true;
  } catch {
    return false;
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
