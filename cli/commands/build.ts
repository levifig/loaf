import { Command } from "commander";
import { existsSync, readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { parse as parseYaml } from "yaml";

const __dirname = dirname(fileURLToPath(import.meta.url));

// ANSI color helpers
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;
const cyan = (s: string) => `\x1b[36m${s}\x1b[0m`;

interface TargetModule {
  build(ctx: BuildContext): Promise<void>;
}

interface BuildContext {
  config: Record<string, unknown>;
  targetConfig: Record<string, unknown>;
  targetsConfig: Record<string, unknown>;
  rootDir: string;
  srcDir: string;
  distDir: string;
  targetName: string;
}

function findRootDir(): string {
  // Walk up from __dirname to find package.json with name "loaf"
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
  throw new Error("Could not find loaf root directory (no package.json with name 'loaf')");
}

function loadYamlConfig(path: string): Record<string, unknown> {
  if (!existsSync(path)) return {};
  return parseYaml(readFileSync(path, "utf-8")) as Record<string, unknown>;
}

// Available target names — order determines build order
const TARGET_NAMES = ["claude-code", "opencode", "cursor", "codex", "gemini"];

async function loadTarget(targetName: string, rootDir: string): Promise<TargetModule> {
  // Import from the JS build system (still vanilla JS at this stage)
  const targetPath = join(rootDir, "build", "targets", `${targetName}.js`);
  if (!existsSync(targetPath)) {
    throw new Error(`Target module not found: ${targetPath}`);
  }
  return import(targetPath) as Promise<TargetModule>;
}

async function buildTarget(
  targetName: string,
  rootDir: string,
  srcDir: string,
  distDir: string,
  hooksConfig: Record<string, unknown>,
  targetsConfig: Record<string, unknown>,
): Promise<void> {
  const targetModule = await loadTarget(targetName, rootDir);

  // Claude Code outputs to repo root, others to dist/
  const outputDir =
    targetName === "claude-code" ? rootDir : join(distDir, targetName);

  const targetConfig =
    (targetsConfig as { targets?: Record<string, Record<string, unknown>> })
      .targets?.[targetName] || {};

  await targetModule.build({
    config: hooksConfig,
    targetConfig,
    targetsConfig,
    rootDir,
    srcDir,
    distDir: outputDir,
    targetName,
  });
}

export function registerBuildCommand(program: Command): void {
  program
    .command("build")
    .description("Build skill distributions for agent harnesses")
    .option("-t, --target <name>", "Build a specific target only")
    .action(async (options: { target?: string }) => {
      const startTime = Date.now();
      const rootDir = findRootDir();
      const contentDir = join(rootDir, "content");
      const configDir = join(rootDir, "config");
      const distDir = join(rootDir, "dist");

      console.log(`\n${bold("loaf build")}\n`);

      // Validate target if specified
      if (options.target && !TARGET_NAMES.includes(options.target)) {
        console.error(
          `${red("error:")} Unknown target ${bold(options.target)}\n` +
          `${gray("Valid targets:")} ${TARGET_NAMES.join(", ")}`
        );
        process.exit(1);
      }

      // Load config
      const hooksConfigPath = join(configDir, "hooks.yaml");
      if (!existsSync(hooksConfigPath)) {
        console.error(`${red("error:")} Hooks config not found: ${hooksConfigPath}`);
        process.exit(1);
      }

      const hooksConfig = loadYamlConfig(hooksConfigPath);
      const targetsConfig = loadYamlConfig(join(configDir, "targets.yaml"));

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
            targetsConfig,
          );
          const elapsed = ((Date.now() - targetStart) / 1000).toFixed(2);
          process.stdout.write(`\r  ${green("✓")} ${targetName} ${gray(`(${elapsed}s)`)}\n`);
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          process.stdout.write(`\r  ${red("✗")} ${targetName}\n`);
          console.error(`    ${red(message)}`);
          failed = true;
        }
      }

      const totalElapsed = ((Date.now() - startTime) / 1000).toFixed(2);
      console.log();

      if (failed) {
        console.error(`${red("Build failed")} ${gray(`(${totalElapsed}s)`)}`);
        process.exit(1);
      }

      console.log(`${green("Build complete")} ${gray(`(${totalElapsed}s)`)}`);
    });
}
