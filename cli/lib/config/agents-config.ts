/**
 * Read/write `.agents/loaf.json` for project configuration and integration toggles.
 *
 * This is the single typed surface for `loaf.json`. All writers — including the
 * souls library — go through here so format conventions (2-space indent,
 * trailing newline, key preservation) live in one place.
 */

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { join } from "path";

/**
 * Default soul when `loaf.json` has no `soul:` field on a *fresh* install.
 * The legacy upgrade path uses `"fellowship"` instead — see
 * `cli/lib/install/install-soul.ts`.
 */
export const DEFAULT_SOUL = "none";

export interface LoafConfig {
  /**
   * Active orchestrator soul (SPEC-033). Names a directory inside
   * `content/souls/<name>/` whose `SOUL.md` is the catalog source of truth.
   *
   * Optional in the schema so existing repos that pre-date SPEC-033 still
   * parse cleanly. Reads should fall back to `DEFAULT_SOUL` ("none") when
   * absent. The legacy-upgrade path (`installSoul`) is the only place that
   * cares about the missing-field branch.
   */
  soul?: string;
  knowledge?: {
    local?: string[];
    staleness_threshold_days?: number;
    imports?: string[];
  };
  integrations?: Record<string, { enabled: boolean }>;
  [key: string]: unknown;
}

/** Absolute path to `.agents/loaf.json` for a project root. */
export function loafConfigPath(projectRoot: string): string {
  return join(projectRoot, ".agents", "loaf.json");
}

export function readLoafConfig(projectRoot: string): LoafConfig {
  const p = loafConfigPath(projectRoot);
  if (!existsSync(p)) return {};
  try {
    const raw = readFileSync(p, "utf-8");
    return JSON.parse(raw) as LoafConfig;
  } catch {
    return {};
  }
}

/**
 * Read `loaf.json` and parse it as a generic record, preserving every key.
 * Internal helper for writers that need to round-trip unknown fields.
 */
function readLoafConfigRaw(projectRoot: string): Record<string, unknown> {
  const p = loafConfigPath(projectRoot);
  if (!existsSync(p)) return {};
  try {
    return JSON.parse(readFileSync(p, "utf-8")) as Record<string, unknown>;
  } catch {
    // Corrupt JSON — overwrite rather than crash. Same posture as
    // `readLoafConfig` above.
    return {};
  }
}

/**
 * Write `loaf.json`, ensuring the `.agents/` directory exists. Format:
 * 2-space indent + trailing newline. Single source of truth for the file
 * format — every writer in the codebase delegates here.
 */
function writeLoafConfigRaw(
  projectRoot: string,
  next: Record<string, unknown>,
): void {
  const agentsDir = join(projectRoot, ".agents");
  if (!existsSync(agentsDir)) {
    mkdirSync(agentsDir, { recursive: true });
  }
  writeFileSync(
    loafConfigPath(projectRoot),
    `${JSON.stringify(next, null, 2)}\n`,
    "utf-8",
  );
}

export function mergeLoafConfigIntegrations(
  projectRoot: string,
  updates: Partial<{ linear: { enabled: boolean }; serena: { enabled: boolean } }>,
): void {
  const existing = readLoafConfig(projectRoot);
  const integrations = {
    ...existing.integrations,
  };
  if (updates.linear !== undefined) {
    integrations.linear = updates.linear;
  }
  if (updates.serena !== undefined) {
    integrations.serena = updates.serena;
  }
  const next: LoafConfig = {
    ...existing,
    integrations,
  };
  writeLoafConfigRaw(projectRoot, next as Record<string, unknown>);
}

/**
 * Read the `soul` field from `loaf.json`.
 *
 * Returns `null` when the file is missing, unreadable, or has no `soul:`
 * field. Callers (e.g. `loaf soul current`) are responsible for applying
 * the `DEFAULT_SOUL` fallback. The legacy-upgrade path in `installSoul`
 * also relies on the null branch to detect pre-SPEC-033 repos.
 */
export function getActiveSoul(projectRoot: string): string | null {
  const cfg = readLoafConfig(projectRoot);
  return typeof cfg.soul === "string" && cfg.soul.length > 0 ? cfg.soul : null;
}

/**
 * Write `soul: <name>` into `loaf.json`, preserving any existing keys.
 * Creates the file (and the `.agents/` directory) if missing.
 */
export function setActiveSoul(projectRoot: string, name: string): void {
  const existing = readLoafConfigRaw(projectRoot);
  const next = { ...existing, soul: name };
  writeLoafConfigRaw(projectRoot, next);
}
