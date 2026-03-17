#!/usr/bin/env node
var __defProp = Object.defineProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};

// cli/index.ts
import { Command } from "commander";
import { readFileSync as readFileSync19 } from "fs";
import { join as join23, dirname as dirname8 } from "path";
import { fileURLToPath as fileURLToPath4 } from "url";

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
import { existsSync as existsSync13, readFileSync as readFileSync13 } from "fs";
import { join as join14, dirname as dirname5 } from "path";
import { fileURLToPath as fileURLToPath3 } from "url";
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
import { join as join13, dirname as dirname4 } from "path";
import { readFileSync as readFileSync12 } from "fs";
import { execFileSync as execFileSync2 } from "child_process";
import { fileURLToPath as fileURLToPath2 } from "url";
var LOAF_MARKER_FILE2 = ".loaf-version";
function getVersion2() {
  const __dirname4 = dirname4(fileURLToPath2(import.meta.url));
  for (const candidate of [
    join13(__dirname4, "..", "package.json"),
    join13(__dirname4, "..", "..", "package.json"),
    join13(__dirname4, "..", "..", "..", "package.json")
  ]) {
    try {
      const pkg = JSON.parse(readFileSync12(candidate, "utf-8"));
      if (pkg.name === "loaf") return pkg.version;
    } catch {
      continue;
    }
  }
  return "0.0.0";
}
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
  writeFileSync8(join13(configDir, LOAF_MARKER_FILE2), `${getVersion2()}
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
var __dirname2 = dirname5(fileURLToPath3(import.meta.url));
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
      const pkg = JSON.parse(readFileSync13(pkgPath, "utf-8"));
      if (pkg.name === "loaf") return dir;
    } catch {
    }
    const parent = dirname5(dir);
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

// cli/commands/init.ts
import {
  existsSync as existsSync15,
  mkdirSync as mkdirSync9,
  writeFileSync as writeFileSync9,
  symlinkSync,
  lstatSync,
  realpathSync
} from "fs";
import { join as join16, relative, dirname as dirname6 } from "path";
import { createInterface as createInterface2 } from "readline";

// cli/lib/detect/project.ts
import { existsSync as existsSync14, readFileSync as readFileSync14 } from "fs";
import { join as join15 } from "path";
function safeReadFile(path) {
  try {
    return readFileSync14(path, "utf-8");
  } catch {
    return "";
  }
}
function hasLanguage(languages, name) {
  return languages.some((l) => l.name === name);
}
function hasJsFamily(languages) {
  return hasLanguage(languages, "TypeScript") || hasLanguage(languages, "JavaScript");
}
function detectLanguages(cwd) {
  const languages = [];
  if (existsSync14(join15(cwd, "tsconfig.json"))) {
    languages.push({ name: "TypeScript", confidence: "high", indicator: "tsconfig.json" });
  } else if (existsSync14(join15(cwd, "package.json"))) {
    const content = safeReadFile(join15(cwd, "package.json"));
    if (content.includes('"typescript"') || content.includes('"ts-node"')) {
      languages.push({ name: "TypeScript", confidence: "medium", indicator: "package.json (typescript in deps)" });
    }
  }
  if (!hasLanguage(languages, "TypeScript")) {
    if (existsSync14(join15(cwd, "package.json"))) {
      languages.push({ name: "JavaScript", confidence: "high", indicator: "package.json" });
    }
  }
  if (existsSync14(join15(cwd, "pyproject.toml"))) {
    languages.push({ name: "Python", confidence: "high", indicator: "pyproject.toml" });
  } else if (existsSync14(join15(cwd, "setup.py"))) {
    languages.push({ name: "Python", confidence: "high", indicator: "setup.py" });
  } else if (existsSync14(join15(cwd, "requirements.txt"))) {
    languages.push({ name: "Python", confidence: "medium", indicator: "requirements.txt" });
  } else if (existsSync14(join15(cwd, "uv.lock"))) {
    languages.push({ name: "Python", confidence: "medium", indicator: "uv.lock" });
  } else if (existsSync14(join15(cwd, "Pipfile"))) {
    languages.push({ name: "Python", confidence: "medium", indicator: "Pipfile" });
  }
  if (existsSync14(join15(cwd, "Gemfile"))) {
    languages.push({ name: "Ruby", confidence: "high", indicator: "Gemfile" });
  } else if (existsSync14(join15(cwd, ".ruby-version"))) {
    languages.push({ name: "Ruby", confidence: "medium", indicator: ".ruby-version" });
  } else if (existsSync14(join15(cwd, ".ruby-gemset"))) {
    languages.push({ name: "Ruby", confidence: "medium", indicator: ".ruby-gemset" });
  }
  if (existsSync14(join15(cwd, "go.mod"))) {
    languages.push({ name: "Go", confidence: "high", indicator: "go.mod" });
  }
  if (existsSync14(join15(cwd, "Cargo.toml"))) {
    languages.push({ name: "Rust", confidence: "high", indicator: "Cargo.toml" });
  }
  return languages;
}
function detectFrameworks(cwd, languages) {
  const frameworks = [];
  if (hasJsFamily(languages)) {
    const lang = hasLanguage(languages, "TypeScript") ? "TypeScript" : "JavaScript";
    const nextConfigs = ["next.config.js", "next.config.mjs", "next.config.ts"];
    const nextIndicator = nextConfigs.find((f) => existsSync14(join15(cwd, f)));
    if (nextIndicator) {
      frameworks.push({ name: "Next.js", language: lang, indicator: nextIndicator });
    }
    if (!nextIndicator && existsSync14(join15(cwd, "package.json"))) {
      const content = safeReadFile(join15(cwd, "package.json"));
      if (content.includes('"react"')) {
        frameworks.push({ name: "React", language: lang, indicator: "package.json (react in deps)" });
      }
    }
  }
  if (hasLanguage(languages, "Python")) {
    const pyprojectContent = safeReadFile(join15(cwd, "pyproject.toml"));
    const requirementsContent = safeReadFile(join15(cwd, "requirements.txt"));
    const pyDeps = pyprojectContent + "\n" + requirementsContent;
    if (pyDeps.includes("fastapi")) {
      frameworks.push({ name: "FastAPI", language: "Python", indicator: "fastapi in deps" });
    }
    if (existsSync14(join15(cwd, "manage.py")) || pyDeps.includes("django")) {
      const indicator = existsSync14(join15(cwd, "manage.py")) ? "manage.py" : "django in deps";
      frameworks.push({ name: "Django", language: "Python", indicator });
    }
    if (pyDeps.includes("flask")) {
      frameworks.push({ name: "Flask", language: "Python", indicator: "flask in deps" });
    }
  }
  if (hasLanguage(languages, "Ruby")) {
    if (existsSync14(join15(cwd, "config", "routes.rb")) || existsSync14(join15(cwd, "bin", "rails"))) {
      const indicator = existsSync14(join15(cwd, "config", "routes.rb")) ? "config/routes.rb" : "bin/rails";
      frameworks.push({ name: "Rails", language: "Ruby", indicator });
    }
  }
  return frameworks;
}
function detectExistingStructure(cwd) {
  const docFiles = [
    "docs/VISION.md",
    "docs/STRATEGY.md",
    "docs/ARCHITECTURE.md",
    "README.md"
  ];
  const existingDocs = docFiles.filter((f) => existsSync14(join15(cwd, f)));
  return {
    hasAgentsDir: existsSync14(join15(cwd, ".agents")),
    hasAgentsMd: existsSync14(join15(cwd, ".agents", "AGENTS.md")),
    hasDocsDir: existsSync14(join15(cwd, "docs")),
    hasChangelog: existsSync14(join15(cwd, "CHANGELOG.md")),
    hasClaudeDir: existsSync14(join15(cwd, ".claude")),
    hasLoafJson: existsSync14(join15(cwd, ".agents", "loaf.json")),
    existingDocs
  };
}
function detectProject(cwd) {
  const languages = detectLanguages(cwd);
  const frameworks = detectFrameworks(cwd, languages);
  const existing = detectExistingStructure(cwd);
  return { languages, frameworks, existing };
}

// cli/commands/init.ts
var yellow2 = (s) => `\x1B[33m${s}\x1B[0m`;
function withinProject(cwd, fullPath) {
  let check = fullPath;
  while (!existsSync15(check) && check !== cwd) {
    check = dirname6(check);
  }
  try {
    const realCheck = realpathSync(check);
    const realCwd = realpathSync(cwd);
    return realCheck === realCwd || realCheck.startsWith(realCwd + "/");
  } catch {
    return false;
  }
}
var bold3 = (s) => `\x1B[1m${s}\x1B[0m`;
var green3 = (s) => `\x1B[32m${s}\x1B[0m`;
var gray3 = (s) => `\x1B[90m${s}\x1B[0m`;
var cyan2 = (s) => `\x1B[36m${s}\x1B[0m`;
var SKILL_MAP = {
  TypeScript: ["typescript-development"],
  Python: ["python-development"],
  Ruby: ["ruby-development"],
  Go: ["go-development"],
  "Next.js": ["typescript-development", "interface-design"],
  React: ["typescript-development", "interface-design"],
  FastAPI: ["python-development", "database-design"],
  Django: ["python-development", "database-design"],
  Rails: ["ruby-development", "database-design"],
  Flask: ["python-development"]
};
var SCAFFOLD_DIRS = [
  ".agents",
  ".agents/sessions",
  ".agents/ideas",
  ".agents/specs",
  ".agents/tasks",
  "docs",
  "docs/knowledge",
  "docs/decisions"
];
var SCAFFOLD_FILES = [
  [
    ".agents/AGENTS.md",
    () => `# Project Instructions

> Agent instructions for this project. Customize per your needs.

## Quick Start

<!-- Add build/run commands here -->

## Project Structure

<!-- Describe your project layout -->

## Development Practices

<!-- Add coding conventions, testing approach, etc. -->

## Key Decisions

<!-- Link to ADRs in docs/decisions/ -->
`
  ],
  [
    ".agents/loaf.json",
    () => JSON.stringify(
      {
        version: "1.0.0",
        initialized: (/* @__PURE__ */ new Date()).toISOString()
      },
      null,
      2
    ) + "\n"
  ],
  [
    "docs/VISION.md",
    () => `# Vision

## Purpose
<!-- Why does this project exist? What problem does it solve? -->

## Target Users
<!-- Who is this for? -->

## Success Criteria
<!-- How do you know when you've succeeded? -->

## Non-Goals
<!-- What is explicitly out of scope? -->
`
  ],
  [
    "docs/STRATEGY.md",
    () => `# Strategy

## Current Focus
<!-- What are you working on right now and why? -->

## Priorities
<!-- Ordered list of what matters most -->

## Constraints
<!-- Budget, timeline, team size, technical limitations -->

## Open Questions
<!-- Unresolved strategic decisions -->
`
  ],
  [
    "docs/ARCHITECTURE.md",
    () => `# Architecture

## Overview
<!-- High-level system description -->

## Components
<!-- Key components and their responsibilities -->

## Data Flow
<!-- How data moves through the system -->

## Technology Choices
<!-- Key technology decisions and rationale -->

## Deployment
<!-- How the system is deployed -->
`
  ],
  [
    "CHANGELOG.md",
    () => `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]
`
  ]
];
function askYesNo2(question) {
  if (!process.stdin.isTTY) {
    return Promise.resolve(false);
  }
  const rl = createInterface2({
    input: process.stdin,
    output: process.stdout
  });
  return new Promise((resolve) => {
    let resolved = false;
    rl.on("close", () => {
      if (!resolved) {
        resolved = true;
        resolve(false);
      }
    });
    rl.question(question, (answer) => {
      resolved = true;
      rl.close();
      resolve(answer.trim().toLowerCase().startsWith("y"));
    });
  });
}
function printDetected(info) {
  console.log(`  ${bold3("Detected:")}`);
  if (info.languages.length === 0 && info.frameworks.length === 0) {
    console.log(`    ${gray3("No languages or frameworks detected")}`);
  } else {
    for (const lang of info.languages) {
      console.log(`    ${green3("\u2713")} ${lang.name} ${gray3(`(${lang.indicator})`)}`);
    }
    for (const fw of info.frameworks) {
      console.log(`    ${green3("\u2713")} ${fw.name} ${gray3(`(${fw.indicator})`)}`);
    }
  }
  console.log();
  console.log(`  ${bold3("Existing:")}`);
  const checks = [
    [info.existing.hasAgentsDir, ".agents/ directory"],
    [info.existing.hasAgentsMd, ".agents/AGENTS.md"],
    [info.existing.hasDocsDir, "docs/ directory"],
    [info.existing.hasChangelog, "CHANGELOG.md"],
    [info.existing.hasClaudeDir, ".claude/ directory"],
    [info.existing.hasLoafJson, ".agents/loaf.json"]
  ];
  for (const [exists, label] of checks) {
    if (exists) {
      console.log(`    ${green3("\u2713")} ${label}`);
    } else {
      console.log(`    ${gray3("\u2717")} ${label}`);
    }
  }
}
function scaffoldDirs(cwd) {
  const created = [];
  const skipped = [];
  for (const dir of SCAFFOLD_DIRS) {
    const fullPath = join16(cwd, dir);
    if (!existsSync15(fullPath)) {
      if (!withinProject(cwd, fullPath)) {
        skipped.push(dir + "/");
        continue;
      }
      mkdirSync9(fullPath, { recursive: true });
      created.push(dir + "/");
    }
  }
  return { created, skipped };
}
function scaffoldFiles(cwd) {
  const created = [];
  const skipped = [];
  for (const [relPath, contentFn] of SCAFFOLD_FILES) {
    const fullPath = join16(cwd, relPath);
    if (!existsSync15(fullPath)) {
      if (!withinProject(cwd, fullPath)) {
        skipped.push(relPath);
        continue;
      }
      const parentDir = dirname6(fullPath);
      if (!existsSync15(parentDir)) {
        mkdirSync9(parentDir, { recursive: true });
      }
      writeFileSync9(fullPath, contentFn(), "utf-8");
      created.push(relPath);
    }
  }
  return { created, skipped };
}
function getRecommendedSkills(info) {
  const skills = /* @__PURE__ */ new Set(["foundations"]);
  for (const lang of info.languages) {
    const mapped = SKILL_MAP[lang.name];
    if (mapped) {
      for (const s of mapped) skills.add(s);
    }
  }
  for (const fw of info.frameworks) {
    const mapped = SKILL_MAP[fw.name];
    if (mapped) {
      for (const s of mapped) skills.add(s);
    }
  }
  return Array.from(skills);
}
function fileOrSymlinkExists(path) {
  try {
    lstatSync(path);
    return true;
  } catch {
    return false;
  }
}
function registerInitCommand(program2) {
  program2.command("init").description("Initialize a project with Loaf structure").option("--no-symlinks", "Skip symlink creation prompts").action(async (options) => {
    const cwd = process.cwd();
    console.log(`
${bold3("loaf init")}
`);
    process.stdout.write(`  ${cyan2("scanning")} project...

`);
    const info = detectProject(cwd);
    printDetected(info);
    console.log();
    const dirs = scaffoldDirs(cwd);
    const files = scaffoldFiles(cwd);
    const allSkipped = [...dirs.skipped, ...files.skipped];
    if (dirs.created.length > 0 || files.created.length > 0) {
      console.log(`  ${bold3("Creating:")}`);
      for (const dir of dirs.created) {
        console.log(`    ${green3("+")} ${dir}`);
      }
      for (const file of files.created) {
        console.log(`    ${green3("+")} ${file}`);
      }
      console.log();
    } else {
      console.log(`  ${gray3("Nothing to create \u2014 all files exist")}
`);
    }
    if (allSkipped.length > 0) {
      console.log(`  ${yellow2("Skipped")} (symlink points outside project):`);
      for (const s of allSkipped) {
        console.log(`    ${yellow2("!")} ${s}`);
      }
      console.log();
    }
    if (options.symlinks) {
      const agentsMdPath = join16(cwd, ".agents", "AGENTS.md");
      if (existsSync15(agentsMdPath)) {
        console.log(`  ${bold3("Symlinks:")}`);
        const claudeSymlink = join16(cwd, ".claude", "CLAUDE.md");
        if (!fileOrSymlinkExists(claudeSymlink)) {
          const yes = await askYesNo2(
            `    Create .claude/CLAUDE.md \u2192 .agents/AGENTS.md? [y/N] `
          );
          if (yes) {
            const claudeDir = join16(cwd, ".claude");
            if (!existsSync15(claudeDir)) {
              mkdirSync9(claudeDir, { recursive: true });
            }
            const relTarget = relative(claudeDir, agentsMdPath);
            symlinkSync(relTarget, claudeSymlink);
            console.log(`    ${green3("\u2713")} Created .claude/CLAUDE.md`);
          }
        }
        const rootSymlink = join16(cwd, "AGENTS.md");
        if (!fileOrSymlinkExists(rootSymlink)) {
          const yes = await askYesNo2(
            `    Create ./AGENTS.md \u2192 .agents/AGENTS.md? [y/N] `
          );
          if (yes) {
            const relTarget = relative(cwd, agentsMdPath);
            symlinkSync(relTarget, rootSymlink);
            console.log(`    ${green3("\u2713")} Created ./AGENTS.md`);
          }
        }
        console.log();
      }
    }
    const skills = getRecommendedSkills(info);
    if (skills.length > 0) {
      console.log(`  ${bold3("Recommended skills:")}`);
      const stackParts = [];
      for (const lang of info.languages) {
        if (SKILL_MAP[lang.name]) stackParts.push(lang.name);
      }
      for (const fw of info.frameworks) {
        if (SKILL_MAP[fw.name]) stackParts.push(fw.name);
      }
      const nonFoundation = skills.filter((s) => s !== "foundations");
      if (nonFoundation.length > 0) {
        const stackLabel = stackParts.length > 0 ? gray3(`(for ${stackParts.join(" + ")})`) : "";
        console.log(
          `    \u2022 ${nonFoundation.join(", ")}  ${stackLabel}`
        );
      }
      console.log(`    \u2022 foundations  ${gray3("(always)")}`);
      console.log();
    }
    console.log(`  ${green3("\u2713")} Project initialized
`);
    console.log(`  ${bold3("Next steps:")}`);
    console.log(`    1. Edit .agents/AGENTS.md with your project details`);
    console.log(`    2. Run ${cyan2("loaf install")} to set up your AI tools`);
    console.log();
  });
}

// cli/commands/release.ts
import { execFileSync as execFileSync4 } from "child_process";
import {
  existsSync as existsSync17,
  readFileSync as readFileSync16,
  writeFileSync as writeFileSync10,
  readdirSync as readdirSync9,
  mkdtempSync,
  unlinkSync
} from "fs";
import { join as join18 } from "path";
import { tmpdir } from "os";
import { createInterface as createInterface3 } from "readline";

// cli/lib/release/commits.ts
import { execFileSync as execFileSync3 } from "child_process";
var CONVENTIONAL_RE = /^(\w+)(\(.+?\))?(!)?:\s*(.+)$/;
var SECTION_MAP = {
  feat: "Added",
  fix: "Fixed",
  refactor: "Changed",
  perf: "Changed",
  docs: null,
  chore: null,
  ci: null,
  test: null,
  build: null,
  style: null
};
var BREAKING_BODY_RE = /^BREAKING[ -]CHANGE:/m;
function getLastTag(cwd) {
  try {
    const tag = execFileSync3("git", ["describe", "--tags", "--abbrev=0"], {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"]
    }).trim();
    return tag || null;
  } catch {
    return null;
  }
}
function getCommitsSince(cwd, tag) {
  const format = "%h%x00%s%x00%B%x00";
  const args = tag ? ["log", `${tag}..HEAD`, `--format=${format}`] : ["log", `--format=${format}`];
  let output;
  try {
    output = execFileSync3("git", args, {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
      maxBuffer: 10 * 1024 * 1024
    });
  } catch {
    return [];
  }
  if (!output.trim()) {
    return [];
  }
  const chunks = output.split("\0\n").filter((c) => c.trim());
  const commits = [];
  for (const chunk of chunks) {
    const parts = chunk.split("\0");
    if (parts.length < 2) continue;
    const hash = parts[0].trim();
    const subject = parts[1].trim();
    const body = (parts[2] || "").trim();
    if (!hash || !subject) continue;
    commits.push(parseCommit(hash, subject, body));
  }
  return commits;
}
function parseCommit(hash, subject, body) {
  const match = subject.match(CONVENTIONAL_RE);
  if (!match) {
    return {
      hash,
      type: "",
      message: subject,
      breaking: BREAKING_BODY_RE.test(body),
      section: BREAKING_BODY_RE.test(body) ? "Breaking Changes" : "Other",
      raw: subject
    };
  }
  const type = match[1];
  const bangIndicator = !!match[3];
  const message = match[4];
  const breakingFromBody = BREAKING_BODY_RE.test(body);
  const breaking = bangIndicator || breakingFromBody;
  let section;
  if (breaking) {
    section = "Breaking Changes";
  } else if (type in SECTION_MAP) {
    section = SECTION_MAP[type];
  } else {
    section = "Other";
  }
  return {
    hash,
    type,
    message,
    breaking,
    section,
    raw: subject
  };
}
function suggestBump(commits) {
  if (commits.some((c) => c.breaking)) {
    return "major";
  }
  if (commits.some((c) => c.section === "Added")) {
    return "minor";
  }
  return "patch";
}

// cli/lib/release/version.ts
import { existsSync as existsSync16, readFileSync as readFileSync15 } from "fs";
import { join as join17, relative as relative2 } from "path";
function parseSemVer(version) {
  const hyphenIndex = version.indexOf("-");
  const core = hyphenIndex === -1 ? version : version.slice(0, hyphenIndex);
  const prerelease = hyphenIndex === -1 ? void 0 : version.slice(hyphenIndex + 1);
  const parts = core.split(".");
  if (parts.length !== 3) return null;
  const [major, minor, patch] = parts.map(Number);
  if ([major, minor, patch].some((n) => !Number.isInteger(n) || n < 0)) {
    return null;
  }
  if (prerelease !== void 0 && prerelease.length === 0) return null;
  return { major, minor, patch, ...prerelease ? { prerelease } : {} };
}
function formatSemVer(ver) {
  const core = `${ver.major}.${ver.minor}.${ver.patch}`;
  return ver.prerelease ? `${core}-${ver.prerelease}` : core;
}
function bumpVersion(current, bump) {
  const ver = parseSemVer(current);
  if (!ver) return null;
  switch (bump) {
    case "major":
      return formatSemVer({ major: ver.major + 1, minor: 0, patch: 0 });
    case "minor":
      return formatSemVer({
        major: ver.major,
        minor: ver.minor + 1,
        patch: 0
      });
    case "patch":
      return formatSemVer({
        major: ver.major,
        minor: ver.minor,
        patch: ver.patch + 1
      });
    case "prerelease": {
      if (!ver.prerelease) return null;
      const dotIndex = ver.prerelease.lastIndexOf(".");
      if (dotIndex === -1) {
        return formatSemVer({ ...ver, prerelease: `${ver.prerelease}.1` });
      }
      const label = ver.prerelease.slice(0, dotIndex);
      const numStr = ver.prerelease.slice(dotIndex + 1);
      const num = Number(numStr);
      if (!Number.isInteger(num) || num < 0) {
        return formatSemVer({ ...ver, prerelease: `${ver.prerelease}.1` });
      }
      return formatSemVer({ ...ver, prerelease: `${label}.${num + 1}` });
    }
    case "release": {
      if (!ver.prerelease) return null;
      return formatSemVer({
        major: ver.major,
        minor: ver.minor,
        patch: ver.patch
      });
    }
  }
}
function readTomlVersion(content, sectionName) {
  const lines = content.split("\n");
  const sectionPattern = new RegExp(`^\\[${escapeRegex(sectionName)}\\]`);
  let inSection = false;
  for (const line of lines) {
    if (sectionPattern.test(line.trim())) {
      inSection = true;
      continue;
    }
    if (inSection) {
      if (/^\[/.test(line.trim())) break;
      const match = line.match(/^version\s*=\s*"([^"]+)"/);
      if (match) return match[1];
    }
  }
  return null;
}
function replaceTomlVersion(content, sectionName, newVersion) {
  const lines = content.split("\n");
  const sectionPattern = new RegExp(`^\\[${escapeRegex(sectionName)}\\]`);
  let inSection = false;
  let replaced = false;
  const result = lines.map((line) => {
    if (sectionPattern.test(line.trim())) {
      inSection = true;
      return line;
    }
    if (inSection && !replaced) {
      if (/^\[/.test(line.trim())) {
        inSection = false;
        return line;
      }
      if (/^version\s*=\s*"[^"]+"/.test(line)) {
        replaced = true;
        return line.replace(
          /^(version\s*=\s*)"[^"]+"/,
          `$1"${newVersion}"`
        );
      }
    }
    return line;
  });
  return result.join("\n");
}
function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
var CANDIDATES = [
  {
    relativePath: "package.json",
    format: "json",
    source: { type: "json" }
  },
  {
    relativePath: "pyproject.toml",
    format: "toml-regex",
    source: { type: "toml", section: "project" }
  },
  {
    relativePath: "Cargo.toml",
    format: "toml-regex",
    source: { type: "toml", section: "package" }
  },
  {
    relativePath: ".agents/loaf.json",
    format: "json",
    source: { type: "json" }
  }
];
function detectVersionFiles(cwd) {
  const ecosystemFiles = [];
  let loafFile = null;
  for (const candidate of CANDIDATES) {
    const absolutePath = join17(cwd, candidate.relativePath);
    if (!existsSync16(absolutePath)) continue;
    try {
      const content = readFileSync15(absolutePath, "utf-8");
      let version;
      if (candidate.source.type === "json") {
        const parsed = JSON.parse(content);
        version = parsed.version;
      } else {
        const result = readTomlVersion(content, candidate.source.section);
        if (result) version = result;
      }
      if (!version) continue;
      const file = {
        path: absolutePath,
        relativePath: relative2(cwd, absolutePath),
        format: candidate.format,
        currentVersion: version
      };
      if (candidate.relativePath === ".agents/loaf.json") {
        loafFile = file;
      } else {
        ecosystemFiles.push(file);
      }
    } catch {
      continue;
    }
  }
  if (ecosystemFiles.length === 0 && loafFile) {
    return [loafFile];
  }
  return ecosystemFiles;
}
function prepareVersionUpdates(files, newVersion) {
  const updates = [];
  for (const file of files) {
    try {
      const content = readFileSync15(file.path, "utf-8");
      let updated;
      if (file.format === "json") {
        const parsed = JSON.parse(content);
        parsed.version = newVersion;
        updated = JSON.stringify(parsed, null, 2) + "\n";
      } else {
        const section = tomlSectionForPath(file.relativePath);
        if (!section) continue;
        updated = replaceTomlVersion(content, section, newVersion);
      }
      updates.push([file.path, updated]);
    } catch {
      continue;
    }
  }
  return updates;
}
function tomlSectionForPath(relativePath) {
  const normalized = relativePath.replace(/\\/g, "/");
  if (normalized === "pyproject.toml") return "project";
  if (normalized === "Cargo.toml") return "package";
  return null;
}

// cli/lib/release/changelog.ts
var SECTION_ORDER = [
  "Breaking Changes",
  "Added",
  "Changed",
  "Fixed",
  "Other"
];
var UNRELEASED_RE = /^## \[unreleased\]/i;
function groupBySection(commits) {
  const groups = /* @__PURE__ */ new Map();
  for (const commit of commits) {
    if (commit.section === null) continue;
    const existing = groups.get(commit.section);
    if (existing) {
      existing.push(commit);
    } else {
      groups.set(commit.section, [commit]);
    }
  }
  return groups;
}
function capitalize(str) {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
}
function formatEntry(commit) {
  return `- ${capitalize(commit.message)} (${commit.hash})`;
}
function generateChangelogSection(version, date, commits) {
  const groups = groupBySection(commits);
  const lines = [];
  lines.push(`## [${version}] - ${date}`);
  for (const section of SECTION_ORDER) {
    const sectionCommits = groups.get(section);
    if (!sectionCommits || sectionCommits.length === 0) continue;
    lines.push("");
    lines.push(`### ${section}`);
    for (const commit of sectionCommits) {
      lines.push(formatEntry(commit));
    }
  }
  return lines.join("\n");
}
function insertIntoChangelog(existingContent, newSection) {
  const lines = existingContent.split("\n");
  let unreleasedIndex = -1;
  for (let i = 0; i < lines.length; i++) {
    if (UNRELEASED_RE.test(lines[i].trim())) {
      unreleasedIndex = i;
      break;
    }
  }
  if (unreleasedIndex === -1) return null;
  let nextReleaseIndex = -1;
  for (let i = unreleasedIndex + 1; i < lines.length; i++) {
    if (/^## \[/.test(lines[i].trim())) {
      nextReleaseIndex = i;
      break;
    }
  }
  const before = lines.slice(0, unreleasedIndex + 1);
  const after = nextReleaseIndex === -1 ? [] : lines.slice(nextReleaseIndex);
  const result = [
    ...before,
    "",
    newSection,
    "",
    ...after
  ];
  return result.join("\n");
}
function createChangelog(releaseSection) {
  const lines = [
    "# Changelog",
    "",
    "All notable changes to this project will be documented in this file.",
    "",
    "The format is based on [Keep a Changelog](https://keepachangelog.com/).",
    "",
    "## [Unreleased]",
    "",
    releaseSection,
    ""
  ];
  return lines.join("\n");
}

// cli/commands/release.ts
var bold4 = (s) => `\x1B[1m${s}\x1B[0m`;
var green4 = (s) => `\x1B[32m${s}\x1B[0m`;
var red3 = (s) => `\x1B[31m${s}\x1B[0m`;
var yellow3 = (s) => `\x1B[33m${s}\x1B[0m`;
var gray4 = (s) => `\x1B[90m${s}\x1B[0m`;
var cyan3 = (s) => `\x1B[36m${s}\x1B[0m`;
function askYesNo3(question) {
  if (!process.stdin.isTTY) {
    return Promise.resolve(false);
  }
  const rl = createInterface3({
    input: process.stdin,
    output: process.stdout
  });
  return new Promise((resolve) => {
    let resolved = false;
    rl.on("close", () => {
      if (!resolved) {
        resolved = true;
        resolve(false);
      }
    });
    rl.question(question, (answer) => {
      resolved = true;
      rl.close();
      resolve(answer.trim().toLowerCase().startsWith("y"));
    });
  });
}
async function askChoice(question, options, defaultChoice) {
  if (!process.stdin.isTTY) {
    return defaultChoice;
  }
  const rl = createInterface3({
    input: process.stdin,
    output: process.stdout
  });
  return new Promise((resolve) => {
    let resolved = false;
    rl.on("close", () => {
      if (!resolved) {
        resolved = true;
        resolve(defaultChoice);
      }
    });
    rl.question(question, (answer) => {
      resolved = true;
      rl.close();
      const num = parseInt(answer.trim(), 10);
      if (num >= 1 && num <= options.length) {
        resolve(options[num - 1]);
      } else {
        resolve(defaultChoice);
      }
    });
  });
}
function isGitRepo(cwd) {
  try {
    execFileSync4("git", ["rev-parse", "--is-inside-work-tree"], {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"]
    });
    return true;
  } catch {
    return false;
  }
}
function isGhAvailable() {
  try {
    execFileSync4("which", ["gh"], {
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"]
    });
    return true;
  } catch {
    return false;
  }
}
function scanIncompleteTasks(cwd) {
  const tasksDir = join18(cwd, ".agents", "tasks");
  if (!existsSync17(tasksDir)) return [];
  const incomplete = [];
  try {
    const files = readdirSync9(tasksDir).filter((f) => f.endsWith(".md"));
    for (const file of files) {
      try {
        const content = readFileSync16(join18(tasksDir, file), "utf-8");
        const lines = content.split("\n").slice(0, 20);
        for (const line of lines) {
          const match = line.match(/^status:\s*(.+)/);
          if (match) {
            const status = match[1].trim();
            if (status !== "complete" && status !== "archived") {
              incomplete.push({ filename: file, status });
            }
            break;
          }
        }
      } catch {
        continue;
      }
    }
  } catch {
  }
  return incomplete;
}
function getEditor() {
  return process.env.VISUAL || process.env.EDITOR || null;
}
function registerReleaseCommand(program2) {
  program2.command("release").description("Create a new release with changelog, version bump, and tag").option("--dry-run", "Preview release without making changes").action(async (options) => {
    const cwd = process.cwd();
    console.log(`
${bold4("loaf release")}
`);
    if (!isGitRepo(cwd)) {
      console.error(`  ${red3("error:")} Not a git repository`);
      process.exit(1);
    }
    process.stdout.write(`  ${cyan3("Analyzing")}...

`);
    const lastTag = getLastTag(cwd);
    const commits = getCommitsSince(cwd, lastTag);
    console.log(`  Last tag: ${lastTag ? bold4(lastTag) : gray4("(none)")}`);
    console.log(`  Commits since tag: ${bold4(String(commits.length))}`);
    console.log();
    if (commits.length === 0) {
      console.log(`  ${gray4("No unreleased changes found.")}
`);
      process.exit(0);
    }
    for (const commit of commits) {
      if (commit.section === null) {
        console.log(`  ${gray4(`${commit.raw} (${commit.hash})`)}  ${gray4("[filtered]")}`);
      } else {
        console.log(`  ${green4(`${commit.raw} (${commit.hash})`)}`);
      }
    }
    console.log();
    const versionFiles = detectVersionFiles(cwd);
    if (versionFiles.length === 0) {
      console.error(`  ${red3("error:")} No version files found`);
      process.exit(1);
    }
    const incompleteTasks = scanIncompleteTasks(cwd);
    const commitBump = suggestBump(commits);
    const currentVersion = versionFiles[0].currentVersion;
    const parsed = parseSemVer(currentVersion);
    const isPrerelease = parsed !== null && parsed.prerelease !== void 0;
    let bump;
    let newVersion;
    if (isPrerelease) {
      console.log(`  ${bold4("Current pre-release:")} ${currentVersion}`);
      console.log();
      console.log(`  ${bold4("Bump options:")}`);
      console.log(`    ${cyan3("1.")} prerelease \u2192 ${bumpVersion(currentVersion, "prerelease")}`);
      console.log(`    ${cyan3("2.")} release    \u2192 ${bumpVersion(currentVersion, "release")}`);
      console.log(`    ${cyan3("3.")} ${commitBump.padEnd(10)} \u2192 ${bumpVersion(currentVersion, commitBump)} ${gray4("(based on commits)")}`);
      console.log();
      const choice = await askChoice(
        `  Bump type [1/2/3]: `,
        ["prerelease", "release", commitBump],
        "prerelease"
      );
      bump = choice;
      newVersion = bumpVersion(currentVersion, bump);
    } else {
      bump = commitBump;
      newVersion = bumpVersion(currentVersion, bump);
    }
    if (!newVersion) {
      console.error(`  ${red3("error:")} Could not compute new version from "${currentVersion}"`);
      process.exit(1);
    }
    const today = (/* @__PURE__ */ new Date()).toISOString().slice(0, 10);
    let changelogSection = generateChangelogSection(newVersion, today, commits);
    const editor = getEditor();
    if (editor && process.stdin.isTTY) {
      try {
        const tmpDir = mkdtempSync(join18(tmpdir(), "loaf-release-"));
        const tmpFile = join18(tmpDir, "CHANGELOG_SECTION.md");
        writeFileSync10(tmpFile, changelogSection, "utf-8");
        execFileSync4(editor, [tmpFile], { stdio: "inherit" });
        changelogSection = readFileSync16(tmpFile, "utf-8");
        unlinkSync(tmpFile);
        console.log(`  ${green4("Edited changelog accepted.")}`);
        console.log();
      } catch {
        console.log(`  ${yellow3("Editor failed \u2014 using generated changelog.")}`);
        console.log();
      }
    } else {
      console.log(`  ${bold4("Generated changelog:")}
`);
      for (const line of changelogSection.split("\n")) {
        console.log(`  ${line}`);
      }
      console.log();
      console.log(`  ${gray4("(Set $EDITOR to edit before confirming)")}`);
      console.log();
    }
    const tagName = `v${newVersion}`;
    const ghAvailable = isGhAvailable();
    console.log(`  ${bold4("Version files:")}`);
    for (const file of versionFiles) {
      console.log(`    \u2022 ${file.relativePath} (${file.currentVersion} \u2192 ${newVersion})`);
    }
    console.log();
    if (incompleteTasks.length > 0) {
      console.log(`  ${bold4("Incomplete tasks:")} ${incompleteTasks.length}`);
      for (const task of incompleteTasks) {
        console.log(`    ${yellow3("\u26A0")} ${task.filename} (status: ${task.status})`);
      }
      console.log();
    }
    const bumpReasons = {
      major: "breaking changes detected",
      minor: "new features detected",
      patch: "bug fixes only",
      prerelease: "development milestone",
      release: "stable release"
    };
    console.log(`  Suggested bump: ${bold4(bump)} (${bumpReasons[bump]})`);
    console.log(`  New version: ${bold4(newVersion)}`);
    console.log();
    console.log(`  ${bold4("Actions:")}`);
    let actionNum = 1;
    console.log(`    ${actionNum++}. Update version in ${versionFiles.length} file(s)`);
    console.log(`    ${actionNum++}. Update CHANGELOG.md`);
    console.log(`    ${actionNum++}. Run loaf build`);
    console.log(`    ${actionNum++}. Commit release artifacts`);
    console.log(`    ${actionNum++}. Create git tag ${tagName}`);
    if (ghAvailable) {
      console.log(`    ${actionNum++}. Create GitHub release draft (gh available)`);
    } else {
      console.log(`    ${gray4(`${actionNum++}. Create GitHub release draft (gh not available \u2014 skipped)`)}`);
    }
    console.log();
    if (options.dryRun) {
      console.log(`  ${cyan3("--dry-run:")} No changes made.
`);
      process.exit(0);
    }
    const confirmed = await askYesNo3(`  Proceed with release ${bold4(tagName)}? [y/N] `);
    if (!confirmed) {
      console.log(`
  ${gray4("Release cancelled.")}
`);
      process.exit(0);
    }
    console.log();
    console.log(`  ${bold4("Executing:")}`);
    try {
      const updates = prepareVersionUpdates(versionFiles, newVersion);
      for (const [filePath, content] of updates) {
        writeFileSync10(filePath, content, "utf-8");
        const relPath = versionFiles.find((f) => f.path === filePath)?.relativePath ?? filePath;
        const oldVer = versionFiles.find((f) => f.path === filePath)?.currentVersion ?? "?";
        console.log(`    ${green4("\u2713")} Updated ${relPath} (${oldVer} \u2192 ${newVersion})`);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`    ${red3("\u2717")} Failed to update version files: ${message}`);
      process.exit(1);
    }
    try {
      const changelogPath = join18(cwd, "CHANGELOG.md");
      let changelogContent;
      if (existsSync17(changelogPath)) {
        const existing = readFileSync16(changelogPath, "utf-8");
        const inserted = insertIntoChangelog(existing, changelogSection);
        if (inserted) {
          changelogContent = inserted;
        } else {
          changelogContent = existing + "\n" + changelogSection + "\n";
        }
      } else {
        changelogContent = createChangelog(changelogSection);
      }
      writeFileSync10(changelogPath, changelogContent, "utf-8");
      console.log(`    ${green4("\u2713")} Updated CHANGELOG.md`);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`    ${red3("\u2717")} Failed to update CHANGELOG.md: ${message}`);
      process.exit(1);
    }
    try {
      execFileSync4(process.execPath, [process.argv[1], "build"], {
        cwd,
        stdio: "inherit"
      });
      console.log(`    ${green4("\u2713")} Built all targets`);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`    ${red3("\u2717")} Build failed: ${message}`);
      process.exit(1);
    }
    try {
      execFileSync4("git", ["add", "-A"], {
        cwd,
        stdio: ["ignore", "pipe", "ignore"]
      });
      execFileSync4(
        "git",
        ["commit", "-m", `release: ${tagName}`],
        { cwd, stdio: ["ignore", "pipe", "ignore"] }
      );
      console.log(`    ${green4("\u2713")} Committed release artifacts`);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`    ${red3("\u2717")} Failed to commit release: ${message}`);
      process.exit(1);
    }
    try {
      execFileSync4("git", ["tag", "-a", tagName, "-m", `Release ${newVersion}`], {
        cwd,
        stdio: ["ignore", "pipe", "ignore"]
      });
      console.log(`    ${green4("\u2713")} Created tag ${tagName}`);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`    ${red3("\u2717")} Failed to create tag: ${message}`);
      process.exit(1);
    }
    if (ghAvailable) {
      try {
        execFileSync4(
          "gh",
          [
            "release",
            "create",
            tagName,
            "--draft",
            "--title",
            `v${newVersion}`,
            "--notes",
            changelogSection
          ],
          { cwd, stdio: "inherit" }
        );
        console.log(`    ${green4("\u2713")} Created GitHub release draft`);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`    ${red3("\u2717")} Failed to create GitHub release: ${message}`);
        process.exit(1);
      }
    } else {
      console.log(`    ${gray4("-")} GitHub release skipped (gh not available)`);
    }
    console.log();
    console.log(`  ${green4("\u2713")} Release ${bold4(tagName)} complete
`);
  });
}

// cli/commands/task.ts
import { existsSync as existsSync20, mkdirSync as mkdirSync10, readFileSync as readFileSync18, writeFileSync as writeFileSync12 } from "fs";
import { join as join21 } from "path";
import matter9 from "gray-matter";

// cli/lib/tasks/resolve.ts
import { existsSync as existsSync19 } from "fs";
import { join as join20, dirname as dirname7 } from "path";

// cli/lib/tasks/migrate.ts
import { existsSync as existsSync18, readFileSync as readFileSync17, writeFileSync as writeFileSync11, readdirSync as readdirSync10 } from "fs";
import { join as join19, basename as basename5, relative as relative3 } from "path";
import matter8 from "gray-matter";

// cli/lib/tasks/parser.ts
import { basename as basename4 } from "path";
import matter7 from "gray-matter";

// cli/lib/tasks/types.ts
var TASK_STATUSES = [
  "todo",
  "in_progress",
  "blocked",
  "review",
  "done"
];
var SPEC_STATUSES = [
  "drafting",
  "approved",
  "implementing",
  "complete"
];
var TASK_PRIORITIES = ["P0", "P1", "P2", "P3"];

// cli/lib/tasks/parser.ts
var yellow4 = (s) => `\x1B[33m${s}\x1B[0m`;
var TASK_STATUS_ALIASES = {
  "complete": "done",
  "completed": "done",
  "archived": "done",
  "in-progress": "in_progress",
  "in progress": "in_progress",
  "wip": "in_progress",
  "pending": "todo",
  "waiting": "blocked"
};
var SPEC_STATUS_ALIASES = {
  "draft": "drafting",
  "done": "complete",
  "completed": "complete",
  "archived": "complete",
  "implemented": "complete",
  "in-progress": "implementing",
  "in_progress": "implementing"
};
function normalizeTaskStatus(raw) {
  if (!raw) return "todo";
  const lower = raw.trim().toLowerCase();
  if (TASK_STATUSES.includes(lower)) {
    return lower;
  }
  return TASK_STATUS_ALIASES[lower] ?? "todo";
}
function normalizeTaskPriority(raw) {
  if (!raw) return "P2";
  const upper = raw.trim().toUpperCase();
  if (TASK_PRIORITIES.includes(upper)) {
    return upper;
  }
  return "P2";
}
function normalizeSpecStatus(raw) {
  if (!raw) return "drafting";
  const lower = raw.trim().toLowerCase();
  if (SPEC_STATUSES.includes(lower)) {
    return lower;
  }
  return SPEC_STATUS_ALIASES[lower] ?? "drafting";
}
function normalizeDate(value) {
  if (!value) return (/* @__PURE__ */ new Date()).toISOString();
  if (value instanceof Date) {
    return value.toISOString();
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed.includes("T")) {
      return trimmed;
    }
    if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
      return `${trimmed}T00:00:00Z`;
    }
    const parsed = new Date(trimmed);
    if (!isNaN(parsed.getTime())) {
      return parsed.toISOString();
    }
  }
  return (/* @__PURE__ */ new Date()).toISOString();
}
function parseTaskFilename(filePath) {
  const name = basename4(filePath, ".md");
  const match = name.match(/^(TASK-\d+)(?:-(.+))?$/);
  if (!match) {
    return { id: null, slug: name };
  }
  return {
    id: match[1],
    slug: match[2] ?? ""
  };
}
function parseSpecFilename(filePath) {
  const name = basename4(filePath, ".md");
  const match = name.match(/^(SPEC-\d+)/);
  return { id: match ? match[1] : null };
}
function parseTaskFile(filePath, content) {
  try {
    const { data } = matter7(content);
    const fm = data;
    const { id: filenameId, slug } = parseTaskFilename(filePath);
    const id = fm.id || filenameId;
    if (!id) {
      console.error(`  ${yellow4("warn:")} Could not determine task ID for ${basename4(filePath)}`);
      return null;
    }
    const now = (/* @__PURE__ */ new Date()).toISOString();
    const status = normalizeTaskStatus(fm.status);
    const entry = {
      title: fm.title || basename4(filePath, ".md"),
      slug,
      spec: fm.spec || null,
      status,
      priority: normalizeTaskPriority(fm.priority),
      depends_on: Array.isArray(fm.depends_on) ? fm.depends_on : [],
      files: Array.isArray(fm.files) ? fm.files : [],
      verify: fm.verify || null,
      done: fm.done || null,
      session: fm.session || null,
      created: normalizeDate(fm.created),
      updated: normalizeDate(fm.updated ?? fm.created),
      completed_at: status === "done" ? fm.completed_at ? normalizeDate(fm.completed_at) : normalizeDate(fm.updated ?? fm.created ?? now) : null,
      file: basename4(filePath)
    };
    return { id, entry };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`  ${yellow4("warn:")} Failed to parse ${basename4(filePath)}: ${message}`);
    return null;
  }
}
function parseSpecFile(filePath, content) {
  try {
    const { data } = matter7(content);
    const fm = data;
    const { id: filenameId } = parseSpecFilename(filePath);
    const id = fm.id || filenameId;
    if (!id) {
      console.error(`  ${yellow4("warn:")} Could not determine spec ID for ${basename4(filePath)}`);
      return null;
    }
    const entry = {
      title: fm.title || basename4(filePath, ".md"),
      status: normalizeSpecStatus(fm.status),
      appetite: fm.appetite || null,
      requirement: fm.requirement || null,
      source: fm.source || null,
      created: normalizeDate(fm.created),
      file: basename4(filePath)
    };
    return { id, entry };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`  ${yellow4("warn:")} Failed to parse ${basename4(filePath)}: ${message}`);
    return null;
  }
}

// cli/lib/tasks/migrate.ts
var yellow5 = (s) => `\x1B[33m${s}\x1B[0m`;
var gray5 = (s) => `\x1B[90m${s}\x1B[0m`;
function collectFiles(dir, prefix) {
  if (!existsSync18(dir)) return [];
  try {
    return readdirSync10(dir).filter((f) => f.startsWith(prefix) && f.endsWith(".md")).map((f) => join19(dir, f));
  } catch {
    return [];
  }
}
function collectFilesDeep(dir, prefix) {
  if (!existsSync18(dir)) return [];
  const results = [];
  try {
    const entries = readdirSync10(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isFile() && entry.name.startsWith(prefix) && entry.name.endsWith(".md")) {
        results.push(join19(dir, entry.name));
      } else if (entry.isDirectory()) {
        try {
          const subEntries = readdirSync10(join19(dir, entry.name));
          for (const sub of subEntries) {
            if (sub.startsWith(prefix) && sub.endsWith(".md")) {
              results.push(join19(dir, entry.name, sub));
            }
          }
        } catch {
        }
      }
    }
  } catch {
  }
  return results;
}
function extractNumber(id) {
  const match = id.match(/\d+$/);
  return match ? parseInt(match[0], 10) : 0;
}
function buildIndexFromFiles(agentsDir) {
  const tasksDir = join19(agentsDir, "tasks");
  const specsDir = join19(agentsDir, "specs");
  const tasksArchiveDir = join19(tasksDir, "archive");
  const specsArchiveDir = join19(specsDir, "archive");
  const tasks = {};
  const specs = {};
  let maxTaskNum = 0;
  const activeTaskFiles = collectFiles(tasksDir, "TASK-");
  const archivedTaskFiles = collectFilesDeep(tasksArchiveDir, "TASK-");
  for (const filePath of activeTaskFiles) {
    const content = readFileSync17(filePath, "utf-8");
    const result = parseTaskFile(filePath, content);
    if (result) {
      result.entry.file = basename5(filePath);
      tasks[result.id] = result.entry;
      maxTaskNum = Math.max(maxTaskNum, extractNumber(result.id));
    }
  }
  for (const filePath of archivedTaskFiles) {
    const content = readFileSync17(filePath, "utf-8");
    const result = parseTaskFile(filePath, content);
    if (result) {
      result.entry.file = relative3(tasksDir, filePath);
      tasks[result.id] = result.entry;
      maxTaskNum = Math.max(maxTaskNum, extractNumber(result.id));
    }
  }
  const activeSpecFiles = collectFiles(specsDir, "SPEC-");
  const archivedSpecFiles = collectFilesDeep(specsArchiveDir, "SPEC-");
  for (const filePath of activeSpecFiles) {
    const content = readFileSync17(filePath, "utf-8");
    const result = parseSpecFile(filePath, content);
    if (result) {
      result.entry.file = basename5(filePath);
      specs[result.id] = result.entry;
    }
  }
  for (const filePath of archivedSpecFiles) {
    const content = readFileSync17(filePath, "utf-8");
    const result = parseSpecFile(filePath, content);
    if (result) {
      result.entry.file = relative3(specsDir, filePath);
      specs[result.id] = result.entry;
    }
  }
  return {
    version: 1,
    next_id: maxTaskNum + 1,
    tasks,
    specs
  };
}
function loadIndex(indexPath) {
  if (!existsSync18(indexPath)) return null;
  try {
    const content = readFileSync17(indexPath, "utf-8");
    const parsed = JSON.parse(content);
    if (typeof parsed.version !== "number" || typeof parsed.next_id !== "number" || typeof parsed.tasks !== "object" || typeof parsed.specs !== "object") {
      console.error(`  ${yellow5("warn:")} TASKS.json has invalid shape, ignoring`);
      return null;
    }
    return parsed;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`  ${yellow5("warn:")} Failed to read TASKS.json: ${message}`);
    return null;
  }
}
function saveIndex(indexPath, index) {
  const content = JSON.stringify(index, null, 2) + "\n";
  writeFileSync11(indexPath, content, "utf-8");
}
function taskEntryToFrontmatter(id, entry) {
  const fm = {
    id,
    title: entry.title
  };
  if (entry.spec) fm.spec = entry.spec;
  fm.status = entry.status;
  fm.priority = entry.priority;
  if (entry.created) fm.created = entry.created;
  if (entry.updated) fm.updated = entry.updated;
  if (entry.depends_on.length > 0) fm.depends_on = entry.depends_on;
  if (entry.files.length > 0) fm.files = entry.files;
  if (entry.verify) fm.verify = entry.verify;
  if (entry.done) fm.done = entry.done;
  if (entry.session) fm.session = entry.session;
  if (entry.completed_at) fm.completed_at = entry.completed_at;
  return fm;
}
function specEntryToFrontmatter(id, entry) {
  const fm = {
    id,
    title: entry.title
  };
  if (entry.source) fm.source = entry.source;
  if (entry.created) fm.created = entry.created;
  fm.status = entry.status;
  if (entry.appetite) fm.appetite = entry.appetite;
  if (entry.requirement) fm.requirement = entry.requirement;
  return fm;
}
function frontmatterEquals(a, b) {
  return JSON.stringify(a) === JSON.stringify(b);
}
function resolveFilePath(agentsDir, subdir, relFile) {
  return join19(agentsDir, subdir, relFile);
}
function syncFrontmatterFromIndex(agentsDir, index) {
  for (const [id, entry] of Object.entries(index.tasks)) {
    const filePath = resolveFilePath(agentsDir, "tasks", entry.file);
    if (!existsSync18(filePath)) {
      console.error(`  ${gray5("skip:")} ${entry.file} not found on disk`);
      continue;
    }
    try {
      const raw = readFileSync17(filePath, "utf-8");
      const { data: existingFm, content: body } = matter8(raw);
      const newFm = taskEntryToFrontmatter(id, entry);
      if (frontmatterEquals(existingFm, newFm)) continue;
      const updated = matter8.stringify(body, newFm);
      writeFileSync11(filePath, updated, "utf-8");
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`  ${yellow5("warn:")} Failed to sync ${entry.file}: ${message}`);
    }
  }
  for (const [id, entry] of Object.entries(index.specs)) {
    const filePath = resolveFilePath(agentsDir, "specs", entry.file);
    if (!existsSync18(filePath)) {
      console.error(`  ${gray5("skip:")} ${entry.file} not found on disk`);
      continue;
    }
    try {
      const raw = readFileSync17(filePath, "utf-8");
      const { data: existingFm, content: body } = matter8(raw);
      const newFm = specEntryToFrontmatter(id, entry);
      if (frontmatterEquals(existingFm, newFm)) continue;
      const updated = matter8.stringify(body, newFm);
      writeFileSync11(filePath, updated, "utf-8");
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`  ${yellow5("warn:")} Failed to sync ${entry.file}: ${message}`);
    }
  }
}
function findOrphans(agentsDir, index) {
  const tasksDir = join19(agentsDir, "tasks");
  const specsDir = join19(agentsDir, "specs");
  const tasksArchiveDir = join19(tasksDir, "archive");
  const specsArchiveDir = join19(specsDir, "archive");
  const orphanTasks = [];
  const orphanSpecs = [];
  const knownTaskIds = new Set(Object.keys(index.tasks));
  const knownSpecIds = new Set(Object.keys(index.specs));
  const checkTaskFiles = (dir, baseDir, deep) => {
    const files = deep ? collectFilesDeep(dir, "TASK-") : collectFiles(dir, "TASK-");
    for (const filePath of files) {
      const content = readFileSync17(filePath, "utf-8");
      const result = parseTaskFile(filePath, content);
      if (result && !knownTaskIds.has(result.id)) {
        result.entry.file = relative3(baseDir, filePath);
        orphanTasks.push(result);
      }
    }
  };
  checkTaskFiles(tasksDir, tasksDir, false);
  checkTaskFiles(tasksArchiveDir, tasksDir, true);
  const checkSpecFiles = (dir, baseDir, deep) => {
    const files = deep ? collectFilesDeep(dir, "SPEC-") : collectFiles(dir, "SPEC-");
    for (const filePath of files) {
      const content = readFileSync17(filePath, "utf-8");
      const result = parseSpecFile(filePath, content);
      if (result && !knownSpecIds.has(result.id)) {
        result.entry.file = relative3(baseDir, filePath);
        orphanSpecs.push(result);
      }
    }
  };
  checkSpecFiles(specsDir, specsDir, false);
  checkSpecFiles(specsArchiveDir, specsDir, true);
  return { tasks: orphanTasks, specs: orphanSpecs };
}

// cli/lib/tasks/resolve.ts
function findAgentsDir(startDir = process.cwd()) {
  let current = startDir;
  while (true) {
    const candidate = join20(current, ".agents");
    if (existsSync19(candidate)) {
      return candidate;
    }
    const parent = dirname7(current);
    if (parent === current) {
      return null;
    }
    current = parent;
  }
}
function getOrBuildIndex(agentsDir) {
  const indexPath = join20(agentsDir, "TASKS.json");
  if (existsSync19(indexPath)) {
    const index2 = loadIndex(indexPath);
    if (index2) return index2;
  }
  const index = buildIndexFromFiles(agentsDir);
  saveIndex(indexPath, index);
  return index;
}

// cli/commands/task.ts
var bold5 = (s) => `\x1B[1m${s}\x1B[0m`;
var green5 = (s) => `\x1B[32m${s}\x1B[0m`;
var red4 = (s) => `\x1B[31m${s}\x1B[0m`;
var yellow6 = (s) => `\x1B[33m${s}\x1B[0m`;
var gray6 = (s) => `\x1B[90m${s}\x1B[0m`;
var cyan4 = (s) => `\x1B[36m${s}\x1B[0m`;
var STATUS_DISPLAY_ORDER = [
  "in_progress",
  "blocked",
  "todo",
  "review",
  "done"
];
var STATUS_LABELS = {
  in_progress: "In Progress",
  blocked: "Blocked",
  todo: "Todo",
  review: "Review",
  done: "Done"
};
var STATUS_COLORS = {
  in_progress: yellow6,
  blocked: red4,
  todo: cyan4,
  review: gray6,
  done: green5
};
var PRIORITY_COLORS = {
  P0: red4,
  P1: yellow6,
  P2: cyan4,
  P3: gray6
};
function sortTasks(tasks) {
  return tasks.sort((a, b) => {
    const pA = TASK_PRIORITIES.indexOf(a[1].priority);
    const pB = TASK_PRIORITIES.indexOf(b[1].priority);
    if (pA !== pB) return pA - pB;
    const dateA = a[1].updated || a[1].created || "";
    const dateB = b[1].updated || b[1].created || "";
    return dateB.localeCompare(dateA);
  });
}
function countSpecs(index) {
  const byStatus = {
    drafting: 0,
    approved: 0,
    implementing: 0,
    complete: 0
  };
  for (const spec of Object.values(index.specs)) {
    byStatus[spec.status]++;
  }
  return {
    total: Object.keys(index.specs).length,
    byStatus
  };
}
function generateSlug(title) {
  return title.toLowerCase().replace(/[`'"]/g, "").replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 50);
}
function registerTaskCommand(program2) {
  const task = program2.command("task").description("Manage project tasks");
  task.command("list").description("Show task board grouped by status").option("--json", "Output raw JSON").action(async (options) => {
    const agentsDir = findAgentsDir();
    if (!agentsDir) {
      console.error(`  ${red4("error:")} Could not find .agents/ directory`);
      process.exit(1);
    }
    const index = getOrBuildIndex(agentsDir);
    if (options.json) {
      const indexPath = join21(agentsDir, "TASKS.json");
      if (existsSync20(indexPath)) {
        process.stdout.write(readFileSync18(indexPath, "utf-8"));
      } else {
        process.stdout.write(JSON.stringify(index, null, 2) + "\n");
      }
      return;
    }
    const taskEntries = Object.entries(index.tasks);
    if (taskEntries.length === 0) {
      console.log(`
  ${gray6("No tasks found.")}
`);
      return;
    }
    console.log(`
  ${bold5("loaf task list")}
`);
    const grouped = {
      in_progress: [],
      blocked: [],
      todo: [],
      review: [],
      done: []
    };
    for (const [id, entry] of taskEntries) {
      const status = entry.status;
      if (grouped[status]) {
        grouped[status].push([id, entry]);
      } else {
        grouped.todo.push([id, entry]);
      }
    }
    const specIds = /* @__PURE__ */ new Set();
    for (const status of STATUS_DISPLAY_ORDER) {
      const tasks = sortTasks(grouped[status]);
      const colorFn = STATUS_COLORS[status];
      const label = STATUS_LABELS[status];
      console.log(`  ${bold5(colorFn(`${label} (${tasks.length})`))}`);
      if (tasks.length === 0) {
      } else {
        for (const [id, entry] of tasks) {
          const priorityColor = PRIORITY_COLORS[entry.priority] || gray6;
          const specRef = entry.spec ? gray6(entry.spec) : "";
          if (entry.spec) specIds.add(entry.spec);
          const idCol = bold5(id.padEnd(10));
          const prioCol = priorityColor(entry.priority.padEnd(4));
          const titleCol = entry.title;
          console.log(`    ${idCol}${prioCol}${titleCol}  ${specRef}`);
        }
      }
      console.log();
    }
    const totalTasks = taskEntries.length;
    const totalSpecs = specIds.size;
    console.log(`  Total: ${bold5(String(totalTasks))} tasks across ${bold5(String(totalSpecs))} specs
`);
  });
  task.command("show").description("Display a single task's details").argument("<id>", "Task ID (e.g., TASK-019)").option("--json", "Output task entry as JSON").action(async (id, options) => {
    const agentsDir = findAgentsDir();
    if (!agentsDir) {
      console.error(`  ${red4("error:")} Could not find .agents/ directory`);
      process.exit(1);
    }
    const index = getOrBuildIndex(agentsDir);
    const entry = index.tasks[id];
    if (!entry) {
      console.error(`  ${red4("error:")} ${id} not found in index`);
      process.exit(1);
    }
    if (options.json) {
      process.stdout.write(JSON.stringify({ id, ...entry }, null, 2) + "\n");
      return;
    }
    console.log(`
  ${bold5("loaf task show")} ${id}
`);
    const priorityColor = PRIORITY_COLORS[entry.priority] || gray6;
    const statusColor = STATUS_COLORS[entry.status] || gray6;
    console.log(`  ${bold5(`${id}: ${entry.title}`)}`);
    console.log();
    const metaParts = [
      `Status: ${statusColor(entry.status)}`,
      `Priority: ${priorityColor(entry.priority)}`
    ];
    if (entry.spec) metaParts.push(`Spec: ${entry.spec}`);
    console.log(`  ${metaParts.join(gray6(" \xB7 "))}`);
    const dateParts = [];
    if (entry.created) dateParts.push(`Created: ${entry.created.slice(0, 10)}`);
    if (entry.updated) dateParts.push(`Updated: ${entry.updated.slice(0, 10)}`);
    if (dateParts.length > 0) {
      console.log(`  ${dateParts.join(gray6(" \xB7 "))}`);
    }
    if (entry.depends_on.length > 0) {
      console.log(`  Depends on: ${entry.depends_on.join(", ")}`);
    }
    console.log(`  File: .agents/tasks/${entry.file}`);
    const filePath = join21(agentsDir, "tasks", entry.file);
    if (!existsSync20(filePath)) {
      console.log();
      console.log(`  ${gray6("(no detail file)")}`);
      console.log();
      return;
    }
    try {
      const raw = readFileSync18(filePath, "utf-8");
      const { content: body } = matter9(raw);
      const trimmedBody = body.replace(/^\n+/, "").replace(/\n+$/, "");
      if (trimmedBody.length > 0) {
        console.log();
        console.log(`  ${"\u2500".repeat(60)}`);
        console.log();
        const lines = trimmedBody.split("\n");
        for (const line of lines) {
          console.log(`  ${line}`);
        }
      }
      console.log();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.error(`  ${yellow6("warn:")} Failed to read ${entry.file}: ${message}`);
      console.log();
    }
  });
  task.command("status").description("Show task summary counts").action(async () => {
    const agentsDir = findAgentsDir();
    if (!agentsDir) {
      console.error(`  ${red4("error:")} Could not find .agents/ directory`);
      process.exit(1);
    }
    const index = getOrBuildIndex(agentsDir);
    console.log(`
  ${bold5("loaf task status")}
`);
    const taskCounts = {
      in_progress: 0,
      blocked: 0,
      todo: 0,
      review: 0,
      done: 0
    };
    for (const entry of Object.values(index.tasks)) {
      if (taskCounts[entry.status] !== void 0) {
        taskCounts[entry.status]++;
      }
    }
    const totalTasks = Object.keys(index.tasks).length;
    const taskParts = STATUS_DISPLAY_ORDER.map((status) => {
      const count = taskCounts[status];
      const colorFn = STATUS_COLORS[status];
      return `${colorFn(String(count))} ${status}`;
    });
    console.log(`  Tasks:  ${taskParts.join(gray6(" \xB7 "))}  ${gray6(`(${totalTasks} total)`)}`);
    const specInfo = countSpecs(index);
    const specStatusOrder = ["drafting", "approved", "implementing", "complete"];
    const specStatusColors = {
      drafting: yellow6,
      approved: cyan4,
      implementing: yellow6,
      complete: green5
    };
    const specParts = specStatusOrder.map((status) => {
      const count = specInfo.byStatus[status];
      const colorFn = specStatusColors[status];
      return `${colorFn(String(count))} ${status}`;
    });
    console.log(`  Specs:  ${specParts.join(gray6(" \xB7 "))}  ${gray6(`(${specInfo.total} total)`)}`);
    console.log();
  });
  task.command("create").description("Create a new task").requiredOption("--title <title>", "Task title").option("--spec <id>", "Associated spec ID (e.g., SPEC-010)").option("--priority <level>", "Priority level (P0/P1/P2/P3)", "P2").option("--depends-on <ids>", "Comma-separated task IDs").action(async (options) => {
    const agentsDir = findAgentsDir();
    if (!agentsDir) {
      console.error(`  ${red4("error:")} Could not find .agents/ directory`);
      process.exit(1);
    }
    const index = getOrBuildIndex(agentsDir);
    const indexPath = join21(agentsDir, "TASKS.json");
    const priority = options.priority;
    if (!TASK_PRIORITIES.includes(priority)) {
      console.error(`  ${red4("error:")} Invalid priority "${options.priority}". Must be one of: ${TASK_PRIORITIES.join(", ")}`);
      process.exit(1);
    }
    const spec = options.spec || null;
    if (spec && !index.specs[spec]) {
      console.error(`  ${red4("error:")} Spec "${spec}" not found in index`);
      process.exit(1);
    }
    const dependsOn = [];
    if (options.dependsOn) {
      for (const depId of options.dependsOn.split(",").map((s) => s.trim())) {
        if (!index.tasks[depId]) {
          console.error(`  ${red4("error:")} Dependency "${depId}" not found in index`);
          process.exit(1);
        }
        dependsOn.push(depId);
      }
    }
    const nextId = index.next_id;
    const taskId = `TASK-${String(nextId).padStart(3, "0")}`;
    const slug = generateSlug(options.title);
    const now = (/* @__PURE__ */ new Date()).toISOString();
    const fileName = `${taskId}-${slug}.md`;
    const entry = {
      title: options.title,
      slug,
      spec,
      status: "todo",
      priority,
      depends_on: dependsOn,
      files: [],
      verify: null,
      done: null,
      session: null,
      created: now,
      updated: now,
      completed_at: null,
      file: fileName
    };
    index.tasks[taskId] = entry;
    index.next_id = nextId + 1;
    saveIndex(indexPath, index);
    const tasksDir = join21(agentsDir, "tasks");
    if (!existsSync20(tasksDir)) {
      mkdirSync10(tasksDir, { recursive: true });
    }
    const frontmatterData = {
      id: taskId,
      title: options.title,
      status: "todo",
      priority,
      created: now,
      updated: now
    };
    if (spec) frontmatterData.spec = spec;
    if (dependsOn.length > 0) frontmatterData.depends_on = dependsOn;
    const body = `
# ${taskId}: ${options.title}

## Description

<!-- Describe the task here -->

## Acceptance Criteria

- [ ]

## Verification

\`\`\`bash
# Add verification command
\`\`\`
`;
    const mdContent = matter9.stringify(body, frontmatterData);
    writeFileSync12(join21(tasksDir, fileName), mdContent, "utf-8");
    console.log(`
  ${bold5("loaf task create")}
`);
    console.log(`  ${green5("\u2713")} Created ${bold5(taskId)}: ${options.title}`);
    console.log(`    File: .agents/tasks/${fileName}`);
    const details = [];
    if (spec) details.push(`Spec: ${spec}`);
    details.push(`Priority: ${priority}`);
    if (dependsOn.length > 0) details.push(`Depends on: ${dependsOn.join(", ")}`);
    console.log(`    ${details.join(gray6(" \xB7 "))}`);
    console.log();
  });
  task.command("update").description("Update a task's metadata").argument("<id>", "Task ID to update (e.g., TASK-031)").option("--status <status>", "New status: todo, in_progress, blocked, review, done").option("--priority <level>", "New priority: P0, P1, P2, P3").option("--depends-on <ids>", "Replace depends_on (comma-separated task IDs)").option("--session <file>", 'Set or clear session reference (use "none" to clear)').option("--spec <id>", "Set or change associated spec").action(async (id, options) => {
    const agentsDir = findAgentsDir();
    if (!agentsDir) {
      console.error(`  ${red4("error:")} Could not find .agents/ directory`);
      process.exit(1);
    }
    if (options.status === void 0 && options.priority === void 0 && options.dependsOn === void 0 && options.session === void 0 && options.spec === void 0) {
      console.error(`  ${red4("error:")} No updates specified. Use --status, --priority, --depends-on, --session, or --spec`);
      process.exit(1);
    }
    const index = getOrBuildIndex(agentsDir);
    const entry = index.tasks[id];
    if (!entry) {
      console.error(`  ${red4("error:")} ${id} not found in index`);
      process.exit(1);
    }
    const changes = [];
    if (options.status !== void 0) {
      if (!TASK_STATUSES.includes(options.status)) {
        console.error(`  ${red4("error:")} Invalid status "${options.status}". Valid: ${TASK_STATUSES.join(", ")}`);
        process.exit(1);
      }
      const oldStatus = entry.status;
      const newStatus = options.status;
      if (newStatus === "done" && oldStatus !== "done") {
        entry.completed_at = (/* @__PURE__ */ new Date()).toISOString();
      } else if (newStatus !== "done" && oldStatus === "done") {
        entry.completed_at = null;
      }
      entry.status = newStatus;
      changes.push({ field: "Status", from: oldStatus, to: newStatus });
    }
    if (options.priority !== void 0) {
      if (!TASK_PRIORITIES.includes(options.priority)) {
        console.error(`  ${red4("error:")} Invalid priority "${options.priority}". Valid: ${TASK_PRIORITIES.join(", ")}`);
        process.exit(1);
      }
      const oldPriority = entry.priority;
      const newPriority = options.priority;
      entry.priority = newPriority;
      changes.push({ field: "Priority", from: oldPriority, to: newPriority });
    }
    if (options.dependsOn !== void 0) {
      const newDeps = options.dependsOn.split(",").map((s) => s.trim()).filter((s) => s.length > 0);
      for (const depId of newDeps) {
        if (!index.tasks[depId]) {
          console.error(`  ${red4("error:")} Unknown task ID "${depId}" in --depends-on`);
          process.exit(1);
        }
      }
      const oldDeps = entry.depends_on.length > 0 ? entry.depends_on.join(", ") : "(none)";
      entry.depends_on = newDeps;
      changes.push({ field: "Depends on", from: oldDeps, to: newDeps.length > 0 ? newDeps.join(", ") : "(none)" });
    }
    if (options.session !== void 0) {
      const oldSession = entry.session || "(none)";
      const newSession = options.session === "none" ? null : options.session;
      entry.session = newSession;
      changes.push({ field: "Session", from: oldSession, to: newSession || "(none)" });
    }
    if (options.spec !== void 0) {
      if (options.spec !== "none" && !index.specs[options.spec]) {
        console.error(`  ${red4("error:")} Unknown spec "${options.spec}". Use \`loaf spec list\` to see valid IDs.`);
        process.exit(1);
      }
      const oldSpec = entry.spec || "(none)";
      const newSpec = options.spec === "none" ? null : options.spec;
      entry.spec = newSpec;
      changes.push({ field: "Spec", from: oldSpec, to: newSpec || "(none)" });
    }
    entry.updated = (/* @__PURE__ */ new Date()).toISOString();
    const indexPath = join21(agentsDir, "TASKS.json");
    saveIndex(indexPath, index);
    syncFrontmatterFromIndex(agentsDir, index);
    console.log(`
  ${bold5("loaf task update")}
`);
    console.log(`  ${green5("\u2713")} Updated ${bold5(id)}: ${entry.title}`);
    for (const change of changes) {
      if (change.from === change.to) {
        console.log(`    ${change.field}: ${change.from} ${gray6("(unchanged)")}`);
      } else {
        console.log(`    ${change.field}: ${change.from} \u2192 ${change.to}`);
      }
    }
    const providedFields = new Set(changes.map((c) => c.field));
    if (!providedFields.has("Status")) {
      console.log(`    Status: ${entry.status} ${gray6("(unchanged)")}`);
    }
    if (!providedFields.has("Priority")) {
      console.log(`    Priority: ${entry.priority} ${gray6("(unchanged)")}`);
    }
    console.log();
  });
  task.command("sync").description("Rebuild TASKS.json from .md files, or import orphans").option("--import", "Import orphan .md files not in the index").action(async (options) => {
    const agentsDir = findAgentsDir();
    if (!agentsDir) {
      console.error(`  ${red4("error:")} Could not find .agents/ directory`);
      process.exit(1);
    }
    const indexPath = join21(agentsDir, "TASKS.json");
    console.log(`
  ${bold5("loaf task sync")}
`);
    if (options.import) {
      const index = getOrBuildIndex(agentsDir);
      const orphans = findOrphans(agentsDir, index);
      const totalOrphans = orphans.tasks.length + orphans.specs.length;
      if (totalOrphans === 0) {
        console.log(`  No orphan files found.`);
        console.log();
        return;
      }
      console.log(`  Found ${totalOrphans} orphan file(s):`);
      for (const orphan of orphans.tasks) {
        console.log(`    ${green5("+")} ${orphan.entry.file}`);
      }
      for (const orphan of orphans.specs) {
        console.log(`    ${green5("+")} ${orphan.entry.file}`);
      }
      let maxTaskNum = index.next_id - 1;
      for (const orphan of orphans.tasks) {
        index.tasks[orphan.id] = orphan.entry;
        const num = extractOrphanNumber(orphan.id);
        if (num > maxTaskNum) maxTaskNum = num;
      }
      for (const orphan of orphans.specs) {
        index.specs[orphan.id] = orphan.entry;
      }
      if (maxTaskNum >= index.next_id) {
        index.next_id = maxTaskNum + 1;
      }
      saveIndex(indexPath, index);
      const importedParts = [];
      if (orphans.tasks.length > 0) importedParts.push(`${orphans.tasks.length} task(s)`);
      if (orphans.specs.length > 0) importedParts.push(`${orphans.specs.length} spec(s)`);
      console.log();
      console.log(`  ${green5("\u2713")} Imported ${importedParts.join(" and ")} into TASKS.json`);
      console.log();
    } else {
      const index = buildIndexFromFiles(agentsDir);
      saveIndex(indexPath, index);
      const statusCounts = {};
      for (const entry of Object.values(index.tasks)) {
        statusCounts[entry.status] = (statusCounts[entry.status] || 0) + 1;
      }
      const totalTasks = Object.keys(index.tasks).length;
      const totalSpecs = Object.keys(index.specs).length;
      const countParts = STATUS_DISPLAY_ORDER.map((s) => {
        const count = statusCounts[s] || 0;
        return `${count} ${s}`;
      });
      console.log(`  ${green5("\u2713")} Rebuilt TASKS.json from .md files`);
      console.log(`    Tasks: ${totalTasks} (${countParts.join(", ")})`);
      console.log(`    Specs: ${totalSpecs}`);
      console.log();
    }
  });
}
function extractOrphanNumber(id) {
  const match = id.match(/\d+$/);
  return match ? parseInt(match[0], 10) : 0;
}

// cli/commands/spec.ts
import { existsSync as existsSync21 } from "fs";
import { join as join22 } from "path";
var bold6 = (s) => `\x1B[1m${s}\x1B[0m`;
var green6 = (s) => `\x1B[32m${s}\x1B[0m`;
var red5 = (s) => `\x1B[31m${s}\x1B[0m`;
var yellow7 = (s) => `\x1B[33m${s}\x1B[0m`;
var cyan5 = (s) => `\x1B[36m${s}\x1B[0m`;
var gray7 = (s) => `\x1B[90m${s}\x1B[0m`;
var STATUS_ORDER = [
  "implementing",
  "approved",
  "drafting",
  "complete"
];
var STATUS_COLORS2 = {
  implementing: yellow7,
  approved: cyan5,
  drafting: gray7,
  complete: green6
};
var STATUS_LABELS2 = {
  implementing: "Implementing",
  approved: "Approved",
  drafting: "Drafting",
  complete: "Complete"
};
function resolveIndex(agentsDir) {
  const indexPath = join22(agentsDir, "TASKS.json");
  if (existsSync21(indexPath)) {
    const index2 = loadIndex(indexPath);
    if (index2) return index2;
    console.error(`  ${red5("error:")} TASKS.json exists but is invalid`);
    process.exit(1);
  }
  const index = buildIndexFromFiles(agentsDir);
  saveIndex(indexPath, index);
  return index;
}
function computeTaskCounts(index) {
  const counts = {};
  for (const [, task] of Object.entries(index.tasks)) {
    if (!task.spec) continue;
    if (!counts[task.spec]) {
      counts[task.spec] = { todo: 0, in_progress: 0, done: 0 };
    }
    const c = counts[task.spec];
    if (task.status === "done") {
      c.done++;
    } else if (task.status === "in_progress") {
      c.in_progress++;
    } else {
      c.todo++;
    }
  }
  return counts;
}
function formatTaskCounts(counts) {
  if (!counts || counts.todo === 0 && counts.in_progress === 0 && counts.done === 0) {
    return gray7("(none)");
  }
  const parts = [
    counts.todo > 0 ? yellow7(String(counts.todo)) : gray7("0"),
    " todo \xB7 ",
    counts.in_progress > 0 ? cyan5(String(counts.in_progress)) : gray7("0"),
    " in_progress \xB7 ",
    counts.done > 0 ? green6(String(counts.done)) : gray7("0"),
    " done"
  ];
  return parts.join("");
}
function registerSpecCommand(program2) {
  const spec = program2.command("spec").description("Manage project specs");
  spec.command("list").description("Show specs with status and task counts").option("--json", "Output raw JSON").action(async (options) => {
    const agentsDir = findAgentsDir();
    if (!agentsDir) {
      console.error(`  ${red5("error:")} No .agents/ directory found`);
      process.exit(1);
    }
    const index = resolveIndex(agentsDir);
    if (options.json) {
      console.log(JSON.stringify(index.specs, null, 2));
      return;
    }
    console.log(`
${bold6("  loaf spec list")}
`);
    const specEntries = Object.entries(index.specs);
    if (specEntries.length === 0) {
      console.log(`  ${gray7("No specs found.")}
`);
      return;
    }
    const taskCounts = computeTaskCounts(index);
    const grouped = {
      implementing: [],
      approved: [],
      drafting: [],
      complete: []
    };
    for (const [id, entry] of specEntries) {
      const status = entry.status;
      if (grouped[status]) {
        grouped[status].push([id, entry]);
      } else {
        grouped.drafting.push([id, entry]);
      }
    }
    for (const status of STATUS_ORDER) {
      grouped[status].sort((a, b) => a[0].localeCompare(b[0]));
    }
    for (const status of STATUS_ORDER) {
      const entries = grouped[status];
      if (entries.length === 0) continue;
      const colorFn = STATUS_COLORS2[status];
      const label = STATUS_LABELS2[status];
      console.log(`  ${bold6(colorFn(`${label} (${entries.length})`))}`);
      for (const [id, entry] of entries) {
        const appetite = entry.appetite ? gray7(entry.appetite) : gray7("TBD");
        console.log(`    ${bold6(id)}  ${entry.title}  ${appetite}`);
        console.log(`              Tasks: ${formatTaskCounts(taskCounts[id])}`);
      }
      console.log();
    }
    console.log(`  Total: ${bold6(String(specEntries.length))} specs
`);
  });
}

// cli/index.ts
var __dirname3 = dirname8(fileURLToPath4(import.meta.url));
function getVersion3() {
  for (const candidate of [
    join23(__dirname3, "..", "package.json"),
    join23(__dirname3, "..", "..", "package.json")
  ]) {
    try {
      const pkg = JSON.parse(readFileSync19(candidate, "utf-8"));
      if (pkg.name === "loaf") return pkg.version;
    } catch {
      continue;
    }
  }
  return "0.0.0";
}
var program = new Command();
program.name("loaf").description("Loaf \u2014 Levi's Opinionated Agentic Framework").version(getVersion3(), "-v, --version");
registerBuildCommand(program);
registerInstallCommand(program);
registerInitCommand(program);
registerReleaseCommand(program);
registerTaskCommand(program);
registerSpecCommand(program);
if (process.argv.length <= 2) {
  program.outputHelp();
  process.exit(0);
}
program.parse();
//# sourceMappingURL=index.js.map