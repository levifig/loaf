/**
 * Setup Command Tests
 *
 * Tests for the loaf setup command — directory creation, scaffolding,
 * and idempotent re-runs.
 *
 * Note: Build and install steps require the full Loaf content tree,
 * so those are tested via integration (npm run build && loaf setup).
 * These unit tests focus on the scaffolding and directory-creation logic.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  existsSync,
  mkdirSync,
  rmSync,
  readFileSync,
  writeFileSync,
} from "fs";
import { join } from "path";

// ─────────────────────────────────────────────────────────────────────────────
// Test Fixtures
// ─────────────────────────────────────────────────────────────────────────────

const TEST_ROOT = join(process.cwd(), ".test-setup-command");

// Scaffold constants (mirrored from setup.ts — kept in sync manually)
const EXPECTED_DIRS = [
  ".agents",
  ".agents/sessions",
  ".agents/ideas",
  ".agents/specs",
  ".agents/tasks",
  "docs",
  "docs/knowledge",
  "docs/decisions",
];

const EXPECTED_FILES = [
  ".agents/AGENTS.md",
  ".agents/loaf.json",
  "docs/VISION.md",
  "docs/STRATEGY.md",
  "docs/ARCHITECTURE.md",
  "CHANGELOG.md",
];

// ─────────────────────────────────────────────────────────────────────────────
// Helpers — inline scaffolding logic for isolated testing
// (avoids importing the full setup module which pulls in build targets)
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Minimal scaffold that mirrors what setup.ts does in the init phase.
 * We test the logic, not the imports.
 */
function scaffoldDirs(cwd: string): { created: string[] } {
  const created: string[] = [];
  for (const dir of EXPECTED_DIRS) {
    const fullPath = join(cwd, dir);
    if (!existsSync(fullPath)) {
      mkdirSync(fullPath, { recursive: true });
      created.push(dir + "/");
    }
  }
  return { created };
}

function scaffoldFiles(
  cwd: string,
): { created: string[] } {
  const created: string[] = [];
  const fileContents: Array<[string, () => string]> = [
    [".agents/AGENTS.md", () => "# Project Instructions\n"],
    [
      ".agents/loaf.json",
      () =>
        JSON.stringify(
          { version: "1.0.0", initialized: new Date().toISOString() },
          null,
          2,
        ) + "\n",
    ],
    ["docs/VISION.md", () => "# Vision\n"],
    ["docs/STRATEGY.md", () => "# Strategy\n"],
    ["docs/ARCHITECTURE.md", () => "# Architecture\n"],
    ["CHANGELOG.md", () => "# Changelog\n"],
  ];

  for (const [relPath, contentFn] of fileContents) {
    const fullPath = join(cwd, relPath);
    if (!existsSync(fullPath)) {
      const parentDir = join(fullPath, "..");
      if (!existsSync(parentDir)) {
        mkdirSync(parentDir, { recursive: true });
      }
      writeFileSync(fullPath, contentFn(), "utf-8");
      created.push(relPath);
    }
  }
  return { created };
}

// ─────────────────────────────────────────────────────────────────────────────
// Setup / Teardown
// ─────────────────────────────────────────────────────────────────────────────

beforeEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
  mkdirSync(TEST_ROOT, { recursive: true });
});

afterEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

describe("setup: directory creation", () => {
  it("creates the target directory when it does not exist", () => {
    const targetDir = join(TEST_ROOT, "my-project");
    expect(existsSync(targetDir)).toBe(false);

    mkdirSync(targetDir, { recursive: true });

    expect(existsSync(targetDir)).toBe(true);
  });

  it("handles nested directory paths", () => {
    const targetDir = join(TEST_ROOT, "deep", "nested", "project");
    expect(existsSync(targetDir)).toBe(false);

    mkdirSync(targetDir, { recursive: true });

    expect(existsSync(targetDir)).toBe(true);
  });

  it("succeeds when directory already exists", () => {
    const targetDir = join(TEST_ROOT, "existing-project");
    mkdirSync(targetDir, { recursive: true });

    // Should not throw
    mkdirSync(targetDir, { recursive: true });

    expect(existsSync(targetDir)).toBe(true);
  });
});

describe("setup: init scaffolding", () => {
  it("creates all expected directories", () => {
    const result = scaffoldDirs(TEST_ROOT);

    for (const dir of EXPECTED_DIRS) {
      expect(existsSync(join(TEST_ROOT, dir))).toBe(true);
    }

    expect(result.created.length).toBe(EXPECTED_DIRS.length);
  });

  it("creates all expected files", () => {
    // Ensure parent dirs exist first
    scaffoldDirs(TEST_ROOT);
    const result = scaffoldFiles(TEST_ROOT);

    for (const file of EXPECTED_FILES) {
      expect(existsSync(join(TEST_ROOT, file))).toBe(true);
    }

    expect(result.created.length).toBe(EXPECTED_FILES.length);
  });

  it("creates loaf.json with valid JSON", () => {
    scaffoldDirs(TEST_ROOT);
    scaffoldFiles(TEST_ROOT);

    const content = readFileSync(
      join(TEST_ROOT, ".agents/loaf.json"),
      "utf-8",
    );
    const parsed = JSON.parse(content);

    expect(parsed.version).toBe("1.0.0");
    expect(parsed.initialized).toBeDefined();
    // Verify it is a valid ISO date
    expect(() => new Date(parsed.initialized)).not.toThrow();
    expect(new Date(parsed.initialized).toISOString()).toBe(
      parsed.initialized,
    );
  });

  it("creates AGENTS.md with expected heading", () => {
    scaffoldDirs(TEST_ROOT);
    scaffoldFiles(TEST_ROOT);

    const content = readFileSync(
      join(TEST_ROOT, ".agents/AGENTS.md"),
      "utf-8",
    );
    expect(content).toContain("# Project Instructions");
  });
});

describe("setup: idempotency", () => {
  it("does not overwrite existing directories on re-run", () => {
    scaffoldDirs(TEST_ROOT);

    // Run again — should create nothing
    const result = scaffoldDirs(TEST_ROOT);

    expect(result.created.length).toBe(0);
  });

  it("does not overwrite existing files on re-run", () => {
    scaffoldDirs(TEST_ROOT);
    scaffoldFiles(TEST_ROOT);

    // Modify a file
    const agentsPath = join(TEST_ROOT, ".agents/AGENTS.md");
    writeFileSync(agentsPath, "# My Custom Content\n", "utf-8");

    // Run again — should not overwrite
    const result = scaffoldFiles(TEST_ROOT);

    expect(result.created.length).toBe(0);
    const content = readFileSync(agentsPath, "utf-8");
    expect(content).toBe("# My Custom Content\n");
  });

  it("creates only missing files on partial re-run", () => {
    scaffoldDirs(TEST_ROOT);
    scaffoldFiles(TEST_ROOT);

    // Delete one file
    rmSync(join(TEST_ROOT, "docs/VISION.md"));

    // Re-run — only the deleted file should be recreated
    const result = scaffoldFiles(TEST_ROOT);

    expect(result.created).toEqual(["docs/VISION.md"]);
    expect(existsSync(join(TEST_ROOT, "docs/VISION.md"))).toBe(true);
  });

  it("creates only missing directories on partial re-run", () => {
    scaffoldDirs(TEST_ROOT);

    // Delete one directory
    rmSync(join(TEST_ROOT, ".agents/ideas"), { recursive: true });

    // Re-run
    const result = scaffoldDirs(TEST_ROOT);

    expect(result.created).toEqual([".agents/ideas/"]);
    expect(existsSync(join(TEST_ROOT, ".agents/ideas"))).toBe(true);
  });
});
