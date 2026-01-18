/**
 * Sidecar Metadata System
 *
 * Provides utilities for loading and applying target-specific transforms.
 * Sidecars are optional YAML files that override defaults for specific targets.
 *
 * Example sidecar: pm.opencode.yaml
 * ```yaml
 * frontmatter:
 *   name: PM
 *   mode: primary
 *   tools:
 *     remove: ["Task(*)"]
 *     add: { Task: true }
 * ```
 */

import { readFileSync, existsSync } from "fs";
import { join, dirname, basename } from "path";
import { parse as parseYaml } from "yaml";

/**
 * Load sidecar config for a source file and target, merged with defaults
 *
 * @param {string} sourcePath - Path to the source file (e.g., src/agents/pm.md)
 * @param {string} target - Target name (e.g., 'opencode', 'cursor')
 * @param {Object} defaults - Default config from targets.yaml
 * @returns {Object} Merged config (defaults + sidecar overrides)
 */
export function loadSidecar(sourcePath, target, defaults = {}) {
  // Build sidecar path: pm.md -> pm.opencode.yaml
  const dir = dirname(sourcePath);
  const baseName = basename(sourcePath, ".md");
  const sidecarPath = join(dir, `${baseName}.${target}.yaml`);

  // Start with defaults
  const config = structuredClone(defaults);

  // If sidecar exists, merge its config
  if (existsSync(sidecarPath)) {
    const sidecarContent = readFileSync(sidecarPath, "utf-8");
    const sidecarConfig = parseYaml(sidecarContent);

    // Deep merge sidecar into defaults
    deepMerge(config, sidecarConfig);
  }

  return config;
}

/**
 * Load skill sidecar config
 *
 * @param {string} skillDir - Path to skill directory (e.g., src/skills/python)
 * @param {string} target - Target name
 * @param {Object} defaults - Default config from targets.yaml
 * @returns {Object} Merged config
 */
export function loadSkillSidecar(skillDir, target, defaults = {}) {
  const sidecarPath = join(skillDir, `SKILL.${target}.yaml`);

  const config = structuredClone(defaults);

  if (existsSync(sidecarPath)) {
    const sidecarContent = readFileSync(sidecarPath, "utf-8");
    const sidecarConfig = parseYaml(sidecarContent);
    deepMerge(config, sidecarConfig);
  }

  return config;
}

/**
 * Apply frontmatter transforms based on config
 *
 * Handles:
 * - Direct value overrides (name, mode, etc.)
 * - Tools transformation (array-to-record, remove, add)
 *
 * @param {Object} frontmatter - Original frontmatter from source
 * @param {Object} config - Transform config (from sidecar or defaults)
 * @returns {Object} Transformed frontmatter
 */
export function applyFrontmatterTransforms(frontmatter, config) {
  const result = structuredClone(frontmatter);

  if (!config.frontmatter) {
    return result;
  }

  const fm = config.frontmatter;

  // Apply direct value overrides
  for (const [key, value] of Object.entries(fm)) {
    if (key === "tools") continue; // Handle tools separately
    result[key] = value;
  }

  // Apply tools transforms
  if (fm.tools) {
    result.tools = applyToolsTransform(result.tools, fm.tools);
  }

  return result;
}

/**
 * Apply tools-specific transforms
 *
 * @param {Array|Object} tools - Original tools (usually array)
 * @param {Object} config - Tools transform config
 * @returns {Array|Object} Transformed tools
 */
function applyToolsTransform(tools, config) {
  if (!tools) return tools;

  let result = Array.isArray(tools) ? [...tools] : { ...tools };

  // Remove specified tools (supports glob patterns like "Task(*)")
  if (config.remove && Array.isArray(result)) {
    const removePatterns = config.remove;
    result = result.filter((tool) => {
      return !removePatterns.some((pattern) => matchesPattern(tool, pattern));
    });
  }

  // Transform array to record if specified
  if (config.transform === "array-to-record" && Array.isArray(result)) {
    const record = {};
    for (const tool of result) {
      record[tool] = true;
    }
    result = record;
  }

  // Add new tools
  if (config.add) {
    if (typeof result === "object" && !Array.isArray(result)) {
      // Record format - merge in additions
      Object.assign(result, config.add);
    } else if (Array.isArray(result)) {
      // Array format - add tool names
      for (const [tool, enabled] of Object.entries(config.add)) {
        if (enabled && !result.includes(tool)) {
          result.push(tool);
        }
      }
    }
  }

  return result;
}

/**
 * Check if a tool name matches a pattern
 *
 * Supports:
 * - Exact match: "Task" matches "Task"
 * - Glob pattern: "Task(*)" matches "Task(backend-dev)", "Task(frontend-dev)", etc.
 *
 * @param {string} tool - Tool name to check
 * @param {string} pattern - Pattern to match against
 * @returns {boolean} True if matches
 */
function matchesPattern(tool, pattern) {
  // Exact match
  if (tool === pattern) return true;

  // Convert glob pattern to regex
  // "Task(*)" -> /^Task\(.*\)$/
  if (pattern.includes("*")) {
    const regexPattern = pattern
      .replace(/[.+?^${}()|[\]\\]/g, "\\$&") // Escape special chars
      .replace(/\\\*/g, ".*"); // Convert * to .*
    const regex = new RegExp(`^${regexPattern}$`);
    return regex.test(tool);
  }

  return false;
}

/**
 * Deep merge source into target
 *
 * @param {Object} target - Object to merge into (modified in place)
 * @param {Object} source - Object to merge from
 */
function deepMerge(target, source) {
  for (const [key, value] of Object.entries(source)) {
    if (
      value &&
      typeof value === "object" &&
      !Array.isArray(value) &&
      target[key] &&
      typeof target[key] === "object"
    ) {
      deepMerge(target[key], value);
    } else {
      target[key] = value;
    }
  }
}

/**
 * Load targets.yaml configuration
 *
 * @param {string} configDir - Path to config directory (e.g., src/config)
 * @returns {Object} Targets configuration
 */
export function loadTargetsConfig(configDir) {
  const configPath = join(configDir, "targets.yaml");
  if (!existsSync(configPath)) {
    return { targets: {} };
  }
  const content = readFileSync(configPath, "utf-8");
  return parseYaml(content);
}

/**
 * Get defaults for a specific target and file type
 *
 * @param {Object} targetsConfig - Full targets.yaml config
 * @param {string} target - Target name
 * @param {string} fileType - 'agents' or 'skills'
 * @returns {Object} Default config for that target/type combo
 */
export function getTargetDefaults(targetsConfig, target, fileType) {
  return targetsConfig.targets?.[target]?.defaults?.[fileType] || {};
}
