/**
 * Cursor Build Target
 *
 * Generates Cursor rules structure:
 * dist/cursor/
 * └── .cursor/
 *     └── rules/
 *         ├── python.mdc
 *         ├── typescript.mdc
 *         ├── rails.mdc
 *         └── ...
 *
 * Uses sidecar metadata for globs and truncation settings.
 * See src/config/targets.yaml for defaults, skills can override via SKILL.cursor.yaml.
 *
 * Note: Cursor doesn't support hooks, so we include manual check instructions.
 */

import {
  mkdirSync,
  readFileSync,
  writeFileSync,
  existsSync,
  readdirSync,
} from "fs";
import { join } from "path";
import { loadSkillSidecar, getTargetDefaults } from "../lib/sidecar.js";

const TARGET_NAME = "cursor";

/**
 * Build Cursor distribution
 */
export async function build({
  config,
  targetConfig,
  targetsConfig,
  rootDir,
  srcDir,
  distDir,
}) {
  const rulesDir = join(distDir, ".cursor", "rules");
  mkdirSync(rulesDir, { recursive: true });

  // Get skill defaults for this target
  const skillDefaults = getTargetDefaults(targetsConfig, TARGET_NAME, "skills");

  // Generate a rule file for each skill
  const skillsDir = join(srcDir, "skills");
  if (existsSync(skillsDir)) {
    const skills = readdirSync(skillsDir, { withFileTypes: true })
      .filter((d) => d.isDirectory())
      .map((d) => d.name);

    for (const skill of skills) {
      generateSkillRule(skill, srcDir, rulesDir, config, skillDefaults);
    }
  }

  // Generate a combined agents rule
  generateAgentsRule(srcDir, rulesDir);
}

/**
 * Generate .mdc rule file for a skill
 */
function generateSkillRule(skillName, srcDir, rulesDir, config, defaults) {
  const skillDir = join(srcDir, "skills", skillName);
  const skillMdPath = join(skillDir, "SKILL.md");

  if (!existsSync(skillMdPath)) {
    return;
  }

  const skillContent = readFileSync(skillMdPath, "utf-8");

  // Load sidecar config merged with defaults
  const sidecarConfig = loadSkillSidecar(skillDir, TARGET_NAME, defaults);

  // Get globs from sidecar or defaults
  const globs = sidecarConfig.globs || ["**/*"];
  const truncateConfig = sidecarConfig.truncate || { max_chars: 1500, max_files: 5 };

  // Find associated hooks for manual check instructions
  const pluginGroup = Object.entries(config["plugin-groups"]).find(
    ([, cfg]) => cfg.skills?.includes(skillName)
  );

  let hookInstructions = "";
  if (pluginGroup) {
    const [, cfg] = pluginGroup;
    const preHookIds = cfg.hooks?.["pre-tool"] || [];
    const postHookIds = cfg.hooks?.["post-tool"] || [];

    if (preHookIds.length > 0 || postHookIds.length > 0) {
      hookInstructions = `
## Quality Checks (Manual)

Since Cursor doesn't support automated hooks, run these checks manually:

### Before Committing
${preHookIds
  .map((id) => {
    const hook = config.hooks["pre-tool"].find((h) => h.id === id);
    return hook ? `- ${hook.description}` : null;
  })
  .filter(Boolean)
  .join("\n")}

### After Changes
${postHookIds
  .map((id) => {
    const hook = config.hooks["post-tool"].find((h) => h.id === id);
    return hook ? `- ${hook.description}` : null;
  })
  .filter(Boolean)
  .join("\n")}
`;
    }
  }

  // Read reference files with truncation from config
  const referenceDir = join(skillDir, "reference");
  let referenceContent = "";
  if (existsSync(referenceDir)) {
    const refFiles = readdirSync(referenceDir)
      .filter((f) => f.endsWith(".md"))
      .slice(0, truncateConfig.max_files);

    for (const refFile of refFiles) {
      const refPath = join(referenceDir, refFile);
      const content = readFileSync(refPath, "utf-8");
      referenceContent += `\n\n---\n\n${content}`;
    }
  }

  // Format globs for YAML
  const globsYaml = globs.map((g) => `  - "${g}"`).join("\n");

  // Build MDC content
  const mdcContent = `---
description: ${skillName} development patterns and best practices
globs:
${globsYaml}
---

${skillContent}
${hookInstructions}
${referenceContent}
`;

  writeFileSync(join(rulesDir, `${skillName}.mdc`), mdcContent);
}

/**
 * Generate agents rule file
 */
function generateAgentsRule(srcDir, rulesDir) {
  const agentsDir = join(srcDir, "agents");
  if (!existsSync(agentsDir)) {
    return;
  }

  const agentFiles = readdirSync(agentsDir).filter((f) => f.endsWith(".md"));

  let content = `---
description: Agent roles and responsibilities
globs:
  - "**/*"
---

# Available Agents

These are the specialized agents and their responsibilities:

`;

  for (const agentFile of agentFiles) {
    const agentPath = join(agentsDir, agentFile);
    const agentContent = readFileSync(agentPath, "utf-8");

    // Extract frontmatter
    const frontmatterMatch = agentContent.match(/^---\n([\s\S]*?)\n---/);
    if (frontmatterMatch) {
      const frontmatter = frontmatterMatch[1];
      const nameMatch = frontmatter.match(/name:\s*(.+)/);
      const descMatch = frontmatter.match(/description:\s*(.+)/);

      if (nameMatch && descMatch) {
        content += `## ${nameMatch[1]}\n\n${descMatch[1]}\n\n`;
      }
    }
  }

  writeFileSync(join(rulesDir, "agents.mdc"), content);
}
