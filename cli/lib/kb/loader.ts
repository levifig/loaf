/**
 * Knowledge File Loader
 *
 * Scans configured directories for .md files with knowledge frontmatter,
 * parses them via gray-matter, and returns KnowledgeFile objects.
 * Uses the same frontmatter parsing approach as cli/lib/tasks/parser.ts.
 */

import { existsSync, readdirSync, readFileSync } from "fs";
import { join, relative } from "path";
import matter from "gray-matter";

import type { KbConfig, KnowledgeFile, KnowledgeFrontmatter } from "./types.js";

// ANSI color helpers (matching project conventions)
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;

/**
 * Load all knowledge files from directories configured in KbConfig.
 *
 * For each directory in `config.local`:
 * - Resolves the path relative to `gitRoot`
 * - Scans for .md files (non-recursive)
 * - Parses YAML frontmatter via gray-matter
 * - Skips files without frontmatter or without a `topics` field
 * - Returns KnowledgeFile objects with absolute and relative paths
 *
 * Missing directories are skipped with a warning (not an error).
 */
export function loadKnowledgeFiles(
  gitRoot: string,
  config: KbConfig,
): KnowledgeFile[] {
  const files: KnowledgeFile[] = [];

  for (const dir of config.local) {
    const absDir = join(gitRoot, dir);

    if (!existsSync(absDir)) {
      console.warn(`  ${yellow("warn:")} KB directory not found: ${dir}`);
      continue;
    }

    let entries: string[];
    try {
      entries = readdirSync(absDir);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.warn(`  ${yellow("warn:")} Failed to read directory ${dir}: ${message}`);
      continue;
    }

    for (const entry of entries) {
      if (!entry.endsWith(".md")) continue;

      const absPath = join(absDir, entry);
      const relPath = relative(gitRoot, absPath);

      try {
        const raw = readFileSync(absPath, "utf-8");
        const { data, content } = matter(raw);

        // Skip files without frontmatter or without topics
        if (!data || typeof data !== "object") continue;
        if (!Array.isArray(data.topics) || data.topics.length === 0) continue;

        const frontmatter: KnowledgeFrontmatter = {
          topics: data.topics,
          last_reviewed: normalizeDate(data.last_reviewed),
          covers: Array.isArray(data.covers) ? data.covers : undefined,
          consumers: Array.isArray(data.consumers) ? data.consumers : undefined,
          depends_on: Array.isArray(data.depends_on) ? data.depends_on : undefined,
          implementation_status: normalizeImplementationStatus(data.implementation_status),
        };

        files.push({
          path: absPath,
          relativePath: relPath,
          frontmatter,
          content,
        });
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.warn(`  ${yellow("warn:")} Failed to parse ${relPath}: ${message}`);
      }
    }
  }

  return files;
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Normalize a date value to an ISO 8601 date string.
 * Handles Date objects (gray-matter auto-parsing), ISO strings, bare dates.
 * Preserves invalid/missing values as-is so downstream validation can flag them.
 */
function normalizeDate(value: unknown): string {
  if (!value) return "";

  if (value instanceof Date) {
    return value.toISOString().slice(0, 10);
  }

  if (typeof value === "string") {
    const trimmed = value.trim();

    // Already a bare date
    if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
      return trimmed;
    }

    // ISO timestamp — extract date portion
    if (trimmed.includes("T")) {
      return trimmed.slice(0, 10);
    }

    // Try parsing
    const parsed = new Date(trimmed);
    if (!isNaN(parsed.getTime())) {
      return parsed.toISOString().slice(0, 10);
    }
  }

  // Preserve the raw value so validate can report it
  return String(value);
}

/**
 * Normalize implementation_status, preserving invalid values for validation.
 */
function normalizeImplementationStatus(
  value: unknown,
): string | undefined {
  if (typeof value !== "string") return undefined;
  return value.trim().toLowerCase();
}
