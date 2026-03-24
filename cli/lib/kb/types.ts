/**
 * Knowledge Base Types
 *
 * Type definitions for the knowledge management system. Knowledge files
 * are .md files with YAML frontmatter containing topic metadata, staleness
 * tracking, and coverage hints. The KB config in .agents/loaf.json defines
 * which directories to scan and staleness thresholds.
 */

// ─────────────────────────────────────────────────────────────────────────────
// Implementation Status
// ─────────────────────────────────────────────────────────────────────────────

/** Lifecycle status of a knowledge file */
export type ImplementationStatus = "in-progress" | "stable" | "deprecated";

export const IMPLEMENTATION_STATUSES: ImplementationStatus[] = [
  "in-progress",
  "stable",
  "deprecated",
];

// ─────────────────────────────────────────────────────────────────────────────
// Knowledge File Frontmatter
// ─────────────────────────────────────────────────────────────────────────────

/** YAML frontmatter schema for knowledge .md files */
export interface KnowledgeFrontmatter {
  /** Topic tags for routing and discovery (min 1 required) */
  topics: string[];
  /** ISO 8601 date of last human review */
  last_reviewed: string;
  /** Glob patterns for files this knowledge covers (resolved from git root) */
  covers?: string[];
  /** Agent routing hints */
  consumers?: string[];
  /** Cross-references to other knowledge files */
  depends_on?: string[];
  /** Lifecycle status (raw value — may be invalid; validator checks) */
  implementation_status?: string;
}

// ─────────────────────────────────────────────────────────────────────────────
// Loaded Knowledge File
// ─────────────────────────────────────────────────────────────────────────────

/** A parsed knowledge file with resolved paths and content */
export interface KnowledgeFile {
  /** Absolute path to the file */
  path: string;
  /** Path relative to git root */
  relativePath: string;
  /** Parsed YAML frontmatter */
  frontmatter: KnowledgeFrontmatter;
  /** Markdown body (without frontmatter) */
  content: string;
}

// ─────────────────────────────────────────────────────────────────────────────
// KB Config (from .agents/loaf.json)
// ─────────────────────────────────────────────────────────────────────────────

/** Import entry for external knowledge sources */
export interface KbImport {
  name: string;
}

/** Knowledge base configuration extracted from .agents/loaf.json */
export interface KbConfig {
  /** Directory paths relative to git root */
  local: string[];
  /** Days before a file is considered stale */
  staleness_threshold_days: number;
  /** External knowledge imports */
  imports: KbImport[];
}

// ─────────────────────────────────────────────────────────────────────────────
// Result Types (for downstream commands: TASK-034, TASK-035)
// ─────────────────────────────────────────────────────────────────────────────

/** Staleness check result for a single knowledge file */
export interface StalenessResult {
  file: KnowledgeFile;
  isStale: boolean;
  /** Whether the file has a `covers:` field (false = can't determine staleness) */
  hasCoverage: boolean;
  commitCount: number;
  lastCommitAuthor?: string;
  lastCommitDate?: string;
}

/** Validation result for a single knowledge file */
export interface ValidationResult {
  file: KnowledgeFile;
  errors: ValidationIssue[];
  warnings: ValidationIssue[];
}

/** A single validation issue (error or warning) */
export interface ValidationIssue {
  field: string;
  message: string;
}
