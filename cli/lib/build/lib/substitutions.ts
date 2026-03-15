/**
 * Agent Name Substitution System
 *
 * Resolves {{AGENT:slug}} placeholders in content with target-specific display
 * names from agent sidecars.
 */

import { readFileSync, readdirSync, existsSync } from "fs";
import { join, basename } from "path";
import { parse as parseYaml } from "yaml";

/**
 * Build a { slug: displayName } map by scanning agent sidecars.
 * Display name comes from the sidecar's `name` field.
 */
export function buildAgentMap(
  srcDir: string,
  target: string,
): Record<string, string> {
  const agentsDir = join(srcDir, "agents");
  const map: Record<string, string> = {};

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
        const sidecar = parseYaml(content) as Record<string, unknown> | null;
        map[slug] = (sidecar?.name as string) || slug;
      } catch {
        map[slug] = slug;
      }
    } else {
      map[slug] = slug;
    }
  }

  return map;
}

/**
 * Replace all {{AGENT:slug}} placeholders in content with display names.
 */
export function substituteAgentNames(
  content: string,
  agentMap: Record<string, string>,
): string {
  return content.replace(/\{\{AGENT:([^}]+)\}\}/g, (_match, slug: string) => {
    if (slug in agentMap) {
      return agentMap[slug];
    }
    console.warn(
      `[loaf] Unknown agent placeholder: {{AGENT:${slug}}} — using slug as-is`,
    );
    return slug;
  });
}
