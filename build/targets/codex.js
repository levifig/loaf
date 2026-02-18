/**
 * Codex Build Target
 *
 * Generates Codex-compatible skills:
 * dist/codex/
 * └── skills/
 *     ├── python/
 *     │   ├── SKILL.md         # version + Codex frontmatter (globs)
 *     │   ├── references/
 *     │   └── scripts/
 *     └── ...
 *
 * Codex only supports skills - no commands, agents, or hooks.
 * Optional SKILL.codex.yaml sidecar adds Codex-specific metadata like globs.
 */

import {
  mkdirSync,
  cpSync,
  writeFileSync,
  readFileSync,
  existsSync,
  readdirSync,
  rmSync,
} from "fs";
import matter from "gray-matter";
import { join } from "path";
import { parse as parseYaml } from "yaml";
import { loadSkillFrontmatter } from "../lib/sidecar.js";
import { getVersion, injectVersion } from "../lib/version.js";
import { buildAgentMap, substituteAgentNames } from "../lib/substitutions.js";
import { copySharedTemplates } from "../lib/shared-templates.js";

/**
 * Substitute command placeholders with Codex unscoped commands
 *
 * Placeholders:
 * - {{IMPLEMENT_CMD}} -> /implement
 * - {{RESUME_CMD}} -> /resume
 * - {{ORCHESTRATE_CMD}} -> /implement
 */
function substituteCommands(content) {
  return content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/resume")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}

/**
 * Copy references directory with command substitution for markdown files
 */
function copyReferencesWithSubstitution(srcDir, destDir) {
  mkdirSync(destDir, { recursive: true });

  const entries = readdirSync(srcDir, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = join(srcDir, entry.name);
    const destPath = join(destDir, entry.name);

    if (entry.isDirectory()) {
      copyReferencesWithSubstitution(srcPath, destPath);
    } else if (entry.name.endsWith(".md")) {
      // Apply substitutions to markdown files
      const content = readFileSync(srcPath, "utf-8");
      writeFileSync(destPath, substituteAgentNames(substituteCommands(content), AGENT_MAP));
    } else {
      // Copy non-markdown files as-is
      cpSync(srcPath, destPath);
    }
  }
}

const TARGET_NAME = "codex";

// Agent name map is loaded dynamically from sidecars at build time
let AGENT_MAP = {};

/**
 * Build Codex distribution
 */
export async function build({ rootDir, srcDir, distDir, targetsConfig }) {
  const version = getVersion(rootDir);

  // Build agent name map from sidecars
  AGENT_MAP = buildAgentMap(srcDir, TARGET_NAME);

  const skillsDir = join(distDir, "skills");

  // Clean existing directory
  if (existsSync(skillsDir)) {
    rmSync(skillsDir, { recursive: true });
  }
  mkdirSync(skillsDir, { recursive: true });

  // Copy all skills with version injection
  copySkills(srcDir, skillsDir, version, targetsConfig);
}

/**
 * Copy all skills with version injection and Codex-specific frontmatter
 */
function copySkills(srcDir, destDir, version, targetsConfig) {
  const src = join(srcDir, "skills");

  if (!existsSync(src)) {
    return;
  }

  const skills = readdirSync(src, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillSrc = join(src, skill);
    const skillDest = join(destDir, skill);
    mkdirSync(skillDest, { recursive: true });

    // Load frontmatter from SKILL.md
    const baseFrontmatter = loadSkillFrontmatter(skillSrc);

    // Load optional Codex sidecar (e.g., for globs)
    const sidecarFrontmatter = loadCodexSkillSidecar(skillSrc);

    // Merge and inject version
    const frontmatter = injectVersion(
      { ...baseFrontmatter, ...sidecarFrontmatter },
      version
    );

    // Read SKILL.md body (strip existing frontmatter if any)
    const skillMdPath = join(skillSrc, "SKILL.md");
    if (existsSync(skillMdPath)) {
      const content = readFileSync(skillMdPath, "utf-8");
      const { content: body } = matter(content);

      // Write SKILL.md with merged frontmatter + version, command and agent name substitution
      const transformed = substituteAgentNames(
        substituteCommands(matter.stringify(body, frontmatter)),
        AGENT_MAP
      );
      writeFileSync(join(skillDest, "SKILL.md"), transformed);
    }

    // Copy references directory with command substitution
    const refSrc = join(skillSrc, "references");
    const refDest = join(skillDest, "references");
    if (existsSync(refSrc)) {
      copyReferencesWithSubstitution(refSrc, refDest);
    }

    // Copy scripts directory
    const scriptsSrc = join(skillSrc, "scripts");
    const scriptsDest = join(skillDest, "scripts");
    if (existsSync(scriptsSrc)) {
      cpSync(scriptsSrc, scriptsDest, { recursive: true });
    }

    // Copy templates directory with substitution
    const templatesSrc = join(skillSrc, "templates");
    const templatesDest = join(skillDest, "templates");
    if (existsSync(templatesSrc)) {
      copyReferencesWithSubstitution(templatesSrc, templatesDest);
    }

    // Copy shared templates for this skill (won't overwrite skill-specific ones)
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, (content) =>
      substituteAgentNames(substituteCommands(content), AGENT_MAP)
    );
  }
}

/**
 * Load optional Codex sidecar for skill
 *
 * Codex sidecars typically contain:
 * - globs: Array of file patterns to match (e.g., ["*.py", "pyproject.toml"])
 */
function loadCodexSkillSidecar(skillDir) {
  const sidecarPath = join(skillDir, `SKILL.${TARGET_NAME}.yaml`);

  if (!existsSync(sidecarPath)) {
    return {};
  }

  const content = readFileSync(sidecarPath, "utf-8");
  return parseYaml(content) || {};
}
