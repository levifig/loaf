/**
 * KB Validation Tests
 *
 * Tests for validateLoadedFiles() and findSkippedFiles() — checking
 * frontmatter for errors and warnings.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mkdirSync, writeFileSync, rmSync } from "fs";
import { join } from "path";

import { validateLoadedFiles, findSkippedFiles } from "./validate.js";
import type { KnowledgeFile, KnowledgeFrontmatter } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

/** Build a minimal KnowledgeFile for testing */
function makeFile(overrides: Partial<KnowledgeFrontmatter> = {}, relPath = "docs/knowledge/test.md"): KnowledgeFile {
  const frontmatter: KnowledgeFrontmatter = {
    topics: ["testing"],
    last_reviewed: "2026-03-01",
    ...overrides,
  };

  return {
    path: `/fake/root/${relPath}`,
    relativePath: relPath,
    frontmatter,
    content: "# Test\n\nBody content.",
  };
}

const FAKE_GIT_ROOT = "/fake/root";

// ─────────────────────────────────────────────────────────────────────────────
// Mock child_process.execFileSync for git ls-files
// ─────────────────────────────────────────────────────────────────────────────

vi.mock("child_process", () => ({
  execFileSync: vi.fn((_cmd: string, args: string[]) => {
    // Simulate git ls-files: return empty for "nonexistent/**", non-empty for others
    const glob = args[args.length - 1];
    if (glob.includes("nonexistent") || glob.includes("missing")) {
      return "";
    }
    return "some/file.ts\n";
  }),
}));

// ─────────────────────────────────────────────────────────────────────────────
// Mock fs.existsSync for depends_on checks
// ─────────────────────────────────────────────────────────────────────────────

vi.mock("fs", async () => {
  const actual = await vi.importActual("fs");
  return {
    ...actual,
    existsSync: vi.fn((path: string) => {
      // Only return false for paths containing "nonexistent" or "broken"
      if (typeof path === "string" && (path.includes("nonexistent") || path.includes("broken"))) {
        return false;
      }
      return true;
    }),
  };
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: validateLoadedFiles
// ─────────────────────────────────────────────────────────────────────────────

describe("validateLoadedFiles", () => {
  it("returns no errors or warnings for valid frontmatter", () => {
    const file = makeFile({
      topics: ["api", "rest"],
      last_reviewed: "2026-03-01",
      covers: ["cli/**/*.ts"],
      depends_on: ["docs/knowledge/other.md"],
      implementation_status: "stable",
    });

    const results = validateLoadedFiles(FAKE_GIT_ROOT, [file]);

    expect(results).toHaveLength(1);
    expect(results[0].errors).toHaveLength(0);
    expect(results[0].warnings).toHaveLength(0);
  });

  it("produces an error for invalid last_reviewed date", () => {
    const file = makeFile({ last_reviewed: "not-a-date" });

    const results = validateLoadedFiles(FAKE_GIT_ROOT, [file]);

    expect(results).toHaveLength(1);
    expect(results[0].errors).toHaveLength(1);
    expect(results[0].errors[0].field).toBe("last_reviewed");
    expect(results[0].errors[0].message).toContain("Invalid date format");
  });

  it("produces a warning for covers glob matching nothing", () => {
    const file = makeFile({ covers: ["nonexistent/**/*.ts"] });

    const results = validateLoadedFiles(FAKE_GIT_ROOT, [file]);

    expect(results).toHaveLength(1);
    expect(results[0].warnings).toHaveLength(1);
    expect(results[0].warnings[0].field).toBe("covers");
    expect(results[0].warnings[0].message).toContain("matches no tracked files");
  });

  it("produces a warning for broken depends_on reference", () => {
    const file = makeFile({ depends_on: ["docs/knowledge/broken-ref.md"] });

    const results = validateLoadedFiles(FAKE_GIT_ROOT, [file]);

    expect(results).toHaveLength(1);
    expect(results[0].warnings).toHaveLength(1);
    expect(results[0].warnings[0].field).toBe("depends_on");
    expect(results[0].warnings[0].message).toContain("does not exist");
  });

  it("produces a warning for unrecognized implementation_status", () => {
    // The loader normalizes unrecognized values to undefined, but if
    // somehow an invalid value slips through, the validator catches it.
    // We test by directly setting a bad value via type assertion.
    const file = makeFile({
      implementation_status: "banana" as KnowledgeFrontmatter["implementation_status"],
    });

    const results = validateLoadedFiles(FAKE_GIT_ROOT, [file]);

    expect(results).toHaveLength(1);
    expect(results[0].warnings).toHaveLength(1);
    expect(results[0].warnings[0].field).toBe("implementation_status");
    expect(results[0].warnings[0].message).toContain("Unrecognized value");
  });

  it("does not warn for valid covers globs", () => {
    const file = makeFile({ covers: ["cli/**/*.ts", "config/*.yaml"] });

    const results = validateLoadedFiles(FAKE_GIT_ROOT, [file]);

    expect(results).toHaveLength(1);
    expect(results[0].warnings).toHaveLength(0);
  });

  it("validates multiple files independently", () => {
    const good = makeFile({}, "docs/knowledge/good.md");
    const bad = makeFile({ covers: ["missing/**"] }, "docs/knowledge/bad.md");

    const results = validateLoadedFiles(FAKE_GIT_ROOT, [good, bad]);

    expect(results).toHaveLength(2);

    const goodResult = results.find((r) => r.file.relativePath.includes("good"));
    const badResult = results.find((r) => r.file.relativePath.includes("bad"));

    expect(goodResult?.warnings).toHaveLength(0);
    expect(badResult?.warnings).toHaveLength(1);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: findSkippedFiles
// ─────────────────────────────────────────────────────────────────────────────

// findSkippedFiles reads actual files from disk, so we use the same temp
// directory pattern as loader.test.ts.

describe("findSkippedFiles", () => {
  const TEST_ROOT = join(process.cwd(), ".test-kb-validate");

  function writeKbFile(dir: string, name: string, frontmatter: string, body = ""): void {
    const dirPath = join(TEST_ROOT, dir);
    mkdirSync(dirPath, { recursive: true });
    writeFileSync(join(dirPath, name), `---\n${frontmatter}\n---\n${body}`, "utf-8");
  }

  beforeEach(() => {
    mkdirSync(TEST_ROOT, { recursive: true });
  });

  afterEach(() => {
    rmSync(TEST_ROOT, { recursive: true, force: true });
  });

  it("detects files missing topics", () => {
    writeKbFile("kb", "no-topics.md", 'last_reviewed: "2026-03-01"\ncovers:\n  - src/**');

    const results = findSkippedFiles(TEST_ROOT, {
      local: ["kb"],
      staleness_threshold_days: 30,
      imports: [],
    });

    expect(results).toHaveLength(1);
    expect(results[0].errors.some((e) => e.field === "topics")).toBe(true);
  });

  it("detects files with empty topics array", () => {
    writeKbFile("kb", "empty-topics.md", 'topics: []\nlast_reviewed: "2026-03-01"');

    const results = findSkippedFiles(TEST_ROOT, {
      local: ["kb"],
      staleness_threshold_days: 30,
      imports: [],
    });

    expect(results).toHaveLength(1);
    expect(results[0].errors.some((e) => e.field === "topics")).toBe(true);
  });

  it("detects files missing last_reviewed", () => {
    writeKbFile("kb", "no-reviewed.md", "topics:\n  - testing");

    const results = findSkippedFiles(TEST_ROOT, {
      local: ["kb"],
      staleness_threshold_days: 30,
      imports: [],
    });

    expect(results).toHaveLength(1);
    expect(results[0].errors.some((e) => e.field === "last_reviewed")).toBe(true);
  });

  it("skips files with no frontmatter", () => {
    const dirPath = join(TEST_ROOT, "kb");
    mkdirSync(dirPath, { recursive: true });
    writeFileSync(join(dirPath, "readme.md"), "# Just a readme\n\nNo frontmatter here.", "utf-8");

    const results = findSkippedFiles(TEST_ROOT, {
      local: ["kb"],
      staleness_threshold_days: 30,
      imports: [],
    });

    expect(results).toHaveLength(0);
  });

  it("skips valid files that the loader would accept", () => {
    writeKbFile("kb", "valid.md", 'topics:\n  - testing\nlast_reviewed: "2026-03-01"');

    const results = findSkippedFiles(TEST_ROOT, {
      local: ["kb"],
      staleness_threshold_days: 30,
      imports: [],
    });

    expect(results).toHaveLength(0);
  });
});
