/**
 * Regression tests for `rebuildTaskIndex` (PR #49 Codex HOLD).
 *
 * The defect: the original SPEC-036 refactor replaced `current.tasks` and
 * `current.specs` with the pre-scan snapshot from disk. If a concurrent
 * `task create` committed its TASKS.json entry INSIDE the lock (the index
 * write happens before the .md file is written, which lands outside the
 * lock), and that index commit happened DURING our outside scan, the fresh
 * entry would be silently dropped at the in-lock swap.
 *
 * The fix: scan-window-aware union merge.
 *
 *   - Scanned (on-disk) entries are authoritative.
 *   - PLUS entries minted DURING the scan window (preserved even though no
 *     .md exists yet — the .md write hasn't landed).
 *   - DROP pre-scan entries that aren't on disk (the whole point of
 *     `refresh` is to reconcile with disk truth).
 */

import { describe, it, expect, afterEach } from "vitest";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "fs";
import { join } from "path";
import { tmpdir } from "os";
import matter from "gray-matter";

import { rebuildTaskIndex } from "./task.js";
import { saveIndex } from "../lib/tasks/migrate.js";
import type { TaskIndex, TaskEntry } from "../lib/tasks/types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Fixture helpers
// ─────────────────────────────────────────────────────────────────────────────

let tempDir: string;

function makeAgentsDir(): string {
  tempDir = mkdtempSync(join(tmpdir(), "loaf-task-rebuild-"));
  const agentsDir = join(tempDir, ".agents");
  mkdirSync(agentsDir, { recursive: true });
  mkdirSync(join(agentsDir, "tasks"), { recursive: true });
  return agentsDir;
}

function taskEntry(id: string, overrides: Partial<TaskEntry> = {}): TaskEntry {
  return {
    title: `Task ${id}`,
    slug: "seeded",
    spec: null,
    status: "todo",
    priority: "P2",
    depends_on: [],
    files: [],
    verify: null,
    done: null,
    session: null,
    created: "2026-05-19T00:00:00Z",
    updated: "2026-05-19T00:00:00Z",
    completed_at: null,
    file: `${id}-seeded.md`,
    ...overrides,
  };
}

function writeTaskMd(agentsDir: string, id: string): void {
  const fm = {
    id,
    title: `Task ${id}`,
    status: "todo",
    priority: "P2",
    created: "2026-05-19T00:00:00Z",
    updated: "2026-05-19T00:00:00Z",
  };
  const body = `\n# ${id}: seeded\n`;
  writeFileSync(
    join(agentsDir, "tasks", `${id}-seeded.md`),
    matter.stringify(body, fm),
    "utf-8",
  );
}

afterEach(() => {
  if (tempDir) {
    rmSync(tempDir, { recursive: true, force: true });
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

describe("rebuildTaskIndex — scan-window-aware union merge", () => {
  it("preserves an in-lock entry minted DURING the scan window (.md not yet on disk)", () => {
    // Seed TASK-001..TASK-010 with matching .md files. We model the race
    // window by placing TASK-011 in the index with `next_id` still at 11
    // — exactly the state the file would be in BETWEEN our pre-scan
    // snapshot (which captures `scanStartNextId = 11`) and the in-lock
    // re-read in the real race:
    //
    //   - Real race: pre-scan snapshot reads next_id=11; concurrent
    //     `task create` then commits TASK-011 + sets next_id=12 inside its
    //     lock; the .md write lands AFTER its lock release; our scan
    //     misses TASK-011.md; our in-lock read sees next_id=12 with
    //     TASK-011 present.
    //   - In-process model: snapshot and in-lock read see the same file
    //     content because there's no concurrent writer. Setting next_id=11
    //     ensures `scanStartNextId = 11`, putting TASK-011 (parsed id=11)
    //     INSIDE the preservation window (id >= 11). This is the load-
    //     bearing condition: a defective implementation that replaces
    //     `current.tasks` with `scannedIndex.tasks` will drop TASK-011
    //     even though it should be preserved.
    const agentsDir = makeAgentsDir();
    const seedIndex: TaskIndex = {
      version: 1,
      next_id: 11,
      tasks: {},
      specs: {},
    };
    for (let i = 1; i <= 10; i++) {
      const id = `TASK-${String(i).padStart(3, "0")}`;
      seedIndex.tasks[id] = taskEntry(id);
      writeTaskMd(agentsDir, id);
    }
    // TASK-011 is in TASKS.json with NO matching .md — the freshly-minted
    // entry whose .md write hasn't landed yet.
    seedIndex.tasks["TASK-011"] = taskEntry("TASK-011");
    saveIndex(join(agentsDir, "TASKS.json"), seedIndex);

    const result = rebuildTaskIndex(agentsDir);

    // Defect would have dropped TASK-011 here (the in-lock swap replaces
    // current.tasks with the disk scan, which never saw TASK-011.md).
    expect(result.tasks["TASK-011"]).toBeDefined();
    expect(result.tasks["TASK-011"].title).toBe("Task TASK-011");
    // next_id stays monotonic: at least max(parsed-id)+1 = 12.
    expect(result.next_id).toBeGreaterThanOrEqual(12);
    // All seeded tasks still present.
    for (let i = 1; i <= 10; i++) {
      const id = `TASK-${String(i).padStart(3, "0")}`;
      expect(result.tasks[id]).toBeDefined();
    }
  });

  it("DROPS a pre-scan entry that exists in TASKS.json but has no .md file (the legitimate cleanup case)", () => {
    // Seed TASK-001..TASK-010 with matching .md files. Plus seed a STALE
    // TASK-005-bogus entry into TASKS.json that has no corresponding .md
    // and an ID below scanStartNextId — the cleanup case.
    const agentsDir = makeAgentsDir();
    const seedIndex: TaskIndex = {
      version: 1,
      next_id: 50,
      tasks: {},
      specs: {},
    };
    for (let i = 1; i <= 10; i++) {
      const id = `TASK-${String(i).padStart(3, "0")}`;
      seedIndex.tasks[id] = taskEntry(id);
      writeTaskMd(agentsDir, id);
    }
    // Stale entry: ID < scanStartNextId (50), no .md file.
    seedIndex.tasks["TASK-042"] = taskEntry("TASK-042", { title: "Stale" });
    saveIndex(join(agentsDir, "TASKS.json"), seedIndex);

    const result = rebuildTaskIndex(agentsDir);

    // Window semantics: TASK-042 was in the pre-scan snapshot AND has
    // id-number < scanStartNextId (=50). The scan didn't find its .md, so
    // it should be dropped.
    expect(result.tasks["TASK-042"]).toBeUndefined();
    // The 10 real tasks survive.
    expect(Object.keys(result.tasks).length).toBe(10);
  });

  it("monotonic next_id across scan, snapshot, and merged max id", () => {
    const agentsDir = makeAgentsDir();
    const seedIndex: TaskIndex = {
      version: 1,
      next_id: 200,
      tasks: {},
      specs: {},
    };
    for (let i = 1; i <= 5; i++) {
      const id = `TASK-${String(i).padStart(3, "0")}`;
      seedIndex.tasks[id] = taskEntry(id);
      writeTaskMd(agentsDir, id);
    }
    saveIndex(join(agentsDir, "TASKS.json"), seedIndex);

    const result = rebuildTaskIndex(agentsDir);

    // Even though only TASK-001..TASK-005 exist on disk, next_id must not
    // roll back below the pre-existing 200 (a concurrent allocator already
    // moved past us).
    expect(result.next_id).toBeGreaterThanOrEqual(200);
  });
});
