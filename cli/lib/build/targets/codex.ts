/**
 * Codex Build Target
 *
 * Codex only supports skills — no commands, agents, or hooks.
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
import { loadSkillFrontmatter, loadTargetSkillSidecar } from "../lib/sidecar.js";
import { getVersion, injectVersion } from "../lib/version.js";
import { buildAgentMap, substituteAgentNames } from "../lib/substitutions.js";
import { copySharedTemplates } from "../lib/shared-templates.js";
import { copyDirWithTransform } from "../lib/copy-utils.js";
import type { BuildContext } from "../types.js";

const TARGET_NAME = "codex";

function substituteCommands(content: string): string {
  return content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/resume")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}

export async function build({ rootDir, srcDir, distDir, targetsConfig }: BuildContext): Promise<void> {
  const version = getVersion(rootDir);
  const agentMap = buildAgentMap(srcDir, TARGET_NAME);
  const transformMd = (content: string) =>
    substituteAgentNames(substituteCommands(content), agentMap);

  const skillsDir = join(distDir, "skills");

  if (existsSync(skillsDir)) {
    rmSync(skillsDir, { recursive: true });
  }
  mkdirSync(skillsDir, { recursive: true });

  const src = join(srcDir, "skills");
  if (!existsSync(src)) return;

  const skills = readdirSync(src, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillSrc = join(src, skill);
    const skillDest = join(skillsDir, skill);
    mkdirSync(skillDest, { recursive: true });

    const baseFrontmatter = loadSkillFrontmatter(skillSrc);
    const sidecarFrontmatter = loadTargetSkillSidecar(skillSrc, TARGET_NAME);
    const frontmatter = injectVersion(
      { ...baseFrontmatter, ...sidecarFrontmatter },
      version,
    );

    const skillMdPath = join(skillSrc, "SKILL.md");
    if (existsSync(skillMdPath)) {
      const content = readFileSync(skillMdPath, "utf-8");
      const { content: body } = matter(content);
      writeFileSync(
        join(skillDest, "SKILL.md"),
        transformMd(matter.stringify(body, frontmatter)),
      );
    }

    for (const subdir of ["references", "templates"]) {
      const subSrc = join(skillSrc, subdir);
      if (existsSync(subSrc)) {
        copyDirWithTransform(subSrc, join(skillDest, subdir), transformMd);
      }
    }

    const scriptsSrc = join(skillSrc, "scripts");
    if (existsSync(scriptsSrc)) {
      cpSync(scriptsSrc, join(skillDest, "scripts"), { recursive: true });
    }

    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}
