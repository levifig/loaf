/**
 * Knowledge File Validation
 *
 * Validates knowledge file frontmatter for errors (missing required fields,
 * bad date formats) and warnings (unresolvable globs, broken depends_on,
 * unrecognized implementation_status).
 *
 * Returns pure data — the command layer formats output.
 */

import { execFileSync } from "child_process";
import { existsSync, readdirSync, readFileSync } from "fs";
import { dirname, join, relative } from "path";
import matter from "gray-matter";

import type {
  KbConfig,
  KnowledgeFile,
  KnowledgeFrontmatter,
  ValidationResult,
  ValidationIssue,
} from "./types.js";
import { IMPLEMENTATION_STATUSES, type ImplementationStatus } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Validate already-loaded knowledge files for warnings.
 *
 * These files passed the loader's requirements (have topics, etc.) so we
 * only check for soft issues: unresolvable globs, broken depends_on,
 * unrecognized implementation_status.
 */
export function validateLoadedFiles(
  gitRoot: string,
  files: KnowledgeFile[],
): ValidationResult[] {
  return files.map((file) => validateLoadedFile(gitRoot, file));
}

/**
 * Scan configured directories for .md files that have YAML frontmatter but
 * were skipped by the loader (missing required fields). Returns
 * ValidationResult objects with errors for the missing fields.
 *
 * This catches files that authors intended to be knowledge files but have
 * incomplete frontmatter.
 */
export function findSkippedFiles(
  gitRoot: string,
  config: KbConfig,
): ValidationResult[] {
  const results: ValidationResult[] = [];

  for (const dir of config.local) {
    const absDir = join(gitRoot, dir);

    if (!existsSync(absDir)) continue;

    let entries: string[];
    try {
      entries = readdirSync(absDir);
    } catch {
      continue;
    }

    for (const entry of entries) {
      if (!entry.endsWith(".md")) continue;

      const absPath = join(absDir, entry);
      const relPath = relative(gitRoot, absPath);

      try {
        const raw = readFileSync(absPath, "utf-8");
        const { data, content } = matter(raw);

        // Skip files without any YAML frontmatter — these aren't knowledge files
        if (!data || typeof data !== "object" || Object.keys(data).length === 0) {
          continue;
        }

        // Check for missing/empty topics
        const hasTopics = Array.isArray(data.topics) && data.topics.length > 0;

        // Check for missing/empty last_reviewed
        const hasLastReviewed = data.last_reviewed !== undefined && data.last_reviewed !== null && data.last_reviewed !== "";

        // If the loader would have accepted this file, skip it — it's validated elsewhere
        if (hasTopics && hasLastReviewed) continue;

        const errors: ValidationIssue[] = [];

        if (!data.topics) {
          errors.push({ field: "topics", message: "Missing required field" });
        } else if (!Array.isArray(data.topics)) {
          errors.push({ field: "topics", message: `Must be an array, got ${typeof data.topics}: "${data.topics}"` });
        } else if (data.topics.length === 0) {
          errors.push({ field: "topics", message: "Must contain at least one topic" });
        }

        if (!hasLastReviewed) {
          errors.push({ field: "last_reviewed", message: "Missing required field" });
        }

        // Build a minimal KnowledgeFile for reporting
        const frontmatter: KnowledgeFrontmatter = {
          topics: Array.isArray(data.topics) ? data.topics : [],
          last_reviewed: typeof data.last_reviewed === "string" ? data.last_reviewed : "",
          covers: Array.isArray(data.covers) ? data.covers : undefined,
          depends_on: Array.isArray(data.depends_on) ? data.depends_on : undefined,
          implementation_status: undefined,
        };

        results.push({
          file: { path: absPath, relativePath: relPath, frontmatter, content },
          errors,
          warnings: [],
        });
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        results.push({
          file: {
            path: absPath,
            relativePath: relPath,
            frontmatter: { topics: [], last_reviewed: "" },
            content: "",
          },
          errors: [{ field: "frontmatter", message: `Failed to parse: ${message}` }],
          warnings: [],
        });
      }
    }
  }

  return results;
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Validate a single loaded knowledge file for warnings.
 */
function validateLoadedFile(gitRoot: string, file: KnowledgeFile): ValidationResult {
  const warnings: ValidationIssue[] = [];
  const errors: ValidationIssue[] = [];
  const fm = file.frontmatter;

  // Check last_reviewed date is valid (strict YYYY-MM-DD with round-trip check)
  if (fm.last_reviewed) {
    const dateStr = fm.last_reviewed;
    const dateMatch = /^\d{4}-\d{2}-\d{2}$/.test(dateStr);
    if (!dateMatch) {
      errors.push({ field: "last_reviewed", message: `Invalid date format (expected YYYY-MM-DD): "${dateStr}"` });
    } else {
      // Round-trip check: reject impossible dates like 2026-02-30
      const parsed = new Date(dateStr + "T00:00:00Z");
      const roundTrip = parsed.toISOString().slice(0, 10);
      if (roundTrip !== dateStr) {
        errors.push({ field: "last_reviewed", message: `Invalid calendar date: "${dateStr}"` });
      }
    }
  }

  // Check covers globs resolve to tracked files
  if (fm.covers) {
    for (const glob of fm.covers) {
      if (!globMatchesTrackedFiles(gitRoot, glob)) {
        warnings.push({
          field: "covers",
          message: `Glob pattern matches no tracked files: "${glob}"`,
        });
      }
    }
  }

  // Check depends_on references exist (resolve from git root or file's directory)
  if (fm.depends_on) {
    for (const dep of fm.depends_on) {
      const fromRoot = join(gitRoot, dep);
      const fromFileDir = join(dirname(file.path), dep);
      if (!existsSync(fromRoot) && !existsSync(fromFileDir)) {
        warnings.push({
          field: "depends_on",
          message: `Referenced file does not exist: "${dep}"`,
        });
      }
    }
  }

  // Check implementation_status is recognized
  if (fm.implementation_status !== undefined) {
    if (!IMPLEMENTATION_STATUSES.includes(fm.implementation_status as ImplementationStatus)) {
      warnings.push({
        field: "implementation_status",
        message: `Unrecognized value: "${fm.implementation_status}"`,
      });
    }
  }

  return { file, errors, warnings };
}

/**
 * Check if a glob pattern matches any git-tracked files.
 * Uses `git ls-files` which respects .gitignore and only returns tracked files.
 */
function globMatchesTrackedFiles(gitRoot: string, glob: string): boolean {
  try {
    const result = execFileSync("git", ["ls-files", "--", glob], {
      cwd: gitRoot,
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return result.trim().length > 0;
  } catch {
    return false;
  }
}
