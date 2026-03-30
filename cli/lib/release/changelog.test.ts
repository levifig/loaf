/**
 * Release Changelog Tests
 *
 * Tests for changelog generation and insertion into existing CHANGELOG.md files.
 */

import { describe, it, expect } from "vitest";
import type { ParsedCommit } from "./commits.js";
import {
  groupBySection,
  generateChangelogSection,
  insertIntoChangelog,
  createChangelog,
} from "./changelog.js";

// ─────────────────────────────────────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────────────────────────────────────

function makeCommit(
  overrides: Partial<ParsedCommit> & { hash: string; message: string },
): ParsedCommit {
  return {
    type: "feat",
    breaking: false,
    section: "Added",
    raw: `feat: ${overrides.message}`,
    ...overrides,
  };
}

// ─────────────────────────────────────────────────────────────────────────────
// groupBySection
// ─────────────────────────────────────────────────────────────────────────────

describe("groupBySection", () => {
  it("groups commits by section", () => {
    const commits = [
      makeCommit({ hash: "abc", message: "add feature", section: "Added" }),
      makeCommit({ hash: "def", message: "fix bug", section: "Fixed" }),
      makeCommit({ hash: "ghi", message: "add another", section: "Added" }),
    ];

    const groups = groupBySection(commits);
    expect(groups.get("Added")).toHaveLength(2);
    expect(groups.get("Fixed")).toHaveLength(1);
  });

  it("filters out null-section commits", () => {
    const commits = [
      makeCommit({ hash: "abc", message: "add feature", section: "Added" }),
      makeCommit({ hash: "def", message: "chore stuff", section: null }),
    ];

    const groups = groupBySection(commits);
    expect(groups.size).toBe(1);
    expect(groups.has("Added")).toBe(true);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// generateChangelogSection
// ─────────────────────────────────────────────────────────────────────────────

describe("generateChangelogSection", () => {
  it("generates a versioned section with grouped entries", () => {
    const commits = [
      makeCommit({ hash: "abc", message: "new feature", section: "Added" }),
      makeCommit({ hash: "def", message: "fix crash", section: "Fixed" }),
    ];

    const section = generateChangelogSection("2.0.0-dev.7", "2026-03-30", commits);

    expect(section).toContain("## [2.0.0-dev.7] - 2026-03-30");
    expect(section).toContain("### Added");
    expect(section).toContain("- New feature (abc)");
    expect(section).toContain("### Fixed");
    expect(section).toContain("- Fix crash (def)");
  });

  it("omits sections with no commits", () => {
    const commits = [
      makeCommit({ hash: "abc", message: "fix only", section: "Fixed" }),
    ];

    const section = generateChangelogSection("1.0.1", "2026-03-30", commits);
    expect(section).not.toContain("### Added");
    expect(section).toContain("### Fixed");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// insertIntoChangelog
// ─────────────────────────────────────────────────────────────────────────────

describe("insertIntoChangelog", () => {
  it("inserts below [Unreleased] and preserves it", () => {
    const existing = [
      "# Changelog",
      "",
      "## [Unreleased]",
      "",
      "## [1.0.0] - 2026-01-01",
      "",
      "### Added",
      "- Initial release",
    ].join("\n");

    const newSection = "## [1.1.0] - 2026-03-30\n\n### Added\n- New feature (abc)";
    const result = insertIntoChangelog(existing, newSection);

    expect(result).not.toBeNull();
    // [Unreleased] is preserved at the top
    expect(result).toContain("## [Unreleased]");
    // New section appears between [Unreleased] and old version
    const lines = result!.split("\n");
    const unreleasedIdx = lines.findIndex((l) => l.includes("[Unreleased]"));
    const newSectionIdx = lines.findIndex((l) => l.includes("[1.1.0]"));
    const oldSectionIdx = lines.findIndex((l) => l.includes("[1.0.0]"));
    expect(unreleasedIdx).toBeLessThan(newSectionIdx);
    expect(newSectionIdx).toBeLessThan(oldSectionIdx);
  });

  it("returns null when no [Unreleased] marker exists", () => {
    const existing = "# Changelog\n\n## [1.0.0] - 2026-01-01\n\n### Added\n- Initial";
    const result = insertIntoChangelog(existing, "## [1.1.0] - 2026-03-30");
    expect(result).toBeNull();
  });

  it("is case-insensitive for [Unreleased]", () => {
    const existing = "# Changelog\n\n## [unreleased]\n\n## [1.0.0] - 2026-01-01";
    const result = insertIntoChangelog(existing, "## [1.1.0] - 2026-03-30");
    expect(result).not.toBeNull();
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// createChangelog
// ─────────────────────────────────────────────────────────────────────────────

describe("createChangelog", () => {
  it("creates a new changelog with [Unreleased] section", () => {
    const section = "## [1.0.0] - 2026-03-30\n\n### Added\n- First feature";
    const result = createChangelog(section);

    expect(result).toContain("# Changelog");
    expect(result).toContain("## [Unreleased]");
    expect(result).toContain("## [1.0.0] - 2026-03-30");
  });
});
