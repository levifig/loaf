/**
 * OpenCode Build Target
 *
 * Generates flat OpenCode structure:
 * dist/opencode/
 * ├── skill/
 * │   ├── python/
 * │   ├── typescript/
 * │   └── ...
 * ├── agent/
 * │   ├── pm.md
 * │   ├── backend-dev.md
 * │   └── ...
 * ├── command/
 * │   ├── start-session.md
 * │   └── ...
 * └── plugin/
 *     └── hooks.js
 *
 * Reads frontmatter from sidecars (e.g., pm.opencode.yaml, SKILL.opencode.yaml)
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
import { loadAgentSidecar, loadSkillFrontmatter } from "../lib/sidecar.js";
import { parse as parseYaml } from "yaml";
import { getVersion } from "../lib/version.js";
import { buildAgentMap, substituteAgentNames } from "../lib/substitutions.js";
import { copySharedTemplates } from "../lib/shared-templates.js";

/**
 * Substitute command placeholders with OpenCode unscoped commands
 *
 * Placeholders:
 * - {{IMPLEMENT_CMD}} -> /implement
 * - {{RESUME_CMD}} -> /resume
 * - {{ORCHESTRATE_CMD}} -> /implement
 */
function substituteCommands(content) {
  return content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/resume")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
}

/**
 * Copy references directory with command substitution for markdown files
 */
function copyReferencesWithSubstitution(srcDir, destDir) {
  mkdirSync(destDir, { recursive: true });

  const entries = readdirSync(srcDir, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = join(srcDir, entry.name);
    const destPath = join(destDir, entry.name);

    if (entry.isDirectory()) {
      copyReferencesWithSubstitution(srcPath, destPath);
    } else if (entry.name.endsWith(".md")) {
      // Apply substitutions to markdown files
      const content = readFileSync(srcPath, "utf-8");
      writeFileSync(destPath, substituteAgentNames(substituteCommands(content), AGENT_MAP));
    } else {
      // Copy non-markdown files as-is
      cpSync(srcPath, destPath);
    }
  }
}

const TARGET_NAME = "opencode";

// Version is loaded dynamically from package.json at build time
let VERSION = "0.0.0";

// Agent name map is loaded dynamically from sidecars at build time
let AGENT_MAP = {};

/**
 * Build OpenCode distribution
 */
export async function build({
  config,
  targetConfig,
  targetsConfig,
  rootDir,
  srcDir,
  distDir,
}) {
  // Load version from package.json at build time
  VERSION = getVersion(rootDir);

  // Build agent name map from sidecars
  AGENT_MAP = buildAgentMap(srcDir, TARGET_NAME);

  // Clean and create dist directory
  if (existsSync(distDir)) {
    rmSync(distDir, { recursive: true });
  }
  mkdirSync(distDir, { recursive: true });

  // Copy skills with frontmatter from sidecars
  copySkills(srcDir, distDir, targetsConfig);

  // Copy and transform agents with frontmatter from sidecars
  copyAgents(srcDir, distDir);

  // Generate commands from skills with OpenCode sidecars
  generateCommandsFromSkills(srcDir, distDir);

  // Generate hooks.js
  generateHooks(config, srcDir, distDir);
}

/**
 * Copy all skills with frontmatter from sidecars
 */
function copySkills(srcDir, distDir, targetsConfig) {
  const src = join(srcDir, "skills");
  const dest = join(distDir, "skill");

  if (!existsSync(src)) {
    return;
  }

  mkdirSync(dest, { recursive: true });

  const skills = readdirSync(src, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillSrc = join(src, skill);
    const skillDest = join(dest, skill);
    mkdirSync(skillDest, { recursive: true });

    // Load frontmatter from SKILL.md (standard format works for OpenCode)
    const frontmatter = loadSkillFrontmatter(skillSrc);

    // Read SKILL.md body (strip existing frontmatter if any)
    const skillMdPath = join(skillSrc, "SKILL.md");
    if (existsSync(skillMdPath)) {
      const content = readFileSync(skillMdPath, "utf-8");
      const { content: body } = matter(content);

      // Write SKILL.md with frontmatter, command and agent name substitution
      const transformed = substituteAgentNames(
        substituteCommands(matter.stringify(body, frontmatter)),
        AGENT_MAP
      );
      writeFileSync(join(skillDest, "SKILL.md"), transformed);
    }

    // Copy references directory with command substitution
    const refSrc = join(skillSrc, "references");
    const refDest = join(skillDest, "references");
    if (existsSync(refSrc)) {
      copyReferencesWithSubstitution(refSrc, refDest);
    }

    // Copy scripts directory
    const scriptsSrc = join(skillSrc, "scripts");
    const scriptsDest = join(skillDest, "scripts");
    if (existsSync(scriptsSrc)) {
      cpSync(scriptsSrc, scriptsDest, { recursive: true });
    }

    // Copy templates directory with substitution
    const templatesSrc = join(skillSrc, "templates");
    const templatesDest = join(skillDest, "templates");
    if (existsSync(templatesSrc)) {
      copyReferencesWithSubstitution(templatesSrc, templatesDest);
    }

    // Copy shared templates for this skill (won't overwrite skill-specific ones)
    copySharedTemplates(skill, skillDest, srcDir, targetsConfig, (content) =>
      substituteAgentNames(substituteCommands(content), AGENT_MAP)
    );
  }
}

/**
 * Copy and transform all agents with frontmatter from sidecars
 */
function copyAgents(srcDir, distDir) {
  const src = join(srcDir, "agents");
  const dest = join(distDir, "agent");

  if (!existsSync(src)) {
    return;
  }

  mkdirSync(dest, { recursive: true });

  const files = readdirSync(src).filter((f) => f.endsWith(".md"));

  for (const file of files) {
    const srcPath = join(src, file);
    const destPath = join(dest, file);

    // Read source body (strip existing frontmatter if any)
    const content = readFileSync(srcPath, "utf-8");
    const { content: body } = matter(content);

    // Load frontmatter from sidecar
    const frontmatter = loadAgentSidecar(srcPath, TARGET_NAME);

    // Reconstruct the file with sidecar frontmatter and agent name substitution
    const transformed = substituteAgentNames(
      matter.stringify(body, frontmatter),
      AGENT_MAP
    );
    writeFileSync(destPath, transformed);
  }
}

/**
 * Generate OpenCode commands from skills that have SKILL.opencode.yaml
 *
 * Skills with an OpenCode sidecar emit command files to dist/opencode/command/.
 * Routing metadata (agent, subtask, model) comes from the sidecar.
 * Body content and description come from SKILL.md.
 */
function generateCommandsFromSkills(srcDir, distDir) {
  const skillsSrc = join(srcDir, "skills");
  const commandsDest = join(distDir, "command");

  if (!existsSync(skillsSrc)) {
    return;
  }

  mkdirSync(commandsDest, { recursive: true });

  const skills = readdirSync(skillsSrc, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const skill of skills) {
    const skillDir = join(skillsSrc, skill);
    const sidecarPath = join(skillDir, "SKILL.opencode.yaml");

    // Only generate commands for skills with OpenCode sidecar
    if (!existsSync(sidecarPath)) {
      continue;
    }

    const skillMdPath = join(skillDir, "SKILL.md");
    if (!existsSync(skillMdPath)) {
      continue;
    }

    // Load SKILL.md frontmatter and body
    const content = readFileSync(skillMdPath, "utf-8");
    const { content: body, data: skillFrontmatter } = matter(content);

    // Load OpenCode sidecar for routing metadata
    const sidecarContent = readFileSync(sidecarPath, "utf-8");
    const sidecar = parseYaml(sidecarContent) || {};

    // Merge: skill description + sidecar routing + version
    const mergedFrontmatter = {
      description: skillFrontmatter.description || "",
      ...sidecar,
      version: VERSION,
    };

    // Resolve agent name placeholders in frontmatter agent field
    if (mergedFrontmatter.agent && typeof mergedFrontmatter.agent === "string") {
      mergedFrontmatter.agent = substituteAgentNames(mergedFrontmatter.agent, AGENT_MAP);
    }

    // Rewrite relative links: command files live in command/ but templates and
    // references live under skill/{name}/, so remap paths accordingly
    const relinked = body
      .replace(/\]\(templates\//g, `](../skill/${skill}/templates/`)
      .replace(/\]\(references\//g, `](../skill/${skill}/references/`);

    // Write command file with merged frontmatter, command and agent name substitution
    const transformed = substituteAgentNames(
      substituteCommands(matter.stringify(relinked, mergedFrontmatter)),
      AGENT_MAP
    );
    writeFileSync(join(commandsDest, `${skill}.md`), transformed);
  }
}

/**
 * Generate OpenCode hooks.js
 *
 * OpenCode hook format:
 * - tool.execute.pre.<matcher>
 * - tool.execute.post.<matcher>
 * - session.start
 * - session.end
 */
function generateHooks(config, srcDir, distDir) {
  const pluginDir = join(distDir, "plugin");
  mkdirSync(pluginDir, { recursive: true });

  // Copy hooks lib and scripts as-is (hooks compare against runtime $AGENT_TYPE slugs)
  const hooksSrc = join(srcDir, "hooks");
  const hooksDest = join(pluginDir, "hooks");
  if (existsSync(hooksSrc)) {
    cpSync(hooksSrc, hooksDest, { recursive: true });
  }

  // Generate hooks.js
  const hooksJs = generateHooksJs(config);
  writeFileSync(join(pluginDir, "hooks.js"), hooksJs);
}

/**
 * Generate hooks.js content
 *
 * OpenCode plugin format: export default async function that returns hooks object
 * Hook names: tool.execute.before, tool.execute.after, event
 * Hook signature: (input, output) => Promise<void>
 */
function generateHooksJs(config) {
  const preToolHooks = config.hooks["pre-tool"] || [];
  const postToolHooks = config.hooks["post-tool"] || [];
  const sessionHooks = config.hooks.session || [];

  // Group pre-tool by matcher
  const preToolByMatcher = {};
  for (const hook of preToolHooks) {
    const matcher = hook.matcher || "Edit|Write";
    if (!preToolByMatcher[matcher]) {
      preToolByMatcher[matcher] = [];
    }
    preToolByMatcher[matcher].push(hook);
  }

  // Group post-tool by matcher
  const postToolByMatcher = {};
  for (const hook of postToolHooks) {
    const matcher = hook.matcher || "Edit|Write";
    if (!postToolByMatcher[matcher]) {
      postToolByMatcher[matcher] = [];
    }
    postToolByMatcher[matcher].push(hook);
  }

  return `/**
 * OpenCode Plugin - Agent Skills Hooks
 * Auto-generated by loaf build system
 *
 * This plugin provides quality gates and automation hooks.
 */

import { execFileSync } from 'child_process';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const HOOKS_DIR = join(__dirname, 'hooks');

/**
 * Execute a hook script safely
 */
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

/**
 * Check if tool matches pattern
 */
function matchesTool(toolName, pattern) {
  const patterns = pattern.split('|');
  return patterns.includes(toolName);
}

/**
 * Pre-tool hooks by matcher
 */
const preToolHooks = {
${Object.entries(preToolByMatcher)
  .map(
    ([matcher, hooks]) => `  '${matcher}': [
${hooks.map((h) => `    { id: '${h.id}', script: '${getScriptFilename(h.script)}', timeout: ${h.timeout || 60000} },`).join("\n")}
  ],`
  )
  .join("\n")}
};

/**
 * Post-tool hooks by matcher
 */
const postToolHooks = {
${Object.entries(postToolByMatcher)
  .map(
    ([matcher, hooks]) => `  '${matcher}': [
${hooks.map((h) => `    { id: '${h.id}', script: '${getScriptFilename(h.script)}', timeout: ${h.timeout || 60000} },`).join("\n")}
  ],`
  )
  .join("\n")}
};

/**
 * Session hooks
 */
const sessionHooks = {
${sessionHooks.map((h) => `  '${h.event.toLowerCase()}': { id: '${h.id}', script: '${getScriptFilename(h.script)}', timeout: ${h.timeout || 60000} },`).join("\n")}
};

/**
 * OpenCode Plugin Entry Point
 *
 * Receives context: { client, $, directory, project, worktree }
 * Returns hooks object
 */
export default async function AgentSkillsPlugin({ client, $ }) {
  return {
    /**
     * Called before tool execution
     */
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

    /**
     * Called after tool execution
     */
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

    /**
     * Called on session events
     */
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

/**
 * Get script filename from path
 */
function getScriptFilename(scriptPath) {
  const parts = scriptPath.split("/");
  // Keep the subdirectory structure: pre-tool/script.sh, post-tool/script.sh
  return parts.slice(-2).join("/");
}
