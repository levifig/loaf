/**
 * Cursor Build Target
 *
 * Cursor supports: skills, agents (subagents), hooks
 * Generates hooks.json for Cursor's hook system.
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
import { join, dirname, basename } from "path";
import { parse as parseYaml } from "yaml";
import { loadSkillFrontmatter, loadTargetSkillSidecar, loadAgentSidecarOptional } from "../lib/sidecar.js";
import { getVersion, injectVersion } from "../lib/version.js";

import { copySharedTemplates } from "../lib/shared-templates.js";
import { copyDirWithTransform } from "../lib/copy-utils.js";
import type { BuildContext, HooksConfig, HookDefinition } from "../types.js";

const TARGET_NAME = "cursor";

const DEFAULT_AGENT_FRONTMATTER = {
  model: "inherit",
  is_background: true,
};

function substituteCommands(content: string): string {
  return content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/resume")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}

export async function build({
  config,
  targetConfig,
  targetsConfig,
  rootDir,
  srcDir,
  distDir,
}: BuildContext): Promise<void> {
  const version = getVersion(rootDir);
  const transformMd = (content: string) => substituteCommands(content);

  const skillsDir = join(distDir, "skills");
  const agentsDir = join(distDir, "agents");
  const hooksDir = join(distDir, "hooks");

  // Remove stale commands directory from previous builds
  const staleCommandsDir = join(distDir, "commands");
  if (existsSync(staleCommandsDir)) {
    rmSync(staleCommandsDir, { recursive: true });
  }

  for (const dir of [skillsDir, agentsDir, hooksDir]) {
    if (existsSync(dir)) rmSync(dir, { recursive: true });
    mkdirSync(dir, { recursive: true });
  }

  copySkills(srcDir, skillsDir, version, targetsConfig, transformMd);
  copyAgents(srcDir, agentsDir, targetConfig, version);
  copyHooks(srcDir, hooksDir);
  generateHooksJson(config as HooksConfig, distDir);

  // Copy plugin-root templates (e.g. soul.md for SessionStart hook)
  const soulTemplateSrc = join(srcDir, "templates", "soul.md");
  if (existsSync(soulTemplateSrc)) {
    const templatesDir = join(distDir, "templates");
    mkdirSync(templatesDir, { recursive: true });
    cpSync(soulTemplateSrc, join(templatesDir, "soul.md"));
  }
}

function copySkills(
  srcDir: string,
  destDir: string,
  version: string,
  targetsConfig: BuildContext["targetsConfig"],
  transformMd: (content: string) => string,
): void {
  const src = join(srcDir, "skills");
  if (!existsSync(src)) return;

  const skills = readdirSync(src, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillSrc = join(src, skill);
    const skillDest = join(destDir, skill);
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

    // Cursor-specific: assets directory
    const assetsSrc = join(skillSrc, "assets");
    if (existsSync(assetsSrc)) {
      cpSync(assetsSrc, join(skillDest, "assets"), { recursive: true });
    }

    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}

function copyAgents(
  srcDir: string,
  destDir: string,
  targetConfig: BuildContext["targetConfig"],
  version: string,
): void {
  const src = join(srcDir, "agents");
  if (!existsSync(src)) return;

  const files = readdirSync(src).filter((f) => f.endsWith(".md"));

  for (const file of files) {
    const srcPath = join(src, file);
    const destPath = join(destDir, file);
    const agentName = file.replace(".md", "");

    const content = readFileSync(srcPath, "utf-8");
    const { content: body, data: sourceFrontmatter } = matter(content);

    const sidecarFrontmatter = loadAgentSidecarOptional(srcPath, TARGET_NAME);

    const defaults =
      (targetConfig as Record<string, unknown>)?.defaults
        ? ((targetConfig as { defaults?: { agents?: { frontmatter?: Record<string, unknown> } } }).defaults?.agents?.frontmatter || DEFAULT_AGENT_FRONTMATTER)
        : DEFAULT_AGENT_FRONTMATTER;

    const frontmatter: Record<string, unknown> = {
      ...defaults,
      name: (sourceFrontmatter as Record<string, unknown>).name || agentName,
      description:
        (sourceFrontmatter as Record<string, unknown>).description ||
        `${agentName} agent for specialized tasks`,
      ...sidecarFrontmatter,
    };

    const bodyWithFooter = body.trim() + `\n\n---\nversion: ${version}\n`;
    writeFileSync(destPath, matter.stringify(bodyWithFooter, frontmatter));
  }
}

function copyHooks(srcDir: string, destDir: string): void {
  const hooksSrc = join(srcDir, "hooks");
  if (!existsSync(hooksSrc)) return;

  const subdirs = ["pre-tool", "post-tool", "session", "lib", "instructions"];

  for (const subdir of subdirs) {
    const subSrc = join(hooksSrc, subdir);
    const subDest = join(destDir, subdir);

    if (existsSync(subSrc)) {
      mkdirSync(subDest, { recursive: true });
      copyHookFiles(subSrc, subDest);
    }
  }

  // Copy top-level hook files
  const entries = readdirSync(hooksSrc, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.isFile()) {
      cpSync(join(hooksSrc, entry.name), join(destDir, entry.name));
    }
  }
}

/**
 * Copy hook files with Cursor-specific override support.
 * If foo.cursor.sh exists, it replaces foo.sh in the output.
 */
function copyHookFiles(srcDir: string, destDir: string): void {
  const entries = readdirSync(srcDir, { withFileTypes: true });

  const cursorOverrides = new Set<string>();
  for (const entry of entries) {
    if (entry.isFile() && entry.name.endsWith(".cursor.sh")) {
      cursorOverrides.add(entry.name.replace(".cursor.sh", ".sh"));
    }
  }

  for (const entry of entries) {
    if (entry.isDirectory()) {
      const subSrc = join(srcDir, entry.name);
      const subDest = join(destDir, entry.name);
      mkdirSync(subDest, { recursive: true });
      copyHookFiles(subSrc, subDest);
    } else if (entry.isFile()) {
      if (entry.name.endsWith(".cursor.sh")) {
        const destName = entry.name.replace(".cursor.sh", ".sh");
        cpSync(join(srcDir, entry.name), join(destDir, destName));
      } else if (!cursorOverrides.has(entry.name)) {
        cpSync(join(srcDir, entry.name), join(destDir, entry.name));
      }
    }
  }
}

function mapSessionEvent(event: string): string {
  const mapping: Record<string, string> = {
    SessionStart: "sessionStart",
    SessionEnd: "sessionEnd",
    PreCompact: "preCompact",
  };
  return mapping[event] || event.toLowerCase();
}

function getHookCommand(hook: HookDefinition): string {
  const scriptPath = hook.script!.replace(/^hooks\//, "");
  const basePath = "$HOME/.cursor/hooks";

  if (hook.script!.endsWith(".py")) {
    return `python3 ${basePath}/${scriptPath}`;
  } else if (hook.script!.endsWith(".ts")) {
    return `bun run ${basePath}/${scriptPath}`;
  }
  return `bash ${basePath}/${scriptPath}`;
}

function generateHooksJson(config: HooksConfig, distDir: string): void {
  const preToolHooks = config.hooks["pre-tool"] || [];
  const postToolHooks = config.hooks["post-tool"] || [];
  const sessionHooks = config.hooks.session || [];

  const hooksJson: Record<string, unknown> = {
    version: 1,
    hooks: {} as Record<string, unknown>,
  };

  const hooks = hooksJson.hooks as Record<string, unknown>;

  // Filter out prompt hooks — Cursor only supports command (script) hooks
  const commandPreToolHooks = preToolHooks.filter((h) => h.type !== "prompt");
  const commandPostToolHooks = postToolHooks.filter((h) => h.type !== "prompt");

  if (commandPreToolHooks.length > 0) {
    hooks.preToolUse = commandPreToolHooks.map((hook) => ({
      command: getHookCommand(hook),
      timeout: Math.floor((hook.timeout || 60000) / 1000),
      ...(hook.matcher && { matcher: hook.matcher }),
    }));
  }

  if (commandPostToolHooks.length > 0) {
    hooks.postToolUse = commandPostToolHooks.map((hook) => ({
      command: getHookCommand(hook),
      timeout: 30,
      ...(hook.matcher && { matcher: hook.matcher }),
    }));
  }

  for (const hook of sessionHooks) {
    const eventName = mapSessionEvent(hook.event || "");
    if (!hooks[eventName]) hooks[eventName] = [];
    (hooks[eventName] as unknown[]).push({
      command: getHookCommand(hook),
      timeout: Math.floor((hook.timeout || 60000) / 1000),
    });
  }

  writeFileSync(
    join(distDir, "hooks.json"),
    JSON.stringify(hooksJson, null, 2),
  );
}
