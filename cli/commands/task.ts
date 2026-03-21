/**
 * loaf task command
 *
 * Subcommands for listing and summarizing project tasks from TASKS.json.
 */

import { Command } from "commander";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { join } from "path";
import matter from "gray-matter";

import { findAgentsDir, getOrBuildIndex } from "../lib/tasks/resolve.js";
import { buildIndexFromFiles, saveIndex, syncFrontmatterFromIndex, findOrphans } from "../lib/tasks/migrate.js";
import type { TaskIndex, TaskEntry, TaskStatus, TaskPriority, SpecStatus } from "../lib/tasks/types.js";
import { TASK_STATUSES, TASK_PRIORITIES } from "../lib/tasks/types.js";
import { generateSlug } from "../lib/tasks/slug.js";
import {
  STATUS_DISPLAY_ORDER,
  STATUS_LABELS,
  getDisplayStatuses,
  sortTasks,
  countActiveTasks,
  groupTasksByStatus,
} from "../lib/tasks/task-display.js";

// ANSI color helpers (matching project conventions)
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;
const cyan = (s: string) => `\x1b[36m${s}\x1b[0m`;

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────


/** Color functions for status headers */
const STATUS_COLORS: Record<TaskStatus, (s: string) => string> = {
  in_progress: yellow,
  blocked: red,
  todo: cyan,
  review: gray,
  done: green,
};

/** Color functions for priority labels */
const PRIORITY_COLORS: Record<TaskPriority, (s: string) => string> = {
  P0: red,
  P1: yellow,
  P2: cyan,
  P3: gray,
};


/**
 * Count unique spec IDs referenced by tasks.
 */
function countSpecs(index: TaskIndex): { total: number; byStatus: Record<SpecStatus, number> } {
  const byStatus: Record<SpecStatus, number> = {
    drafting: 0,
    approved: 0,
    implementing: 0,
    complete: 0,
  };

  for (const spec of Object.values(index.specs)) {
    byStatus[spec.status]++;
  }

  return {
    total: Object.keys(index.specs).length,
    byStatus,
  };
}


// ─────────────────────────────────────────────────────────────────────────────
// Command
// ─────────────────────────────────────────────────────────────────────────────

export function registerTaskCommand(program: Command): void {
  const task = program
    .command("task")
    .description("Manage project tasks");

  // ── loaf task list ──────────────────────────────────────────────────────

  task
    .command("list")
    .description("Show task board grouped by status")
    .option("--json", "Output raw JSON")
    .option("--active", "Hide completed tasks")
    .action(async (options: { json?: boolean; active?: boolean }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const index = getOrBuildIndex(agentsDir);

      // --json: output as JSON and exit
      if (options.json) {
        if (options.active) {
          // Filter out done tasks when --active is set
          const filtered = { ...index, tasks: {} as Record<string, TaskEntry> };
          for (const [id, entry] of Object.entries(index.tasks)) {
            if (entry.status !== "done") {
              filtered.tasks[id] = entry;
            }
          }
          process.stdout.write(JSON.stringify(filtered, null, 2) + "\n");
        } else {
          const indexPath = join(agentsDir, "TASKS.json");
          if (existsSync(indexPath)) {
            process.stdout.write(readFileSync(indexPath, "utf-8"));
          } else {
            process.stdout.write(JSON.stringify(index, null, 2) + "\n");
          }
        }
        return;
      }

      const taskEntries = Object.entries(index.tasks);

      if (taskEntries.length === 0) {
        console.log(`\n  ${gray("No tasks found.")}\n`);
        return;
      }

      console.log(`\n  ${bold("loaf task list")}\n`);

      // Group tasks by status
      const grouped = groupTasksByStatus(taskEntries);

      // Collect unique spec IDs for the footer
      const specIds = new Set<string>();

      // Filter statuses when --active is set
      const displayStatuses = getDisplayStatuses(!!options.active);

      // Display each status group
      for (const status of displayStatuses) {
        const tasks = sortTasks(grouped[status]);
        const colorFn = STATUS_COLORS[status];
        const label = STATUS_LABELS[status];

        console.log(`  ${bold(colorFn(`${label} (${tasks.length})`))}`)

        if (tasks.length === 0) {
          // Empty group — just the header
        } else {
          for (const [id, entry] of tasks) {
            const priorityColor = PRIORITY_COLORS[entry.priority] || gray;
            const specRef = entry.spec ? gray(entry.spec) : "";

            if (entry.spec) specIds.add(entry.spec);

            // Pad columns for alignment
            const idCol = bold(id.padEnd(10));
            const prioCol = priorityColor(entry.priority.padEnd(4));
            const titleCol = entry.title;

            console.log(`    ${idCol}${prioCol}${titleCol}  ${specRef}`);
          }
        }

        console.log();
      }

      // Footer
      if (options.active) {
        const activeCount = countActiveTasks(taskEntries);
        console.log(`  Total: ${bold(String(activeCount))} active tasks across ${bold(String(specIds.size))} specs\n`);
      } else {
        const totalTasks = taskEntries.length;
        const totalSpecs = specIds.size;
        console.log(`  Total: ${bold(String(totalTasks))} tasks across ${bold(String(totalSpecs))} specs\n`);
      }
    });

  // ── loaf task show ─────────────────────────────────────────────────────

  task
    .command("show")
    .description("Display a single task's details")
    .argument("<id>", "Task ID (e.g., TASK-019)")
    .option("--json", "Output task entry as JSON")
    .action(async (id: string, options: { json?: boolean }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const index = getOrBuildIndex(agentsDir);

      const entry = index.tasks[id];
      if (!entry) {
        console.error(`  ${red("error:")} ${id} not found in index`);
        process.exit(1);
      }

      // --json: dump task entry as JSON and exit
      if (options.json) {
        process.stdout.write(JSON.stringify({ id, ...entry }, null, 2) + "\n");
        return;
      }

      // ── Metadata header ──────────────────────────────────────────────────

      console.log(`\n  ${bold("loaf task show")} ${id}\n`);

      const priorityColor = PRIORITY_COLORS[entry.priority] || gray;
      const statusColor = STATUS_COLORS[entry.status] || gray;

      console.log(`  ${bold(`${id}: ${entry.title}`)}`);
      console.log();

      const metaParts: string[] = [
        `Status: ${statusColor(entry.status)}`,
        `Priority: ${priorityColor(entry.priority)}`,
      ];
      if (entry.spec) metaParts.push(`Spec: ${entry.spec}`);
      console.log(`  ${metaParts.join(gray(" \u00b7 "))}`);

      const dateParts: string[] = [];
      if (entry.created) dateParts.push(`Created: ${entry.created.slice(0, 10)}`);
      if (entry.updated) dateParts.push(`Updated: ${entry.updated.slice(0, 10)}`);
      if (dateParts.length > 0) {
        console.log(`  ${dateParts.join(gray(" \u00b7 "))}`);
      }

      if (entry.depends_on.length > 0) {
        console.log(`  Depends on: ${entry.depends_on.join(", ")}`);
      }

      console.log(`  File: .agents/tasks/${entry.file}`);

      // ── Body content ─────────────────────────────────────────────────────

      const filePath = join(agentsDir, "tasks", entry.file);
      if (!existsSync(filePath)) {
        console.log();
        console.log(`  ${gray("(no detail file)")}`);
        console.log();
        return;
      }

      try {
        const raw = readFileSync(filePath, "utf-8");
        const { content: body } = matter(raw);

        // Strip leading/trailing blank lines, preserve internal formatting
        const trimmedBody = body.replace(/^\n+/, "").replace(/\n+$/, "");

        if (trimmedBody.length > 0) {
          console.log();
          console.log(`  ${"─".repeat(60)}`);
          console.log();

          // Indent each line by 2 spaces for consistent display
          const lines = trimmedBody.split("\n");
          for (const line of lines) {
            console.log(`  ${line}`);
          }
        }

        console.log();
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`  ${yellow("warn:")} Failed to read ${entry.file}: ${message}`);
        console.log();
      }
    });

  // ── loaf task status ────────────────────────────────────────────────────

  task
    .command("status")
    .description("Show task summary counts")
    .action(async () => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const index = getOrBuildIndex(agentsDir);

      console.log(`\n  ${bold("loaf task status")}\n`);

      // Task counts by status
      const taskCounts: Record<TaskStatus, number> = {
        in_progress: 0,
        blocked: 0,
        todo: 0,
        review: 0,
        done: 0,
      };

      for (const entry of Object.values(index.tasks)) {
        if (taskCounts[entry.status] !== undefined) {
          taskCounts[entry.status]++;
        }
      }

      const totalTasks = Object.keys(index.tasks).length;

      const taskParts = STATUS_DISPLAY_ORDER.map((status) => {
        const count = taskCounts[status];
        const colorFn = STATUS_COLORS[status];
        return `${colorFn(String(count))} ${status}`;
      });

      console.log(`  Tasks:  ${taskParts.join(gray(" \u00b7 "))}  ${gray(`(${totalTasks} total)`)}`);

      // Spec counts by status
      const specInfo = countSpecs(index);

      const specStatusOrder: SpecStatus[] = ["drafting", "approved", "implementing", "complete"];
      const specStatusColors: Record<SpecStatus, (s: string) => string> = {
        drafting: yellow,
        approved: cyan,
        implementing: yellow,
        complete: green,
      };

      const specParts = specStatusOrder.map((status) => {
        const count = specInfo.byStatus[status];
        const colorFn = specStatusColors[status];
        return `${colorFn(String(count))} ${status}`;
      });

      console.log(`  Specs:  ${specParts.join(gray(" \u00b7 "))}  ${gray(`(${specInfo.total} total)`)}`);
      console.log();
    });

  // ── loaf task create ────────────────────────────────────────────────────

  task
    .command("create")
    .description("Create a new task")
    .requiredOption("--title <title>", "Task title")
    .option("--spec <id>", "Associated spec ID (e.g., SPEC-010)")
    .option("--priority <level>", "Priority level (P0/P1/P2/P3)", "P2")
    .option("--depends-on <ids>", "Comma-separated task IDs")
    .action(async (options: {
      title: string;
      spec?: string;
      priority: string;
      dependsOn?: string;
    }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const index = getOrBuildIndex(agentsDir);
      const indexPath = join(agentsDir, "TASKS.json");

      // ── Validate priority ───────────────────────────────────────────────

      const priority = options.priority as TaskPriority;
      if (!TASK_PRIORITIES.includes(priority)) {
        console.error(`  ${red("error:")} Invalid priority "${options.priority}". Must be one of: ${TASK_PRIORITIES.join(", ")}`);
        process.exit(1);
      }

      // ── Validate spec exists ────────────────────────────────────────────

      const spec = options.spec || null;
      if (spec && !index.specs[spec]) {
        console.error(`  ${red("error:")} Spec "${spec}" not found in index`);
        process.exit(1);
      }

      // ── Validate depends-on IDs exist ───────────────────────────────────

      const dependsOn: string[] = [];
      if (options.dependsOn) {
        for (const depId of options.dependsOn.split(",").map((s) => s.trim())) {
          if (!index.tasks[depId]) {
            console.error(`  ${red("error:")} Dependency "${depId}" not found in index`);
            process.exit(1);
          }
          dependsOn.push(depId);
        }
      }

      // ── Generate ID and slug ────────────────────────────────────────────

      const nextId = index.next_id;
      const taskId = `TASK-${String(nextId).padStart(3, "0")}`;
      const slug = generateSlug(options.title);
      const now = new Date().toISOString();

      // ── Create TaskEntry ────────────────────────────────────────────────

      const fileName = `${taskId}-${slug}.md`;
      const entry: TaskEntry = {
        title: options.title,
        slug,
        spec,
        status: "todo",
        priority,
        depends_on: dependsOn,
        files: [],
        verify: null,
        done: null,
        session: null,
        created: now,
        updated: now,
        completed_at: null,
        file: fileName,
      };

      // ── Update index ────────────────────────────────────────────────────

      index.tasks[taskId] = entry;
      index.next_id = nextId + 1;
      saveIndex(indexPath, index);

      // ── Create .md file ─────────────────────────────────────────────────

      const tasksDir = join(agentsDir, "tasks");
      if (!existsSync(tasksDir)) {
        mkdirSync(tasksDir, { recursive: true });
      }

      const frontmatterData: Record<string, unknown> = {
        id: taskId,
        title: options.title,
        status: "todo",
        priority,
        created: now,
        updated: now,
      };
      if (spec) frontmatterData.spec = spec;
      if (dependsOn.length > 0) frontmatterData.depends_on = dependsOn;

      const body = `
# ${taskId}: ${options.title}

## Description

<!-- Describe the task here -->

## Acceptance Criteria

- [ ]

## Verification

\`\`\`bash
# Add verification command
\`\`\`
`;

      const mdContent = matter.stringify(body, frontmatterData);
      writeFileSync(join(tasksDir, fileName), mdContent, "utf-8");

      // ── Print confirmation ──────────────────────────────────────────────

      console.log(`\n  ${bold("loaf task create")}\n`);
      console.log(`  ${green("\u2713")} Created ${bold(taskId)}: ${options.title}`);
      console.log(`    File: .agents/tasks/${fileName}`);

      const details: string[] = [];
      if (spec) details.push(`Spec: ${spec}`);
      details.push(`Priority: ${priority}`);
      if (dependsOn.length > 0) details.push(`Depends on: ${dependsOn.join(", ")}`);
      console.log(`    ${details.join(gray(" \u00b7 "))}`);
      console.log();
    });

  // ── loaf task update ────────────────────────────────────────────────────

  task
    .command("update")
    .description("Update a task's metadata")
    .argument("<id>", "Task ID to update (e.g., TASK-031)")
    .option("--status <status>", "New status: todo, in_progress, blocked, review, done")
    .option("--priority <level>", "New priority: P0, P1, P2, P3")
    .option("--depends-on <ids>", "Replace depends_on (comma-separated task IDs)")
    .option("--session <file>", "Set or clear session reference (use \"none\" to clear)")
    .option("--spec <id>", "Set or change associated spec")
    .action(async (id: string, options: {
      status?: string;
      priority?: string;
      dependsOn?: string;
      session?: string;
      spec?: string;
    }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      // Validate that at least one flag was provided
      if (
        options.status === undefined &&
        options.priority === undefined &&
        options.dependsOn === undefined &&
        options.session === undefined &&
        options.spec === undefined
      ) {
        console.error(`  ${red("error:")} No updates specified. Use --status, --priority, --depends-on, --session, or --spec`);
        process.exit(1);
      }

      const index = getOrBuildIndex(agentsDir);

      // Validate task ID exists
      const entry = index.tasks[id];
      if (!entry) {
        console.error(`  ${red("error:")} ${id} not found in index`);
        process.exit(1);
      }

      // Track changes for confirmation output
      const changes: Array<{ field: string; from: string; to: string }> = [];

      // ── Validate and apply --status ───────────────────────────────────
      if (options.status !== undefined) {
        if (!TASK_STATUSES.includes(options.status as TaskStatus)) {
          console.error(`  ${red("error:")} Invalid status "${options.status}". Valid: ${TASK_STATUSES.join(", ")}`);
          process.exit(1);
        }

        const oldStatus = entry.status;
        const newStatus = options.status as TaskStatus;

        // Handle completed_at transitions
        if (newStatus === "done" && oldStatus !== "done") {
          entry.completed_at = new Date().toISOString();
        } else if (newStatus !== "done" && oldStatus === "done") {
          entry.completed_at = null;
        }

        entry.status = newStatus;
        changes.push({ field: "Status", from: oldStatus, to: newStatus });
      }

      // ── Validate and apply --priority ─────────────────────────────────
      if (options.priority !== undefined) {
        if (!TASK_PRIORITIES.includes(options.priority as TaskPriority)) {
          console.error(`  ${red("error:")} Invalid priority "${options.priority}". Valid: ${TASK_PRIORITIES.join(", ")}`);
          process.exit(1);
        }

        const oldPriority = entry.priority;
        const newPriority = options.priority as TaskPriority;
        entry.priority = newPriority;
        changes.push({ field: "Priority", from: oldPriority, to: newPriority });
      }

      // ── Validate and apply --depends-on ───────────────────────────────
      if (options.dependsOn !== undefined) {
        const newDeps = options.dependsOn
          .split(",")
          .map((s) => s.trim())
          .filter((s) => s.length > 0);

        for (const depId of newDeps) {
          if (!index.tasks[depId]) {
            console.error(`  ${red("error:")} Unknown task ID "${depId}" in --depends-on`);
            process.exit(1);
          }
        }

        const oldDeps = entry.depends_on.length > 0 ? entry.depends_on.join(", ") : "(none)";
        entry.depends_on = newDeps;
        changes.push({ field: "Depends on", from: oldDeps, to: newDeps.length > 0 ? newDeps.join(", ") : "(none)" });
      }

      // ── Apply --session ───────────────────────────────────────────────
      if (options.session !== undefined) {
        const oldSession = entry.session || "(none)";
        const newSession = options.session === "none" ? null : options.session;
        entry.session = newSession;
        changes.push({ field: "Session", from: oldSession, to: newSession || "(none)" });
      }

      // ── Apply --spec ──────────────────────────────────────────────────
      if (options.spec !== undefined) {
        if (options.spec !== "none" && !index.specs[options.spec]) {
          console.error(`  ${red("error:")} Unknown spec "${options.spec}". Use \`loaf spec list\` to see valid IDs.`);
          process.exit(1);
        }
        const oldSpec = entry.spec || "(none)";
        const newSpec = options.spec === "none" ? null : options.spec;
        entry.spec = newSpec;
        changes.push({ field: "Spec", from: oldSpec, to: newSpec || "(none)" });
      }

      // ── Update timestamp and persist ──────────────────────────────────
      entry.updated = new Date().toISOString();

      const indexPath = join(agentsDir, "TASKS.json");
      saveIndex(indexPath, index);
      syncFrontmatterFromIndex(agentsDir, index);

      // ── Print confirmation ────────────────────────────────────────────
      console.log(`\n  ${bold("loaf task update")}\n`);
      console.log(`  ${green("\u2713")} Updated ${bold(id)}: ${entry.title}`);

      for (const change of changes) {
        if (change.from === change.to) {
          console.log(`    ${change.field}: ${change.from} ${gray("(unchanged)")}`);
        } else {
          console.log(`    ${change.field}: ${change.from} \u2192 ${change.to}`);
        }
      }

      // Show unchanged fields that weren't provided
      const providedFields = new Set(changes.map((c) => c.field));

      if (!providedFields.has("Status")) {
        console.log(`    Status: ${entry.status} ${gray("(unchanged)")}`);
      }
      if (!providedFields.has("Priority")) {
        console.log(`    Priority: ${entry.priority} ${gray("(unchanged)")}`);
      }

      console.log();
    });

  // ── loaf task sync ──────────────────────────────────────────────────────

  task
    .command("sync")
    .description("Rebuild TASKS.json from .md files, or import orphans")
    .option("--import", "Import orphan .md files not in the index")
    .action(async (options: { import?: boolean }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const indexPath = join(agentsDir, "TASKS.json");

      console.log(`\n  ${bold("loaf task sync")}\n`);

      if (options.import) {
        // ── Import mode: find orphans and merge ──────────────────────────

        const index = getOrBuildIndex(agentsDir);
        const orphans = findOrphans(agentsDir, index);

        const totalOrphans = orphans.tasks.length + orphans.specs.length;

        if (totalOrphans === 0) {
          console.log(`  No orphan files found.`);
          console.log();
          return;
        }

        console.log(`  Found ${totalOrphans} orphan file(s):`);

        for (const orphan of orphans.tasks) {
          console.log(`    ${green("+")} ${orphan.entry.file}`);
        }
        for (const orphan of orphans.specs) {
          console.log(`    ${green("+")} ${orphan.entry.file}`);
        }

        // Merge orphan tasks into the index
        let maxTaskNum = index.next_id - 1;
        for (const orphan of orphans.tasks) {
          index.tasks[orphan.id] = orphan.entry;
          const num = extractOrphanNumber(orphan.id);
          if (num > maxTaskNum) maxTaskNum = num;
        }

        // Merge orphan specs into the index
        for (const orphan of orphans.specs) {
          index.specs[orphan.id] = orphan.entry;
        }

        // Update next_id if any imported task has a higher number
        if (maxTaskNum >= index.next_id) {
          index.next_id = maxTaskNum + 1;
        }

        saveIndex(indexPath, index);

        const importedParts: string[] = [];
        if (orphans.tasks.length > 0) importedParts.push(`${orphans.tasks.length} task(s)`);
        if (orphans.specs.length > 0) importedParts.push(`${orphans.specs.length} spec(s)`);

        console.log();
        console.log(`  ${green("\u2713")} Imported ${importedParts.join(" and ")} into TASKS.json`);
        console.log();
      } else {
        // ── Full rebuild mode ────────────────────────────────────────────

        const index = buildIndexFromFiles(agentsDir);
        saveIndex(indexPath, index);

        // Count tasks by status
        const statusCounts: Record<string, number> = {};
        for (const entry of Object.values(index.tasks)) {
          statusCounts[entry.status] = (statusCounts[entry.status] || 0) + 1;
        }

        const totalTasks = Object.keys(index.tasks).length;
        const totalSpecs = Object.keys(index.specs).length;

        const countParts = STATUS_DISPLAY_ORDER.map((s) => {
          const count = statusCounts[s] || 0;
          return `${count} ${s}`;
        });

        console.log(`  ${green("\u2713")} Rebuilt TASKS.json from .md files`);
        console.log(`    Tasks: ${totalTasks} (${countParts.join(", ")})`);
        console.log(`    Specs: ${totalSpecs}`);
        console.log();
      }
    });
}

/**
 * Extract the numeric portion of a task/spec ID for next_id calculation.
 * "TASK-019" -> 19, "SPEC-010" -> 10
 */
function extractOrphanNumber(id: string): number {
  const match = id.match(/\d+$/);
  return match ? parseInt(match[0], 10) : 0;
}
