/**
 * loaf install command
 *
 * Detects AI tools and installs Loaf distributions to their config directories.
 * Replaces install.sh for post-build installation.
 */

import { Command } from "commander";
import { existsSync, readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { createInterface } from "readline";
import {
  detectTools,
  detectClaudeCode,
  DEFAULT_CONFIG_DIRS,
  isDevMode,
  type DetectedTool,
} from "../lib/detect/tools.js";
import { INSTALLERS } from "../lib/install/installer.js";
import { installFencedSectionsForTargets } from "../lib/install/fenced-section.js";

const __dirname = dirname(fileURLToPath(import.meta.url));

// ANSI colors
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;
const cyan = (s: string) => `\x1b[36m${s}\x1b[0m`;
const white = (s: string) => `\x1b[97m${s}\x1b[0m`;

function findRootDir(): string {
  let dir = __dirname;
  for (let i = 0; i < 10; i++) {
    const pkgPath = join(dir, "package.json");
    try {
      const pkg = JSON.parse(readFileSync(pkgPath, "utf-8"));
      if (pkg.name === "loaf") return dir;
    } catch {
      // not found
    }
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error("Could not find loaf root directory");
}

function askYesNo(question: string): Promise<boolean> {
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer.trim().toLowerCase().startsWith("y"));
    });
  });
}

const VALID_TARGETS = Object.keys(DEFAULT_CONFIG_DIRS);

export function registerInstallCommand(program: Command): void {
  program
    .command("install")
    .description("Install Loaf to detected AI tool configurations")
    .option("--to <target>", 'Target to install to (or "all")')
    .option("--upgrade", "Update only already-installed targets")
    .action(async (options: { to?: string; upgrade?: boolean }) => {
      const rootDir = findRootDir();
      const distDir = join(rootDir, "dist");
      const devMode = isDevMode(rootDir);

      console.log(`\n${bold("loaf install")}\n`);

      // Detect tools
      const hasClaudeCode = detectClaudeCode();
      const tools = detectTools();

      // Show detection results
      if (hasClaudeCode) {
        console.log(`  ${green("✓")} Claude Code detected`);
        if (devMode) {
          console.log(
            `    ${gray("Test locally:")} ${white(`/plugin marketplace add ${rootDir}`)}`,
          );
        } else {
          console.log(
            `    ${gray("Install via:")} ${white("/plugin marketplace add levifig/loaf")}`,
          );
        }
        console.log();
      }

      for (const tool of tools) {
        const status = tool.installed
          ? ` ${yellow("(installed)")}`
          : "";
        console.log(`  ${green("✓")} ${tool.name} detected${status}`);
      }

      if (tools.length === 0 && !hasClaudeCode) {
        console.log(`  ${gray("No AI tools detected")}`);
        console.log();
        return;
      }
      console.log();

      // Determine targets to install
      let selectedTargets: string[];

      if (options.to === "all") {
        selectedTargets = tools.map((t) => t.key);
      } else if (options.to) {
        if (!VALID_TARGETS.includes(options.to) && options.to !== "all") {
          console.error(
            `${red("error:")} Unknown target ${bold(options.to)}\n` +
              `${gray("Valid targets:")} ${VALID_TARGETS.join(", ")}, all`,
          );
          process.exit(1);
        }

        // Ensure config dir exists for the target
        const tool = tools.find((t) => t.key === options.to);
        selectedTargets = [options.to];

        if (!tool) {
          console.log(
            `  ${yellow("⚡")} ${options.to} was not auto-detected; installing to ${DEFAULT_CONFIG_DIRS[options.to]}`,
          );
        }
      } else if (options.upgrade) {
        selectedTargets = tools
          .filter((t) => t.installed)
          .map((t) => t.key);

        if (selectedTargets.length === 0) {
          console.log(`  ${gray("No installed targets to upgrade")}`);
          console.log();
          return;
        }
        console.log(`  ${gray("Upgrading:")} ${selectedTargets.join(", ")}`);
      } else {
        // Interactive selection
        selectedTargets = [];
        for (const tool of tools) {
          const status = tool.installed ? ` ${yellow("(installed)")}` : "";
          const yes = await askYesNo(
            `  Install to ${bold(tool.name)}${status}? [y/N] `,
          );
          if (yes) {
            selectedTargets.push(tool.key);
          }
        }
      }

      if (selectedTargets.length === 0) {
        console.log(`  ${gray("No targets selected")}`);
        console.log();
        return;
      }

      console.log();

      // Install each target
      const installedTargets: string[] = [];

      for (const target of selectedTargets) {
        const tool = tools.find((t) => t.key === target);
        const configDir = tool?.configDir || DEFAULT_CONFIG_DIRS[target];
        const targetDistDir = join(distDir, target);

        if (!existsSync(targetDistDir)) {
          console.log(
            `  ${red("✗")} ${target} — no build output found. Run ${bold("loaf build")} first.`,
          );
          continue;
        }

        const installer = INSTALLERS[target];
        if (!installer) {
          console.log(`  ${red("✗")} ${target} — no installer available`);
          continue;
        }

        try {
          installer(targetDistDir, configDir, options.upgrade ?? false);
          console.log(`  ${green("✓")} ${target} installed to ${gray(configDir)}`);
          installedTargets.push(target);
        } catch (error) {
          const msg = error instanceof Error ? error.message : String(error);
          console.log(`  ${red("✗")} ${target} — ${msg}`);
        }
      }

      console.log();

      // Install fenced sections to project files (only for successfully installed targets)
      const projectRoot = process.cwd();
      const fencedResults = installFencedSectionsForTargets(
        installedTargets,
        projectRoot,
        options.upgrade ?? false
      );

      // Report fenced section results
      let hasFencedOutput = false;
      for (const [target, result] of Object.entries(fencedResults)) {
        if (result.action === "error") {
          console.log(
            `  ${red("✗")} ${target} project file — ${result.error}`
          );
          hasFencedOutput = true;
        } else if (result.action === "created") {
          console.log(
            `  ${green("✓")} ${target} created project file with Loaf framework section`
          );
          hasFencedOutput = true;
        } else if (result.action === "appended") {
          console.log(
            `  ${green("✓")} ${target} added Loaf framework section to project file`
          );
          hasFencedOutput = true;
        } else if (result.action === "updated") {
          console.log(
            `  ${green("✓")} ${target} updated Loaf framework section in project file (v${result.version})`
          );
          hasFencedOutput = true;
        } else if (result.action === "skipped") {
          console.log(
            `  ${gray("○")} ${target} Loaf framework section already current (v${result.version})`
          );
          hasFencedOutput = true;
        }
      }

      if (hasFencedOutput) {
        console.log();
      }
    });
}
