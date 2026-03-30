/**
 * Release Commits Tests
 *
 * Tests for conventional commit parsing, bump suggestion,
 * and the commit gathering functions used by `loaf release`.
 */

import { describe, it, expect } from "vitest";
import { suggestBump } from "./commits.js";
import type { ParsedCommit } from "./commits.js";

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
// suggestBump
// ─────────────────────────────────────────────────────────────────────────────

describe("suggestBump", () => {
  it("suggests major when breaking changes present", () => {
    const commits = [
      makeCommit({ hash: "a", message: "breaking thing", breaking: true, section: "Breaking Changes" }),
      makeCommit({ hash: "b", message: "normal feat", section: "Added" }),
    ];
    expect(suggestBump(commits)).toBe("major");
  });

  it("suggests minor when features present (no breaking)", () => {
    const commits = [
      makeCommit({ hash: "a", message: "new feature", section: "Added" }),
      makeCommit({ hash: "b", message: "fix bug", section: "Fixed" }),
    ];
    expect(suggestBump(commits)).toBe("minor");
  });

  it("suggests patch when only fixes present", () => {
    const commits = [
      makeCommit({ hash: "a", message: "fix bug", type: "fix", section: "Fixed" }),
      makeCommit({ hash: "b", message: "another fix", type: "fix", section: "Fixed" }),
    ];
    expect(suggestBump(commits)).toBe("patch");
  });

  it("suggests patch for empty-ish commits (no features, no breaking)", () => {
    const commits = [
      makeCommit({ hash: "a", message: "misc", type: "chore", section: null }),
    ];
    expect(suggestBump(commits)).toBe("patch");
  });
});
