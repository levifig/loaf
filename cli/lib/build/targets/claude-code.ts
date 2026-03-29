/**
 * Claude Code Build Target
 *
 * Generates a single unified plugin at repo root:
 * plugins/loaf/
 * ├── .claude-plugin/plugin.json
 * ├── agents/
 * ├── skills/
 * └── hooks/
 *
 * Also creates .claude-plugin/marketplace.json at repo root.
 * Scoping: /loaf:implement, Task(loaf:backend-dev)
 */

import { mkdirSync, cpSync, writeFileSync, readFileSync, existsSync, readdirSync, rmSync } from "fs";
import matter from "gray-matter";
import { join } from "path";
import { loadAgentSidecar, loadSkillFrontmatter, loadSkillExtensions, mergeSkillFrontmatter } from "../lib/sidecar.js";
import { getVersion } from "../lib/version.js";
import { buildAgentMap, substituteAgentNames } from "../lib/substitutions.js";
import { copySharedTemplates } from "../lib/shared-templates.js";
import { copyDirWithTransform, discoverAgents, discoverSkills } from "../lib/copy-utils.js";
import type { BuildContext, HooksConfig, HookDefinition } from "../types.js";

const TARGET_NAME = "claude-code";
const PLUGIN_NAME = "loaf";
const PLUGIN_DESCRIPTION = "Loaf - An Opinionated Agentic Framework";
const REPOSITORY = "https://github.com/levifig/loaf";

const LSP_SERVERS: Record<string, unknown> = {
  go: {
    command: "gopls",
    args: ["serve"],
    extensionToLanguage: { ".go": "go" },
  },
  python: {
    command: "pyright-langserver",
    args: ["--stdio"],
    extensionToLanguage: { ".py": "python", ".pyi": "python" },
  },
  typescript: {
    command: "typescript-language-server",
    args: ["--stdio"],
    extensionToLanguage: {
      ".ts": "typescript",
      ".tsx": "typescriptreact",
      ".js": "javascript",
      ".jsx": "javascriptreact",
    },
  },
  ruby: {
    command: "solargraph",
    args: ["stdio"],
    extensionToLanguage: { ".rb": "ruby", ".rake": "ruby", ".gemspec": "ruby" },
  },
};

const MCP_SERVERS: Record<string, unknown> = {
  "sequential-thinking": {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-sequential-thinking"],
  },
  linear: {
    command: "bash",
    args: ["${CLAUDE_PLUGIN_ROOT}/hooks/linear-mcp.sh"],
  },
  serena: {
    command: "uvx",
    args: ["--from", "git+https://github.com/oraios/serena", "serena", "start-mcp-server"],
  },
};

/**
 * Substitute command references with Claude Code scoped commands.
 * Generic slash commands: /breakdown -> /loaf:breakdown (for known commands only).
 */
function substituteCommands(content: string, knownCommands: string[] = []): string {
  let result = content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/loaf:implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/loaf:resume-session")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/loaf:implement");

  for (const cmd of knownCommands) {
    const pattern = new RegExp(`(?<!/\\w+:)\\/${cmd}(?=\\s|\\)|\\]|,|$|\`)`, "g");
    result = result.replace(pattern, `/loaf:${cmd}`);
  }

  return result;
}

let VERSION = "0.0.0";

export async function build({ config, targetsConfig, rootDir, srcDir, distDir }: BuildContext): Promise<void> {
  VERSION = getVersion(rootDir);

  // distDir is the repo root for claude-code target
  const pluginsDir = join(distDir, "plugins");
  const marketplaceDir = join(distDir, ".claude-plugin");

  if (existsSync(pluginsDir)) rmSync(pluginsDir, { recursive: true });
  if (existsSync(marketplaceDir)) rmSync(marketplaceDir, { recursive: true });

  mkdirSync(pluginsDir, { recursive: true });
  mkdirSync(marketplaceDir, { recursive: true });

  createMarketplace(marketplaceDir);
  buildUnifiedPlugin(config as HooksConfig, srcDir, pluginsDir, targetsConfig);
}

function createMarketplace(marketplaceDir: string): void {
  const marketplace = {
    name: "levifig-loaf",
    owner: { name: "Levi Figueira", email: "me@levifig.com" },
    metadata: { description: PLUGIN_DESCRIPTION, version: VERSION },
    plugins: [
      {
        name: PLUGIN_NAME,
        description: PLUGIN_DESCRIPTION,
        source: `./plugins/${PLUGIN_NAME}`,
        version: VERSION,
        license: "MIT",
        repository: REPOSITORY,
      },
    ],
  };

  writeFileSync(join(marketplaceDir, "marketplace.json"), JSON.stringify(marketplace, null, 2));
}

function buildUnifiedPlugin(config: HooksConfig, srcDir: string, pluginsDir: string, targetsConfig: BuildContext["targetsConfig"]): void {
  const pluginDir = join(pluginsDir, PLUGIN_NAME);
  mkdirSync(pluginDir, { recursive: true });

  const allAgents = discoverAgents(srcDir);
  const allSkills = discoverSkills(srcDir);

  const knownCommands = allSkills.filter((skill) => {
    const extensions = loadSkillExtensions(join(srcDir, "skills", skill));
    return extensions["user-invocable"] !== false;
  });

  const agentMap = buildAgentMap(srcDir, TARGET_NAME);

  createPluginJson(config, pluginDir);
  copyAgents(allAgents, srcDir, pluginDir, agentMap);
  copySkills(allSkills, srcDir, pluginDir, knownCommands, agentMap, targetsConfig);
  copyAllHooks(config, srcDir, pluginDir);

  writeFileSync(join(pluginDir, ".lsp.json"), JSON.stringify(LSP_SERVERS, null, 2));

  const setupSrc = join(srcDir, "SETUP.md");
  if (existsSync(setupSrc)) {
    cpSync(setupSrc, join(pluginDir, "SETUP.md"));
  }
}

function groupByMatcher(hooks: HookDefinition[]): Record<string, HookDefinition[]> {
  const groups: Record<string, HookDefinition[]> = {};
  for (const hook of hooks) {
    const matcher = hook.matcher || "Edit|Write";
    if (!groups[matcher]) groups[matcher] = [];
    groups[matcher].push(hook);
  }
  return groups;
}

function getHookPath(hook: HookDefinition): string {
  const parts = hook.script!.split("/");
  const filename = parts[parts.length - 1];
  return `hooks/${filename}`;
}

function getHookCommand(hook: HookDefinition): string {
  const hookPath = getHookPath(hook);
  const filename = hookPath.split("/").pop()!;

  if (filename.endsWith(".py")) {
    return `python3 \${CLAUDE_PLUGIN_ROOT}/${hookPath}`;
  }
  return `bash \${CLAUDE_PLUGIN_ROOT}/${hookPath}`;
}

function createPluginJson(config: HooksConfig, pluginDir: string): void {
  const pluginJson: Record<string, unknown> = {
    name: PLUGIN_NAME,
    version: VERSION,
    description: PLUGIN_DESCRIPTION,
    repository: REPOSITORY,
    license: "MIT",
    hooks: {} as Record<string, unknown>,
    mcpServers: MCP_SERVERS,
  };

  const hooks = pluginJson.hooks as Record<string, unknown>;
  const allPreToolHooks = config.hooks["pre-tool"] || [];
  const allPostToolHooks = config.hooks["post-tool"] || [];
  const allSessionHooks = config.hooks.session || [];

  if (allPreToolHooks.length > 0) {
    const preToolByMatcher = groupByMatcher(allPreToolHooks);
    hooks.PreToolUse = Object.entries(preToolByMatcher).map(([matcher, hookList]) => ({
      matcher,
      hooks: hookList.map((h) => {
        if (h.type === "prompt") {
          return {
            type: "prompt" as const,
            prompt: h.prompt!,
            ...(h.if && { if: h.if }),
            ...(h.timeout && { timeout: h.timeout }),
          };
        }
        return {
          type: "command" as const,
          command: getHookCommand(h),
          ...(h.if && { if: h.if }),
          ...(h.timeout && { timeout: h.timeout }),
          ...(h.description && { description: h.description }),
        };
      }),
    }));
  }

  if (allPostToolHooks.length > 0) {
    const postToolByMatcher = groupByMatcher(allPostToolHooks);
    hooks.PostToolUse = Object.entries(postToolByMatcher).map(([matcher, hookList]) => ({
      matcher,
      hooks: hookList.map((h) => ({
        type: "command",
        command: getHookCommand(h),
        ...(h.description && { description: h.description }),
      })),
    }));
  }

  if (allSessionHooks.length > 0) {
    for (const hook of allSessionHooks) {
      const eventName = hook.event!;
      if (!hooks[eventName]) hooks[eventName] = [];
      (hooks[eventName] as unknown[]).push({
        hooks: [
          {
            type: "command",
            command: getHookCommand(hook),
            ...(hook.description && { description: hook.description }),
            ...(hook.timeout && { timeout: hook.timeout }),
          },
        ],
      });
    }
  }

  const pluginJsonDir = join(pluginDir, ".claude-plugin");
  mkdirSync(pluginJsonDir, { recursive: true });
  writeFileSync(join(pluginJsonDir, "plugin.json"), JSON.stringify(pluginJson, null, 2));
}

function copyAgents(agents: string[], srcDir: string, pluginDir: string, agentMap: Record<string, string>): void {
  const agentsDir = join(pluginDir, "agents");
  mkdirSync(agentsDir, { recursive: true });

  for (const agent of agents) {
    const srcPath = join(srcDir, "agents", `${agent}.md`);
    const destPath = join(agentsDir, `${agent}.md`);

    if (!existsSync(srcPath)) continue;

    const frontmatter = loadAgentSidecar(srcPath, TARGET_NAME);
    const content = readFileSync(srcPath, "utf-8");
    const { content: body } = matter(content);

    const transformed = substituteAgentNames(matter.stringify(body, frontmatter), agentMap);
    writeFileSync(destPath, transformed);
  }
}

function copySkills(skills: string[], srcDir: string, pluginDir: string, knownCommands: string[], agentMap: Record<string, string>, targetsConfig: BuildContext["targetsConfig"]): void {
  const skillsDir = join(pluginDir, "skills");
  mkdirSync(skillsDir, { recursive: true });

  const transformMd = (content: string) => substituteAgentNames(substituteCommands(content, knownCommands), agentMap);

  for (const skill of skills) {
    const skillSrc = join(srcDir, "skills", skill);
    const skillDest = join(skillsDir, skill);

    if (!existsSync(skillSrc)) continue;

    mkdirSync(skillDest, { recursive: true });

    const baseFrontmatter = loadSkillFrontmatter(skillSrc);
    const extensions = loadSkillExtensions(skillSrc);
    const frontmatter = mergeSkillFrontmatter(baseFrontmatter, extensions);

    const skillMdPath = join(skillSrc, "SKILL.md");
    if (existsSync(skillMdPath)) {
      const content = readFileSync(skillMdPath, "utf-8");
      const { content: body } = matter(content);
      writeFileSync(join(skillDest, "SKILL.md"), transformMd(matter.stringify(body, frontmatter)));
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

function copyAllHooks(config: HooksConfig, srcDir: string, pluginDir: string): void {
  const hooksDir = join(pluginDir, "hooks");
  mkdirSync(hooksDir, { recursive: true });

  // Copy lib directory as-is
  const libSrc = join(srcDir, "hooks", "lib");
  if (existsSync(libSrc)) {
    cpSync(libSrc, join(hooksDir, "lib"), { recursive: true });
  }

  // Collect all hook IDs
  const allHookIds = new Set<string>();
  for (const hook of config.hooks["pre-tool"] || []) allHookIds.add(hook.id);
  for (const hook of config.hooks["post-tool"] || []) allHookIds.add(hook.id);
  for (const hook of config.hooks.session || []) allHookIds.add(hook.id);

  // Find and copy each hook script
  for (const hookId of allHookIds) {
    const hookDef = config.hooks["pre-tool"]?.find((h) => h.id === hookId) || config.hooks["post-tool"]?.find((h) => h.id === hookId) || config.hooks.session?.find((h) => h.id === hookId);

    if (hookDef && hookDef.script) {
      const parts = hookDef.script.split("/");
      const filename = parts[parts.length - 1];
      const src = join(srcDir, hookDef.script);
      const dest = join(hooksDir, filename);
      if (existsSync(src)) cpSync(src, dest);
    }
  }

  // Copy Linear MCP wrapper
  const linearMcpSrc = join(srcDir, "hooks", "linear-mcp.sh");
  if (existsSync(linearMcpSrc)) {
    cpSync(linearMcpSrc, join(hooksDir, "linear-mcp.sh"));
  }

  // Copy subagent hooks as-is
  const subagentHooksSrc = join(srcDir, "hooks", "subagent");
  if (existsSync(subagentHooksSrc)) {
    const files = readdirSync(subagentHooksSrc);
    for (const file of files) {
      cpSync(join(subagentHooksSrc, file), join(hooksDir, file));
    }
  }

  // Copy instructions directory (used by workflow hooks for inline output)
  const instructionsSrc = join(srcDir, "hooks", "instructions");
  if (existsSync(instructionsSrc)) {
    cpSync(instructionsSrc, join(hooksDir, "instructions"), { recursive: true });
  }
}
