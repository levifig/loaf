/**
 * Shared Template Distribution
 *
 * Copies shared templates from content/templates/ into skill output directories
 * based on the shared-templates config in targets.yaml.
 */

import { existsSync, readFileSync, writeFileSync, mkdirSync } from "fs";
import { join } from "path";
import type { TargetsConfig } from "../types.js";

/**
 * Copy shared templates to a skill's output templates/ directory.
 *
 * Skill-specific templates take precedence — shared templates
 * will not overwrite existing files.
 */
export function copySharedTemplates(
  skillName: string,
  skillDest: string,
  srcDir: string,
  targetsConfig: TargetsConfig,
  transformFn?: (content: string) => string,
): void {
  const sharedTemplates = targetsConfig?.["shared-templates"] || {};

  for (const [templateFile, skills] of Object.entries(sharedTemplates)) {
    if (!Array.isArray(skills) || !skills.includes(skillName)) {
      continue;
    }

    const templateSrc = join(srcDir, "templates", templateFile);
    if (!existsSync(templateSrc)) {
      continue;
    }

    const templatesDest = join(skillDest, "templates");
    const destPath = join(templatesDest, templateFile);

    // Don't overwrite skill-specific templates
    if (existsSync(destPath)) {
      continue;
    }

    mkdirSync(templatesDest, { recursive: true });

    let content = readFileSync(templateSrc, "utf-8");
    if (transformFn && templateFile.endsWith(".md")) {
      content = transformFn(content);
    }
    writeFileSync(destPath, content);
  }
}
