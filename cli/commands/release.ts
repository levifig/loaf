/**
 * loaf release command
 *
 * Orchestrates the full release workflow: analyze commits, generate changelog,
 * bump versions, build, tag, and create GitHub release draft.
 */

import { Command } from "commander";
import { execFileSync } from "child_process";
import {
  existsSync,
  readFileSync,
  writeFileSync,
  readdirSync,
  mkdtempSync,
  unlinkSync,
} from "fs";
import { join } from "path";
import { tmpdir } from "os";
import { createInterface } from "readline";

import { askYesNo } from "../lib/prompts.js";
import { getLastTag, getCommitsSince, suggestBump } from "../lib/release/commits.js";
import type { ParsedCommit } from "../lib/release/commits.js";
import {
  detectVersionFiles,
  bumpVersion,
  prepareVersionUpdates,
  parseSemVer,
} from "../lib/release/version.js";
import type { VersionFile, BumpType } from "../lib/release/version.js";
import {
  generateChangelogSection,
  insertIntoChangelog,
  createChangelog,
} from "../lib/release/changelog.js";
import {
  validateBumpType,
  validateBaseRef,
  normalizeSkipFlags,
} from "../lib/release/options.js";
import type { ReleaseOptions } from "../lib/release/options.js";

// ANSI color helpers
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;
const cyan = (s: string) => `\x1b[36m${s}\x1b[0m`;

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

async function askChoice(
  question: string,
  options: BumpType[],
  defaultChoice: BumpType,
): Promise<BumpType> {
  if (!process.stdin.isTTY) {
    return defaultChoice;
  }

  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    let resolved = false;
    rl.on("close", () => {
      if (!resolved) {
        resolved = true;
        resolve(defaultChoice);
      }
    });
    rl.question(question, (answer) => {
      resolved = true;
      rl.close();
      const num = parseInt(answer.trim(), 10);
      if (num >= 1 && num <= options.length) {
        resolve(options[num - 1]);
      } else {
        resolve(defaultChoice);
      }
    });
  });
}

function isGitRepo(cwd: string): boolean {
  try {
    execFileSync("git", ["rev-parse", "--is-inside-work-tree"], {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
    });
    return true;
  } catch {
    return false;
  }
}

function isGhAvailable(): boolean {
  try {
    execFileSync("which", ["gh"], {
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
    });
    return true;
  } catch {
    return false;
  }
}

interface IncompleteTask {
  filename: string;
  status: string;
}

function scanIncompleteTasks(cwd: string): IncompleteTask[] {
  const tasksDir = join(cwd, ".agents", "tasks");
  if (!existsSync(tasksDir)) return [];

  const incomplete: IncompleteTask[] = [];

  try {
    const files = readdirSync(tasksDir).filter((f) => f.endsWith(".md"));

    for (const file of files) {
      try {
        const content = readFileSync(join(tasksDir, file), "utf-8");
        const lines = content.split("\n").slice(0, 20);

        for (const line of lines) {
          const match = line.match(/^status:\s*(.+)/);
          if (match) {
            const status = match[1].trim();
            if (status !== "complete" && status !== "archived") {
              incomplete.push({ filename: file, status });
            }
            break;
          }
        }
      } catch {
        // Can't read file — skip
        continue;
      }
    }
  } catch {
    // Can't read directory — skip
  }

  return incomplete;
}

function getEditor(): string | null {
  return process.env.VISUAL || process.env.EDITOR || null;
}

// ─────────────────────────────────────────────────────────────────────────────
// Command
// ─────────────────────────────────────────────────────────────────────────────

export function registerReleaseCommand(program: Command): void {
  program
    .command("release")
    .description("Create a new release with changelog, version bump, and tag")
    .option("--dry-run", "Preview release without making changes")
    .option("--bump <type>", "Skip interactive bump choice (prerelease, release, major, minor, patch)")
    .option("--base <ref>", "Use commits since <ref> instead of last tag (e.g. main)")
    .option("--no-tag", "Skip git tag creation")
    .option("--no-gh", "Skip GitHub release draft")
    .action(async (options: ReleaseOptions) => {
      const cwd = process.cwd();

      console.log(`\n${bold("loaf release")}\n`);

      // ── Pre-flight checks ──────────────────────────────────────────────

      if (!isGitRepo(cwd)) {
        console.error(`  ${red("error:")} Not a git repository`);
        process.exit(1);
      }

      // ── Phase 1: Gather ────────────────────────────────────────────────

      process.stdout.write(`  ${cyan("Analyzing")}...\n\n`);

      let baseRef = options.base ?? getLastTag(cwd);

      // Validate --base ref exists before proceeding (may resolve to origin/<ref>)
      if (options.base) {
        try {
          baseRef = validateBaseRef(cwd, options.base);
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          console.error(`  ${red("error:")} ${message}`);
          process.exit(1);
        }
      }

      const commits = getCommitsSince(cwd, baseRef);

      if (options.base) {
        const baseRefLabel = baseRef === options.base
          ? `${bold(options.base)} (via --base flag)`
          : `${bold(options.base)} (resolved to ${bold(baseRef!)} via --base flag)`;
        console.log(`  Base ref: ${baseRefLabel}`);
      } else {
        console.log(`  Last tag: ${baseRef ? bold(baseRef) : gray("(none)")}`);
      }
      console.log(`  Commits since ${options.base ? "base" : "tag"}: ${bold(String(commits.length))}`);
      console.log();

      if (commits.length === 0) {
        console.log(`  ${gray("No unreleased changes found.")}\n`);
        process.exit(0);
      }

      // Print each commit
      for (const commit of commits) {
        if (commit.section === null) {
          console.log(`  ${gray(`${commit.raw} (${commit.hash})`)}  ${gray("[filtered]")}`);
        } else {
          console.log(`  ${green(`${commit.raw} (${commit.hash})`)}`);
        }
      }
      console.log();

      // Detect version files
      const versionFiles = detectVersionFiles(cwd);
      if (versionFiles.length === 0) {
        console.error(`  ${red("error:")} No version files found`);
        process.exit(1);
      }

      // Scan for incomplete tasks
      const incompleteTasks = scanIncompleteTasks(cwd);

      // ── Phase 2: Generate ──────────────────────────────────────────────

      const commitBump = suggestBump(commits);
      const currentVersion = versionFiles[0].currentVersion;
      const parsed = parseSemVer(currentVersion);
      const isPrerelease = parsed !== null && parsed.prerelease !== undefined;

      let bump: BumpType;
      let newVersion: string | null;

      // Validate --bump flag if provided
      if (options.bump) {
        try {
          bump = validateBumpType(options.bump);
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          console.error(`  ${red("error:")} ${message}`);
          process.exit(1);
        }
        newVersion = bumpVersion(currentVersion, bump);
        console.log(`  Bump type: ${bold(bump)} (via --bump flag)`);
        console.log();
      } else if (isPrerelease) {
        // On a pre-release version — default to bumping the prerelease counter
        // Show available options
        console.log(`  ${bold("Current pre-release:")} ${currentVersion}`);
        console.log();
        console.log(`  ${bold("Bump options:")}`);
        console.log(`    ${cyan("1.")} prerelease → ${bumpVersion(currentVersion, "prerelease")}`);
        console.log(`    ${cyan("2.")} release    → ${bumpVersion(currentVersion, "release")}`);
        console.log(`    ${cyan("3.")} ${commitBump.padEnd(10)} → ${bumpVersion(currentVersion, commitBump)} ${gray("(based on commits)")}`);
        console.log();

        // Ask user to choose (with non-interactive fallback defaulting to "prerelease")
        const choice = await askChoice(
          `  Bump type [1/2/3]: `,
          ["prerelease", "release", commitBump],
          "prerelease",
        );
        bump = choice;
        newVersion = bumpVersion(currentVersion, bump);
      } else {
        // Normal release — use commit-based suggestion
        bump = commitBump;
        newVersion = bumpVersion(currentVersion, bump);
      }

      if (!newVersion) {
        console.error(`  ${red("error:")} Could not compute new version from "${currentVersion}"`);
        process.exit(1);
      }

      const today = new Date().toISOString().slice(0, 10);
      let changelogSection = generateChangelogSection(newVersion, today, commits);

      // Editor workflow
      const editor = getEditor();
      if (editor && process.stdin.isTTY) {
        try {
          const tmpDir = mkdtempSync(join(tmpdir(), "loaf-release-"));
          const tmpFile = join(tmpDir, "CHANGELOG_SECTION.md");
          writeFileSync(tmpFile, changelogSection, "utf-8");

          execFileSync(editor, [tmpFile], { stdio: "inherit" });

          changelogSection = readFileSync(tmpFile, "utf-8");
          unlinkSync(tmpFile);

          console.log(`  ${green("Edited changelog accepted.")}`);
          console.log();
        } catch {
          // Editor failed — fall through to terminal preview
          console.log(`  ${yellow("Editor failed — using generated changelog.")}`);
          console.log();
        }
      } else {
        console.log(`  ${bold("Generated changelog:")}\n`);
        // Indent each line for display
        for (const line of changelogSection.split("\n")) {
          console.log(`  ${line}`);
        }
        console.log();
        console.log(`  ${gray("(Set $EDITOR to edit before confirming)")}`);
        console.log();
      }

      // ── Phase 3: Present ───────────────────────────────────────────────

      const tagName = `v${newVersion}`;
      const ghAvailable = isGhAvailable();

      // Version files summary
      console.log(`  ${bold("Version files:")}`);
      for (const file of versionFiles) {
        console.log(`    \u2022 ${file.relativePath} (${file.currentVersion} \u2192 ${newVersion})`);
      }
      console.log();

      // Incomplete tasks warning
      if (incompleteTasks.length > 0) {
        console.log(`  ${bold("Incomplete tasks:")} ${incompleteTasks.length}`);
        for (const task of incompleteTasks) {
          console.log(`    ${yellow("\u26A0")} ${task.filename} (status: ${task.status})`);
        }
        console.log();
      }

      // Bump summary
      const bumpReasons: Record<string, string> = {
        major: "breaking changes detected",
        minor: "new features detected",
        patch: "bug fixes only",
        prerelease: "development milestone",
        release: "stable release",
      };
      console.log(`  Suggested bump: ${bold(bump)} (${bumpReasons[bump]})`);
      console.log(`  New version: ${bold(newVersion)}`);
      console.log();

      // Determine which steps are enabled
      const { skipTag, skipGh } = normalizeSkipFlags(options);

      // Action list
      console.log(`  ${bold("Actions:")}`);
      let actionNum = 1;
      console.log(`    ${actionNum++}. Update version in ${versionFiles.length} file(s)`);
      console.log(`    ${actionNum++}. Update CHANGELOG.md`);
      console.log(`    ${actionNum++}. Run loaf build`);
      console.log(`    ${actionNum++}. Commit release artifacts`);
      if (skipTag) {
        console.log(`    ${gray(`${actionNum++}. Create git tag ${tagName} (--no-tag — skipped)`)}`);
      } else {
        console.log(`    ${actionNum++}. Create git tag ${tagName}`);
      }
      if (skipGh) {
        console.log(`    ${gray(`${actionNum++}. Create GitHub release draft (--no-gh — skipped)`)}`);
      } else if (ghAvailable) {
        console.log(`    ${actionNum++}. Create GitHub release draft (gh available)`);
      } else {
        console.log(`    ${gray(`${actionNum++}. Create GitHub release draft (gh not available — skipped)`)}`);
      }
      console.log();

      // Dry run exit
      if (options.dryRun) {
        console.log(`  ${cyan("--dry-run:")} No changes made.\n`);
        process.exit(0);
      }

      // Confirmation
      const confirmed = await askYesNo(`  Proceed with release ${bold(tagName)}? [y/N] `);
      if (!confirmed) {
        console.log(`\n  ${gray("Release cancelled.")}\n`);
        process.exit(0);
      }

      console.log();

      // ── Phase 4: Execute ───────────────────────────────────────────────

      console.log(`  ${bold("Executing:")}`);

      // Step 1: Write version files
      try {
        const updates = prepareVersionUpdates(versionFiles, newVersion);
        for (const [filePath, content] of updates) {
          writeFileSync(filePath, content, "utf-8");
          const relPath = versionFiles.find((f) => f.path === filePath)?.relativePath ?? filePath;
          const oldVer = versionFiles.find((f) => f.path === filePath)?.currentVersion ?? "?";
          console.log(`    ${green("\u2713")} Updated ${relPath} (${oldVer} \u2192 ${newVersion})`);
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`    ${red("\u2717")} Failed to update version files: ${message}`);
        process.exit(1);
      }

      // Step 2: Write changelog
      try {
        const changelogPath = join(cwd, "CHANGELOG.md");
        let changelogContent: string;

        if (existsSync(changelogPath)) {
          const existing = readFileSync(changelogPath, "utf-8");
          const inserted = insertIntoChangelog(existing, changelogSection);
          if (inserted) {
            changelogContent = inserted;
          } else {
            // No [Unreleased] marker — append after header
            changelogContent = existing + "\n" + changelogSection + "\n";
          }
        } else {
          changelogContent = createChangelog(changelogSection);
        }

        writeFileSync(changelogPath, changelogContent, "utf-8");
        console.log(`    ${green("\u2713")} Updated CHANGELOG.md`);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`    ${red("\u2717")} Failed to update CHANGELOG.md: ${message}`);
        process.exit(1);
      }

      // Step 3: Run loaf build
      try {
        execFileSync(process.execPath, [process.argv[1], "build"], {
          cwd,
          stdio: "inherit",
        });
        console.log(`    ${green("\u2713")} Built all targets`);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`    ${red("\u2717")} Build failed: ${message}`);
        process.exit(1);
      }

      // Step 4: Commit release artifacts
      try {
        execFileSync("git", ["add", "-A"], {
          cwd,
          stdio: ["ignore", "pipe", "ignore"],
        });
        execFileSync(
          "git",
          ["commit", "-m", `release: ${tagName}`],
          { cwd, stdio: ["ignore", "pipe", "ignore"] },
        );
        console.log(`    ${green("\u2713")} Committed release artifacts`);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`    ${red("\u2717")} Failed to commit release: ${message}`);
        process.exit(1);
      }

      // Step 5: Create git tag
      if (skipTag) {
        console.log(`    ${gray("-")} Git tag skipped (--no-tag)`);
      } else {
        try {
          execFileSync("git", ["tag", "-a", tagName, "-m", `Release ${newVersion}`], {
            cwd,
            stdio: ["ignore", "pipe", "ignore"],
          });
          console.log(`    ${green("\u2713")} Created tag ${tagName}`);
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          console.error(`    ${red("\u2717")} Failed to create tag: ${message}`);
          process.exit(1);
        }
      }

      // Step 6: Create GitHub release draft
      if (skipGh) {
        console.log(`    ${gray("-")} GitHub release skipped (--no-gh)`);
      } else if (ghAvailable) {
        try {
          execFileSync(
            "gh",
            [
              "release",
              "create",
              tagName,
              "--draft",
              "--title",
              `v${newVersion}`,
              "--notes",
              changelogSection,
            ],
            { cwd, stdio: "inherit" },
          );
          console.log(`    ${green("\u2713")} Created GitHub release draft`);
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          console.error(`    ${red("\u2717")} Failed to create GitHub release: ${message}`);
          process.exit(1);
        }
      } else {
        console.log(`    ${gray("-")} GitHub release skipped (gh not available)`);
      }

      console.log();
      console.log(`  ${green("\u2713")} Release ${bold(tagName)} complete\n`);
    });
}
