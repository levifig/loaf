/**
 * Task Display Tests
 *
 * Tests for display helpers: status filtering (--active flag), task sorting,
 * grouping, and active task counting.
 */

import { describe, it, expect } from "vitest";

import {
  STATUS_DISPLAY_ORDER,
  STATUS_LABELS,
  getDisplayStatuses,
  sortTasks,
  countActiveTasks,
  groupTasksByStatus,
} from "./task-display.js";
import type { TaskEntry, TaskStatus } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

function taskEntry(overrides: Partial<TaskEntry> = {}): TaskEntry {
  return {
    title: "Test task",
    slug: "test",
    spec: null,
    status: "todo",
    priority: "P2",
    depends_on: [],
    files: [],
    verify: null,
    done: null,
    session: null,
    created: "2026-03-10T08:00:00Z",
    updated: "2026-03-10T08:00:00Z",
    completed_at: null,
    file: "TASK-001-test.md",
    ...overrides,
  };
}

// ─────────────────────────────────────────────────────────────────────────────
// STATUS_DISPLAY_ORDER & STATUS_LABELS
// ─────────────────────────────────────────────────────────────────────────────

describe("STATUS_DISPLAY_ORDER", () => {
  it("contains all five statuses in display order", () => {
    expect(STATUS_DISPLAY_ORDER).toEqual([
      "in_progress",
      "blocked",
      "todo",
      "review",
      "done",
    ]);
  });

  it("includes 'done' as the last status", () => {
    expect(STATUS_DISPLAY_ORDER[STATUS_DISPLAY_ORDER.length - 1]).toBe("done");
  });
});

describe("STATUS_LABELS", () => {
  it("maps every status to a human-readable label", () => {
    expect(STATUS_LABELS.in_progress).toBe("In Progress");
    expect(STATUS_LABELS.blocked).toBe("Blocked");
    expect(STATUS_LABELS.todo).toBe("Todo");
    expect(STATUS_LABELS.review).toBe("Review");
    expect(STATUS_LABELS.done).toBe("Done");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// getDisplayStatuses (--active flag behavior)
// ─────────────────────────────────────────────────────────────────────────────

describe("getDisplayStatuses", () => {
  it("returns all statuses when active is false", () => {
    const statuses = getDisplayStatuses(false);
    expect(statuses).toEqual(STATUS_DISPLAY_ORDER);
    expect(statuses).toContain("done");
  });

  it("excludes 'done' when active is true", () => {
    const statuses = getDisplayStatuses(true);
    expect(statuses).not.toContain("done");
  });

  it("preserves display order when active is true", () => {
    const statuses = getDisplayStatuses(true);
    expect(statuses).toEqual(["in_progress", "blocked", "todo", "review"]);
  });

  it("returns exactly 4 statuses when active is true", () => {
    const statuses = getDisplayStatuses(true);
    expect(statuses).toHaveLength(4);
  });

  it("returns exactly 5 statuses when active is false", () => {
    const statuses = getDisplayStatuses(false);
    expect(statuses).toHaveLength(5);
  });

  it("does not mutate STATUS_DISPLAY_ORDER", () => {
    const before = [...STATUS_DISPLAY_ORDER];
    getDisplayStatuses(true);
    getDisplayStatuses(false);
    expect(STATUS_DISPLAY_ORDER).toEqual(before);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// sortTasks
// ─────────────────────────────────────────────────────────────────────────────

describe("sortTasks", () => {
  it("sorts by priority: P0 before P1 before P2 before P3", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ priority: "P3" })],
      ["TASK-002", taskEntry({ priority: "P0" })],
      ["TASK-003", taskEntry({ priority: "P2" })],
      ["TASK-004", taskEntry({ priority: "P1" })],
    ];

    const sorted = sortTasks(tasks);
    expect(sorted.map(([id]) => id)).toEqual([
      "TASK-002", // P0
      "TASK-004", // P1
      "TASK-003", // P2
      "TASK-001", // P3
    ]);
  });

  it("sorts by updated date (newest first) within same priority", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ priority: "P2", updated: "2026-03-10T08:00:00Z" })],
      ["TASK-002", taskEntry({ priority: "P2", updated: "2026-03-12T08:00:00Z" })],
      ["TASK-003", taskEntry({ priority: "P2", updated: "2026-03-11T08:00:00Z" })],
    ];

    const sorted = sortTasks(tasks);
    expect(sorted.map(([id]) => id)).toEqual([
      "TASK-002", // newest
      "TASK-003",
      "TASK-001", // oldest
    ]);
  });

  it("falls back to created date when updated is missing", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ priority: "P2", updated: "", created: "2026-03-10T00:00:00Z" })],
      ["TASK-002", taskEntry({ priority: "P2", updated: "", created: "2026-03-12T00:00:00Z" })],
    ];

    const sorted = sortTasks(tasks);
    expect(sorted.map(([id]) => id)).toEqual(["TASK-002", "TASK-001"]);
  });

  it("returns empty array for empty input", () => {
    expect(sortTasks([])).toEqual([]);
  });

  it("returns single-element array unchanged", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry()],
    ];
    const sorted = sortTasks(tasks);
    expect(sorted).toHaveLength(1);
    expect(sorted[0][0]).toBe("TASK-001");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// countActiveTasks
// ─────────────────────────────────────────────────────────────────────────────

describe("countActiveTasks", () => {
  it("counts all tasks when none are done", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ status: "todo" })],
      ["TASK-002", taskEntry({ status: "in_progress" })],
      ["TASK-003", taskEntry({ status: "blocked" })],
    ];

    expect(countActiveTasks(tasks)).toBe(3);
  });

  it("excludes done tasks from the count", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ status: "todo" })],
      ["TASK-002", taskEntry({ status: "done" })],
      ["TASK-003", taskEntry({ status: "in_progress" })],
      ["TASK-004", taskEntry({ status: "done" })],
    ];

    expect(countActiveTasks(tasks)).toBe(2);
  });

  it("returns 0 when all tasks are done", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ status: "done" })],
      ["TASK-002", taskEntry({ status: "done" })],
    ];

    expect(countActiveTasks(tasks)).toBe(0);
  });

  it("returns 0 for empty task list", () => {
    expect(countActiveTasks([])).toBe(0);
  });

  it("counts review tasks as active", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ status: "review" })],
    ];

    expect(countActiveTasks(tasks)).toBe(1);
  });

  it("handles mixed statuses correctly", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ status: "todo" })],
      ["TASK-002", taskEntry({ status: "in_progress" })],
      ["TASK-003", taskEntry({ status: "blocked" })],
      ["TASK-004", taskEntry({ status: "review" })],
      ["TASK-005", taskEntry({ status: "done" })],
    ];

    expect(countActiveTasks(tasks)).toBe(4);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// groupTasksByStatus
// ─────────────────────────────────────────────────────────────────────────────

describe("groupTasksByStatus", () => {
  it("groups tasks into correct status buckets", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ status: "todo" })],
      ["TASK-002", taskEntry({ status: "in_progress" })],
      ["TASK-003", taskEntry({ status: "done" })],
      ["TASK-004", taskEntry({ status: "todo" })],
    ];

    const grouped = groupTasksByStatus(tasks);

    expect(grouped.todo).toHaveLength(2);
    expect(grouped.in_progress).toHaveLength(1);
    expect(grouped.done).toHaveLength(1);
    expect(grouped.blocked).toHaveLength(0);
    expect(grouped.review).toHaveLength(0);
  });

  it("returns empty arrays for all statuses when no tasks given", () => {
    const grouped = groupTasksByStatus([]);

    expect(grouped.todo).toHaveLength(0);
    expect(grouped.in_progress).toHaveLength(0);
    expect(grouped.blocked).toHaveLength(0);
    expect(grouped.review).toHaveLength(0);
    expect(grouped.done).toHaveLength(0);
  });

  it("preserves task ID and entry in grouped output", () => {
    const entry = taskEntry({ status: "blocked", title: "Blocked task" });
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-042", entry],
    ];

    const grouped = groupTasksByStatus(tasks);

    expect(grouped.blocked).toHaveLength(1);
    expect(grouped.blocked[0][0]).toBe("TASK-042");
    expect(grouped.blocked[0][1].title).toBe("Blocked task");
  });

  it("places tasks with unknown status into the todo bucket", () => {
    // Force an unknown status to test the fallback
    const entry = taskEntry({ status: "banana" as TaskStatus });
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", entry],
    ];

    const grouped = groupTasksByStatus(tasks);

    expect(grouped.todo).toHaveLength(1);
    expect(grouped.todo[0][0]).toBe("TASK-001");
  });

  it("groups all five valid statuses correctly", () => {
    const tasks: Array<[string, TaskEntry]> = [
      ["TASK-001", taskEntry({ status: "todo" })],
      ["TASK-002", taskEntry({ status: "in_progress" })],
      ["TASK-003", taskEntry({ status: "blocked" })],
      ["TASK-004", taskEntry({ status: "review" })],
      ["TASK-005", taskEntry({ status: "done" })],
    ];

    const grouped = groupTasksByStatus(tasks);

    expect(grouped.todo).toHaveLength(1);
    expect(grouped.in_progress).toHaveLength(1);
    expect(grouped.blocked).toHaveLength(1);
    expect(grouped.review).toHaveLength(1);
    expect(grouped.done).toHaveLength(1);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Integration: --active flag end-to-end filtering
// ─────────────────────────────────────────────────────────────────────────────

describe("--active flag filtering (combined behavior)", () => {
  const tasks: Array<[string, TaskEntry]> = [
    ["TASK-001", taskEntry({ status: "todo", spec: "SPEC-001" })],
    ["TASK-002", taskEntry({ status: "in_progress", spec: "SPEC-001" })],
    ["TASK-003", taskEntry({ status: "done", spec: "SPEC-002" })],
    ["TASK-004", taskEntry({ status: "done", spec: "SPEC-002" })],
    ["TASK-005", taskEntry({ status: "blocked" })],
  ];

  it("active mode: grouped output does not include done tasks in iterated statuses", () => {
    const grouped = groupTasksByStatus(tasks);
    const displayStatuses = getDisplayStatuses(true);

    // Collect all tasks that would be displayed
    const displayed: string[] = [];
    for (const status of displayStatuses) {
      for (const [id] of grouped[status]) {
        displayed.push(id);
      }
    }

    expect(displayed).toContain("TASK-001");
    expect(displayed).toContain("TASK-002");
    expect(displayed).toContain("TASK-005");
    expect(displayed).not.toContain("TASK-003");
    expect(displayed).not.toContain("TASK-004");
  });

  it("active mode: countActiveTasks returns correct count", () => {
    expect(countActiveTasks(tasks)).toBe(3);
  });

  it("default mode: all tasks are included in iterated statuses", () => {
    const grouped = groupTasksByStatus(tasks);
    const displayStatuses = getDisplayStatuses(false);

    const displayed: string[] = [];
    for (const status of displayStatuses) {
      for (const [id] of grouped[status]) {
        displayed.push(id);
      }
    }

    expect(displayed).toHaveLength(5);
    expect(displayed).toContain("TASK-003");
    expect(displayed).toContain("TASK-004");
  });

  it("active mode: done tasks still exist in grouped output but are not iterated", () => {
    const grouped = groupTasksByStatus(tasks);
    const displayStatuses = getDisplayStatuses(true);

    // Done tasks are in the grouped record but not in displayStatuses
    expect(grouped.done).toHaveLength(2);
    expect(displayStatuses).not.toContain("done");
  });
});
