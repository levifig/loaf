/**
 * Interactive soul selection for `loaf install` (SPEC-033, T14).
 *
 * Wraps the non-interactive `installSoul()` flow with a prompt that lets the
 * user pick a catalog soul on the **fresh-install path** (no existing
 * `.agents/SOUL.md` and no `soul:` field in `loaf.json`). Off the fresh path
 * we silently no-op so legacy upgrades and already-configured projects keep
 * the existing behavior.
 *
 * The prompt is intentionally tiny — one numbered choice with `none` pre-
 * selected as the default. Pressing Enter without choosing matches the
 * spec's "fresh installs default to none" contract, so the interactive
 * surface degrades gracefully into the non-interactive path.
 *
 * Selection persists by:
 *   1. Copying the chosen soul's catalog `SOUL.md` to `.agents/SOUL.md`.
 *   2. Writing `soul: <name>` into `.agents/loaf.json`.
 *
 * After this runs, `installSoul()` will see both files configured and
 * return `noop` — by design, so the wrapping is transparent to the
 * downstream call site.
 */

import { existsSync } from "fs";

import { askChoice } from "../prompts.js";
import {
  copySoulToProject,
  listSouls,
  localSoulPath,
  type SoulEntry,
} from "../souls/index.js";
import {
  DEFAULT_SOUL,
  getActiveSoul,
  setActiveSoul,
} from "../config/agents-config.js";

/** ANSI helpers — duplicated locally to keep this module zero-dep on install.ts. */
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;

export interface SoulPromptOptions {
  /** Project root containing `.agents/`. */
  projectRoot: string;
  /**
   * True when interactive mode is active for this install. Computed by the
   * caller from `--interactive` / `--no-interactive` / `--yes` / TTY. When
   * false, this helper is a no-op and returns `null`.
   */
  interactive: boolean;
  /** Override the catalog root (tests). Falls through to auto-resolution. */
  loafRoot?: string;
}

export type SoulPromptOutcome =
  /** Interactive prompt ran and the user's choice was applied. */
  | { action: "prompted"; soul: string }
  /** Skipped because interactive=false. */
  | { action: "skipped-non-interactive" }
  /** Skipped because not a fresh install (legacy upgrade or already-configured). */
  | { action: "skipped-not-fresh" }
  /** Catalog has no souls — bail silently and let `installSoul` handle defaults. */
  | { action: "skipped-empty-catalog" };

/**
 * Detect whether this install is on the fresh path (no `.agents/SOUL.md`
 * file *and* no `soul:` field in `loaf.json`). Mirrors the gating logic in
 * `installSoul()` but inverted: returns true only when the fresh-install
 * branch would fire.
 */
function isFreshInstall(projectRoot: string): boolean {
  const hasLocalSoul = existsSync(localSoulPath(projectRoot));
  const activeSoul = getActiveSoul(projectRoot);
  return !hasLocalSoul && activeSoul === null;
}

/**
 * Render a single catalog entry as a numbered choice line:
 *
 *   1. none — A neutral, function-only soul — describes...
 *   2. fellowship — The Warden
 */
function formatSoulChoice(entry: SoulEntry, index: number): string {
  const num = `${index + 1}.`;
  const desc = entry.description ? ` ${gray("—")} ${entry.description}` : "";
  return `  ${num} ${bold(entry.name)}${desc}`;
}

/**
 * Sort the catalog so the default soul (`none`) appears first. This is the
 * one that pressing Enter selects, so showing it on top matches the user's
 * expectation. Other souls follow in alphabetical order.
 */
export function orderSoulsForPrompt(
  souls: SoulEntry[],
  defaultName: string,
): SoulEntry[] {
  const sorted = [...souls].sort((a, b) => a.name.localeCompare(b.name));
  const defaultIdx = sorted.findIndex((s) => s.name === defaultName);
  if (defaultIdx <= 0) return sorted;
  const [def] = sorted.splice(defaultIdx, 1);
  return [def, ...sorted];
}

/**
 * Prompt the user to pick a soul from the catalog and persist their choice.
 *
 * Returns a structured outcome the caller can use to decide whether to print
 * a status line. Filesystem side effects only happen on the `prompted`
 * branch.
 */
export async function promptAndApplySoul(
  options: SoulPromptOptions,
): Promise<SoulPromptOutcome> {
  if (!options.interactive) {
    return { action: "skipped-non-interactive" };
  }

  if (!isFreshInstall(options.projectRoot)) {
    return { action: "skipped-not-fresh" };
  }

  const catalog = listSouls(options.loafRoot);
  if (catalog.length === 0) {
    return { action: "skipped-empty-catalog" };
  }

  const ordered = orderSoulsForPrompt(catalog, DEFAULT_SOUL);
  const fallback =
    ordered.find((s) => s.name === DEFAULT_SOUL) ?? ordered[0];

  console.log(`  ${bold("Choose an orchestrator soul:")}`);
  const chosen = await askChoice<SoulEntry>(
    `  Soul [1-${ordered.length}, default ${fallback.name}]: `,
    ordered,
    formatSoulChoice,
    fallback,
  );

  copySoulToProject(chosen.name, options.projectRoot, options.loafRoot);
  setActiveSoul(options.projectRoot, chosen.name);

  return { action: "prompted", soul: chosen.name };
}
