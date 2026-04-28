/**
 * loaf soul command
 *
 * Manage the configured agent soul — the orchestrator identity that
 * agent profiles read at spawn (SPEC-033).
 *
 * Subcommands:
 *   loaf soul list            List catalog souls with one-line descriptions
 *   loaf soul current         Print the active soul (defaults to "none")
 *   loaf soul show <name>     Print a catalog SOUL.md to stdout
 *   loaf soul use <name>      Copy a catalog SOUL.md to .agents/SOUL.md and
 *                             record `soul: <name>` in .agents/loaf.json.
 *                             Refuses to overwrite a diverged local file
 *                             without --force.
 *
 * Schema definition for `loaf.json` and `loaf install` integration are
 * separate (TASK-131); this command only manipulates SOUL.md and reads /
 * merges the `soul` field in loaf.json.
 */

import { Command } from "commander";
import { execFileSync } from "child_process";

import {
  checkDivergence,
  copySoulToProject,
  listSouls,
  localSoulPath,
  readActiveSoul,
  readSoul,
  writeActiveSoul,
} from "../lib/souls/index.js";

// ANSI color helpers
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;

/** Default soul when `loaf.json` has no `soul:` field (per SPEC-033). */
const DEFAULT_SOUL = "none";

/**
 * Locate the project root. Mirrors `findProjectRoot` in install.ts: prefer
 * git toplevel, fall back to cwd.
 */
function findProjectRoot(): string {
  try {
    return execFileSync("git", ["rev-parse", "--show-toplevel"], {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
  } catch {
    return process.cwd();
  }
}

function dieUnknownSoul(name: string): never {
  console.error(`${red("error:")} Unknown soul: ${bold(name)}`);
  const available = listSouls()
    .map((s) => s.name)
    .join(", ");
  if (available) {
    console.error(`${gray("Available:")} ${available}`);
  }
  process.exit(1);
}

export function registerSoulCommand(program: Command): void {
  const soul = program
    .command("soul")
    .description("Manage the configured agent soul (orchestrator identity)");

  // ─── loaf soul list ───────────────────────────────────────────────────
  soul
    .command("list")
    .description("List available souls in the catalog")
    .action(() => {
      const souls = listSouls();
      if (souls.length === 0) {
        console.log(gray("No souls found in catalog."));
        return;
      }
      for (const s of souls) {
        console.log(`${bold(s.name)} — ${s.description}`);
      }
    });

  // ─── loaf soul current ─────────────────────────────────────────────────
  soul
    .command("current")
    .description("Print the active soul name")
    .action(() => {
      const projectRoot = findProjectRoot();
      const active = readActiveSoul(projectRoot) ?? DEFAULT_SOUL;
      console.log(active);
    });

  // ─── loaf soul show <name> ────────────────────────────────────────────
  soul
    .command("show")
    .description("Print a catalog SOUL.md to stdout (no filesystem writes)")
    .argument("<name>", "Catalog soul name")
    .action((name: string) => {
      try {
        process.stdout.write(readSoul(name));
      } catch {
        dieUnknownSoul(name);
      }
    });

  // ─── loaf soul use <name> ─────────────────────────────────────────────
  soul
    .command("use")
    .description("Activate a catalog soul: copies SOUL.md and updates loaf.json")
    .argument("<name>", "Catalog soul name")
    .option("--force", "Overwrite a diverged local SOUL.md")
    .action((name: string, options: { force?: boolean }) => {
      // Validate the soul exists before touching anything.
      const available = listSouls();
      if (!available.some((s) => s.name === name)) {
        dieUnknownSoul(name);
      }

      const projectRoot = findProjectRoot();
      const localPath = localSoulPath(projectRoot);

      const divergence = checkDivergence(localPath);
      if (divergence.diverged && !options.force) {
        console.error(
          `${red("error:")} local SOUL.md diverges from catalog; use --force to override`,
        );
        process.exit(1);
      }

      copySoulToProject(name, projectRoot);
      writeActiveSoul(projectRoot, name);

      console.log(`${green("✓")} Activated soul: ${bold(name)}`);
      console.log(`  ${gray("→")} ${localPath}`);
    });
}
