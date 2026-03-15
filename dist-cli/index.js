#!/usr/bin/env node
var __defProp = Object.defineProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};

// cli/index.ts
import { Command } from "commander";
import { readFileSync as readFileSync13 } from "fs";
import { join as join15, dirname as dirname5 } from "path";
import { fileURLToPath as fileURLToPath3 } from "url";

// cli/commands/build.ts
import { existsSync as existsSync10, readFileSync as readFileSync11 } from "fs";
import { join as join11, dirname as dirname3 } from "path";
import { fileURLToPath } from "url";
import { parse as parseYaml4 } from "yaml";

// cli/lib/build/targets/claude-code.ts
var claude_code_exports = {};
__export(claude_code_exports, {
  build: () => build
});
import {
  mkdirSync as mkdirSync3,
  cpSync as cpSync2,
  writeFileSync as writeFileSync3,
  readFileSync as readFileSync6,
  existsSync as existsSync5,
  readdirSync as readdirSync3,
  rmSync
} from "fs";
import matter2 from "gray-matter";
import { join as join6 } from "path";

// cli/lib/build/lib/sidecar.ts
import { readFileSync, existsSync } from "fs";
import { join, dirname, basename } from "path";
import { parse as parseYaml } from "yaml";
import matter from "gray-matter";
function loadSkillFrontmatter(skillDir) {
  const skillMdPath = join(skillDir, "SKILL.md");
  if (!existsSync(skillMdPath)) {
    throw new Error(`Missing SKILL.md in ${skillDir}`);
  }
  const content = readFileSync(skillMdPath, "utf-8");
  const { data } = matter(content);
  return data;
}
function loadSkillExtensions(skillDir) {
  const sidecarPath = join(skillDir, "SKILL.claude-code.yaml");
  if (!existsSync(sidecarPath)) {
    return {};
  }
  const content = readFileSync(sidecarPath, "utf-8");
  return parseYaml(content) || {};
}
function mergeSkillFrontmatter(base, extensions) {
  return { ...base, ...extensions };
}
function loadTargetSkillSidecar(skillDir, targetName) {
  const sidecarPath = join(skillDir, `SKILL.${targetName}.yaml`);
  if (!existsSync(sidecarPath)) {
    return {};
  }
  const content = readFileSync(sidecarPath, "utf-8");
  return parseYaml(content) || {};
}
function loadAgentSidecar(sourcePath, target) {
  const dir = dirname(sourcePath);
  const baseName = basename(sourcePath, ".md");
  const sidecarPath = join(dir, `${baseName}.${target}.yaml`);
  if (!existsSync(sidecarPath)) {
    throw new Error(
      `Missing required sidecar: ${sidecarPath}
Agent '${baseName}' requires a sidecar for target '${target}'`
    );
  }
  const content = readFileSync(sidecarPath, "utf-8");
  return parseYaml(content);
}
function loadAgentSidecarOptional(sourcePath, target) {
  const dir = dirname(sourcePath);
  const baseName = basename(sourcePath, ".md");
  const sidecarPath = join(dir, `${baseName}.${target}.yaml`);
  if (!existsSync(sidecarPath)) {
    return {};
  }
  const content = readFileSync(sidecarPath, "utf-8");
  return parseYaml(content) || {};
}

// cli/lib/build/lib/version.ts
import { readFileSync as readFileSync2 } from "fs";
import { join as join2 } from "path";
function getVersion(rootDir) {
  const pkgPath = join2(rootDir, "package.json");
  const pkg = JSON.parse(readFileSync2(pkgPath, "utf-8"));
  return pkg.version;
}
function injectVersion(frontmatter, version) {
  return { ...frontmatter, version };
}

// cli/lib/build/lib/substitutions.ts
import { readFileSync as readFileSync3, readdirSync, existsSync as existsSync2 } from "fs";
import { join as join3, basename as basename2 } from "path";
import { parse as parseYaml2 } from "yaml";
function buildAgentMap(srcDir, target) {
  const agentsDir = join3(srcDir, "agents");
  const map = {};
  if (!existsSync2(agentsDir)) {
    return map;
  }
  const agentFiles = readdirSync(agentsDir).filter((f) => f.endsWith(".md"));
  for (const file of agentFiles) {
    const slug = basename2(file, ".md");
    const sidecarPath = join3(agentsDir, `${slug}.${target}.yaml`);
    if (existsSync2(sidecarPath)) {
      try {
        const content = readFileSync3(sidecarPath, "utf-8");
        const sidecar = parseYaml2(content);
        map[slug] = sidecar?.name || slug;
      } catch {
        map[slug] = slug;
      }
    } else {
      map[slug] = slug;
    }
  }
  return map;
}
function substituteAgentNames(content, agentMap) {
  return content.replace(/\{\{AGENT:([^}]+)\}\}/g, (_match, slug) => {
    if (slug in agentMap) {
      return agentMap[slug];
    }
    console.warn(
      `[loaf] Unknown agent placeholder: {{AGENT:${slug}}} \u2014 using slug as-is`
    );
    return slug;
  });
}

// cli/lib/build/lib/shared-templates.ts
import { existsSync as existsSync3, readFileSync as readFileSync4, writeFileSync, mkdirSync } from "fs";
import { join as join4 } from "path";
function copySharedTemplates(skillName, skillDest, srcDir, targetsConfig, transformFn) {
  const sharedTemplates = targetsConfig?.["shared-templates"] || {};
  for (const [templateFile, skills] of Object.entries(sharedTemplates)) {
    if (!Array.isArray(skills) || !skills.includes(skillName)) {
      continue;
    }
    const templateSrc = join4(srcDir, "templates", templateFile);
    if (!existsSync3(templateSrc)) {
      continue;
    }
    const templatesDest = join4(skillDest, "templates");
    const destPath = join4(templatesDest, templateFile);
    if (existsSync3(destPath)) {
      continue;
    }
    mkdirSync(templatesDest, { recursive: true });
    let content = readFileSync4(templateSrc, "utf-8");
    if (transformFn && templateFile.endsWith(".md")) {
      content = transformFn(content);
    }
    writeFileSync(destPath, content);
  }
}

// cli/lib/build/lib/copy-utils.ts
import {
  mkdirSync as mkdirSync2,
  cpSync,
  writeFileSync as writeFileSync2,
  readFileSync as readFileSync5,
  readdirSync as readdirSync2,
  existsSync as existsSync4
} from "fs";
import { join as join5 } from "path";
function copyDirWithTransform(srcDir, destDir, transformMd) {
  mkdirSync2(destDir, { recursive: true });
  const entries = readdirSync2(srcDir, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = join5(srcDir, entry.name);
    const destPath = join5(destDir, entry.name);
    if (entry.isDirectory()) {
      copyDirWithTransform(srcPath, destPath, transformMd);
    } else if (entry.name.endsWith(".md")) {
      const content = readFileSync5(srcPath, "utf-8");
      writeFileSync2(destPath, transformMd(content));
    } else {
      cpSync(srcPath, destPath);
    }
  }
}
function discoverSkills(srcDir) {
  const skillsDir = join5(srcDir, "skills");
  if (!existsSync4(skillsDir)) return [];
  return readdirSync2(skillsDir).filter((f) => {
    const skillPath = join5(skillsDir, f);
    return existsSync4(join5(skillPath, "SKILL.md")) || existsSync4(join5(skillPath, "references"));
  });
}
function discoverAgents(srcDir) {
  const agentsDir = join5(srcDir, "agents");
  if (!existsSync4(agentsDir)) return [];
  return readdirSync2(agentsDir).filter((f) => f.endsWith(".md")).map((f) => f.replace(".md", ""));
}

// cli/lib/build/targets/claude-code.ts
var TARGET_NAME = "claude-code";
var PLUGIN_NAME = "loaf";
var PLUGIN_DESCRIPTION = "Loaf - Levi's Opinionated Agentic Framework";
var REPOSITORY = "https://github.com/levifig/loaf";
var LSP_SERVERS = {
  go: {
    command: "gopls",
    args: ["serve"],
    extensionToLanguage: { ".go": "go" }
  },
  python: {
    command: "pyright-langserver",
    args: ["--stdio"],
    extensionToLanguage: { ".py": "python", ".pyi": "python" }
  },
  typescript: {
    command: "typescript-language-server",
    args: ["--stdio"],
    extensionToLanguage: {
      ".ts": "typescript",
      ".tsx": "typescriptreact",
      ".js": "javascript",
      ".jsx": "javascriptreact"
    }
  },
  ruby: {
    command: "solargraph",
    args: ["stdio"],
    extensionToLanguage: { ".rb": "ruby", ".rake": "ruby", ".gemspec": "ruby" }
  }
};
var MCP_SERVERS = {
  "sequential-thinking": {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-sequential-thinking"]
  },
  linear: {
    command: "bash",
    args: ["${CLAUDE_PLUGIN_ROOT}/hooks/linear-mcp.sh"]
  },
  serena: {
    command: "uvx",
    args: [
      "--from",
      "git+https://github.com/oraios/serena",
      "serena",
      "start-mcp-server"
    ]
  }
};
function substituteCommands(content, knownCommands = []) {
  let result = content.replace(/\{\{IMPLEMENT_CMD\}\}/g, "/loaf:implement").replace(/\{\{RESUME_CMD\}\}/g, "/loaf:resume-session").replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/loaf:implement");
  for (const cmd of knownCommands) {
    const pattern = new RegExp(
      `(?<!/\\w+:)\\/${cmd}(?=\\s|\\)|\\]|,|$|\`)`,
      "g"
    );
    result = result.replace(pattern, `/loaf:${cmd}`);
  }
  return result;
}
var VERSION = "0.0.0";
async function build({
  config,
  targetsConfig,
  rootDir,
  srcDir,
  distDir
}) {
  VERSION = getVersion(rootDir);
  const pluginsDir = join6(distDir, "plugins");
  const marketplaceDir = join6(distDir, ".claude-plugin");
  if (existsSync5(pluginsDir)) rmSync(pluginsDir, { recursive: true });
  if (existsSync5(marketplaceDir)) rmSync(marketplaceDir, { recursive: true });
  mkdirSync3(pluginsDir, { recursive: true });
  mkdirSync3(marketplaceDir, { recursive: true });
  createMarketplace(marketplaceDir);
  buildUnifiedPlugin(config, srcDir, pluginsDir, targetsConfig);
}
function createMarketplace(marketplaceDir) {
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
        repository: REPOSITORY
      }
    ]
  };
  writeFileSync3(
    join6(marketplaceDir, "marketplace.json"),
    JSON.stringify(marketplace, null, 2)
  );
}
function buildUnifiedPlugin(config, srcDir, pluginsDir, targetsConfig) {
  const pluginDir = join6(pluginsDir, PLUGIN_NAME);
  mkdirSync3(pluginDir, { recursive: true });
  const allAgents = discoverAgents(srcDir);
  const allSkills = discoverSkills(srcDir);
  const knownCommands = allSkills.filter((skill) => {
    const extensions = loadSkillExtensions(join6(srcDir, "skills", skill));
    return extensions["user-invocable"] !== false;
  });
  const agentMap = buildAgentMap(srcDir, TARGET_NAME);
  createPluginJson(config, pluginDir);
  copyAgents(allAgents, srcDir, pluginDir, agentMap);
  copySkills(allSkills, srcDir, pluginDir, knownCommands, agentMap, targetsConfig);
  copyAllHooks(config, srcDir, pluginDir);
  writeFileSync3(
    join6(pluginDir, ".lsp.json"),
    JSON.stringify(LSP_SERVERS, null, 2)
  );
  const setupSrc = join6(srcDir, "SETUP.md");
  if (existsSync5(setupSrc)) {
    cpSync2(setupSrc, join6(pluginDir, "SETUP.md"));
  }
}
function groupByMatcher(hooks) {
  const groups = {};
  for (const hook of hooks) {
    const matcher = hook.matcher || "Edit|Write";
    if (!groups[matcher]) groups[matcher] = [];
    groups[matcher].push(hook);
  }
  return groups;
}
function getHookPath(hook) {
  const parts = hook.script.split("/");
  const filename = parts[parts.length - 1];
  return `hooks/${filename}`;
}
function getHookCommand(hook) {
  const hookPath = getHookPath(hook);
  const filename = hookPath.split("/").pop();
  if (filename.endsWith(".py")) {
    return `python3 \${CLAUDE_PLUGIN_ROOT}/${hookPath}`;
  }
  return `bash \${CLAUDE_PLUGIN_ROOT}/${hookPath}`;
}
function createPluginJson(config, pluginDir) {
  const pluginJson = {
    name: PLUGIN_NAME,
    version: VERSION,
    description: PLUGIN_DESCRIPTION,
    repository: REPOSITORY,
    license: "MIT",
    hooks: {},
    mcpServers: MCP_SERVERS
  };
  const hooks = pluginJson.hooks;
  const allPreToolHooks = config.hooks["pre-tool"] || [];
  const allPostToolHooks = config.hooks["post-tool"] || [];
  const allSessionHooks = config.hooks.session || [];
  if (allPreToolHooks.length > 0) {
    const preToolByMatcher = groupByMatcher(allPreToolHooks);
    hooks.PreToolUse = Object.entries(preToolByMatcher).map(
      ([matcher, hookList]) => ({
        matcher,
        hooks: hookList.map((h) => ({
          type: "command",
          command: getHookCommand(h),
          ...h.timeout && { timeout: h.timeout },
          ...h.description && { description: h.description }
        }))
      })
    );
  }
  if (allPostToolHooks.length > 0) {
    const postToolByMatcher = groupByMatcher(allPostToolHooks);
    hooks.PostToolUse = Object.entries(postToolByMatcher).map(
      ([matcher, hookList]) => ({
        matcher,
        hooks: hookList.map((h) => ({
          type: "command",
          command: getHookCommand(h),
          ...h.description && { description: h.description }
        }))
      })
    );
  }
  if (allSessionHooks.length > 0) {
    for (const hook of allSessionHooks) {
      const eventName = hook.event;
      if (!hooks[eventName]) hooks[eventName] = [];
      hooks[eventName].push({
        hooks: [
          {
            type: "command",
            command: getHookCommand(hook),
            ...hook.description && { description: hook.description },
            ...hook.timeout && { timeout: hook.timeout }
          }
        ]
      });
    }
  }
  const pluginJsonDir = join6(pluginDir, ".claude-plugin");
  mkdirSync3(pluginJsonDir, { recursive: true });
  writeFileSync3(
    join6(pluginJsonDir, "plugin.json"),
    JSON.stringify(pluginJson, null, 2)
  );
}
function copyAgents(agents, srcDir, pluginDir, agentMap) {
  const agentsDir = join6(pluginDir, "agents");
  mkdirSync3(agentsDir, { recursive: true });
  for (const agent of agents) {
    const srcPath = join6(srcDir, "agents", `${agent}.md`);
    const destPath = join6(agentsDir, `${agent}.md`);
    if (!existsSync5(srcPath)) continue;
    const frontmatter = loadAgentSidecar(srcPath, TARGET_NAME);
    const content = readFileSync6(srcPath, "utf-8");
    const { content: body } = matter2(content);
    const transformed = substituteAgentNames(
      matter2.stringify(body, frontmatter),
      agentMap
    );
    writeFileSync3(destPath, transformed);
  }
}
function copySkills(skills, srcDir, pluginDir, knownCommands, agentMap, targetsConfig) {
  const skillsDir = join6(pluginDir, "skills");
  mkdirSync3(skillsDir, { recursive: true });
  const transformMd = (content) => substituteAgentNames(
    substituteCommands(content, knownCommands),
    agentMap
  );
  for (const skill of skills) {
    const skillSrc = join6(srcDir, "skills", skill);
    const skillDest = join6(skillsDir, skill);
    if (!existsSync5(skillSrc)) continue;
    mkdirSync3(skillDest, { recursive: true });
    const baseFrontmatter = loadSkillFrontmatter(skillSrc);
    const extensions = loadSkillExtensions(skillSrc);
    const frontmatter = mergeSkillFrontmatter(baseFrontmatter, extensions);
    const skillMdPath = join6(skillSrc, "SKILL.md");
    if (existsSync5(skillMdPath)) {
      const content = readFileSync6(skillMdPath, "utf-8");
      const { content: body } = matter2(content);
      writeFileSync3(
        join6(skillDest, "SKILL.md"),
        transformMd(matter2.stringify(body, frontmatter))
      );
    }
    for (const subdir of ["references", "templates"]) {
      const subSrc = join6(skillSrc, subdir);
      if (existsSync5(subSrc)) {
        copyDirWithTransform(subSrc, join6(skillDest, subdir), transformMd);
      }
    }
    const scriptsSrc = join6(skillSrc, "scripts");
    if (existsSync5(scriptsSrc)) {
      cpSync2(scriptsSrc, join6(skillDest, "scripts"), { recursive: true });
    }
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}
function copyAllHooks(config, srcDir, pluginDir) {
  const hooksDir = join6(pluginDir, "hooks");
  mkdirSync3(hooksDir, { recursive: true });
  const libSrc = join6(srcDir, "hooks", "lib");
  if (existsSync5(libSrc)) {
    cpSync2(libSrc, join6(hooksDir, "lib"), { recursive: true });
  }
  const allHookIds = /* @__PURE__ */ new Set();
  for (const hook of config.hooks["pre-tool"] || []) allHookIds.add(hook.id);
  for (const hook of config.hooks["post-tool"] || []) allHookIds.add(hook.id);
  for (const hook of config.hooks.session || []) allHookIds.add(hook.id);
  for (const hookId of allHookIds) {
    const hookDef = config.hooks["pre-tool"]?.find((h) => h.id === hookId) || config.hooks["post-tool"]?.find((h) => h.id === hookId) || config.hooks.session?.find((h) => h.id === hookId);
    if (hookDef) {
      const parts = hookDef.script.split("/");
      const filename = parts[parts.length - 1];
      const src = join6(srcDir, hookDef.script);
      const dest = join6(hooksDir, filename);
      if (existsSync5(src)) cpSync2(src, dest);
    }
  }
  const linearMcpSrc = join6(srcDir, "hooks", "linear-mcp.sh");
  if (existsSync5(linearMcpSrc)) {
    cpSync2(linearMcpSrc, join6(hooksDir, "linear-mcp.sh"));
  }
  const subagentHooksSrc = join6(srcDir, "hooks", "subagent");
  if (existsSync5(subagentHooksSrc)) {
    const files = readdirSync3(subagentHooksSrc);
    for (const file of files) {
      cpSync2(join6(subagentHooksSrc, file), join6(hooksDir, file));
    }
  }
}

// cli/lib/build/targets/opencode.ts
var opencode_exports = {};
__export(opencode_exports, {
  build: () => build2
});
import {
  mkdirSync as mkdirSync4,
  cpSync as cpSync3,
  writeFileSync as writeFileSync4,
  readFileSync as readFileSync7,
  readdirSync as readdirSync4,
  existsSync as existsSync6,
  rmSync as rmSync2
} from "fs";
import matter3 from "gray-matter";
import { join as join7 } from "path";
import { parse as parseYaml3 } from "yaml";
var TARGET_NAME2 = "opencode";
function substituteCommands2(content) {
  return content.replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement").replace(/\{\{RESUME_CMD\}\}/g, "/resume").replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}
async function build2({
  config,
  targetsConfig,
  rootDir,
  srcDir,
  distDir
}) {
  const version = getVersion(rootDir);
  const agentMap = buildAgentMap(srcDir, TARGET_NAME2);
  const transformMd = (content) => substituteAgentNames(substituteCommands2(content), agentMap);
  if (existsSync6(distDir)) {
    rmSync2(distDir, { recursive: true });
  }
  mkdirSync4(distDir, { recursive: true });
  copySkills2(srcDir, distDir, targetsConfig, transformMd);
  copyAgents2(srcDir, distDir, agentMap);
  generateCommandsFromSkills(srcDir, distDir, version, agentMap);
  generateHooks(config, srcDir, distDir);
}
function copySkills2(srcDir, distDir, targetsConfig, transformMd) {
  const src = join7(srcDir, "skills");
  const dest = join7(distDir, "skills");
  if (!existsSync6(src)) return;
  mkdirSync4(dest, { recursive: true });
  const skills = readdirSync4(src, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name);
  for (const skill of skills) {
    const skillSrc = join7(src, skill);
    const skillDest = join7(dest, skill);
    mkdirSync4(skillDest, { recursive: true });
    const frontmatter = loadSkillFrontmatter(skillSrc);
    const skillMdPath = join7(skillSrc, "SKILL.md");
    if (existsSync6(skillMdPath)) {
      const content = readFileSync7(skillMdPath, "utf-8");
      const { content: body } = matter3(content);
      writeFileSync4(
        join7(skillDest, "SKILL.md"),
        transformMd(matter3.stringify(body, frontmatter))
      );
    }
    for (const subdir of ["references", "templates"]) {
      const subSrc = join7(skillSrc, subdir);
      if (existsSync6(subSrc)) {
        copyDirWithTransform(subSrc, join7(skillDest, subdir), transformMd);
      }
    }
    const scriptsSrc = join7(skillSrc, "scripts");
    if (existsSync6(scriptsSrc)) {
      cpSync3(scriptsSrc, join7(skillDest, "scripts"), { recursive: true });
    }
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}
function copyAgents2(srcDir, distDir, agentMap) {
  const src = join7(srcDir, "agents");
  const dest = join7(distDir, "agents");
  if (!existsSync6(src)) return;
  mkdirSync4(dest, { recursive: true });
  const files = readdirSync4(src).filter((f) => f.endsWith(".md"));
  for (const file of files) {
    const srcPath = join7(src, file);
    const destPath = join7(dest, file);
    const content = readFileSync7(srcPath, "utf-8");
    const { content: body } = matter3(content);
    const frontmatter = loadAgentSidecar(srcPath, TARGET_NAME2);
    const transformed = substituteAgentNames(
      matter3.stringify(body, frontmatter),
      agentMap
    );
    writeFileSync4(destPath, transformed);
  }
}
function generateCommandsFromSkills(srcDir, distDir, version, agentMap) {
  const skillsSrc = join7(srcDir, "skills");
  const commandsDest = join7(distDir, "commands");
  if (!existsSync6(skillsSrc)) return;
  mkdirSync4(commandsDest, { recursive: true });
  const skills = readdirSync4(skillsSrc, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name);
  for (const skill of skills) {
    const skillDir = join7(skillsSrc, skill);
    const sidecarPath = join7(skillDir, "SKILL.opencode.yaml");
    if (!existsSync6(sidecarPath)) continue;
    const skillMdPath = join7(skillDir, "SKILL.md");
    if (!existsSync6(skillMdPath)) continue;
    const content = readFileSync7(skillMdPath, "utf-8");
    const { content: body, data: skillFrontmatter } = matter3(content);
    const sidecarContent = readFileSync7(sidecarPath, "utf-8");
    const sidecar = parseYaml3(sidecarContent) || {};
    const mergedFrontmatter = {
      description: skillFrontmatter.description || "",
      ...sidecar,
      version
    };
    if (mergedFrontmatter.agent && typeof mergedFrontmatter.agent === "string") {
      mergedFrontmatter.agent = substituteAgentNames(mergedFrontmatter.agent, agentMap);
    }
    const relinked = body.replace(/\]\(templates\//g, `](../skills/${skill}/templates/`).replace(/\]\(references\//g, `](../skills/${skill}/references/`);
    const transformed = substituteAgentNames(
      substituteCommands2(matter3.stringify(relinked, mergedFrontmatter)),
      agentMap
    );
    writeFileSync4(join7(commandsDest, `${skill}.md`), transformed);
  }
}
function getScriptFilename(scriptPath) {
  const parts = scriptPath.split("/");
  return parts.slice(-2).join("/");
}
function generateHooks(config, srcDir, distDir) {
  const pluginDir = join7(distDir, "plugins");
  mkdirSync4(pluginDir, { recursive: true });
  const hooksSrc = join7(srcDir, "hooks");
  const hooksDest = join7(pluginDir, "hooks");
  if (existsSync6(hooksSrc)) {
    cpSync3(hooksSrc, hooksDest, { recursive: true });
  }
  const hooksJs = generateHooksJs(config);
  writeFileSync4(join7(pluginDir, "hooks.js"), hooksJs);
}
function generateHooksJs(config) {
  const preToolHooks = config.hooks["pre-tool"] || [];
  const postToolHooks = config.hooks["post-tool"] || [];
  const sessionHooks = config.hooks.session || [];
  const preToolByMatcher = {};
  for (const hook of preToolHooks) {
    const matcher = hook.matcher || "Edit|Write";
    if (!preToolByMatcher[matcher]) preToolByMatcher[matcher] = [];
    preToolByMatcher[matcher].push(hook);
  }
  const postToolByMatcher = {};
  for (const hook of postToolHooks) {
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
${Object.entries(preToolByMatcher).map(
    ([matcher, hooks]) => `  '${matcher}': [
${hooks.map((h) => `    { id: '${h.id}', script: '${getScriptFilename(h.script)}', timeout: ${h.timeout || 6e4} },`).join("\n")}
  ],`
  ).join("\n")}
};

const postToolHooks = {
${Object.entries(postToolByMatcher).map(
    ([matcher, hooks]) => `  '${matcher}': [
${hooks.map((h) => `    { id: '${h.id}', script: '${getScriptFilename(h.script)}', timeout: ${h.timeout || 6e4} },`).join("\n")}
  ],`
  ).join("\n")}
};

const sessionHooks = {
${sessionHooks.map((h) => `  '${(h.event || "").toLowerCase()}': { id: '${h.id}', script: '${getScriptFilename(h.script)}', timeout: ${h.timeout || 6e4} },`).join("\n")}
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

// cli/lib/build/targets/cursor.ts
var cursor_exports = {};
__export(cursor_exports, {
  build: () => build3
});
import {
  mkdirSync as mkdirSync5,
  cpSync as cpSync4,
  writeFileSync as writeFileSync5,
  readFileSync as readFileSync8,
  existsSync as existsSync7,
  readdirSync as readdirSync5,
  rmSync as rmSync3
} from "fs";
import matter4 from "gray-matter";
import { join as join8 } from "path";
var TARGET_NAME3 = "cursor";
var DEFAULT_AGENT_FRONTMATTER = {
  model: "inherit",
  is_background: true
};
var PM_AGENT_FRONTMATTER = {
  model: "inherit",
  is_background: false
};
function substituteCommands3(content) {
  return content.replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement").replace(/\{\{RESUME_CMD\}\}/g, "/resume").replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}
async function build3({
  config,
  targetConfig,
  targetsConfig,
  rootDir,
  srcDir,
  distDir
}) {
  const version = getVersion(rootDir);
  const agentMap = buildAgentMap(srcDir, TARGET_NAME3);
  const transformMd = (content) => substituteAgentNames(substituteCommands3(content), agentMap);
  const skillsDir = join8(distDir, "skills");
  const agentsDir = join8(distDir, "agents");
  const hooksDir = join8(distDir, "hooks");
  const staleCommandsDir = join8(distDir, "commands");
  if (existsSync7(staleCommandsDir)) {
    rmSync3(staleCommandsDir, { recursive: true });
  }
  for (const dir of [skillsDir, agentsDir, hooksDir]) {
    if (existsSync7(dir)) rmSync3(dir, { recursive: true });
    mkdirSync5(dir, { recursive: true });
  }
  copySkills3(srcDir, skillsDir, version, targetsConfig, transformMd);
  copyAgents3(srcDir, agentsDir, targetConfig, version, agentMap);
  copyHooks(srcDir, hooksDir);
  generateHooksJson(config, distDir);
}
function copySkills3(srcDir, destDir, version, targetsConfig, transformMd) {
  const src = join8(srcDir, "skills");
  if (!existsSync7(src)) return;
  const skills = readdirSync5(src, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name);
  for (const skill of skills) {
    const skillSrc = join8(src, skill);
    const skillDest = join8(destDir, skill);
    mkdirSync5(skillDest, { recursive: true });
    const baseFrontmatter = loadSkillFrontmatter(skillSrc);
    const sidecarFrontmatter = loadTargetSkillSidecar(skillSrc, TARGET_NAME3);
    const frontmatter = injectVersion(
      { ...baseFrontmatter, ...sidecarFrontmatter },
      version
    );
    const skillMdPath = join8(skillSrc, "SKILL.md");
    if (existsSync7(skillMdPath)) {
      const content = readFileSync8(skillMdPath, "utf-8");
      const { content: body } = matter4(content);
      writeFileSync5(
        join8(skillDest, "SKILL.md"),
        transformMd(matter4.stringify(body, frontmatter))
      );
    }
    for (const subdir of ["references", "templates"]) {
      const subSrc = join8(skillSrc, subdir);
      if (existsSync7(subSrc)) {
        copyDirWithTransform(subSrc, join8(skillDest, subdir), transformMd);
      }
    }
    const scriptsSrc = join8(skillSrc, "scripts");
    if (existsSync7(scriptsSrc)) {
      cpSync4(scriptsSrc, join8(skillDest, "scripts"), { recursive: true });
    }
    const assetsSrc = join8(skillSrc, "assets");
    if (existsSync7(assetsSrc)) {
      cpSync4(assetsSrc, join8(skillDest, "assets"), { recursive: true });
    }
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}
function copyAgents3(srcDir, destDir, targetConfig, version, agentMap) {
  const src = join8(srcDir, "agents");
  if (!existsSync7(src)) return;
  const files = readdirSync5(src).filter((f) => f.endsWith(".md"));
  for (const file of files) {
    const srcPath = join8(src, file);
    const destPath = join8(destDir, file);
    const agentName = file.replace(".md", "");
    const content = readFileSync8(srcPath, "utf-8");
    const { content: body, data: sourceFrontmatter } = matter4(content);
    const sidecarFrontmatter = loadAgentSidecarOptional(srcPath, TARGET_NAME3);
    const defaults = agentName === "pm" ? PM_AGENT_FRONTMATTER : targetConfig?.defaults ? targetConfig.defaults?.agents?.frontmatter || DEFAULT_AGENT_FRONTMATTER : DEFAULT_AGENT_FRONTMATTER;
    const frontmatter = {
      ...defaults,
      name: sourceFrontmatter.name || agentName,
      description: sourceFrontmatter.description || `${agentName} agent for specialized tasks`,
      ...sidecarFrontmatter
    };
    const bodyWithFooter = body.trim() + `

---
version: ${version}
`;
    const transformed = substituteAgentNames(
      matter4.stringify(bodyWithFooter, frontmatter),
      agentMap
    );
    writeFileSync5(destPath, transformed);
  }
}
function copyHooks(srcDir, destDir) {
  const hooksSrc = join8(srcDir, "hooks");
  if (!existsSync7(hooksSrc)) return;
  const subdirs = ["pre-tool", "post-tool", "session", "lib"];
  for (const subdir of subdirs) {
    const subSrc = join8(hooksSrc, subdir);
    const subDest = join8(destDir, subdir);
    if (existsSync7(subSrc)) {
      mkdirSync5(subDest, { recursive: true });
      copyHookFiles(subSrc, subDest);
    }
  }
  const entries = readdirSync5(hooksSrc, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.isFile()) {
      cpSync4(join8(hooksSrc, entry.name), join8(destDir, entry.name));
    }
  }
}
function copyHookFiles(srcDir, destDir) {
  const entries = readdirSync5(srcDir, { withFileTypes: true });
  const cursorOverrides = /* @__PURE__ */ new Set();
  for (const entry of entries) {
    if (entry.isFile() && entry.name.endsWith(".cursor.sh")) {
      cursorOverrides.add(entry.name.replace(".cursor.sh", ".sh"));
    }
  }
  for (const entry of entries) {
    if (entry.isDirectory()) {
      const subSrc = join8(srcDir, entry.name);
      const subDest = join8(destDir, entry.name);
      mkdirSync5(subDest, { recursive: true });
      copyHookFiles(subSrc, subDest);
    } else if (entry.isFile()) {
      if (entry.name.endsWith(".cursor.sh")) {
        const destName = entry.name.replace(".cursor.sh", ".sh");
        cpSync4(join8(srcDir, entry.name), join8(destDir, destName));
      } else if (!cursorOverrides.has(entry.name)) {
        cpSync4(join8(srcDir, entry.name), join8(destDir, entry.name));
      }
    }
  }
}
function mapSessionEvent(event) {
  const mapping = {
    SessionStart: "sessionStart",
    SessionEnd: "sessionEnd",
    PreCompact: "preCompact"
  };
  return mapping[event] || event.toLowerCase();
}
function getHookCommand2(hook) {
  const scriptPath = hook.script.replace(/^hooks\//, "");
  const basePath = "$HOME/.cursor/hooks";
  if (hook.script.endsWith(".py")) {
    return `python3 ${basePath}/${scriptPath}`;
  } else if (hook.script.endsWith(".ts")) {
    return `bun run ${basePath}/${scriptPath}`;
  }
  return `bash ${basePath}/${scriptPath}`;
}
function generateHooksJson(config, distDir) {
  const preToolHooks = config.hooks["pre-tool"] || [];
  const postToolHooks = config.hooks["post-tool"] || [];
  const sessionHooks = config.hooks.session || [];
  const hooksJson = {
    version: 1,
    hooks: {}
  };
  const hooks = hooksJson.hooks;
  if (preToolHooks.length > 0) {
    hooks.preToolUse = preToolHooks.map((hook) => ({
      command: getHookCommand2(hook),
      timeout: Math.floor((hook.timeout || 6e4) / 1e3),
      ...hook.matcher && { matcher: hook.matcher }
    }));
  }
  if (postToolHooks.length > 0) {
    hooks.postToolUse = postToolHooks.map((hook) => ({
      command: getHookCommand2(hook),
      timeout: 30,
      ...hook.matcher && { matcher: hook.matcher }
    }));
  }
  for (const hook of sessionHooks) {
    const eventName = mapSessionEvent(hook.event || "");
    if (!hooks[eventName]) hooks[eventName] = [];
    hooks[eventName].push({
      command: getHookCommand2(hook),
      timeout: Math.floor((hook.timeout || 6e4) / 1e3)
    });
  }
  writeFileSync5(
    join8(distDir, "hooks.json"),
    JSON.stringify(hooksJson, null, 2)
  );
}

// cli/lib/build/targets/codex.ts
var codex_exports = {};
__export(codex_exports, {
  build: () => build4
});
import {
  mkdirSync as mkdirSync6,
  cpSync as cpSync5,
  writeFileSync as writeFileSync6,
  readFileSync as readFileSync9,
  existsSync as existsSync8,
  readdirSync as readdirSync6,
  rmSync as rmSync4
} from "fs";
import matter5 from "gray-matter";
import { join as join9 } from "path";
var TARGET_NAME4 = "codex";
function substituteCommands4(content) {
  return content.replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement").replace(/\{\{RESUME_CMD\}\}/g, "/resume").replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}
async function build4({ rootDir, srcDir, distDir, targetsConfig }) {
  const version = getVersion(rootDir);
  const agentMap = buildAgentMap(srcDir, TARGET_NAME4);
  const transformMd = (content) => substituteAgentNames(substituteCommands4(content), agentMap);
  const skillsDir = join9(distDir, "skills");
  if (existsSync8(skillsDir)) {
    rmSync4(skillsDir, { recursive: true });
  }
  mkdirSync6(skillsDir, { recursive: true });
  const src = join9(srcDir, "skills");
  if (!existsSync8(src)) return;
  const skills = readdirSync6(src, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name);
  for (const skill of skills) {
    const skillSrc = join9(src, skill);
    const skillDest = join9(skillsDir, skill);
    mkdirSync6(skillDest, { recursive: true });
    const baseFrontmatter = loadSkillFrontmatter(skillSrc);
    const sidecarFrontmatter = loadTargetSkillSidecar(skillSrc, TARGET_NAME4);
    const frontmatter = injectVersion(
      { ...baseFrontmatter, ...sidecarFrontmatter },
      version
    );
    const skillMdPath = join9(skillSrc, "SKILL.md");
    if (existsSync8(skillMdPath)) {
      const content = readFileSync9(skillMdPath, "utf-8");
      const { content: body } = matter5(content);
      writeFileSync6(
        join9(skillDest, "SKILL.md"),
        transformMd(matter5.stringify(body, frontmatter))
      );
    }
    for (const subdir of ["references", "templates"]) {
      const subSrc = join9(skillSrc, subdir);
      if (existsSync8(subSrc)) {
        copyDirWithTransform(subSrc, join9(skillDest, subdir), transformMd);
      }
    }
    const scriptsSrc = join9(skillSrc, "scripts");
    if (existsSync8(scriptsSrc)) {
      cpSync5(scriptsSrc, join9(skillDest, "scripts"), { recursive: true });
    }
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}

// cli/lib/build/targets/gemini.ts
var gemini_exports = {};
__export(gemini_exports, {
  build: () => build5
});
import {
  mkdirSync as mkdirSync7,
  cpSync as cpSync6,
  writeFileSync as writeFileSync7,
  readFileSync as readFileSync10,
  existsSync as existsSync9,
  readdirSync as readdirSync7,
  rmSync as rmSync5
} from "fs";
import matter6 from "gray-matter";
import { join as join10 } from "path";
var TARGET_NAME5 = "gemini";
function substituteCommands5(content) {
  return content.replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement").replace(/\{\{RESUME_CMD\}\}/g, "/resume").replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}
async function build5({ rootDir, srcDir, distDir, targetsConfig }) {
  const version = getVersion(rootDir);
  const agentMap = buildAgentMap(srcDir, TARGET_NAME5);
  const transformMd = (content) => substituteAgentNames(substituteCommands5(content), agentMap);
  const skillsDir = join10(distDir, "skills");
  if (existsSync9(skillsDir)) {
    rmSync5(skillsDir, { recursive: true });
  }
  mkdirSync7(skillsDir, { recursive: true });
  const src = join10(srcDir, "skills");
  if (!existsSync9(src)) return;
  const skills = readdirSync7(src, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name);
  for (const skill of skills) {
    const skillSrc = join10(src, skill);
    const skillDest = join10(skillsDir, skill);
    mkdirSync7(skillDest, { recursive: true });
    const baseFrontmatter = loadSkillFrontmatter(skillSrc);
    const sidecarFrontmatter = loadTargetSkillSidecar(skillSrc, TARGET_NAME5);
    const frontmatter = injectVersion(
      { ...baseFrontmatter, ...sidecarFrontmatter },
      version
    );
    const skillMdPath = join10(skillSrc, "SKILL.md");
    if (existsSync9(skillMdPath)) {
      const content = readFileSync10(skillMdPath, "utf-8");
      const { content: body } = matter6(content);
      writeFileSync7(
        join10(skillDest, "SKILL.md"),
        transformMd(matter6.stringify(body, frontmatter))
      );
    }
    for (const subdir of ["references", "templates"]) {
      const subSrc = join10(skillSrc, subdir);
      if (existsSync9(subSrc)) {
        copyDirWithTransform(subSrc, join10(skillDest, subdir), transformMd);
      }
    }
    const scriptsSrc = join10(skillSrc, "scripts");
    if (existsSync9(scriptsSrc)) {
      cpSync6(scriptsSrc, join10(skillDest, "scripts"), { recursive: true });
    }
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, transformMd);
  }
}

// cli/commands/build.ts
var __dirname = dirname3(fileURLToPath(import.meta.url));
var bold = (s) => `\x1B[1m${s}\x1B[0m`;
var green = (s) => `\x1B[32m${s}\x1B[0m`;
var red = (s) => `\x1B[31m${s}\x1B[0m`;
var gray = (s) => `\x1B[90m${s}\x1B[0m`;
var cyan = (s) => `\x1B[36m${s}\x1B[0m`;
var TARGETS = {
  "claude-code": claude_code_exports,
  opencode: opencode_exports,
  cursor: cursor_exports,
  codex: codex_exports,
  gemini: gemini_exports
};
function findRootDir() {
  let dir = __dirname;
  for (let i = 0; i < 10; i++) {
    const pkgPath = join11(dir, "package.json");
    try {
      const pkg = JSON.parse(readFileSync11(pkgPath, "utf-8"));
      if (pkg.name === "loaf") return dir;
    } catch {
    }
    const parent = dirname3(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error("Could not find loaf root directory (no package.json with name 'loaf')");
}
function loadYamlConfig(path) {
  if (!existsSync10(path)) return {};
  return parseYaml4(readFileSync11(path, "utf-8"));
}
var TARGET_NAMES = Object.keys(TARGETS);
async function buildTarget(targetName, rootDir, contentDir, distDir, hooksConfig, targetsConfig) {
  const targetModule = TARGETS[targetName];
  if (!targetModule) {
    throw new Error(`Unknown target: ${targetName}`);
  }
  const outputDir = targetName === "claude-code" ? rootDir : join11(distDir, targetName);
  const targetConfig = targetsConfig.targets?.[targetName] || {};
  await targetModule.build({
    config: hooksConfig,
    targetConfig,
    targetsConfig,
    rootDir,
    srcDir: contentDir,
    distDir: outputDir,
    targetName
  });
}
function registerBuildCommand(program2) {
  program2.command("build").description("Build skill distributions for agent harnesses").option("-t, --target <name>", "Build a specific target only").action(async (options) => {
    const startTime = Date.now();
    const rootDir = findRootDir();
    const contentDir = join11(rootDir, "content");
    const configDir = join11(rootDir, "config");
    const distDir = join11(rootDir, "dist");
    console.log(`
${bold("loaf build")}
`);
    if (options.target && !TARGET_NAMES.includes(options.target)) {
      console.error(
        `${red("error:")} Unknown target ${bold(options.target)}
${gray("Valid targets:")} ${TARGET_NAMES.join(", ")}`
      );
      process.exit(1);
    }
    const hooksConfigPath = join11(configDir, "hooks.yaml");
    if (!existsSync10(hooksConfigPath)) {
      console.error(`${red("error:")} Hooks config not found: ${hooksConfigPath}`);
      process.exit(1);
    }
    const hooksConfig = loadYamlConfig(hooksConfigPath);
    const targetsConfig = loadYamlConfig(join11(configDir, "targets.yaml"));
    const targets = options.target ? [options.target] : TARGET_NAMES;
    let failed = false;
    for (const targetName of targets) {
      const targetStart = Date.now();
      process.stdout.write(`  ${cyan("building")} ${targetName}...`);
      try {
        await buildTarget(
          targetName,
          rootDir,
          contentDir,
          distDir,
          hooksConfig,
          targetsConfig
        );
        const elapsed = ((Date.now() - targetStart) / 1e3).toFixed(2);
        process.stdout.write(`\r  ${green("\u2713")} ${targetName} ${gray(`(${elapsed}s)`)}
`);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        process.stdout.write(`\r  ${red("\u2717")} ${targetName}
`);
        console.error(`    ${red(message)}`);
        failed = true;
      }
    }
    const totalElapsed = ((Date.now() - startTime) / 1e3).toFixed(2);
    console.log();
    if (failed) {
      console.error(`${red("Build failed")} ${gray(`(${totalElapsed}s)`)}`);
      process.exit(1);
    }
    console.log(`${green("Build complete")} ${gray(`(${totalElapsed}s)`)}`);
  });
}

// cli/commands/install.ts
import { existsSync as existsSync13, readFileSync as readFileSync12 } from "fs";
import { join as join14, dirname as dirname4 } from "path";
import { fileURLToPath as fileURLToPath2 } from "url";
import { createInterface } from "readline";

// cli/lib/detect/tools.ts
import { existsSync as existsSync11 } from "fs";
import { join as join12 } from "path";
import { execFileSync } from "child_process";
import { platform } from "os";
var HOME = process.env.HOME || process.env.USERPROFILE || "";
var XDG_CONFIG_HOME = process.env.XDG_CONFIG_HOME || join12(HOME, ".config");
var LOAF_MARKER_FILE = ".loaf-version";
function hasCmd(cmd) {
  try {
    execFileSync("which", [cmd], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}
function isMacOS() {
  return platform() === "darwin";
}
function isLoafInstalled(configDir) {
  if (existsSync11(join12(configDir, LOAF_MARKER_FILE))) {
    return true;
  }
  const skillsDir = join12(configDir, "skills");
  for (const skill of ["foundations", "python-development", "python"]) {
    if (existsSync11(join12(skillsDir, skill))) {
      return true;
    }
  }
  return false;
}
function detectClaudeCode() {
  return hasCmd("claude");
}
function detectTools() {
  const tools = [];
  const opencodeConfig = join12(XDG_CONFIG_HOME, "opencode");
  if (existsSync11(opencodeConfig)) {
    tools.push({
      key: "opencode",
      name: "OpenCode",
      configDir: opencodeConfig,
      installed: isLoafInstalled(opencodeConfig),
      detectedVia: "config"
    });
  }
  const cursorConfig = join12(HOME, ".cursor");
  let cursorDetected = false;
  let cursorVia = "";
  if (hasCmd("cursor")) {
    cursorDetected = true;
    cursorVia = "cli";
  } else if (isMacOS() && (existsSync11("/Applications/Cursor.app") || existsSync11(join12(HOME, "Applications/Cursor.app")))) {
    cursorDetected = true;
    cursorVia = "app";
  } else if (existsSync11(cursorConfig)) {
    cursorDetected = true;
    cursorVia = "config";
  }
  if (cursorDetected) {
    tools.push({
      key: "cursor",
      name: "Cursor",
      configDir: cursorConfig,
      installed: isLoafInstalled(cursorConfig),
      detectedVia: cursorVia
    });
  }
  const codexConfig = (process.env.CODEX_HOME || join12(HOME, ".codex")).replace(
    /\/$/,
    ""
  );
  let codexDetected = false;
  let codexVia = "";
  if (hasCmd("codex")) {
    codexDetected = true;
    codexVia = "cli";
  } else if (existsSync11(codexConfig) || existsSync11(join12(HOME, ".codex"))) {
    codexDetected = true;
    codexVia = "config";
  }
  if (codexDetected) {
    tools.push({
      key: "codex",
      name: "Codex",
      configDir: codexConfig,
      installed: isLoafInstalled(codexConfig),
      detectedVia: codexVia
    });
  }
  const geminiConfig = join12(HOME, ".gemini");
  let geminiDetected = false;
  let geminiVia = "";
  if (hasCmd("gemini")) {
    geminiDetected = true;
    geminiVia = "cli";
  } else if (existsSync11(geminiConfig)) {
    geminiDetected = true;
    geminiVia = "config";
  }
  if (geminiDetected) {
    tools.push({
      key: "gemini",
      name: "Gemini",
      configDir: geminiConfig,
      installed: isLoafInstalled(geminiConfig),
      detectedVia: geminiVia
    });
  }
  return tools;
}
var DEFAULT_CONFIG_DIRS = {
  opencode: join12(XDG_CONFIG_HOME, "opencode"),
  cursor: join12(HOME, ".cursor"),
  codex: process.env.CODEX_HOME || join12(HOME, ".codex"),
  gemini: join12(HOME, ".gemini")
};
function isDevMode(rootDir) {
  return existsSync11(join12(rootDir, ".git")) && existsSync11(join12(rootDir, "package.json")) && existsSync11(join12(rootDir, "content", "skills"));
}

// cli/lib/install/installer.ts
import {
  mkdirSync as mkdirSync8,
  cpSync as cpSync7,
  writeFileSync as writeFileSync8,
  rmSync as rmSync6,
  existsSync as existsSync12,
  readdirSync as readdirSync8
} from "fs";
import { join as join13 } from "path";
import { execFileSync as execFileSync2 } from "child_process";
var LOAF_MARKER_FILE2 = ".loaf-version";
var VERSION2 = "2.0.0";
function hasRsync() {
  try {
    execFileSync2("which", ["rsync"], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}
function syncDir(src, dest) {
  mkdirSync8(dest, { recursive: true });
  if (hasRsync()) {
    execFileSync2("rsync", ["-a", "--delete", `${src}/`, `${dest}/`], {
      stdio: "inherit"
    });
  } else {
    const entries = readdirSync8(dest);
    for (const entry of entries) {
      rmSync6(join13(dest, entry), { recursive: true, force: true });
    }
    cpSync7(src, dest, { recursive: true });
  }
}
function writeMarker(configDir) {
  mkdirSync8(configDir, { recursive: true });
  writeFileSync8(join13(configDir, LOAF_MARKER_FILE2), `${VERSION2}
`);
}
function installOpencode(distDir, configDir) {
  const dirs = ["skills", "agents", "commands", "plugins"];
  for (const dir of dirs) {
    const src = join13(distDir, dir);
    const dest = join13(configDir, dir);
    if (existsSync12(src)) {
      syncDir(src, dest);
    }
  }
  writeMarker(configDir);
}
function installCursor(distDir, configDir) {
  const staleCommands = join13(configDir, "commands");
  if (existsSync12(staleCommands)) {
    rmSync6(staleCommands, { recursive: true });
  }
  const skillsSrc = join13(distDir, "skills");
  if (existsSync12(skillsSrc)) {
    syncDir(skillsSrc, join13(configDir, "skills"));
  }
  const agentsSrc = join13(distDir, "agents");
  if (existsSync12(agentsSrc)) {
    syncDir(agentsSrc, join13(configDir, "agents"));
  }
  const hooksSrc = join13(distDir, "hooks.json");
  if (existsSync12(hooksSrc)) {
    mkdirSync8(configDir, { recursive: true });
    cpSync7(hooksSrc, join13(configDir, "hooks.json"));
  }
  const hooksDir = join13(distDir, "hooks");
  if (existsSync12(hooksDir)) {
    syncDir(hooksDir, join13(configDir, "hooks"));
  }
  writeMarker(configDir);
}
function installCodex(distDir, configDir) {
  const skillsSrc = join13(distDir, "skills");
  if (existsSync12(skillsSrc)) {
    syncDir(skillsSrc, join13(configDir, "skills"));
  }
  writeMarker(configDir);
}
function installGemini(distDir, configDir) {
  const skillsSrc = join13(distDir, "skills");
  if (existsSync12(skillsSrc)) {
    syncDir(skillsSrc, join13(configDir, "skills"));
  }
  writeMarker(configDir);
}
var INSTALLERS = {
  opencode: installOpencode,
  cursor: installCursor,
  codex: installCodex,
  gemini: installGemini
};

// cli/commands/install.ts
var __dirname2 = dirname4(fileURLToPath2(import.meta.url));
var bold2 = (s) => `\x1B[1m${s}\x1B[0m`;
var green2 = (s) => `\x1B[32m${s}\x1B[0m`;
var red2 = (s) => `\x1B[31m${s}\x1B[0m`;
var yellow = (s) => `\x1B[33m${s}\x1B[0m`;
var gray2 = (s) => `\x1B[90m${s}\x1B[0m`;
var white = (s) => `\x1B[97m${s}\x1B[0m`;
function findRootDir2() {
  let dir = __dirname2;
  for (let i = 0; i < 10; i++) {
    const pkgPath = join14(dir, "package.json");
    try {
      const pkg = JSON.parse(readFileSync12(pkgPath, "utf-8"));
      if (pkg.name === "loaf") return dir;
    } catch {
    }
    const parent = dirname4(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error("Could not find loaf root directory");
}
function askYesNo(question) {
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout
  });
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer.trim().toLowerCase().startsWith("y"));
    });
  });
}
var VALID_TARGETS = Object.keys(DEFAULT_CONFIG_DIRS);
function registerInstallCommand(program2) {
  program2.command("install").description("Install Loaf to detected AI tool configurations").option("--to <target>", 'Target to install to (or "all")').option("--upgrade", "Update only already-installed targets").action(async (options) => {
    const rootDir = findRootDir2();
    const distDir = join14(rootDir, "dist");
    const devMode = isDevMode(rootDir);
    console.log(`
${bold2("loaf install")}
`);
    const hasClaudeCode = detectClaudeCode();
    const tools = detectTools();
    if (hasClaudeCode) {
      console.log(`  ${green2("\u2713")} Claude Code detected`);
      if (devMode) {
        console.log(
          `    ${gray2("Test locally:")} ${white(`/plugin marketplace add ${rootDir}`)}`
        );
      } else {
        console.log(
          `    ${gray2("Install via:")} ${white("/plugin marketplace add levifig/loaf")}`
        );
      }
      console.log();
    }
    for (const tool of tools) {
      const status = tool.installed ? ` ${yellow("(installed)")}` : "";
      console.log(`  ${green2("\u2713")} ${tool.name} detected${status}`);
    }
    if (tools.length === 0 && !hasClaudeCode) {
      console.log(`  ${gray2("No AI tools detected")}`);
      console.log();
      return;
    }
    console.log();
    let selectedTargets;
    if (options.to === "all") {
      selectedTargets = tools.map((t) => t.key);
    } else if (options.to) {
      if (!VALID_TARGETS.includes(options.to) && options.to !== "all") {
        console.error(
          `${red2("error:")} Unknown target ${bold2(options.to)}
${gray2("Valid targets:")} ${VALID_TARGETS.join(", ")}, all`
        );
        process.exit(1);
      }
      const tool = tools.find((t) => t.key === options.to);
      selectedTargets = [options.to];
      if (!tool) {
        console.log(
          `  ${yellow("\u26A1")} ${options.to} was not auto-detected; installing to ${DEFAULT_CONFIG_DIRS[options.to]}`
        );
      }
    } else if (options.upgrade) {
      selectedTargets = tools.filter((t) => t.installed).map((t) => t.key);
      if (selectedTargets.length === 0) {
        console.log(`  ${gray2("No installed targets to upgrade")}`);
        console.log();
        return;
      }
      console.log(`  ${gray2("Upgrading:")} ${selectedTargets.join(", ")}`);
    } else {
      selectedTargets = [];
      for (const tool of tools) {
        const status = tool.installed ? ` ${yellow("(installed)")}` : "";
        const yes = await askYesNo(
          `  Install to ${bold2(tool.name)}${status}? [y/N] `
        );
        if (yes) {
          selectedTargets.push(tool.key);
        }
      }
    }
    if (selectedTargets.length === 0) {
      console.log(`  ${gray2("No targets selected")}`);
      console.log();
      return;
    }
    console.log();
    for (const target of selectedTargets) {
      const tool = tools.find((t) => t.key === target);
      const configDir = tool?.configDir || DEFAULT_CONFIG_DIRS[target];
      const targetDistDir = join14(distDir, target);
      if (!existsSync13(targetDistDir)) {
        console.log(
          `  ${red2("\u2717")} ${target} \u2014 no build output found. Run ${bold2("loaf build")} first.`
        );
        continue;
      }
      const installer = INSTALLERS[target];
      if (!installer) {
        console.log(`  ${red2("\u2717")} ${target} \u2014 no installer available`);
        continue;
      }
      try {
        installer(targetDistDir, configDir);
        console.log(`  ${green2("\u2713")} ${target} installed to ${gray2(configDir)}`);
      } catch (error) {
        const msg = error instanceof Error ? error.message : String(error);
        console.log(`  ${red2("\u2717")} ${target} \u2014 ${msg}`);
      }
    }
    console.log();
  });
}

// cli/index.ts
var __dirname3 = dirname5(fileURLToPath3(import.meta.url));
function getVersion2() {
  for (const candidate of [
    join15(__dirname3, "..", "package.json"),
    join15(__dirname3, "..", "..", "package.json")
  ]) {
    try {
      const pkg = JSON.parse(readFileSync13(candidate, "utf-8"));
      if (pkg.name === "loaf") return pkg.version;
    } catch {
      continue;
    }
  }
  return "0.0.0";
}
var program = new Command();
program.name("loaf").description("Loaf \u2014 Levi's Opinionated Agentic Framework").version(getVersion2(), "-v, --version");
registerBuildCommand(program);
registerInstallCommand(program);
if (process.argv.length <= 2) {
  program.outputHelp();
  process.exit(0);
}
program.parse();
//# sourceMappingURL=index.js.map