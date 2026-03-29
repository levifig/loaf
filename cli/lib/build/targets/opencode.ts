/**
 * OpenCode Build Target
 *
 * Generates flat OpenCode structure (plural dirs per OpenCode config):
 * dist/opencode/
 * ├── skills/
 * ├── agents/
 * ├── commands/    (generated from skills with OpenCode sidecars)
 * └── plugins/
 *     └── hooks.js
 */

import {
  mkdirSync,
  cpSync,
  writeFileSync,
  readFileSync,
  readdirSync,
  existsSync,
  rmSync,
} from "fs";
import matter from "gray-matter";
import { join } from "path";
import { parse as parseYaml } from "yaml";
import { loadAgentSidecar, loadSkillFrontmatter } from "../lib/sidecar.js";
import { getVersion } from "../lib/version.js";

import { copySharedTemplates } from "../lib/shared-templates.js";
import { copyDirWithTransform } from "../lib/copy-utils.js";
import type { BuildContext, HooksConfig, HookDefinition } from "../types.js";

const TARGET_NAME = "opencode";

function substituteCommands(content: string): string {
  return content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/resume")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}

export async function build({
  config,
  targetsConfig,
  rootDir,
  srcDir,
  distDir,
}: BuildContext): Promise<void> {
  const version = getVersion(rootDir);
  const transformMd = (content: string) => substituteCommands(content);

  if (existsSync(distDir)) {
    rmSync(distDir, { recursive: true });
  }
  mkdirSync(distDir, { recursive: true });

  copySkills(srcDir, distDir, targetsConfig, transformMd);
  copyAgents(srcDir, distDir);
  generateCommandsFromSkills(srcDir, distDir, version);
  generateHooks(config as HooksConfig, srcDir, distDir);

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
  distDir: string,
  targetsConfig: BuildContext["targetsConfig"],
  transformMd: (content: string) => string,
): void {
  const src = join(srcDir, "skills");
  const dest = join(distDir, "skills");

  if (!existsSync(src)) return;

  mkdirSync(dest, { recursive: true });

  const skills = readdirSync(src, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillSrc = join(src, skill);
    const skillDest = join(dest, skill);
    mkdirSync(skillDest, { recursive: true });

    const frontmatter = loadSkillFrontmatter(skillSrc);

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

function copyAgents(
  srcDir: string,
  distDir: string,
): void {
  const src = join(srcDir, "agents");
  const dest = join(distDir, "agents");

  if (!existsSync(src)) return;

  mkdirSync(dest, { recursive: true });

  const files = readdirSync(src).filter((f) => f.endsWith(".md"));

  for (const file of files) {
    const srcPath = join(src, file);
    const destPath = join(dest, file);

    const content = readFileSync(srcPath, "utf-8");
    const { content: body } = matter(content);
    const frontmatter = loadAgentSidecar(srcPath, TARGET_NAME);

    writeFileSync(destPath, matter.stringify(body, frontmatter));
  }
}

function generateCommandsFromSkills(
  srcDir: string,
  distDir: string,
  version: string,
): void {
  const skillsSrc = join(srcDir, "skills");
  const commandsDest = join(distDir, "commands");

  if (!existsSync(skillsSrc)) return;

  mkdirSync(commandsDest, { recursive: true });

  const skills = readdirSync(skillsSrc, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillDir = join(skillsSrc, skill);
    const sidecarPath = join(skillDir, "SKILL.opencode.yaml");

    if (!existsSync(sidecarPath)) continue;

    const skillMdPath = join(skillDir, "SKILL.md");
    if (!existsSync(skillMdPath)) continue;

    const content = readFileSync(skillMdPath, "utf-8");
    const { content: body, data: skillFrontmatter } = matter(content);

    const sidecarContent = readFileSync(sidecarPath, "utf-8");
    const sidecar = (parseYaml(sidecarContent) as Record<string, unknown>) || {};

    const mergedFrontmatter: Record<string, unknown> = {
      description: (skillFrontmatter as Record<string, unknown>).description || "",
      ...sidecar,
      version,
    };

    // Rewrite relative links for command files
    const relinked = body
      .replace(/\]\(templates\//g, `](../skills/${skill}/templates/`)
      .replace(/\]\(references\//g, `](../skills/${skill}/references/`);

    const transformed = substituteCommands(matter.stringify(relinked, mergedFrontmatter));
    writeFileSync(join(commandsDest, `${skill}.md`), transformed);
  }
}

function getScriptFilename(scriptPath: string): string {
  const parts = scriptPath.split("/");
  return parts.slice(-2).join("/");
}

function generateHooks(
  config: HooksConfig,
  srcDir: string,
  distDir: string,
): void {
  const pluginDir = join(distDir, "plugins");
  mkdirSync(pluginDir, { recursive: true });

  const hooksSrc = join(srcDir, "hooks");
  const hooksDest = join(pluginDir, "hooks");
  if (existsSync(hooksSrc)) {
    cpSync(hooksSrc, hooksDest, { recursive: true });
  }

  const hooksJs = generateHooksJs(config);
  writeFileSync(join(pluginDir, "hooks.js"), hooksJs);
}

function generateHooksJs(config: HooksConfig): string {
  const preToolHooks = config.hooks["pre-tool"] || [];
  const postToolHooks = config.hooks["post-tool"] || [];
  const sessionHooks = config.hooks.session || [];

  // Filter out prompt hooks — OpenCode only supports command (script) hooks
  const commandPreToolHooks = preToolHooks.filter((h) => h.type !== "prompt");
  const commandPostToolHooks = postToolHooks.filter((h) => h.type !== "prompt");

  const preToolByMatcher: Record<string, HookDefinition[]> = {};
  for (const hook of commandPreToolHooks) {
    const matcher = hook.matcher || "Edit|Write";
    if (!preToolByMatcher[matcher]) preToolByMatcher[matcher] = [];
    preToolByMatcher[matcher].push(hook);
  }

  const postToolByMatcher: Record<string, HookDefinition[]> = {};
  for (const hook of commandPostToolHooks) {
    const matcher = hook.matcher || "Edit|Write";
    if (!postToolByMatcher[matcher]) postToolByMatcher[matcher] = [];
    postToolByMatcher[matcher].push(hook);
  }

  return `/**
 * OpenCode Plugin - Agent Skills Hooks
 * Auto-generated by loaf build system
 */

import { execFileSync } from 'child_process';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const HOOKS_DIR = join(__dirname, 'hooks');

function runHook(script, toolName, toolInput, timeout = 60000) {
  try {
    const scriptPath = join(HOOKS_DIR, script);
    const interpreter = script.endsWith('.py') ? 'python3' : 'bash';
    const result = execFileSync(interpreter, [scriptPath], {
      cwd: process.cwd(),
      env: {
        ...process.env,
        TOOL_NAME: toolName || '',
        TOOL_INPUT: JSON.stringify(toolInput || {}),
      },
      encoding: 'utf-8',
      timeout,
    });
    return { success: true, output: result };
  } catch (error) {
    return { success: false, error: error.message };
  }
}

function matchesTool(toolName, pattern) {
  const patterns = pattern.split('|');
  return patterns.includes(toolName);
}

const preToolHooks = {
${Object.entries(preToolByMatcher)
  .map(
    ([matcher, hooks]) => `  '${matcher}': [
${hooks.map((h) => `    { id: '${h.id}', script: '${getScriptFilename(h.script!)}', timeout: ${h.timeout || 60000} },`).join("\n")}
  ],`,
  )
  .join("\n")}
};

const postToolHooks = {
${Object.entries(postToolByMatcher)
  .map(
    ([matcher, hooks]) => `  '${matcher}': [
${hooks.map((h) => `    { id: '${h.id}', script: '${getScriptFilename(h.script!)}', timeout: ${h.timeout || 60000} },`).join("\n")}
  ],`,
  )
  .join("\n")}
};

const sessionHooks = {
${sessionHooks.map((h) => `  '${(h.event || "").toLowerCase()}': { id: '${h.id}', script: '${getScriptFilename(h.script!)}', timeout: ${h.timeout || 60000} },`).join("\n")}
};

export default async function AgentSkillsPlugin({ client, $ }) {
  return {
    'tool.execute.before': async (input, output) => {
      const toolName = input?.tool?.name;
      if (!toolName) return;

      for (const [matcher, hookList] of Object.entries(preToolHooks)) {
        if (matchesTool(toolName, matcher)) {
          for (const hook of hookList) {
            const result = runHook(hook.script, toolName, input?.tool?.input, hook.timeout);
            if (!result.success) {
              console.warn(\`[loaf] Hook \${hook.id} failed: \${result.error}\`);
            }
          }
        }
      }
    },

    'tool.execute.after': async (input, output) => {
      const toolName = input?.tool?.name;
      if (!toolName) return;

      for (const [matcher, hookList] of Object.entries(postToolHooks)) {
        if (matchesTool(toolName, matcher)) {
          for (const hook of hookList) {
            const result = runHook(hook.script, toolName, input?.tool?.input, hook.timeout);
            if (!result.success) {
              console.warn(\`[loaf] Hook \${hook.id} failed: \${result.error}\`);
            }
          }
        }
      }
    },

    'event': async ({ event }) => {
      if (event.type === 'session.created' && sessionHooks.sessionstart) {
        runHook(sessionHooks.sessionstart.script, 'session', {}, sessionHooks.sessionstart.timeout);
      }
      if (event.type === 'session.ended' && sessionHooks.sessionend) {
        runHook(sessionHooks.sessionend.script, 'session', {}, sessionHooks.sessionend.timeout);
      }
    },
  };
}
`;
}
