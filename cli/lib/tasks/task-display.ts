/**
 * Task Display Helpers
 *
 * Pure functions for filtering, sorting, and counting tasks in display contexts.
 * Extracted from the task command to enable unit testing.
 */

import type { TaskEntry, TaskStatus, TaskPriority } from "./types.js";
import { TASK_PRIORITIES } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

/** Display order for task statuses in the task board */
export const STATUS_DISPLAY_ORDER: TaskStatus[] = [
  "in_progress",
  "blocked",
  "todo",
  "review",
  "done",
];

/** Human-readable labels for task statuses */
export const STATUS_LABELS: Record<TaskStatus, string> = {
  in_progress: "In Progress",
  blocked: "Blocked",
  todo: "Todo",
  review: "Review",
  done: "Done",
};

// ─────────────────────────────────────────────────────────────────────────────
// Filtering
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Return the list of statuses to display.
 * When `active` is true, the "done" status is excluded.
 */
export function getDisplayStatuses(active: boolean): TaskStatus[] {
  return active
    ? STATUS_DISPLAY_ORDER.filter((s) => s !== "done")
    : STATUS_DISPLAY_ORDER;
}

// ─────────────────────────────────────────────────────────────────────────────
// Sorting
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Sort tasks by priority (P0 first), then by updated date (newest first).
 */
export function sortTasks(
  tasks: Array<[string, TaskEntry]>,
): Array<[string, TaskEntry]> {
  return tasks.sort((a, b) => {
    // Priority: lower index = higher priority
    const pA = TASK_PRIORITIES.indexOf(a[1].priority);
    const pB = TASK_PRIORITIES.indexOf(b[1].priority);
    if (pA !== pB) return pA - pB;

    // Updated date: newest first
    const dateA = a[1].updated || a[1].created || "";
    const dateB = b[1].updated || b[1].created || "";
    return dateB.localeCompare(dateA);
  });
}

// ─────────────────────────────────────────────────────────────────────────────
// Counting
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Count tasks that are not in "done" status.
 */
export function countActiveTasks(
  taskEntries: Array<[string, TaskEntry]>,
): number {
  return taskEntries.filter(([, e]) => e.status !== "done").length;
}

/**
 * Group tasks by status into a record.
 * Unknown statuses are placed in the "todo" bucket.
 */
export function groupTasksByStatus(
  taskEntries: Array<[string, TaskEntry]>,
): Record<TaskStatus, Array<[string, TaskEntry]>> {
  const grouped: Record<TaskStatus, Array<[string, TaskEntry]>> = {
    in_progress: [],
    blocked: [],
    todo: [],
    review: [],
    done: [],
  };

  for (const [id, entry] of taskEntries) {
    const status = entry.status;
    if (grouped[status]) {
      grouped[status].push([id, entry]);
    } else {
      grouped.todo.push([id, entry]);
    }
  }

  return grouped;
}
