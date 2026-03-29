/**
 * Cleanup Scanner Engine
 *
 * Walks .agents/ directories and produces typed recommendations based on
 * the cleanup skill's existing rules. Pure logic — no I/O prompts.
 */

import { existsSync, readFileSync, readdirSync, statSync } from "fs";
import { join, basename } from "path";
import matter from "gray-matter";

import { findAgentsDir } from "../tasks/resolve.js";
import { loadIndex, buildIndexFromFiles, findOrphans } from "../tasks/migrate.js";
import type { TaskIndex } from "../tasks/types.js";
import type {
  ArtifactType,
  CleanupRecommendation,
  ScanResult,
  ScanOptions,
  TypeSummary,
  ArtifactDirectory,
} from "./types.js";
import { ARTIFACT_DIRS, ARTIFACT_TYPES } from "./types.js";

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

/** Read .md files from a directory (non-recursive, excludes archive/) */
function readMdFiles(dir: string): Array<{ path: string; filename: string; frontmatter: Record<string, unknown>; raw: string }> {
  if (!existsSync(dir)) return [];

  const results: Array<{ path: string; filename: string; frontmatter: Record<string, unknown>; raw: string }> = [];

  try {
    const entries = readdirSync(dir);
    for (const name of entries) {
      if (name === "archive" || !name.endsWith(".md")) continue;
      const filePath = join(dir, name);
      try {
        const stat = statSync(filePath);
        if (!stat.isFile()) continue;
        const raw = readFileSync(filePath, "utf-8");
        const { data } = matter(raw);
        results.push({ path: filePath, filename: name, frontmatter: data, raw });
      } catch {
        // Can't read file — skip
      }
    }
  } catch {
    // Can't read directory — skip
  }

  return results;
}

/** Days since a date string (ISO 8601 or similar) */
function daysSince(dateStr: string | undefined): number | null {
  if (!dateStr) return null;
  const date = new Date(dateStr);
  if (isNaN(date.getTime())) return null;
  return Math.floor((Date.now() - date.getTime()) / (1000 * 60 * 60 * 24));
}

/**
 * Normalize a session reference to a bare filename.
 * Templates store paths like ".agents/sessions/FILE.md" — strip the prefix.
 */
function normalizeSessionRef(ref: string): string {
  return ref.replace(/^\.agents\/sessions\/(archive\/)?/, "");
}

/** Get the most recent date from frontmatter (supports nested session.last_updated) */
function lastActivity(fm: Record<string, unknown>): string | undefined {
  // Session files use nested session.last_updated / session.created
  const sessionBlock = fm.session as Record<string, unknown> | undefined;
  if (sessionBlock && typeof sessionBlock === "object") {
    return (sessionBlock.last_updated as string) || (sessionBlock.created as string) || undefined;
  }
  return (fm.updated as string) || (fm.created as string) || undefined;
}

/** Extract status from frontmatter, supporting nested session.status */
function getStatus(fm: Record<string, unknown>): string {
  const sessionBlock = fm.session as Record<string, unknown> | undefined;
  if (sessionBlock && typeof sessionBlock === "object" && sessionBlock.status) {
    return String(sessionBlock.status).toLowerCase();
  }
  return String(fm.status || "").toLowerCase();
}

// ─────────────────────────────────────────────────────────────────────────────
// Per-Artifact Scanners
// ─────────────────────────────────────────────────────────────────────────────

function scanSessions(agentsDir: string): CleanupRecommendation[] {
  const dir = join(agentsDir, "sessions");
  const files = readMdFiles(dir);
  const recs: CleanupRecommendation[] = [];

  for (const file of files) {
    const fm = file.frontmatter;
    const status = getStatus(fm);

    // Check for extractable learnings
    const hasLearnings =
      file.raw.includes("## Key Decisions") ||
      file.raw.includes("## Lessons Learned") ||
      file.raw.includes("lessons_learned") ||
      (fm.traceability && typeof fm.traceability === "object" &&
        "decisions" in (fm.traceability as Record<string, unknown>));

    if (status === "completed" || status === "complete") {
      if (hasLearnings) {
        recs.push({
          type: "session",
          path: file.path,
          filename: file.filename,
          action: "archive",
          reason: "Completed session with extractable learnings",
          hint: "Consider running /crystallize before archiving",
          frontmatter: fm,
        });
      } else {
        recs.push({
          type: "session",
          path: file.path,
          filename: file.filename,
          action: "archive",
          reason: "Completed session",
          frontmatter: fm,
        });
      }
    } else if (status === "cancelled" || status === "abandoned") {
      recs.push({
        type: "session",
        path: file.path,
        filename: file.filename,
        action: "archive",
        reason: `Session ${status} — archive with status preserved`,
        frontmatter: fm,
      });
    } else {
      const days = daysSince(lastActivity(fm));
      if (days !== null && days > 7) {
        recs.push({
          type: "session",
          path: file.path,
          filename: file.filename,
          action: "flag",
          reason: `Stale session — inactive for ${days} days`,
          frontmatter: fm,
        });
      } else {
        recs.push({
          type: "session",
          path: file.path,
          filename: file.filename,
          action: "skip",
          reason: "Active session",
          frontmatter: fm,
        });
      }
    }
  }

  return recs;
}

function scanTasks(agentsDir: string, index: TaskIndex): CleanupRecommendation[] {
  const recs: CleanupRecommendation[] = [];

  // Known spec IDs for orphan detection
  const knownSpecIds = new Set(Object.keys(index.specs));

  for (const [id, entry] of Object.entries(index.tasks)) {
    // Skip already-archived tasks
    if (entry.file.startsWith("archive/")) continue;

    const filePath = join(agentsDir, "tasks", entry.file);

    if (entry.status === "done") {
      recs.push({
        type: "task",
        path: filePath,
        filename: entry.file,
        action: "archive",
        reason: `Task ${id} is done`,
        frontmatter: { id, title: entry.title, status: entry.status, spec: entry.spec },
      });
    } else if (entry.spec && !knownSpecIds.has(entry.spec)) {
      // Orphaned: references a spec that doesn't exist
      recs.push({
        type: "task",
        path: filePath,
        filename: entry.file,
        action: "flag",
        reason: `Task ${id} references missing spec ${entry.spec}`,
        frontmatter: { id, title: entry.title, status: entry.status, spec: entry.spec },
      });
    } else {
      recs.push({
        type: "task",
        path: filePath,
        filename: entry.file,
        action: "skip",
        reason: `Task ${id} is ${entry.status}`,
        frontmatter: { id, title: entry.title, status: entry.status },
      });
    }
  }

  // Also check for filesystem orphans (files not in index)
  const orphans = findOrphans(agentsDir, index);
  for (const orphan of orphans.tasks) {
    recs.push({
      type: "task",
      path: join(agentsDir, "tasks", orphan.entry.file),
      filename: orphan.entry.file,
      action: "flag",
      reason: `Task ${orphan.id} exists on disk but not in index — run loaf task sync --import`,
      frontmatter: { id: orphan.id, title: orphan.entry.title },
    });
  }

  return recs;
}

function scanSpecs(agentsDir: string, index: TaskIndex): CleanupRecommendation[] {
  const recs: CleanupRecommendation[] = [];

  for (const [id, entry] of Object.entries(index.specs)) {
    if (entry.file.startsWith("archive/")) continue;

    const filePath = join(agentsDir, "specs", entry.file);

    if (entry.status === "complete") {
      recs.push({
        type: "spec",
        path: filePath,
        filename: entry.file,
        action: "archive",
        reason: `Spec ${id} is complete`,
        frontmatter: { id, title: entry.title, status: entry.status },
      });
    } else if (entry.status === "drafting") {
      const days = daysSince(entry.created);
      if (days !== null && days > 14) {
        recs.push({
          type: "spec",
          path: filePath,
          filename: entry.file,
          action: "flag",
          reason: `Spec ${id} has been drafting for ${days} days`,
          frontmatter: { id, title: entry.title, status: entry.status },
        });
      } else {
        recs.push({
          type: "spec",
          path: filePath,
          filename: entry.file,
          action: "skip",
          reason: `Spec ${id} is ${entry.status}`,
          frontmatter: { id, title: entry.title, status: entry.status },
        });
      }
    } else {
      recs.push({
        type: "spec",
        path: filePath,
        filename: entry.file,
        action: "skip",
        reason: `Spec ${id} is ${entry.status}`,
        frontmatter: { id, title: entry.title, status: entry.status },
      });
    }
  }

  // Check for filesystem orphans
  const orphans = findOrphans(agentsDir, index);
  for (const orphan of orphans.specs) {
    recs.push({
      type: "spec",
      path: join(agentsDir, "specs", orphan.entry.file),
      filename: orphan.entry.file,
      action: "flag",
      reason: `Spec ${orphan.id} exists on disk but not in index — run loaf task sync --import`,
      frontmatter: { id: orphan.id, title: orphan.entry.title },
    });
  }

  return recs;
}

function scanPlans(agentsDir: string): CleanupRecommendation[] {
  const dir = join(agentsDir, "plans");
  const files = readMdFiles(dir);
  const recs: CleanupRecommendation[] = [];

  // Build sets of session IDs (stem without .md) for cross-referencing.
  // Plan frontmatter stores session IDs (e.g., "20260327-163059-spec-015"),
  // while session files are named like "20260327-163059-spec-015-workflow-hooks.md".
  // Match by checking if any session filename starts with the plan's session ref.
  const sessionsDir = join(agentsDir, "sessions");
  const activeSessionFiles: string[] = [];
  const archivedSessionFiles: string[] = [];

  if (existsSync(sessionsDir)) {
    for (const name of readdirSync(sessionsDir)) {
      if (name === "archive" || !name.endsWith(".md")) continue;
      try {
        const raw = readFileSync(join(sessionsDir, name), "utf-8");
        const { data } = matter(raw);
        const status = getStatus(data);
        if (status === "completed" || status === "complete" || status === "cancelled" || status === "archived") {
          archivedSessionFiles.push(name);
        } else {
          activeSessionFiles.push(name);
        }
      } catch {
        // Skip unreadable
      }
    }
  }

  // Also check archive/ for sessions
  const sessionArchive = join(sessionsDir, "archive");
  if (existsSync(sessionArchive)) {
    try {
      for (const name of readdirSync(sessionArchive)) {
        if (name.endsWith(".md")) archivedSessionFiles.push(name);
      }
    } catch { /* skip */ }
  }

  // Match a session reference against session filenames.
  // Refs can be filenames ("SESSION-001.md") or ID stems ("20260327-163059-spec-015").
  const matchesSession = (ref: string, files: string[]): boolean => {
    // Direct filename match
    if (files.includes(ref)) return true;
    // Stem match: ref is a prefix of a filename (without .md)
    const stem = ref.replace(/\.md$/, "");
    return files.some((f) => f.startsWith(stem));
  };

  for (const file of files) {
    const fm = file.frontmatter;
    const sessionRef = fm.session as string | undefined;

    if (!sessionRef) {
      // No session link — orphaned
      recs.push({
        type: "plan",
        path: file.path,
        filename: file.filename,
        action: "delete",
        reason: "Orphaned plan — no linked session",
        frontmatter: fm,
      });
    } else if (matchesSession(sessionRef, archivedSessionFiles)) {
      recs.push({
        type: "plan",
        path: file.path,
        filename: file.filename,
        action: "delete",
        reason: "Linked session is archived/completed",
        frontmatter: fm,
      });
    } else if (!matchesSession(sessionRef, activeSessionFiles)) {
      recs.push({
        type: "plan",
        path: file.path,
        filename: file.filename,
        action: "delete",
        reason: `Linked session "${sessionRef}" not found`,
        frontmatter: fm,
      });
    } else {
      // Check staleness
      const days = daysSince(lastActivity(fm));
      if (days !== null && days > 14) {
        recs.push({
          type: "plan",
          path: file.path,
          filename: file.filename,
          action: "delete",
          reason: `Stale plan — inactive for ${days} days with active session`,
          frontmatter: fm,
        });
      } else {
        recs.push({
          type: "plan",
          path: file.path,
          filename: file.filename,
          action: "skip",
          reason: "Plan linked to active session",
          frontmatter: fm,
        });
      }
    }
  }

  return recs;
}

function scanDrafts(agentsDir: string): CleanupRecommendation[] {
  const dir = join(agentsDir, "drafts");
  const files = readMdFiles(dir);
  const recs: CleanupRecommendation[] = [];

  for (const file of files) {
    const fm = file.frontmatter;
    const days = daysSince(lastActivity(fm) || fm.created as string);

    if (days !== null && days > 30) {
      const hasSparks = file.raw.includes("## Sparks");
      recs.push({
        type: "draft",
        path: file.path,
        filename: file.filename,
        action: "flag",
        reason: `Stale draft — ${days} days old`,
        hint: hasSparks ? "Contains ## Sparks section — review before deletion" : undefined,
        frontmatter: fm,
      });
    } else {
      recs.push({
        type: "draft",
        path: file.path,
        filename: file.filename,
        action: "skip",
        reason: "Recent draft",
        frontmatter: fm,
      });
    }
  }

  return recs;
}

function scanCouncils(agentsDir: string): CleanupRecommendation[] {
  const dir = join(agentsDir, "councils");
  const files = readMdFiles(dir);
  const recs: CleanupRecommendation[] = [];

  // Build session lookup for orphan detection
  const sessionsDir = join(agentsDir, "sessions");
  const allSessionFiles: string[] = [];
  for (const subdir of [sessionsDir, join(sessionsDir, "archive")]) {
    if (existsSync(subdir)) {
      try {
        for (const name of readdirSync(subdir)) {
          if (name.endsWith(".md")) allSessionFiles.push(name);
        }
      } catch { /* skip */ }
    }
  }

  const matchesAnySession = (ref: string): boolean => {
    if (allSessionFiles.includes(ref)) return true;
    const stem = ref.replace(/\.md$/, "");
    return allSessionFiles.some((f) => f.startsWith(stem));
  };

  for (const file of files) {
    const fm = file.frontmatter;

    // Two council schemas exist:
    //   council-session template: council.created, council.session_reference
    //   orchestration reference:  council.timestamp, council.session
    const councilBlock = fm.council as Record<string, unknown> | undefined;
    const councilDate = (councilBlock?.created || councilBlock?.timestamp) as string | undefined;
    const sessionRef = (councilBlock?.session_reference || councilBlock?.session) as string | undefined;
    const days = daysSince(councilDate || lastActivity(fm));

    // Check for orphaned councils (no linked session or missing session)
    if (sessionRef && !matchesAnySession(normalizeSessionRef(sessionRef))) {
      recs.push({
        type: "council",
        path: file.path,
        filename: file.filename,
        action: "flag",
        reason: `Orphaned council — linked session "${sessionRef}" not found`,
        frontmatter: fm,
      });
    } else if (days !== null && days > 14) {
      // Archive only if council has a Decision/outcome recorded.
      // The council file itself is the source — check for ## Decision content.
      const hasDecision = file.raw.includes("## Decision") &&
        !file.raw.match(/## Decision\s*\n\s*\n\s*(\[To be filled|$)/);

      if (hasDecision && sessionRef && matchesAnySession(normalizeSessionRef(sessionRef))) {
        recs.push({
          type: "council",
          path: file.path,
          filename: file.filename,
          action: "archive",
          reason: `Council is ${days} days old with decision recorded — ready for archive`,
          frontmatter: fm,
        });
      } else {
        recs.push({
          type: "council",
          path: file.path,
          filename: file.filename,
          action: "flag",
          reason: hasDecision
            ? `Stale council — ${days} days old, no linked session`
            : `Stale council — ${days} days old, decision not yet recorded`,
          frontmatter: fm,
        });
      }
    } else {
      recs.push({
        type: "council",
        path: file.path,
        filename: file.filename,
        action: "skip",
        reason: "Recent council",
        frontmatter: fm,
      });
    }
  }

  return recs;
}

function scanReports(agentsDir: string): CleanupRecommendation[] {
  const dir = join(agentsDir, "reports");
  const files = readMdFiles(dir);
  const recs: CleanupRecommendation[] = [];

  // Build set of archived session files for prerequisite check
  const sessionArchiveDir = join(agentsDir, "sessions", "archive");
  const archivedSessionFiles: string[] = [];
  if (existsSync(sessionArchiveDir)) {
    try {
      for (const name of readdirSync(sessionArchiveDir)) {
        if (name.endsWith(".md")) archivedSessionFiles.push(name);
      }
    } catch { /* skip */ }
  }

  const sessionIsArchived = (ref: string): boolean => {
    if (archivedSessionFiles.includes(ref)) return true;
    const stem = ref.replace(/\.md$/, "");
    return archivedSessionFiles.some((f) => f.startsWith(stem));
  };

  for (const file of files) {
    const fm = file.frontmatter;

    // Report template nests metadata under `report:` block
    const reportBlock = fm.report as Record<string, unknown> | undefined;
    const archivedAt = reportBlock?.archived_at || fm.archived_at;
    const reportStatus = reportBlock?.status as string | undefined;
    const processedAt = reportBlock?.processed_at as string | undefined;
    const sessionRef = reportBlock?.session_reference as string | undefined;

    if (archivedAt) {
      recs.push({
        type: "report",
        path: file.path,
        filename: file.filename,
        action: "skip",
        reason: "Already archived",
        frontmatter: fm,
      });
    } else if (reportStatus === "processed" || processedAt) {
      // Archive prerequisites: session_reference required, and that session must be archived
      if (!sessionRef) {
        recs.push({
          type: "report",
          path: file.path,
          filename: file.filename,
          action: "flag",
          reason: "Report is processed but missing session_reference — repair metadata",
          frontmatter: fm,
        });
      } else if (!sessionIsArchived(normalizeSessionRef(sessionRef))) {
        recs.push({
          type: "report",
          path: file.path,
          filename: file.filename,
          action: "skip",
          reason: "Report is processed but linked session is not yet archived",
          frontmatter: fm,
        });
      } else {
        recs.push({
          type: "report",
          path: file.path,
          filename: file.filename,
          action: "archive",
          reason: "Report is processed and prerequisites met — ready for archive",
          frontmatter: fm,
        });
      }
    } else {
      // Unprocessed reports: flag if stale, skip if recent.
      // No age-based archive — reports must be processed first.
      const days = daysSince(lastActivity(fm));
      if (days !== null && days > 14) {
        recs.push({
          type: "report",
          path: file.path,
          filename: file.filename,
          action: "flag",
          reason: `Report is ${days} days old but not yet processed`,
          frontmatter: fm,
        });
      } else {
        recs.push({
          type: "report",
          path: file.path,
          filename: file.filename,
          action: "skip",
          reason: "Recent report",
          frontmatter: fm,
        });
      }
    }
  }

  return recs;
}

// ─────────────────────────────────────────────────────────────────────────────
// Main Scanner
// ─────────────────────────────────────────────────────────────────────────────

/** Scan .agents/ artifacts and produce cleanup recommendations */
export function scanArtifacts(options: ScanOptions): ScanResult {
  const { agentsDir, filter } = options;
  const recommendations: CleanupRecommendation[] = [];
  const warnings: string[] = [];

  // Check directory existence per V1 contract
  for (const dir of ARTIFACT_DIRS) {
    if (filter && !filter.includes(dir.type)) continue;

    const fullPath = join(agentsDir, dir.dirname);
    if (!existsSync(fullPath)) {
      if (dir.required) {
        warnings.push(`Required directory missing: ${dir.dirname}/`);
      }
      // Optional dirs: skip silently
    }
  }

  // Load task index read-only (never write TASKS.json during scan)
  const indexPath = join(agentsDir, "TASKS.json");
  const index = loadIndex(indexPath) ?? buildIndexFromFiles(agentsDir);

  // Run per-type scanners
  const shouldScan = (type: ArtifactType) => !filter || filter.includes(type);

  if (shouldScan("session")) recommendations.push(...scanSessions(agentsDir));
  if (shouldScan("task")) recommendations.push(...scanTasks(agentsDir, index));
  if (shouldScan("spec")) recommendations.push(...scanSpecs(agentsDir, index));
  if (shouldScan("plan")) recommendations.push(...scanPlans(agentsDir));
  if (shouldScan("draft")) recommendations.push(...scanDrafts(agentsDir));
  if (shouldScan("council")) recommendations.push(...scanCouncils(agentsDir));
  if (shouldScan("report")) recommendations.push(...scanReports(agentsDir));

  // Build summary
  const summary: TypeSummary[] = ARTIFACT_TYPES
    .filter((t) => shouldScan(t))
    .map((type) => {
      const typeRecs = recommendations.filter((r) => r.type === type);
      return {
        type,
        total: typeRecs.length,
        archive: typeRecs.filter((r) => r.action === "archive").length,
        delete: typeRecs.filter((r) => r.action === "delete").length,
        flag: typeRecs.filter((r) => r.action === "flag").length,
        skip: typeRecs.filter((r) => r.action === "skip").length,
      };
    })
    .filter((s) => s.total > 0);

  return { recommendations, summary, warnings };
}
