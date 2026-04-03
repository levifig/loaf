/**
 * Codex Build Target
 *
 * Codex supports: skills, hooks (Bash-matching enforcement only)
 * Generates dist/codex/.codex/hooks.json with Bash-only enforcement hooks.
 * Reads from shared intermediate at dist/skills/, merges SKILL.codex.yaml sidecar.
 */

import { mkdirSync, writeFileSync } from "fs";
import { join } from "path";
import { resetTargetOutput } from "../lib/target-output.js";
import { copySkills } from "../lib/skills.js";
import { loadTargetSkillSidecar } from "../lib/sidecar.js";
import { getVersion } from "../lib/version.js";
import type { BuildContext, HooksConfig, SkillFrontmatter } from "../types.js";

const TARGET_NAME = "codex";
const LOAF_HOOK_MARKER = "loaf-managed";

// Enforcement hooks that use `loaf check --hook <id>`
const ENFORCEMENT_HOOKS = new Set([
  "check-secrets",
  "validate-push",
  "validate-commit",
  "workflow-pre-pr",
  "security-audit",
]);

export async function build({
  config,
  rootDir,
  distDir,
  targetsConfig,
}: BuildContext): Promise<void> {
  const version = getVersion(rootDir);
  resetTargetOutput(distDir);

  // Identity transform - commands already substituted in intermediate
  const transformMd = (content: string) => content;

  // Merge sidecar fields into frontmatter
  const mergeFrontmatter = (
    base: SkillFrontmatter,
    skillDir: string,
  ): SkillFrontmatter => {
    const sidecar = loadTargetSkillSidecar(skillDir, TARGET_NAME);
    return { ...base, ...sidecar };
  };

  copySkills({
    srcDir: join(rootDir, "dist"), // Read from dist/skills/ intermediate
    destDir: join(distDir, "skills"),
    targetName: TARGET_NAME,
    version,
    targetsConfig,
    transformMd,
    mergeFrontmatter,
  });

  // Generate Codex hooks.json (Bash-only enforcement hooks)
  generateCodexHooksJson(config as HooksConfig, distDir);
}

/**
 * Generate Codex hooks.json with Bash-only enforcement hooks.
 * Codex platform limitation: Only Bash matcher is supported.
 */
function generateCodexHooksJson(config: HooksConfig, distDir: string): void {
  const preToolHooks = config.hooks["pre-tool"] || [];

  // Filter to only enforcement hooks with Bash matcher
  const enforcementHooks = preToolHooks.filter((hook) => {
    // Must be an enforcement hook
    if (!ENFORCEMENT_HOOKS.has(hook.id)) return false;
    // Must have Bash matcher (Codex limitation)
    const matcher = hook.matcher || "";
    return matcher.includes("Bash");
  });

  if (enforcementHooks.length === 0) {
    // No hooks to generate
    return;
  }

  const hooksJson: Record<string, unknown> = {
    version: 1,
    hooks: {} as Record<string, unknown>,
  };

  const hooks = hooksJson.hooks as Record<string, unknown>;

  // Codex uses PreToolUse (camelCase like Claude Code)
  hooks.PreToolUse = enforcementHooks.map((hook) => {
    const result: Record<string, unknown> = {
      [LOAF_HOOK_MARKER]: true,
      matcher: "Bash",
      command: `loaf check --hook ${hook.id}`,
      timeout: Math.floor((hook.timeout || 30000) / 1000),
      failClosed: hook.failClosed !== false, // Default to true for enforcement
    };

    if (hook.description) {
      result.description = hook.description;
    }

    return result;
  });

  // Ensure .codex directory exists
  const codexDir = join(distDir, ".codex");
  mkdirSync(codexDir, { recursive: true });

  writeFileSync(
    join(codexDir, "hooks.json"),
    JSON.stringify(hooksJson, null, 2),
  );
}
