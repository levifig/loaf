/**
 * KB Loader Tests
 *
 * Tests for loadKnowledgeFiles() — scanning directories and parsing
 * knowledge file frontmatter.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdirSync, writeFileSync, rmSync, existsSync } from "fs";
import { join } from "path";

import { loadKnowledgeFiles } from "./loader.js";
import type { KbConfig } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Test Fixtures
// ─────────────────────────────────────────────────────────────────────────────

const TEST_ROOT = join(process.cwd(), ".test-kb-loader");

const DEFAULT_CONFIG: KbConfig = {
  local: ["kb-a", "kb-b"],
  staleness_threshold_days: 30,
  imports: [],
};

/** Create a .md file with frontmatter in a test directory */
function writeKbFile(dir: string, name: string, frontmatter: string, body: string = ""): void {
  const dirPath = join(TEST_ROOT, dir);
  if (!existsSync(dirPath)) {
    mkdirSync(dirPath, { recursive: true });
  }
  const content = `---\n${frontmatter}\n---\n${body}`;
  writeFileSync(join(dirPath, name), content, "utf-8");
}

/** Create a plain .md file without frontmatter */
function writePlainFile(dir: string, name: string, content: string): void {
  const dirPath = join(TEST_ROOT, dir);
  if (!existsSync(dirPath)) {
    mkdirSync(dirPath, { recursive: true });
  }
  writeFileSync(join(dirPath, name), content, "utf-8");
}

// ─────────────────────────────────────────────────────────────────────────────
// Setup / Teardown
// ─────────────────────────────────────────────────────────────────────────────

beforeEach(() => {
  mkdirSync(TEST_ROOT, { recursive: true });
});

afterEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

describe("loadKnowledgeFiles", () => {
  it("loads files with valid frontmatter", () => {
    writeKbFile("kb-a", "api-design.md",
      'topics:\n  - api\n  - rest\nlast_reviewed: "2026-03-01"',
      "\n# API Design\n\nGuidance for REST APIs.",
    );

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(1);
    expect(files[0].frontmatter.topics).toEqual(["api", "rest"]);
    expect(files[0].frontmatter.last_reviewed).toBe("2026-03-01");
    expect(files[0].content).toContain("API Design");
    expect(files[0].path).toBe(join(TEST_ROOT, "kb-a", "api-design.md"));
    expect(files[0].relativePath).toBe(join("kb-a", "api-design.md"));
  });

  it("skips files without frontmatter (like README.md)", () => {
    writePlainFile("kb-a", "README.md", "# Knowledge Base\n\nJust a readme.");

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(0);
  });

  it("skips files without topics field", () => {
    writeKbFile("kb-a", "no-topics.md",
      'last_reviewed: "2026-03-01"\ncovers:\n  - src/**',
    );

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(0);
  });

  it("skips files with empty topics array", () => {
    writeKbFile("kb-a", "empty-topics.md",
      'topics: []\nlast_reviewed: "2026-03-01"',
    );

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(0);
  });

  it("handles missing directories gracefully", () => {
    // kb-a and kb-b don't exist — should warn but not throw
    const files = loadKnowledgeFiles(TEST_ROOT, {
      ...DEFAULT_CONFIG,
      local: ["nonexistent-dir"],
    });

    expect(files).toHaveLength(0);
  });

  it("scans multiple directories", () => {
    writeKbFile("kb-a", "file-a.md",
      'topics:\n  - alpha\nlast_reviewed: "2026-03-01"',
    );
    writeKbFile("kb-b", "file-b.md",
      'topics:\n  - beta\nlast_reviewed: "2026-02-15"',
    );

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(2);

    const topics = files.map((f) => f.frontmatter.topics[0]).sort();
    expect(topics).toEqual(["alpha", "beta"]);
  });

  it("returns correct absolute and relative paths", () => {
    writeKbFile("kb-a", "paths-test.md",
      'topics:\n  - testing\nlast_reviewed: "2026-03-01"',
    );

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(1);
    expect(files[0].path).toBe(join(TEST_ROOT, "kb-a", "paths-test.md"));
    expect(files[0].relativePath).toBe(join("kb-a", "paths-test.md"));
  });

  it("parses optional frontmatter fields", () => {
    writeKbFile("kb-a", "full.md", [
      "topics:",
      "  - deployment",
      'last_reviewed: "2026-03-10"',
      "covers:",
      '  - "infra/**"',
      '  - "k8s/**"',
      "consumers:",
      "  - backend-dev",
      "depends_on:",
      "  - docs/knowledge/networking.md",
      "implementation_status: stable",
    ].join("\n"));

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(1);
    const fm = files[0].frontmatter;
    expect(fm.covers).toEqual(["infra/**", "k8s/**"]);
    expect(fm.consumers).toEqual(["backend-dev"]);
    expect(fm.depends_on).toEqual(["docs/knowledge/networking.md"]);
    expect(fm.implementation_status).toBe("stable");
  });

  it("ignores non-.md files", () => {
    writeKbFile("kb-a", "valid.md",
      'topics:\n  - test\nlast_reviewed: "2026-03-01"',
    );

    // Write a .txt file that would match frontmatter if parsed
    const dirPath = join(TEST_ROOT, "kb-a");
    writeFileSync(join(dirPath, "notes.txt"), "---\ntopics:\n  - test\nlast_reviewed: 2026-03-01\n---\n", "utf-8");

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(1);
    expect(files[0].relativePath).toBe(join("kb-a", "valid.md"));
  });

  it("normalizes implementation_status to valid values", () => {
    writeKbFile("kb-a", "deprecated.md",
      'topics:\n  - legacy\nlast_reviewed: "2026-01-01"\nimplementation_status: deprecated',
    );
    writeKbFile("kb-a", "invalid-status.md",
      'topics:\n  - misc\nlast_reviewed: "2026-01-01"\nimplementation_status: banana',
    );

    const files = loadKnowledgeFiles(TEST_ROOT, DEFAULT_CONFIG);

    expect(files).toHaveLength(2);

    const deprecated = files.find((f) => f.relativePath.includes("deprecated"));
    const invalid = files.find((f) => f.relativePath.includes("invalid-status"));

    expect(deprecated?.frontmatter.implementation_status).toBe("deprecated");
    expect(invalid?.frontmatter.implementation_status).toBeUndefined();
  });
});
