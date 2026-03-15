/**
 * Sidecar Metadata System
 *
 * Skills: Frontmatter (name, description) in SKILL.md source file.
 *         Optional extensions for Claude Code in SKILL.claude-code.yaml.
 *
 * Agents: Full frontmatter in sidecars (target-specific formats differ significantly).
 */

import { readFileSync, existsSync } from "fs";
import { join, dirname, basename } from "path";
import { parse as parseYaml } from "yaml";
import matter from "gray-matter";
import type { SkillFrontmatter, TargetsConfig } from "../types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Skills
// ─────────────────────────────────────────────────────────────────────────────

export function loadSkillFrontmatter(skillDir: string): SkillFrontmatter {
  const skillMdPath = join(skillDir, "SKILL.md");

  if (!existsSync(skillMdPath)) {
    throw new Error(`Missing SKILL.md in ${skillDir}`);
  }

  const content = readFileSync(skillMdPath, "utf-8");
  const { data } = matter(content);
  return data as SkillFrontmatter;
}

export function loadSkillExtensions(
  skillDir: string,
): Record<string, unknown> {
  const sidecarPath = join(skillDir, "SKILL.claude-code.yaml");

  if (!existsSync(sidecarPath)) {
    return {};
  }

  const content = readFileSync(sidecarPath, "utf-8");
  return (parseYaml(content) as Record<string, unknown>) || {};
}

export function mergeSkillFrontmatter(
  base: SkillFrontmatter,
  extensions: Record<string, unknown>,
): SkillFrontmatter {
  return { ...base, ...extensions };
}

/**
 * Load optional target-specific sidecar for a skill.
 */
export function loadTargetSkillSidecar(
  skillDir: string,
  targetName: string,
): Record<string, unknown> {
  const sidecarPath = join(skillDir, `SKILL.${targetName}.yaml`);

  if (!existsSync(sidecarPath)) {
    return {};
  }

  const content = readFileSync(sidecarPath, "utf-8");
  return (parseYaml(content) as Record<string, unknown>) || {};
}

// ─────────────────────────────────────────────────────────────────────────────
// Agents
// ─────────────────────────────────────────────────────────────────────────────

export function loadAgentSidecar(
  sourcePath: string,
  target: string,
): Record<string, unknown> {
  const dir = dirname(sourcePath);
  const baseName = basename(sourcePath, ".md");
  const sidecarPath = join(dir, `${baseName}.${target}.yaml`);

  if (!existsSync(sidecarPath)) {
    throw new Error(
      `Missing required sidecar: ${sidecarPath}\n` +
        `Agent '${baseName}' requires a sidecar for target '${target}'`,
    );
  }

  const content = readFileSync(sidecarPath, "utf-8");
  return parseYaml(content) as Record<string, unknown>;
}

/**
 * Load optional agent sidecar (returns empty object if not found).
 */
export function loadAgentSidecarOptional(
  sourcePath: string,
  target: string,
): Record<string, unknown> {
  const dir = dirname(sourcePath);
  const baseName = basename(sourcePath, ".md");
  const sidecarPath = join(dir, `${baseName}.${target}.yaml`);

  if (!existsSync(sidecarPath)) {
    return {};
  }

  const content = readFileSync(sidecarPath, "utf-8");
  return (parseYaml(content) as Record<string, unknown>) || {};
}

// ─────────────────────────────────────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────────────────────────────────────

export function loadTargetsConfig(configDir: string): TargetsConfig {
  const configPath = join(configDir, "targets.yaml");
  if (!existsSync(configPath)) {
    return { targets: {} };
  }
  const content = readFileSync(configPath, "utf-8");
  return parseYaml(content) as TargetsConfig;
}

export function getTargetOutput(
  targetsConfig: TargetsConfig,
  target: string,
): string | null {
  return targetsConfig.targets?.[target]?.output || null;
}
