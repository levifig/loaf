#!/usr/bin/env node

// cli/index.ts
import { Command } from "commander";
import { readFileSync as readFileSync2 } from "fs";
import { join as join2, dirname as dirname2 } from "path";
import { fileURLToPath as fileURLToPath2 } from "url";

// cli/commands/build.ts
import { existsSync, readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { parse as parseYaml } from "yaml";
var __dirname = dirname(fileURLToPath(import.meta.url));
var bold = (s) => `\x1B[1m${s}\x1B[0m`;
var green = (s) => `\x1B[32m${s}\x1B[0m`;
var red = (s) => `\x1B[31m${s}\x1B[0m`;
var gray = (s) => `\x1B[90m${s}\x1B[0m`;
var cyan = (s) => `\x1B[36m${s}\x1B[0m`;
function findRootDir() {
  let dir = __dirname;
  for (let i = 0; i < 10; i++) {
    const pkgPath = join(dir, "package.json");
    try {
      const pkg = JSON.parse(readFileSync(pkgPath, "utf-8"));
      if (pkg.name === "loaf") return dir;
    } catch {
    }
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error("Could not find loaf root directory (no package.json with name 'loaf')");
}
function loadYamlConfig(path) {
  if (!existsSync(path)) return {};
  return parseYaml(readFileSync(path, "utf-8"));
}
var TARGET_NAMES = ["claude-code", "opencode", "cursor", "codex", "gemini"];
async function loadTarget(targetName, rootDir) {
  const targetPath = join(rootDir, "build", "targets", `${targetName}.js`);
  if (!existsSync(targetPath)) {
    throw new Error(`Target module not found: ${targetPath}`);
  }
  return import(targetPath);
}
async function buildTarget(targetName, rootDir, srcDir, distDir, hooksConfig, targetsConfig) {
  const targetModule = await loadTarget(targetName, rootDir);
  const outputDir = targetName === "claude-code" ? rootDir : join(distDir, targetName);
  const targetConfig = targetsConfig.targets?.[targetName] || {};
  await targetModule.build({
    config: hooksConfig,
    targetConfig,
    targetsConfig,
    rootDir,
    srcDir,
    distDir: outputDir,
    targetName
  });
}
function registerBuildCommand(program2) {
  program2.command("build").description("Build skill distributions for agent harnesses").option("-t, --target <name>", "Build a specific target only").action(async (options) => {
    const startTime = Date.now();
    const rootDir = findRootDir();
    const contentDir = join(rootDir, "content");
    const configDir = join(rootDir, "config");
    const distDir = join(rootDir, "dist");
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

// cli/index.ts
var __dirname2 = dirname2(fileURLToPath2(import.meta.url));
function getVersion() {
  for (const candidate of [
    join2(__dirname2, "..", "package.json"),
    join2(__dirname2, "..", "..", "package.json")
  ]) {
    try {
      const pkg = JSON.parse(readFileSync2(candidate, "utf-8"));
      if (pkg.name === "loaf") return pkg.version;
    } catch {
      continue;
    }
  }
  return "0.0.0";
}
var program = new Command();
program.name("loaf").description("Loaf \u2014 Levi's Opinionated Agentic Framework").version(getVersion(), "-v, --version");
registerBuildCommand(program);
if (process.argv.length <= 2) {
  program.outputHelp();
  process.exit(0);
}
program.parse();
//# sourceMappingURL=index.js.map