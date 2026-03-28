/**
 * Cleanup Scanner Types
 *
 * Type definitions for the cleanup scanning engine. The scanner walks
 * .agents/ directories and produces typed recommendations based on
 * the existing cleanup skill's rules.
 */

/** Artifact types that the cleanup scanner recognizes */
export type ArtifactType =
  | "session"
  | "task"
  | "spec"
  | "plan"
  | "draft"
  | "council"
  | "report";

/** All artifact types, ordered for display */
export const ARTIFACT_TYPES: ArtifactType[] = [
  "session",
  "task",
  "spec",
  "plan",
  "draft",
  "council",
  "report",
];

/** Actions the scanner can recommend */
export type RecommendedAction = "archive" | "delete" | "flag" | "skip";

/** A single cleanup recommendation for one artifact */
export interface CleanupRecommendation {
  /** Which artifact type this is */
  type: ArtifactType;
  /** Absolute path to the artifact file */
  path: string;
  /** Filename for display */
  filename: string;
  /** What the scanner recommends */
  action: RecommendedAction;
  /** Human-readable reason for the recommendation */
  reason: string;
  /** Optional hint for the user (e.g., "run /crystallize first") */
  hint?: string;
  /** Frontmatter data for previews */
  frontmatter?: Record<string, unknown>;
}

/** Per-type summary counts */
export interface TypeSummary {
  type: ArtifactType;
  total: number;
  archive: number;
  delete: number;
  flag: number;
  skip: number;
}

/** Full scan result */
export interface ScanResult {
  recommendations: CleanupRecommendation[];
  summary: TypeSummary[];
  warnings: string[];
}

/** Options for the scanner */
export interface ScanOptions {
  /** Path to .agents/ directory */
  agentsDir: string;
  /** Filter to specific artifact types (scan all if omitted) */
  filter?: ArtifactType[];
}

/** Directory metadata for the V1 artifact contract */
export interface ArtifactDirectory {
  type: ArtifactType;
  dirname: string;
  required: boolean;
}

/** V1 artifact directory contract */
export const ARTIFACT_DIRS: ArtifactDirectory[] = [
  { type: "session", dirname: "sessions", required: true },
  { type: "task", dirname: "tasks", required: true },
  { type: "spec", dirname: "specs", required: true },
  { type: "plan", dirname: "plans", required: false },
  { type: "draft", dirname: "drafts", required: false },
  { type: "council", dirname: "councils", required: false },
  { type: "report", dirname: "reports", required: false },
];
