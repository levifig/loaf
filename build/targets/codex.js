/**
 * Codex Build Target
 *
 * Generates Codex skills structure:
 * dist/codex/
 * └── .codex/
 *     └── skills/
 *         ├── python/
 *         ├── typescript/
 *         └── ...
 *
 * Codex uses the canonical skill format, so no transforms are needed.
 */

import { mkdirSync, cpSync, existsSync, rmSync } from "fs";
import { join } from "path";

/**
 * Build Codex distribution
 */
export async function build({ srcDir, distDir }) {
  const skillsDir = join(distDir, ".codex", "skills");
  if (existsSync(skillsDir)) {
    rmSync(skillsDir, { recursive: true });
  }
  mkdirSync(skillsDir, { recursive: true });

  copySkills(srcDir, skillsDir);
}

/**
 * Copy all skills
 */
function copySkills(srcDir, destDir) {
  const src = join(srcDir, "skills");

  if (existsSync(src)) {
    cpSync(src, destDir, { recursive: true });
  }
}
