/**
 * KB Staleness Detection
 *
 * Core innovation: detect when knowledge files are stale by checking if
 * the source code they `covers:` has changed since `last_reviewed`. Uses
 * git log to find commits affecting covered paths, and picomatch for
 * reverse lookups (given a file path, find covering knowledge files).
 */

import { execFileSync } from "child_process";
import picomatch from "picomatch";

import type { KbConfig, KnowledgeFile, StalenessResult } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Forward Lookup — is a knowledge file stale?
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Check whether a single knowledge file is stale.
 *
 * A file is stale when commits have been made to its `covers:` paths since
 * its `last_reviewed` date. Files without `covers:` cannot be checked and
 * return `{ isStale: false, hasCoverage: false }`.
 *
 * Uses a single `git log` invocation per file, passing all covers globs
 * as pathspec arguments.
 */
export function checkStaleness(
  gitRoot: string,
  file: KnowledgeFile,
  _config: KbConfig,
): StalenessResult {
  const covers = file.frontmatter.covers;

  // No covers field — can't determine staleness
  if (!covers || covers.length === 0) {
    return {
      file,
      isStale: false,
      hasCoverage: false,
      commitCount: 0,
    };
  }

  const lastReviewed = file.frontmatter.last_reviewed;

  try {
    const output = execFileSync(
      "git",
      [
        "log",
        `--since=${lastReviewed}`,
        "--format=%H%n%an%n%aI",
        "--",
        ...covers.map((g) => `:(glob)${g}`),
      ],
      {
        cwd: gitRoot,
        encoding: "utf-8",
        stdio: ["pipe", "pipe", "pipe"],
      },
    );

    const parsed = parseGitLogOutput(output);

    if (parsed.commitCount === 0) {
      return {
        file,
        isStale: false,
        hasCoverage: true,
        commitCount: 0,
      };
    }

    return {
      file,
      isStale: true,
      hasCoverage: true,
      commitCount: parsed.commitCount,
      lastCommitAuthor: parsed.lastAuthor,
      lastCommitDate: parsed.lastDate,
    };
  } catch {
    // git log can fail if the repo is corrupted, path doesn't exist, etc.
    // Treat as indeterminate — not stale, but has coverage
    return {
      file,
      isStale: false,
      hasCoverage: true,
      commitCount: 0,
    };
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Batch Check
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Run staleness checks on all knowledge files.
 *
 * Files without `covers:` get `{ isStale: false, hasCoverage: false }`.
 * Files with `covers:` are checked against git log.
 */
export function checkAllStaleness(
  gitRoot: string,
  files: KnowledgeFile[],
  config: KbConfig,
): StalenessResult[] {
  return files.map((file) => checkStaleness(gitRoot, file, config));
}

// ─────────────────────────────────────────────────────────────────────────────
// Reverse Lookup — which knowledge files cover a given path?
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Find all knowledge files whose `covers:` globs match the given file path.
 *
 * Uses picomatch for glob matching. The file path should be relative to
 * the git root (same coordinate system as covers globs).
 */
export function findCoveringFiles(
  files: KnowledgeFile[],
  filePath: string,
): KnowledgeFile[] {
  return files.filter((file) => {
    const covers = file.frontmatter.covers;
    if (!covers || covers.length === 0) return false;

    return covers.some((pattern) => {
      const matcher = picomatch(pattern);
      return matcher(filePath);
    });
  });
}

// ─────────────────────────────────────────────────────────────────────────────
// Git Log Parsing
// ─────────────────────────────────────────────────────────────────────────────

interface ParsedGitLog {
  commitCount: number;
  lastAuthor?: string;
  lastDate?: string;
}

/**
 * Parse git log output formatted with `--format=%H%n%an%n%aI`.
 *
 * Each commit produces 3 lines:
 *   - Commit hash
 *   - Author name
 *   - Author date (ISO 8601)
 *
 * The first commit in the output is the most recent.
 */
export function parseGitLogOutput(output: string): ParsedGitLog {
  const trimmed = output.trim();

  if (trimmed.length === 0) {
    return { commitCount: 0 };
  }

  const lines = trimmed.split("\n");

  // Each commit is 3 lines; ignore any trailing incomplete group
  const commitCount = Math.floor(lines.length / 3);

  if (commitCount === 0) {
    return { commitCount: 0 };
  }

  // First commit in output is the most recent
  const lastAuthor = lines[1];
  const lastDate = lines[2];

  return {
    commitCount,
    lastAuthor,
    lastDate,
  };
}
