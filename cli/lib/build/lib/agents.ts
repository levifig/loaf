/**
 * Agents Module
 *
 * Shared module for copying and transforming agents across all build targets.
 * Extracts common patterns from claude-code.ts, cursor.ts, and opencode.ts
 * target transformers.
 */

import {
  mkdirSync,
  writeFileSync,
  readFileSync,
  existsSync,
  readdirSync,
} from "fs";
import matter from "gray-matter";
import { join } from "path";
import { loadAgentSidecar, loadAgentSidecarOptional } from "./sidecar.js";

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface CopyAgentsOptions {
  /** Source directory containing agents/ subdirectory */
  srcDir: string;
  /** Destination directory for agents output */
  destDir: string;
  /** Target name (e.g., 'claude-code', 'cursor') */
  targetName: string;
  /** Version string to add to output */
  version: string;
  /** Default frontmatter values to merge */
  defaults?: Record<string, unknown>;
  /**
   * Whether a sidecar file is required for each agent.
   * - true (Claude Code): Throws error if sidecar missing
   * - false (Cursor, OpenCode): Uses empty object if sidecar missing
   */
  sidecarRequired?: boolean;
}

// ─────────────────────────────────────────────────────────────────────────────
// Implementation
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Copy all agents from source to destination with frontmatter merging.
 *
 * This function:
 * 1. Discovers agents (*.md files) from srcDir/agents
 * 2. Loads source frontmatter from each agent's markdown file
 * 3. Loads target-specific sidecar (required or optional based on sidecarRequired)
 * 4. Merges: defaults < source frontmatter < sidecar frontmatter
 * 5. Adds version to body footer
 * 6. Writes formatted output
 *
 * Different targets have different requirements:
 * - Claude Code: sidecarRequired=true, no defaults (sidecar has everything)
 * - Cursor: sidecarRequired=false, uses defaults from targets.yaml
 * - OpenCode: sidecarRequired=true (all frontmatter in sidecar)
 */
export function copyAgents(options: CopyAgentsOptions): void {
  const {
    srcDir,
    destDir,
    targetName,
    version,
    defaults = {},
    sidecarRequired = false,
  } = options;

  const src = join(srcDir, "agents");
  if (!existsSync(src)) return;

  const files = readdirSync(src).filter((f) => f.endsWith(".md"));

  for (const file of files) {
    const srcPath = join(src, file);
    const destPath = join(destDir, file);
    const agentName = file.replace(".md", "");

    const content = readFileSync(srcPath, "utf-8");
    const { content: body, data: sourceFrontmatter } = matter(content);

    // Load sidecar (required or optional)
    const sidecarFrontmatter = sidecarRequired
      ? loadAgentSidecar(srcPath, targetName)
      : loadAgentSidecarOptional(srcPath, targetName);

    // Merge frontmatter: defaults < source < sidecar
    // Source provides name/description if not in sidecar
    const mergedFrontmatter: Record<string, unknown> = {
      ...defaults,
      name: (sourceFrontmatter as Record<string, unknown>).name || agentName,
      description:
        (sourceFrontmatter as Record<string, unknown>).description ||
        `${agentName} agent for specialized tasks`,
      ...sidecarFrontmatter,
    };

    // Add version footer
    const bodyWithFooter = body.trim() + `\n\n---\nversion: ${version}\n`;

    writeFileSync(destPath, matter.stringify(bodyWithFooter, mergedFrontmatter));
  }
}
