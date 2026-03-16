/**
 * Changelog Generator
 *
 * Takes parsed conventional commits and produces Keep-a-Changelog formatted
 * markdown. Can insert a new release section into an existing CHANGELOG.md
 * or create one from scratch.
 */

import type { ParsedCommit, ChangelogSection } from "./commits.js";

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

/** Section ordering — Breaking Changes first, Other last. */
const SECTION_ORDER: ChangelogSection[] = [
  "Breaking Changes",
  "Added",
  "Changed",
  "Fixed",
  "Other",
];

const UNRELEASED_RE = /^## \[unreleased\]/i;

// ─────────────────────────────────────────────────────────────────────────────
// Grouping
// ─────────────────────────────────────────────────────────────────────────────

/** Group commits by changelog section, filtering out null-section commits. */
export function groupBySection(
  commits: ParsedCommit[],
): Map<ChangelogSection, ParsedCommit[]> {
  const groups = new Map<ChangelogSection, ParsedCommit[]>();

  for (const commit of commits) {
    if (commit.section === null) continue;

    const existing = groups.get(commit.section);
    if (existing) {
      existing.push(commit);
    } else {
      groups.set(commit.section, [commit]);
    }
  }

  return groups;
}

// ─────────────────────────────────────────────────────────────────────────────
// Section Generation
// ─────────────────────────────────────────────────────────────────────────────

/** Capitalize the first letter of a string. */
function capitalize(str: string): string {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/** Format a single commit as a changelog entry. */
function formatEntry(commit: ParsedCommit): string {
  return `- ${capitalize(commit.message)} (${commit.hash})`;
}

/** Generate a markdown changelog section for a version. */
export function generateChangelogSection(
  version: string,
  date: string,
  commits: ParsedCommit[],
): string {
  const groups = groupBySection(commits);
  const lines: string[] = [];

  lines.push(`## [${version}] - ${date}`);

  for (const section of SECTION_ORDER) {
    const sectionCommits = groups.get(section);
    if (!sectionCommits || sectionCommits.length === 0) continue;

    lines.push("");
    lines.push(`### ${section}`);
    for (const commit of sectionCommits) {
      lines.push(formatEntry(commit));
    }
  }

  return lines.join("\n");
}

// ─────────────────────────────────────────────────────────────────────────────
// Changelog File Operations
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Insert a new release section into an existing CHANGELOG.md content.
 *
 * Replaces [Unreleased] content with empty, adds versioned section below it.
 * Returns null if no [Unreleased] marker found.
 */
export function insertIntoChangelog(
  existingContent: string,
  newSection: string,
): string | null {
  const lines = existingContent.split("\n");

  // Find the [Unreleased] heading
  let unreleasedIndex = -1;
  for (let i = 0; i < lines.length; i++) {
    if (UNRELEASED_RE.test(lines[i].trim())) {
      unreleasedIndex = i;
      break;
    }
  }

  if (unreleasedIndex === -1) return null;

  // Find the next release heading (## [...]) after [Unreleased]
  let nextReleaseIndex = -1;
  for (let i = unreleasedIndex + 1; i < lines.length; i++) {
    if (/^## \[/.test(lines[i].trim())) {
      nextReleaseIndex = i;
      break;
    }
  }

  // Build the replacement:
  //   ## [Unreleased]
  //   <blank>
  //   {newSection}
  //   <blank>
  //   {rest of file from next release onward}
  const before = lines.slice(0, unreleasedIndex + 1);
  const after =
    nextReleaseIndex === -1 ? [] : lines.slice(nextReleaseIndex);

  const result = [
    ...before,
    "",
    newSection,
    "",
    ...after,
  ];

  return result.join("\n");
}

/**
 * Create a fresh CHANGELOG.md with the release section.
 * Used when no CHANGELOG.md exists.
 */
export function createChangelog(releaseSection: string): string {
  const lines = [
    "# Changelog",
    "",
    "All notable changes to this project will be documented in this file.",
    "",
    "The format is based on [Keep a Changelog](https://keepachangelog.com/).",
    "",
    "## [Unreleased]",
    "",
    releaseSection,
    "",
  ];

  return lines.join("\n");
}
