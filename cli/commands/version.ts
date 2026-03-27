/**
 * loaf version command
 *
 * Displays version info, built targets, and content statistics.
 */

import { Command } from "commander";
import { existsSync, readFileSync, readdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { parse as parseYaml } from "yaml";

const __dirname = dirname(fileURLToPath(import.meta.url));

// ANSI color helpers
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;

/** Target definitions: name -> relative output directory from root */
const TARGET_OUTPUTS: Record<string, string> = {
  "claude-code": "plugins/loaf/",
  cursor: "dist/cursor/",
  opencode: "dist/opencode/",
  codex: "dist/codex/",
  gemini: "dist/gemini/",
};

function findRootDir(): string {
  let dir = __dirname;
  for (let i = 0; i < 10; i++) {
    const pkgPath = join(dir, "package.json");
    try {
      const pkg = JSON.parse(readFileSync(pkgPath, "utf-8"));
      if (pkg.name === "loaf") return dir;
    } catch {
      // not found, go up
    }
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error("Could not find loaf root directory");
}

function getVersion(rootDir: string): string {
  try {
    const pkg = JSON.parse(readFileSync(join(rootDir, "package.json"), "utf-8"));
    return pkg.version || "0.0.0";
  } catch {
    return "0.0.0";
  }
}

function countSkills(rootDir: string): number {
  const skillsDir = join(rootDir, "content", "skills");
  if (!existsSync(skillsDir)) return 0;
  return readdirSync(skillsDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .length;
}

function countAgents(rootDir: string): number {
  const agentsDir = join(rootDir, "content", "agents");
  if (!existsSync(agentsDir)) return 0;
  return readdirSync(agentsDir)
    .filter((name) => name.endsWith(".md"))
    .length;
}

function countHooks(rootDir: string): number {
  const hooksPath = join(rootDir, "config", "hooks.yaml");
  if (!existsSync(hooksPath)) return 0;
  try {
    const config = parseYaml(readFileSync(hooksPath, "utf-8")) as {
      hooks?: {
        "pre-tool"?: unknown[];
        "post-tool"?: unknown[];
        session?: unknown[];
      };
    };
    const hooks = config?.hooks;
    if (!hooks) return 0;
    return (
      (hooks["pre-tool"]?.length ?? 0) +
      (hooks["post-tool"]?.length ?? 0) +
      (hooks.session?.length ?? 0)
    );
  } catch {
    return 0;
  }
}

function getBuiltTargets(rootDir: string): Array<{ name: string; path: string }> {
  const built: Array<{ name: string; path: string }> = [];
  for (const [name, relPath] of Object.entries(TARGET_OUTPUTS)) {
    if (existsSync(join(rootDir, relPath))) {
      built.push({ name, path: relPath });
    }
  }
  return built;
}

export function registerVersionCommand(program: Command): void {
  program
    .command("version")
    .description("Show version info and project statistics")
    .action(() => {
      const rootDir = findRootDir();
      const version = getVersion(rootDir);

      console.log(`\n${bold("loaf")} ${version}`);
      console.log(`${gray("node")} ${process.version}`);

      // Targets
      const targets = getBuiltTargets(rootDir);
      if (targets.length > 0) {
        console.log(`\n${bold("Targets:")}`);
        const maxName = Math.max(...targets.map((t) => t.name.length));
        for (const target of targets) {
          console.log(`  ${target.name.padEnd(maxName + 2)}${gray(target.path)}`);
        }
      }

      // Content stats
      const skills = countSkills(rootDir);
      const agents = countAgents(rootDir);
      const hooks = countHooks(rootDir);

      console.log(`\n${bold("Content:")}`);
      console.log(`  Skills:  ${skills}`);
      console.log(`  Agents:  ${agents}`);
      console.log(`  Hooks:   ${hooks}`);
      console.log();
    });
}
