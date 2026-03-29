/**
 * loaf cleanup command
 *
 * Scans .agents/ directories and recommends cleanup actions based on
 * the existing cleanup skill's rules. Supports interactive mode,
 * dry-run, artifact-type filters, and non-TTY pipe-safe output.
 */

import { Command } from "commander";
import { existsSync, mkdirSync, readFileSync, renameSync, rmSync, writeFileSync } from "fs";
import { join, dirname } from "path";
import matter from "gray-matter";

import { isTTY, askYesNo } from "../lib/prompts.js";
import { findAgentsDir, getOrBuildIndex } from "../lib/tasks/resolve.js";
import { archiveTasks, archiveSpecs, saveIndex } from "../lib/tasks/migrate.js";
import { scanArtifacts } from "../lib/cleanup/scanner.js";
import type { ArtifactType, CleanupRecommendation } from "../lib/cleanup/types.js";

// ANSI color helpers
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;
const cyan = (s: string) => `\x1b[36m${s}\x1b[0m`;

// Display names for artifact types
const TYPE_LABELS: Record<ArtifactType, string> = {
  session: "SESSIONS",
  task: "TASKS",
  spec: "SPECS",
  plan: "PLANS",
  draft: "DRAFTS",
  council: "COUNCILS",
  report: "REPORTS",
};

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

/** Format a frontmatter preview (first few fields) for delete confirmation */
function formatPreview(fm: Record<string, unknown> | undefined): string {
  if (!fm) return gray("  (no frontmatter)");
  const keys = Object.keys(fm).slice(0, 3);
  return keys
    .map((k) => {
      const v = fm[k];
      const display = typeof v === "string" ? v : JSON.stringify(v);
      return gray(`  ${k}: ${display}`);
    })
    .join("\n");
}

/**
 * Archive a generic artifact (session, council, report) by moving to archive/.
 * Writes archive metadata into the correct nested block matching the repo's
 * frontmatter conventions: session.*, council.*, report.* respectively.
 */
function archiveGenericArtifact(filePath: string, artifactType: ArtifactType): void {
  const dir = dirname(filePath);
  const archiveDir = join(dir, "archive");
  const filename = filePath.split("/").pop()!;
  const destPath = join(archiveDir, filename);

  mkdirSync(archiveDir, { recursive: true });

  const now = new Date().toISOString();
  const raw = readFileSync(filePath, "utf-8");
  const { data, content } = matter(raw);

  // Write archive metadata into the nested block that matches the artifact type
  const blockKey = artifactType === "session" ? "session"
    : artifactType === "council" ? "council"
    : artifactType === "report" ? "report"
    : null;

  if (blockKey && data[blockKey] && typeof data[blockKey] === "object") {
    const block = data[blockKey] as Record<string, unknown>;
    block.status = "archived";
    block.archived_at = now;
    block.archived_by = "loaf cleanup";
  } else {
    // Fallback for files without a nested block
    data.archived_at = now;
    data.archived_by = "loaf cleanup";
  }

  const updated = matter.stringify(content, data);
  writeFileSync(filePath, updated, "utf-8");

  renameSync(filePath, destPath);
}

// ─────────────────────────────────────────────────────────────────────────────
// Command
// ─────────────────────────────────────────────────────────────────────────────

export function registerCleanupCommand(program: Command): void {
  program
    .command("cleanup")
    .description("Scan .agents/ artifacts and recommend cleanup actions")
    .option("--dry-run", "Show recommendations without prompting for actions")
    .option("--sessions", "Only review sessions")
    .option("--specs", "Only review specs")
    .option("--plans", "Only review plans")
    .option("--drafts", "Only review drafts")
    .action(async (options: {
      dryRun?: boolean;
      sessions?: boolean;
      specs?: boolean;
      plans?: boolean;
      drafts?: boolean;
    }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} No .agents/ directory found`);
        process.exit(1);
      }

      console.log(`\n${bold("loaf cleanup")}\n`);

      // Build filter from flags
      const filter: ArtifactType[] | undefined = (() => {
        const types: ArtifactType[] = [];
        if (options.sessions) types.push("session");
        if (options.specs) types.push("spec");
        if (options.plans) types.push("plan");
        if (options.drafts) types.push("draft");
        return types.length > 0 ? types : undefined;
      })();

      // Scan
      const result = scanArtifacts({ agentsDir, filter });

      // Print warnings
      for (const warning of result.warnings) {
        console.log(`  ${yellow("warn:")} ${warning}`);
      }
      if (result.warnings.length > 0) console.log();

      // Print summary table
      for (const typeSummary of result.summary) {
        const label = TYPE_LABELS[typeSummary.type];
        console.log(`  ${bold(label)} ${gray(`(${typeSummary.total} total)`)}`);

        const actionableRecs = result.recommendations.filter(
          (r) => r.type === typeSummary.type && r.action !== "skip",
        );

        if (typeSummary.archive > 0) {
          const items = actionableRecs
            .filter((r) => r.action === "archive")
            .map((r) => r.filename);
          console.log(`    ${green("Ready to archive:")}  ${typeSummary.archive}  ${gray(items.join(", "))}`);
        }
        if (typeSummary.delete > 0) {
          const items = actionableRecs
            .filter((r) => r.action === "delete")
            .map((r) => r.filename);
          console.log(`    ${red("Ready to delete:")}   ${typeSummary.delete}  ${gray(items.join(", "))}`);
        }
        if (typeSummary.flag > 0) {
          const items = actionableRecs
            .filter((r) => r.action === "flag")
            .map((r) => r.filename);
          console.log(`    ${yellow("Needs review:")}      ${typeSummary.flag}  ${gray(items.join(", "))}`);
        }
        if (typeSummary.skip > 0) {
          console.log(`    ${gray(`Active/current:     ${typeSummary.skip}`)}`);
        }

        console.log();
      }

      // Count actionable items
      const actionable = result.recommendations.filter((r) => r.action !== "skip");

      if (actionable.length === 0) {
        console.log(`  ${green("✓")} Nothing to clean up.\n`);
        return;
      }

      // Dry-run or non-TTY: stop here
      if (options.dryRun || !isTTY()) {
        const mode = options.dryRun ? "--dry-run" : "non-TTY";
        console.log(`  ${cyan(`(${mode})`)} ${actionable.length} actionable item(s). Run interactively to take action.\n`);
        return;
      }

      // ── Interactive mode ────────────────────────────────────────────────

      console.log(`  ${bold("Actions")} — ${actionable.length} item(s)\n`);

      const index = getOrBuildIndex(agentsDir);
      const tasksToArchive: string[] = [];
      const specsToArchive: string[] = [];
      let actionsPerformed = 0;

      for (const rec of actionable) {
        // Header for each item
        console.log(`  ${cyan("→")} ${bold(rec.filename)}`);
        console.log(`    ${rec.reason}`);
        if (rec.hint) console.log(`    ${yellow("hint:")} ${rec.hint}`);

        if (rec.action === "archive") {
          if (rec.type === "task") {
            // Extract task ID from filename
            const match = rec.filename.match(/^(TASK-\d+)/);
            if (match) {
              const confirmed = await askYesNo(`    Archive ${match[1]}? [y/N] `);
              if (confirmed) {
                tasksToArchive.push(match[1]);
                actionsPerformed++;
              }
            }
          } else if (rec.type === "spec") {
            const match = rec.filename.match(/^(SPEC-\d+)/);
            if (match) {
              const confirmed = await askYesNo(`    Archive ${match[1]}? [y/N] `);
              if (confirmed) {
                specsToArchive.push(match[1]);
                actionsPerformed++;
              }
            }
          } else {
            // Generic archive (session, council, report)
            const confirmed = await askYesNo(`    Archive? [y/N] `);
            if (confirmed && existsSync(rec.path)) {
              archiveGenericArtifact(rec.path, rec.type);
              console.log(`    ${green("✓")} Archived`);
              actionsPerformed++;
            }
          }
        } else if (rec.action === "delete") {
          // Show preview before delete
          console.log(formatPreview(rec.frontmatter));
          const confirmed = await askYesNo(`    ${red("Delete")}? [y/N] `);
          if (confirmed && existsSync(rec.path)) {
            rmSync(rec.path);
            console.log(`    ${green("✓")} Deleted`);
            actionsPerformed++;
          }
        } else if (rec.action === "flag") {
          console.log(`    ${gray("(flagged for review — no automatic action)")}`);
        }

        console.log();
      }

      // Batch archive tasks and specs via existing helpers
      if (tasksToArchive.length > 0) {
        const results = archiveTasks(agentsDir, index, tasksToArchive);
        for (const r of results) {
          if (r.status === "archived") {
            console.log(`  ${green("✓")} Archived ${r.id}`);
          } else {
            console.log(`  ${yellow("⚠")} Skipped ${r.id}: ${r.reason}`);
          }
        }
      }

      if (specsToArchive.length > 0) {
        const results = archiveSpecs(agentsDir, index, specsToArchive);
        for (const r of results) {
          if (r.status === "archived") {
            console.log(`  ${green("✓")} Archived ${r.id}`);
          } else {
            console.log(`  ${yellow("⚠")} Skipped ${r.id}: ${r.reason}`);
          }
        }
      }

      // Save index if we archived tasks/specs
      if (tasksToArchive.length > 0 || specsToArchive.length > 0) {
        saveIndex(join(agentsDir, "TASKS.json"), index);
      }

      console.log(`  ${green("✓")} Cleanup complete — ${actionsPerformed} action(s) performed.\n`);
    });
}
