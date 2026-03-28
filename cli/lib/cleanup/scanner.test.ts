/**
 * Cleanup Scanner Tests
 *
 * Tests the scanner engine against temp .agents/ fixtures.
 * Each test creates a minimal directory structure with specific artifact states.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdirSync, mkdtempSync, writeFileSync, rmSync } from "fs";
import { join } from "path";
import { tmpdir } from "os";

import { scanArtifacts } from "./scanner.js";
import type { ArtifactType, CleanupRecommendation } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

let tempDir: string;

function agentsDir(): string {
  return join(tempDir, ".agents");
}

function setupDir(...subpaths: string[]): string {
  const dir = join(agentsDir(), ...subpaths);
  mkdirSync(dir, { recursive: true });
  return dir;
}

function writeArtifact(subdir: string, filename: string, frontmatter: Record<string, unknown>, body = ""): void {
  const dir = setupDir(subdir);
  const lines = Object.entries(frontmatter).map(([k, v]) => `${k}: ${JSON.stringify(v)}`);
  const content = `---\n${lines.join("\n")}\n---\n\n${body}`;
  writeFileSync(join(dir, filename), content, "utf-8");
}

/** Write a minimal TASKS.json */
function writeIndex(tasks: Record<string, unknown> = {}, specs: Record<string, unknown> = {}): void {
  const index = { version: 1, next_id: 100, tasks, specs };
  writeFileSync(join(agentsDir(), "TASKS.json"), JSON.stringify(index, null, 2), "utf-8");
}

function findRec(recs: CleanupRecommendation[], filename: string): CleanupRecommendation | undefined {
  return recs.find((r) => r.filename === filename);
}

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), "loaf-cleanup-test-"));
  // Create required directories
  setupDir("sessions");
  setupDir("tasks");
  setupDir("specs");
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// Sessions
// ─────────────────────────────────────────────────────────────────────────────

describe("sessions", () => {
  it("recommends archive for completed sessions", () => {
    writeArtifact("sessions", "SESSION-001.md", { status: "completed", created: "2026-03-01" });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "SESSION-001.md");
    expect(rec?.action).toBe("archive");
  });

  it("hints at /crystallize for sessions with learnings", () => {
    writeArtifact("sessions", "SESSION-002.md", { status: "completed", created: "2026-03-01" }, "## Key Decisions\n- Did a thing");
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "SESSION-002.md");
    expect(rec?.action).toBe("archive");
    expect(rec?.hint).toContain("crystallize");
  });

  it("flags stale sessions (>7 days inactive)", () => {
    const staleDate = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString();
    writeArtifact("sessions", "SESSION-003.md", { status: "active", updated: staleDate });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "SESSION-003.md");
    expect(rec?.action).toBe("flag");
    expect(rec?.reason).toContain("inactive");
  });

  it("skips active sessions", () => {
    writeArtifact("sessions", "SESSION-004.md", { status: "active", updated: new Date().toISOString() });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "SESSION-004.md");
    expect(rec?.action).toBe("skip");
  });

  it("archives cancelled sessions", () => {
    writeArtifact("sessions", "SESSION-005.md", { status: "cancelled", created: "2026-03-01" });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "SESSION-005.md");
    expect(rec?.action).toBe("archive");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tasks
// ─────────────────────────────────────────────────────────────────────────────

describe("tasks", () => {
  it("recommends archive for done tasks", () => {
    writeIndex(
      {
        "TASK-001": {
          title: "Done task", slug: "done", spec: null, status: "done",
          priority: "P1", depends_on: [], files: [], verify: null, done: null,
          session: null, created: "2026-03-01", updated: "2026-03-01",
          completed_at: "2026-03-10", file: "TASK-001-done.md",
        },
      },
      {},
    );
    writeArtifact("tasks", "TASK-001-done.md", { id: "TASK-001", title: "Done task", status: "done" });
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "TASK-001-done.md");
    expect(rec?.action).toBe("archive");
  });

  it("flags tasks referencing missing specs (orphaned ref)", () => {
    writeIndex(
      {
        "TASK-002": {
          title: "Orphan ref", slug: "orphan", spec: "SPEC-099", status: "todo",
          priority: "P2", depends_on: [], files: [], verify: null, done: null,
          session: null, created: "2026-03-01", updated: "2026-03-01",
          completed_at: null, file: "TASK-002-orphan.md",
        },
      },
      {},
    );
    writeArtifact("tasks", "TASK-002-orphan.md", { id: "TASK-002", title: "Orphan ref", spec: "SPEC-099" });
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "TASK-002-orphan.md");
    expect(rec?.action).toBe("flag");
    expect(rec?.reason).toContain("SPEC-099");
  });

  it("does NOT flag tasks with spec: null (valid ad-hoc tasks)", () => {
    writeIndex(
      {
        "TASK-003": {
          title: "Ad-hoc", slug: "adhoc", spec: null, status: "todo",
          priority: "P2", depends_on: [], files: [], verify: null, done: null,
          session: null, created: "2026-03-01", updated: "2026-03-01",
          completed_at: null, file: "TASK-003-adhoc.md",
        },
      },
      {},
    );
    writeArtifact("tasks", "TASK-003-adhoc.md", { id: "TASK-003", title: "Ad-hoc" });
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "TASK-003-adhoc.md");
    expect(rec?.action).toBe("skip");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Specs
// ─────────────────────────────────────────────────────────────────────────────

describe("specs", () => {
  it("recommends archive for complete specs", () => {
    writeIndex(
      {},
      {
        "SPEC-001": {
          title: "Done spec", status: "complete", appetite: null,
          requirement: null, source: null, created: "2026-03-01",
          file: "SPEC-001-done.md",
        },
      },
    );
    writeArtifact("specs", "SPEC-001-done.md", { id: "SPEC-001", title: "Done spec", status: "complete" });
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "SPEC-001-done.md");
    expect(rec?.action).toBe("archive");
  });

  it("flags stale drafting specs (>14 days)", () => {
    const staleDate = new Date(Date.now() - 20 * 24 * 60 * 60 * 1000).toISOString();
    writeIndex(
      {},
      {
        "SPEC-002": {
          title: "Stale spec", status: "drafting", appetite: null,
          requirement: null, source: null, created: staleDate,
          file: "SPEC-002-stale.md",
        },
      },
    );
    writeArtifact("specs", "SPEC-002-stale.md", { id: "SPEC-002", title: "Stale spec", status: "drafting" });
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "SPEC-002-stale.md");
    expect(rec?.action).toBe("flag");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Plans
// ─────────────────────────────────────────────────────────────────────────────

describe("plans", () => {
  it("recommends delete for orphaned plans (no session link)", () => {
    writeArtifact("plans", "plan-001.md", { title: "Orphan plan", created: "2026-03-01" });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "plan-001.md");
    expect(rec?.action).toBe("delete");
    expect(rec?.reason).toContain("no linked session");
  });

  it("recommends delete for plans linked to completed sessions", () => {
    writeArtifact("sessions", "SESSION-010.md", { status: "completed", created: "2026-03-01" });
    writeArtifact("plans", "plan-002.md", { session: "SESSION-010.md", created: "2026-03-01" });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "plan-002.md");
    expect(rec?.action).toBe("delete");
  });

  it("skips plans linked to active sessions", () => {
    writeArtifact("sessions", "SESSION-011.md", { status: "active", updated: new Date().toISOString() });
    writeArtifact("plans", "plan-003.md", { session: "SESSION-011.md", updated: new Date().toISOString() });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "plan-003.md");
    expect(rec?.action).toBe("skip");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Drafts
// ─────────────────────────────────────────────────────────────────────────────

describe("drafts", () => {
  it("flags stale drafts (>30 days)", () => {
    const staleDate = new Date(Date.now() - 35 * 24 * 60 * 60 * 1000).toISOString();
    writeArtifact("drafts", "draft-001.md", { title: "Old draft", created: staleDate });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "draft-001.md");
    expect(rec?.action).toBe("flag");
    expect(rec?.reason).toContain("days old");
  });

  it("notes sparks section in stale drafts", () => {
    const staleDate = new Date(Date.now() - 35 * 24 * 60 * 60 * 1000).toISOString();
    writeArtifact("drafts", "draft-002.md", { title: "Sparky draft", created: staleDate }, "## Sparks\n- Great idea");
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "draft-002.md");
    expect(rec?.hint).toContain("Sparks");
  });

  it("skips recent drafts", () => {
    writeArtifact("drafts", "draft-003.md", { title: "Fresh draft", created: new Date().toISOString() });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    const rec = findRec(result.recommendations, "draft-003.md");
    expect(rec?.action).toBe("skip");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Missing Directories
// ─────────────────────────────────────────────────────────────────────────────

describe("directory handling", () => {
  it("warns about missing required directories", () => {
    // Remove the sessions dir we created in beforeEach
    rmSync(join(agentsDir(), "sessions"), { recursive: true });
    writeIndex();
    const result = scanArtifacts({ agentsDir: agentsDir() });
    expect(result.warnings).toContain("Required directory missing: sessions/");
  });

  it("skips missing optional directories silently", () => {
    writeIndex();
    // plans/ doesn't exist — should produce no warnings
    const result = scanArtifacts({ agentsDir: agentsDir() });
    expect(result.warnings.filter((w) => w.includes("plans"))).toHaveLength(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Filter
// ─────────────────────────────────────────────────────────────────────────────

describe("filter option", () => {
  it("restricts scan to specified artifact types", () => {
    writeArtifact("sessions", "SESSION-001.md", { status: "completed", created: "2026-03-01" });
    writeArtifact("drafts", "draft-001.md", { title: "Draft", created: new Date().toISOString() });
    writeIndex();

    const result = scanArtifacts({ agentsDir: agentsDir(), filter: ["session"] });
    const types = new Set(result.recommendations.map((r) => r.type));
    expect(types.has("session")).toBe(true);
    expect(types.has("draft")).toBe(false);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Summary
// ─────────────────────────────────────────────────────────────────────────────

describe("summary", () => {
  it("produces correct counts per artifact type", () => {
    writeArtifact("sessions", "SESSION-001.md", { status: "completed", created: "2026-03-01" });
    writeArtifact("sessions", "SESSION-002.md", { status: "active", updated: new Date().toISOString() });
    writeIndex();

    const result = scanArtifacts({ agentsDir: agentsDir() });
    const sessionSummary = result.summary.find((s) => s.type === "session");
    expect(sessionSummary?.total).toBe(2);
    expect(sessionSummary?.archive).toBe(1);
    expect(sessionSummary?.skip).toBe(1);
  });
});
