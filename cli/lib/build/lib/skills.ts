/**
 * Skills Module
 *
 * Shared module for copying and transforming skills across all build targets.
 * Extracts common patterns from claude-code.ts, cursor.ts, opencode.ts,
 * codex.ts, and gemini.ts target transformers.
 */

import {
  mkdirSync,
  cpSync,
  writeFileSync,
  readFileSync,
  existsSync,
  readdirSync,
} from "fs";
import matter from "gray-matter";
import { join } from "path";
import { loadSkillFrontmatter } from "./sidecar.js";
import { copySharedTemplates } from "./shared-templates.js";
import { copyDirWithTransform } from "./copy-utils.js";
import type { SkillFrontmatter, TargetsConfig } from "../types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface CopySkillsOptions {
  /** Source directory containing skills/ subdirectory */
  srcDir: string;
  /** Destination directory for skills output */
  destDir: string;
  /** Target name (e.g., 'claude-code', 'cursor', 'shared') */
  targetName: string;
  /** Version string to inject into frontmatter (optional - if not provided, no version is injected) */
  version?: string;
  /** Full targets.yaml configuration */
  targetsConfig: TargetsConfig;
  /** Transform function for markdown content (e.g., command substitution) */
  transformMd: (content: string) => string;
  /** Additional subdirectories to copy beyond references/, templates/, scripts/ */
  extraDirs?: string[];
  /**
   * Callback to merge additional frontmatter fields.
   * Called after loading base frontmatter, allows targets to inject sidecar fields.
   */
  mergeFrontmatter?: (base: SkillFrontmatter, skillDir: string) => SkillFrontmatter;
}

// ─────────────────────────────────────────────────────────────────────────────
// Implementation
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Copy all skills from source to destination with transformations.
 *
 * This function:
 * 1. Discovers skills from srcDir/skills
 * 2. Loads base frontmatter from each SKILL.md
 * 3. Applies optional mergeFrontmatter callback for target-specific extensions
 * 4. Injects version into frontmatter
 * 5. Applies markdown transforms via transformMd
 * 6. Copies subdirectories: references/, templates/, scripts/
 * 7. Copies any extraDirs specified
 * 8. Distributes shared templates per targets.yaml config
 */
export function copySkills(options: CopySkillsOptions): void {
  const {
    srcDir,
    destDir,
    version,
    targetsConfig,
    transformMd,
    extraDirs = [],
    mergeFrontmatter,
  } = options;

  const src = join(srcDir, "skills");
  if (!existsSync(src)) return;

  const skills = readdirSync(src, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillSrc = join(src, skill);
    const skillDest = join(destDir, skill);
    mkdirSync(skillDest, { recursive: true });

    // Load and merge frontmatter
    let frontmatter = loadSkillFrontmatter(skillSrc);
    if (mergeFrontmatter) {
      frontmatter = mergeFrontmatter(frontmatter, skillSrc);
    }
    // Only inject version if provided (targets add version during their copy step)
    if (version !== undefined) {
      frontmatter = { ...frontmatter, version };
    }

    // Process SKILL.md
    const skillMdPath = join(skillSrc, "SKILL.md");
    if (existsSync(skillMdPath)) {
      const content = readFileSync(skillMdPath, "utf-8");
      const { content: body } = matter(content);
      writeFileSync(
        join(skillDest, "SKILL.md"),
        transformMd(matter.stringify(body, frontmatter)),
      );
    }

    // Copy standard subdirectories with transform
    for (const subdir of ["references", "templates"]) {
      const subSrc = join(skillSrc, subdir);
      if (existsSync(subSrc)) {
        copyDirWithTransform(subSrc, join(skillDest, subdir), transformMd);
      }
    }

    // Copy scripts directory as-is (no transform needed)
    const scriptsSrc = join(skillSrc, "scripts");
    if (existsSync(scriptsSrc)) {
      cpSync(scriptsSrc, join(skillDest, "scripts"), { recursive: true });
    }

    // Copy extra directories as-is
    for (const extraDir of extraDirs) {
      const extraSrc = join(skillSrc, extraDir);
      if (existsSync(extraSrc)) {
        cpSync(extraSrc, join(skillDest, extraDir), { recursive: true });
      }
    }

    // Distribute shared templates
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}
