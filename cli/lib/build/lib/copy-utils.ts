/**
 * File Copy Utilities
 *
 * Shared helpers for copying directories with markdown content transformation.
 */

import {
  mkdirSync,
  cpSync,
  writeFileSync,
  readFileSync,
  readdirSync,
  existsSync,
} from "fs";
import { join } from "path";

/**
 * Recursively copy a directory, applying a transform function to .md files.
 */
export function copyDirWithTransform(
  srcDir: string,
  destDir: string,
  transformMd: (content: string) => string,
): void {
  mkdirSync(destDir, { recursive: true });

  const entries = readdirSync(srcDir, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = join(srcDir, entry.name);
    const destPath = join(destDir, entry.name);

    if (entry.isDirectory()) {
      copyDirWithTransform(srcPath, destPath, transformMd);
    } else if (entry.name.endsWith(".md")) {
      const content = readFileSync(srcPath, "utf-8");
      writeFileSync(destPath, transformMd(content));
    } else {
      cpSync(srcPath, destPath);
    }
  }
}

/**
 * Discover all skill directories (those containing SKILL.md).
 */
export function discoverSkills(srcDir: string): string[] {
  const skillsDir = join(srcDir, "skills");
  if (!existsSync(skillsDir)) return [];

  return readdirSync(skillsDir).filter((f) => {
    const skillPath = join(skillsDir, f);
    return (
      existsSync(join(skillPath, "SKILL.md")) ||
      existsSync(join(skillPath, "references"))
    );
  });
}

/**
 * Discover all agent files (*.md) in agents directory.
 */
export function discoverAgents(srcDir: string): string[] {
  const agentsDir = join(srcDir, "agents");
  if (!existsSync(agentsDir)) return [];

  return readdirSync(agentsDir)
    .filter((f) => f.endsWith(".md"))
    .map((f) => f.replace(".md", ""));
}
