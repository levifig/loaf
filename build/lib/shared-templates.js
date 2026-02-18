/**
 * Shared Template Distribution
 *
 * Copies shared templates from src/templates/ into skill output directories
 * based on the shared-templates config in targets.yaml.
 */

import { existsSync, readFileSync, writeFileSync, mkdirSync } from "fs";
import { join } from "path";

/**
 * Copy shared templates from src/templates/ to a skill's output templates/ directory
 *
 * Shared templates are defined in targets.yaml under shared-templates:
 *   template-file: [skill1, skill2, ...]
 *
 * If a skill is listed for a template, that template is copied to the skill's
 * output templates/ directory (creating it if needed). Skill-specific templates
 * in the skill's own templates/ directory take precedence — shared templates
 * will not overwrite existing files.
 *
 * @param {string} skillName - The skill name (e.g., "implement")
 * @param {string} skillDest - The skill's output directory
 * @param {string} srcDir - The src/ directory (parent of templates/)
 * @param {Object} targetsConfig - Full targets.yaml config
 * @param {Function} [transformFn] - Optional content transform for markdown files
 */
export function copySharedTemplates(
  skillName,
  skillDest,
  srcDir,
  targetsConfig,
  transformFn
) {
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
