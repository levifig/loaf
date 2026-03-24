/**
 * KB Staleness Detection Tests
 *
 * Tests for checkStaleness(), checkAllStaleness(), findCoveringFiles(),
 * and parseGitLogOutput(). Mocks execFileSync for git log output.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";

import type { KbConfig, KnowledgeFile, KnowledgeFrontmatter } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Mock child_process
// ─────────────────────────────────────────────────────────────────────────────

vi.mock("child_process", () => ({
  execFileSync: vi.fn(),
}));

import { execFileSync } from "child_process";
import {
  checkStaleness,
  checkAllStaleness,
  findCoveringFiles,
  parseGitLogOutput,
} from "./staleness.js";

const mockExecFileSync = vi.mocked(execFileSync);

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

const FAKE_GIT_ROOT = "/fake/root";

const DEFAULT_CONFIG: KbConfig = {
  local: ["docs/knowledge"],
  staleness_threshold_days: 30,
  imports: [],
};

/** Build a minimal KnowledgeFile for testing */
function makeFile(
  overrides: Partial<KnowledgeFrontmatter> = {},
  relPath = "docs/knowledge/test.md",
): KnowledgeFile {
  const frontmatter: KnowledgeFrontmatter = {
    topics: ["testing"],
    last_reviewed: "2026-03-01",
    ...overrides,
  };

  return {
    path: `${FAKE_GIT_ROOT}/${relPath}`,
    relativePath: relPath,
    frontmatter,
    content: "# Test\n\nBody content.",
  };
}

/** Sample git log output: 2 commits */
const TWO_COMMITS_OUTPUT = [
  "abc123def456789012345678901234567890abcd",
  "Alice Developer",
  "2026-03-15T10:30:00+00:00",
  "def456abc789012345678901234567890abcdef12",
  "Bob Reviewer",
  "2026-03-10T08:15:00+00:00",
].join("\n");

/** Sample git log output: 1 commit */
const ONE_COMMIT_OUTPUT = [
  "abc123def456789012345678901234567890abcd",
  "Alice Developer",
  "2026-03-15T10:30:00+00:00",
].join("\n");

// ─────────────────────────────────────────────────────────────────────────────
// Tests: parseGitLogOutput
// ─────────────────────────────────────────────────────────────────────────────

describe("parseGitLogOutput", () => {
  it("parses empty output as zero commits", () => {
    const result = parseGitLogOutput("");
    expect(result.commitCount).toBe(0);
    expect(result.lastAuthor).toBeUndefined();
    expect(result.lastDate).toBeUndefined();
  });

  it("parses whitespace-only output as zero commits", () => {
    const result = parseGitLogOutput("   \n  \n  ");
    expect(result.commitCount).toBe(0);
  });

  it("parses a single commit correctly", () => {
    const result = parseGitLogOutput(ONE_COMMIT_OUTPUT);
    expect(result.commitCount).toBe(1);
    expect(result.lastAuthor).toBe("Alice Developer");
    expect(result.lastDate).toBe("2026-03-15T10:30:00+00:00");
  });

  it("parses multiple commits and returns the most recent author/date", () => {
    const result = parseGitLogOutput(TWO_COMMITS_OUTPUT);
    expect(result.commitCount).toBe(2);
    expect(result.lastAuthor).toBe("Alice Developer");
    expect(result.lastDate).toBe("2026-03-15T10:30:00+00:00");
  });

  it("ignores trailing incomplete commit groups", () => {
    // 4 lines = 1 full group + 1 leftover line
    const partial = ONE_COMMIT_OUTPUT + "\nextra-partial-line";
    const result = parseGitLogOutput(partial);
    expect(result.commitCount).toBe(1);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: checkStaleness
// ─────────────────────────────────────────────────────────────────────────────

describe("checkStaleness", () => {
  beforeEach(() => {
    mockExecFileSync.mockReset();
  });

  it("returns not stale with hasCoverage=false when file has no covers", () => {
    const file = makeFile({ covers: undefined });

    const result = checkStaleness(FAKE_GIT_ROOT, file, DEFAULT_CONFIG);

    expect(result.isStale).toBe(false);
    expect(result.hasCoverage).toBe(false);
    expect(result.commitCount).toBe(0);
    // Should not call git at all
    expect(mockExecFileSync).not.toHaveBeenCalled();
  });

  it("returns not stale with hasCoverage=false when covers is empty array", () => {
    const file = makeFile({ covers: [] });

    const result = checkStaleness(FAKE_GIT_ROOT, file, DEFAULT_CONFIG);

    expect(result.isStale).toBe(false);
    expect(result.hasCoverage).toBe(false);
    expect(mockExecFileSync).not.toHaveBeenCalled();
  });

  it("returns stale when commits exist since last_reviewed", () => {
    const file = makeFile({
      covers: ["cli/**/*.ts"],
      last_reviewed: "2026-03-01",
    });

    mockExecFileSync.mockReturnValue(TWO_COMMITS_OUTPUT);

    const result = checkStaleness(FAKE_GIT_ROOT, file, DEFAULT_CONFIG);

    expect(result.isStale).toBe(true);
    expect(result.hasCoverage).toBe(true);
    expect(result.commitCount).toBe(2);
    expect(result.lastCommitAuthor).toBe("Alice Developer");
    expect(result.lastCommitDate).toBe("2026-03-15T10:30:00+00:00");
  });

  it("returns fresh when no commits exist since last_reviewed", () => {
    const file = makeFile({
      covers: ["cli/**/*.ts"],
      last_reviewed: "2026-03-20",
    });

    mockExecFileSync.mockReturnValue("");

    const result = checkStaleness(FAKE_GIT_ROOT, file, DEFAULT_CONFIG);

    expect(result.isStale).toBe(false);
    expect(result.hasCoverage).toBe(true);
    expect(result.commitCount).toBe(0);
    expect(result.lastCommitAuthor).toBeUndefined();
    expect(result.lastCommitDate).toBeUndefined();
  });

  it("passes all covers globs as pathspec args to git log", () => {
    const file = makeFile({
      covers: ["cli/**/*.ts", "config/**/*.yaml", "docs/**"],
      last_reviewed: "2026-03-01",
    });

    mockExecFileSync.mockReturnValue("");

    checkStaleness(FAKE_GIT_ROOT, file, DEFAULT_CONFIG);

    expect(mockExecFileSync).toHaveBeenCalledWith(
      "git",
      [
        "log",
        "--since=2026-03-01",
        "--format=%H%n%an%n%aI",
        "--",
        "cli/**/*.ts",
        "config/**/*.yaml",
        "docs/**",
      ],
      expect.objectContaining({
        cwd: FAKE_GIT_ROOT,
        encoding: "utf-8",
      }),
    );
  });

  it("handles git log failure gracefully (returns fresh)", () => {
    const file = makeFile({
      covers: ["broken-path/**"],
      last_reviewed: "2026-03-01",
    });

    mockExecFileSync.mockImplementation(() => {
      throw new Error("fatal: bad revision");
    });

    const result = checkStaleness(FAKE_GIT_ROOT, file, DEFAULT_CONFIG);

    expect(result.isStale).toBe(false);
    expect(result.hasCoverage).toBe(true);
    expect(result.commitCount).toBe(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: checkAllStaleness
// ─────────────────────────────────────────────────────────────────────────────

describe("checkAllStaleness", () => {
  beforeEach(() => {
    mockExecFileSync.mockReset();
  });

  it("checks all files and returns results in order", () => {
    const staleFile = makeFile(
      { covers: ["src/**"], last_reviewed: "2026-01-01" },
      "docs/knowledge/stale.md",
    );
    const freshFile = makeFile(
      { covers: ["lib/**"], last_reviewed: "2026-03-20" },
      "docs/knowledge/fresh.md",
    );
    const noCoverFile = makeFile(
      { covers: undefined },
      "docs/knowledge/nocover.md",
    );

    // First call (staleFile) returns commits, second call (freshFile) returns empty
    mockExecFileSync
      .mockReturnValueOnce(ONE_COMMIT_OUTPUT)
      .mockReturnValueOnce("");

    const results = checkAllStaleness(
      FAKE_GIT_ROOT,
      [staleFile, freshFile, noCoverFile],
      DEFAULT_CONFIG,
    );

    expect(results).toHaveLength(3);
    expect(results[0].isStale).toBe(true);
    expect(results[0].hasCoverage).toBe(true);
    expect(results[1].isStale).toBe(false);
    expect(results[1].hasCoverage).toBe(true);
    expect(results[2].isStale).toBe(false);
    expect(results[2].hasCoverage).toBe(false);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: findCoveringFiles
// ─────────────────────────────────────────────────────────────────────────────

describe("findCoveringFiles", () => {
  it("returns files whose covers globs match the given path", () => {
    const cliFile = makeFile(
      { covers: ["cli/**/*.ts"] },
      "docs/knowledge/cli-guide.md",
    );
    const configFile = makeFile(
      { covers: ["config/**/*.yaml"] },
      "docs/knowledge/config-guide.md",
    );

    const matches = findCoveringFiles(
      [cliFile, configFile],
      "cli/lib/kb/staleness.ts",
    );

    expect(matches).toHaveLength(1);
    expect(matches[0].relativePath).toBe("docs/knowledge/cli-guide.md");
  });

  it("returns empty array when no covers match", () => {
    const file = makeFile(
      { covers: ["backend/**"] },
      "docs/knowledge/backend.md",
    );

    const matches = findCoveringFiles([file], "frontend/app.tsx");

    expect(matches).toHaveLength(0);
  });

  it("returns multiple files when several cover the same path", () => {
    const broadFile = makeFile(
      { covers: ["cli/**"] },
      "docs/knowledge/broad.md",
    );
    const specificFile = makeFile(
      { covers: ["cli/lib/kb/**"] },
      "docs/knowledge/specific.md",
    );
    const unrelatedFile = makeFile(
      { covers: ["docs/**"] },
      "docs/knowledge/unrelated.md",
    );

    const matches = findCoveringFiles(
      [broadFile, specificFile, unrelatedFile],
      "cli/lib/kb/staleness.ts",
    );

    expect(matches).toHaveLength(2);
    const paths = matches.map((m) => m.relativePath).sort();
    expect(paths).toEqual([
      "docs/knowledge/broad.md",
      "docs/knowledge/specific.md",
    ]);
  });

  it("skips files without covers field", () => {
    const withCovers = makeFile(
      { covers: ["cli/**"] },
      "docs/knowledge/with.md",
    );
    const withoutCovers = makeFile(
      { covers: undefined },
      "docs/knowledge/without.md",
    );

    const matches = findCoveringFiles(
      [withCovers, withoutCovers],
      "cli/index.ts",
    );

    expect(matches).toHaveLength(1);
    expect(matches[0].relativePath).toBe("docs/knowledge/with.md");
  });

  it("skips files with empty covers array", () => {
    const file = makeFile({ covers: [] }, "docs/knowledge/empty.md");

    const matches = findCoveringFiles([file], "cli/index.ts");

    expect(matches).toHaveLength(0);
  });

  it("matches glob patterns with extensions", () => {
    const file = makeFile(
      { covers: ["**/*.yaml"] },
      "docs/knowledge/config.md",
    );

    expect(findCoveringFiles([file], "config/hooks.yaml")).toHaveLength(1);
    expect(findCoveringFiles([file], "config/hooks.ts")).toHaveLength(0);
  });
});
