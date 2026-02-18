/**
 * Agent Name Substitution System
 *
 * Resolves {{AGENT:slug}} placeholders in content with target-specific display
 * names from agent sidecars. Display names come from the `name` field in each
 * agent's target sidecar (e.g., pm.opencode.yaml → name: "PM").
 *
 * If no sidecar exists for a given target, the slug is used as-is.
 */

import { readFileSync, readdirSync, existsSync } from "fs";
import { join, basename } from "path";
import { parse as parseYaml } from "yaml";

/**
 * Build a { slug: displayName } map by scanning agent sidecars
 *
 * Discovers all agent .md files in src/agents/, then reads the corresponding
 * {slug}.{target}.yaml sidecar for each. The display name comes from the
 * sidecar's `name` field. If no sidecar exists, the slug itself is used.
 *
 * @param {string} srcDir - Path to src/ directory
 * @param {string} target - Target name (e.g., 'opencode', 'claude-code')
 * @returns {Object.<string, string>} Map of slug to display name
 */
export function buildAgentMap(srcDir, target) {
  const agentsDir = join(srcDir, "agents");
  const map = {};

  if (!existsSync(agentsDir)) {
    return map;
  }

  const agentFiles = readdirSync(agentsDir).filter((f) => f.endsWith(".md"));

  for (const file of agentFiles) {
    const slug = basename(file, ".md");
    const sidecarPath = join(agentsDir, `${slug}.${target}.yaml`);

    if (existsSync(sidecarPath)) {
      try {
        const content = readFileSync(sidecarPath, "utf-8");
        const sidecar = parseYaml(content);
        map[slug] = sidecar?.name || slug;
      } catch {
        // Malformed sidecar — fall back to slug
        map[slug] = slug;
      }
    } else {
      map[slug] = slug;
    }
  }

  return map;
}

/**
 * Replace all {{AGENT:slug}} placeholders in content with display names
 *
 * Warns to console if a placeholder references a slug not found in the map.
 * Unknown slugs are replaced with the raw slug value so output is never broken.
 *
 * @param {string} content - Content string to process
 * @param {Object.<string, string>} agentMap - Map of slug to display name
 * @returns {string} Content with placeholders resolved
 */
export function substituteAgentNames(content, agentMap) {
  return content.replace(/\{\{AGENT:([^}]+)\}\}/g, (match, slug) => {
    if (slug in agentMap) {
      return agentMap[slug];
    }
    console.warn(
      `[loaf] Unknown agent placeholder: {{AGENT:${slug}}} — using slug as-is`
    );
    return slug;
  });
}
