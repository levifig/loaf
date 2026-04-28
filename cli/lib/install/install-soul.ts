/**
 * Soul integration for `loaf install` (SPEC-033, T6/T7).
 *
 * Three install paths, all non-interactive — the `--interactive` selection
 * surface is owned by TASK-135 and lives elsewhere.
 *
 *   1. **Fresh install** — no `.agents/SOUL.md` exists. Copy the `none`
 *      SOUL.md from the catalog to `.agents/SOUL.md` and write
 *      `soul: none` to `loaf.json`.
 *
 *   2. **Legacy upgrade** — a `.agents/SOUL.md` already exists, but
 *      `loaf.json` has no `soul:` field. The repo predates SPEC-033 and is
 *      almost certainly running the Warden/Fellowship lore. Pin
 *      `soul: fellowship` so existing agents keep working unchanged.
 *      `SOUL.md` is left untouched.
 *
 *   3. **Already-configured** — `.agents/SOUL.md` exists *and* `loaf.json`
 *      has a `soul:` field. No-op for both files.
 *
 * The migration is one-shot. Once `soul:` is set the legacy branch is
 * unreachable; `loaf install` becomes a no-op for these two files on every
 * subsequent run.
 */

import { existsSync } from "fs";

import { DEFAULT_SOUL, getActiveSoul, setActiveSoul } from "../config/agents-config.js";
import { copySoulToProject, localSoulPath } from "../souls/install.js";

/** Soul written to `loaf.json` when upgrading a pre-SPEC-033 repo. */
export const LEGACY_SOUL = "fellowship";

export type InstallSoulAction =
  /** Wrote `none` SOUL.md + `soul: none` (fresh project). */
  | "fresh"
  /** Wrote `soul: fellowship` to loaf.json, left SOUL.md untouched. */
  | "legacy-upgrade"
  /** Both SOUL.md and `soul:` already configured — no changes. */
  | "noop";

export interface InstallSoulResult {
  action: InstallSoulAction;
  /** Soul name now recorded in `loaf.json`. */
  soul: string;
  /** Absolute path to `.agents/SOUL.md`. */
  soulPath: string;
  /** True when `.agents/SOUL.md` was (or already had been) written. */
  soulFileWritten: boolean;
}

/**
 * Run the install-time soul migration for a project.
 *
 * Call this from `loaf install` after the per-target installers have run.
 * Returns a structured result the caller can render in the standard install
 * output style.
 *
 * `loafRoot` is exposed for tests; in production it falls through to the
 * souls catalog auto-resolution.
 */
export function installSoul(
  projectRoot: string,
  loafRoot?: string,
): InstallSoulResult {
  const soulPath = localSoulPath(projectRoot);
  const hasLocalSoul = existsSync(soulPath);
  const activeSoul = getActiveSoul(projectRoot);

  // Path 3: already configured — no-op.
  if (hasLocalSoul && activeSoul !== null) {
    return {
      action: "noop",
      soul: activeSoul,
      soulPath,
      soulFileWritten: true,
    };
  }

  // Path 2: legacy upgrade — existing SOUL.md, no soul: field.
  // Pin to fellowship for zero-config preservation. Don't touch SOUL.md.
  if (hasLocalSoul && activeSoul === null) {
    setActiveSoul(projectRoot, LEGACY_SOUL);
    return {
      action: "legacy-upgrade",
      soul: LEGACY_SOUL,
      soulPath,
      soulFileWritten: true,
    };
  }

  // Path 1: fresh install — write `none` SOUL.md + record it in loaf.json.
  copySoulToProject(DEFAULT_SOUL, projectRoot, loafRoot);
  setActiveSoul(projectRoot, DEFAULT_SOUL);
  return {
    action: "fresh",
    soul: DEFAULT_SOUL,
    soulPath,
    soulFileWritten: true,
  };
}
