/**
 * Gemini Build Target
 *
 * Gemini only supports skills — no commands, agents, or hooks.
 * Reads from shared intermediate at dist/skills/, merges SKILL.gemini.yaml sidecar.
 */

import { join } from "path";
import { copySkills } from "../lib/skills.js";
import { loadTargetSkillSidecar } from "../lib/sidecar.js";
import { getVersion } from "../lib/version.js";
import type { BuildContext, SkillFrontmatter } from "../types.js";

const TARGET_NAME = "gemini";

export async function build({
  rootDir,
  distDir,
  targetsConfig,
}: BuildContext): Promise<void> {
  const version = getVersion(rootDir);

  // Identity transform - commands already substituted in intermediate
  const transformMd = (content: string) => content;

  // Merge sidecar fields into frontmatter
  const mergeFrontmatter = (
    base: SkillFrontmatter,
    skillDir: string,
  ): SkillFrontmatter => {
    const sidecar = loadTargetSkillSidecar(skillDir, TARGET_NAME);
    return { ...base, ...sidecar };
  };

  copySkills({
    srcDir: join(rootDir, "dist"), // Read from dist/skills/ intermediate
    destDir: join(distDir, "skills"),
    targetName: TARGET_NAME,
    version,
    targetsConfig,
    transformMd,
    mergeFrontmatter,
  });
}
