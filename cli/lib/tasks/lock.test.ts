/**
 * Tests for `withTasksJsonLock` (SPEC-036 follow-up, Codex blocker #1).
 *
 * The point of these tests is to lock in the correctness of the
 * read-modify-write barrier under cross-process contention. Two angles:
 *
 *   1. Sequential / in-process: the lock behaves correctly for ordinary
 *      single-process callers (round-trip, error propagation, SKIP_WRITE).
 *
 *   2. Real cross-process concurrency: N child processes each spawn the
 *      bundled `dist-cli/index.js` (via `loaf task create`), allocating
 *      against the same `TASKS.json`. We assert that every ID is distinct
 *      and that `next_id` matches `(start + N)`. This is the test Codex
 *      specifically called out as missing — the previous parallel-ID
 *      allocation test was sequential.
 *
 * The cross-process test invokes the real production binary (not a stub
 * worker) so it exercises the same code path users hit. The trade is that
 * tests in this file run after `pretest: npm run build:cli`, which is
 * already the project's standard test posture.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from "fs";
import { spawn } from "child_process";
import { join } from "path";
import { tmpdir } from "os";

import { SKIP_WRITE, TASKS_JSON_LOCK_FILE, withTasksJsonLock } from "./lock.js";
import { saveIndex } from "./migrate.js";
import type { TaskIndex } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Fixture setup
// ─────────────────────────────────────────────────────────────────────────────

let TEST_ROOT: string;

function makeAgentsDir(name: string, seedNextId = 100): string {
  const dir = realpathSync(mkdtempSync(join(TEST_ROOT, `${name}-`)));
  const agentsDir = join(dir, ".agents");
  mkdirSync(agentsDir, { recursive: true });
  const seed: TaskIndex = {
    version: 1,
    next_id: seedNextId,
    tasks: {},
    specs: {},
  };
  saveIndex(join(agentsDir, "TASKS.json"), seed);
  return agentsDir;
}

beforeEach(() => {
  TEST_ROOT = realpathSync(mkdtempSync(join(tmpdir(), "loaf-tasks-lock-")));
});

afterEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// Lock — sequential / in-process behavior
// ─────────────────────────────────────────────────────────────────────────────

describe("withTasksJsonLock — sequential behavior", () => {
  it("persists mutations made inside the callback", () => {
    const agentsDir = makeAgentsDir("seq-persist");

    withTasksJsonLock(agentsDir, (index) => {
      index.tasks["TASK-100"] = {
        title: "Test",
        slug: "test",
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
        file: "TASK-100-test.md",
      };
      index.next_id = 101;
    });

    const reloaded = JSON.parse(
      readFileSync(join(agentsDir, "TASKS.json"), "utf-8"),
    ) as TaskIndex;
    expect(reloaded.next_id).toBe(101);
    expect(reloaded.tasks["TASK-100"]).toBeDefined();
  });

  it("releases the lock after a successful callback", () => {
    const agentsDir = makeAgentsDir("seq-release");
    withTasksJsonLock(agentsDir, () => undefined);
    expect(existsSync(join(agentsDir, TASKS_JSON_LOCK_FILE))).toBe(false);
  });

  it("releases the lock when the callback throws", () => {
    const agentsDir = makeAgentsDir("seq-throw");
    expect(() =>
      withTasksJsonLock(agentsDir, () => {
        throw new Error("boom");
      }),
    ).toThrow("boom");
    expect(existsSync(join(agentsDir, TASKS_JSON_LOCK_FILE))).toBe(false);
  });

  it("skips the post-callback write when the callback returns SKIP_WRITE", () => {
    const agentsDir = makeAgentsDir("seq-skip");
    const before = readFileSync(join(agentsDir, "TASKS.json"), "utf-8");
    withTasksJsonLock(agentsDir, (index) => {
      index.next_id = 999; // would normally be persisted
      return SKIP_WRITE;
    });
    const after = readFileSync(join(agentsDir, "TASKS.json"), "utf-8");
    expect(after).toBe(before);
  });

  it("skips the post-callback write when the callback returns false", () => {
    const agentsDir = makeAgentsDir("seq-skip-false");
    const before = readFileSync(join(agentsDir, "TASKS.json"), "utf-8");
    withTasksJsonLock(agentsDir, (index) => {
      index.next_id = 999;
      return false;
    });
    const after = readFileSync(join(agentsDir, "TASKS.json"), "utf-8");
    expect(after).toBe(before);
  });

  it("re-reads TASKS.json under the lock (never trusts a stale snapshot)", () => {
    const agentsDir = makeAgentsDir("seq-reread", 100);

    // Out-of-band write between two withTasksJsonLock calls — simulates what
    // a concurrent process would do. The second lock acquisition must see
    // the new value, not the old.
    withTasksJsonLock(agentsDir, (index) => {
      expect(index.next_id).toBe(100);
      index.next_id = 200;
    });

    // External tampering — bypass the lock helper to simulate another
    // process having mutated the file.
    const path = join(agentsDir, "TASKS.json");
    const fresh: TaskIndex = JSON.parse(readFileSync(path, "utf-8"));
    fresh.next_id = 555;
    writeFileSync(path, JSON.stringify(fresh, null, 2) + "\n", "utf-8");

    withTasksJsonLock(agentsDir, (index) => {
      // If the lock helper cached or trusted any in-memory state, this
      // would be 200 (or its derivative). With a fresh re-read inside the
      // lock, it must be the value we just wrote out-of-band.
      expect(index.next_id).toBe(555);
    });
  });

  it("builds TASKS.json from .md files when it doesn't exist yet", () => {
    const dir = realpathSync(mkdtempSync(join(TEST_ROOT, "cold-")));
    const agentsDir = join(dir, ".agents");
    mkdirSync(agentsDir, { recursive: true });
    expect(existsSync(join(agentsDir, "TASKS.json"))).toBe(false);

    withTasksJsonLock(agentsDir, (index) => {
      // First-time observers see a fresh built-from-files index: empty
      // tasks/specs and next_id = 1.
      expect(index.tasks).toEqual({});
      expect(index.specs).toEqual({});
      expect(index.next_id).toBe(1);
      index.next_id = 5;
    });

    expect(existsSync(join(agentsDir, "TASKS.json"))).toBe(true);
    const reloaded = JSON.parse(
      readFileSync(join(agentsDir, "TASKS.json"), "utf-8"),
    ) as TaskIndex;
    expect(reloaded.next_id).toBe(5);
  });

  it("returns the callback's value to the caller", () => {
    const agentsDir = makeAgentsDir("seq-return");
    const minted = withTasksJsonLock(agentsDir, (index) => {
      const id = `TASK-${String(index.next_id).padStart(3, "0")}`;
      index.next_id += 1;
      return id;
    });
    expect(minted).toBe("TASK-100");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Cross-process concurrency: N spawned `loaf task create` against the same lock
// ─────────────────────────────────────────────────────────────────────────────
//
// We spawn the bundled `dist-cli/index.js` so the test exercises the same
// path users hit. The `pretest` script (`npm run build:cli`) guarantees the
// bundle is fresh before tests run.

const CLI_PATH = join(process.cwd(), "dist-cli/index.js");

function spawnTaskCreate(
  agentsDir: string,
  title: string,
): Promise<{ exitCode: number; stdout: string; stderr: string }> {
  return new Promise((resolve) => {
    // We point cwd at the agentsDir's parent (so findAgentsDir() locates the
    // fixture, not the surrounding loaf repo's .agents/).
    const cwd = join(agentsDir, "..");
    const child = spawn(
      "node",
      [CLI_PATH, "task", "create", "--title", title],
      {
        cwd,
        stdio: ["ignore", "pipe", "pipe"],
        env: {
          ...process.env,
          // Prevent any session-side journal nudges from interfering — we
          // only care about the task index write here.
          LOAF_TASK_TEST: "1",
        },
      },
    );
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (d) => (stdout += d.toString()));
    child.stderr.on("data", (d) => (stderr += d.toString()));
    child.on("close", (code) =>
      resolve({ exitCode: code ?? 0, stdout, stderr }),
    );
  });
}

function extractTaskIdFromStdout(stdout: string): string | null {
  // Matches the "Created TASK-NNN: ..." line emitted by `loaf task create`.
  // The line is colorized; strip ANSI before matching.
  const ansiStripped = stdout.replace(/\[[0-9;]*m/g, "");
  const m = ansiStripped.match(/Created\s+(TASK-\d+):/);
  return m ? m[1] : null;
}

describe("withTasksJsonLock — cross-process concurrency", () => {
  it("N concurrent `loaf task create` invocations each mint a distinct TASK ID", async () => {
    const N = 8;
    const agentsDir = makeAgentsDir("concurrent", 100);

    // Spawn all task-create invocations at once.
    const promises = Array.from({ length: N }, (_, i) =>
      spawnTaskCreate(agentsDir, `Concurrent task ${i}`),
    );
    const results = await Promise.all(promises);

    // Every spawn must have exited 0. If a lock acquisition failed, stderr
    // tells us why.
    for (let i = 0; i < results.length; i++) {
      const { exitCode, stderr, stdout } = results[i];
      if (exitCode !== 0) {
        throw new Error(
          `Worker ${i} exited ${exitCode}\nstderr: ${stderr}\nstdout: ${stdout}`,
        );
      }
    }

    // Extract the minted IDs from each spawn's stdout.
    const mintedIds = results.map((r) => extractTaskIdFromStdout(r.stdout));
    expect(mintedIds.every((id) => id !== null)).toBe(true);

    const uniqueIds = new Set(mintedIds as string[]);
    expect(uniqueIds.size).toBe(N); // NO collisions — this is the blocker fix.

    // Every minted ID must fall in [TASK-100, TASK-(99+N)] and the set must
    // be exactly that range — confirms no write got lost.
    const expected = new Set(
      Array.from({ length: N }, (_, i) => `TASK-${String(100 + i).padStart(3, "0")}`),
    );
    expect(uniqueIds).toEqual(expected);

    // Confirm the on-disk index is consistent with N successful allocations.
    const finalIndex = JSON.parse(
      readFileSync(join(agentsDir, "TASKS.json"), "utf-8"),
    ) as TaskIndex;
    expect(finalIndex.next_id).toBe(100 + N);
    expect(Object.keys(finalIndex.tasks)).toHaveLength(N);

    // The lock file must be cleaned up.
    expect(existsSync(join(agentsDir, TASKS_JSON_LOCK_FILE))).toBe(false);

    // Every minted ID must have a corresponding .md file on disk.
    const tasksDir = join(agentsDir, "tasks");
    const taskFiles = readdirSync(tasksDir).filter((f) => f.startsWith("TASK-"));
    expect(taskFiles).toHaveLength(N);
  }, 30000);
});
