/**
 * Release Commits Tests
 *
 * Tests for conventional commit parsing, section mapping, breaking change
 * detection, and bump suggestion.
 */

import { describe, it, expect } from "vitest";
import { parseCommit, suggestBump } from "./commits.js";
import type { ParsedCommit } from "./commits.js";

// ─────────────────────────────────────────────────────────────────────────────
// parseCommit — conventional commit regex + section mapping
// ─────────────────────────────────────────────────────────────────────────────

describe("parseCommit", () => {
  describe("standard types", () => {
    it("parses feat into Added section", () => {
      const result = parseCommit("abc", "feat: add login page", "");
      expect(result).toMatchObject({
        type: "feat",
        message: "add login page",
        breaking: false,
        section: "Added",
      });
    });

    it("parses fix into Fixed section", () => {
      const result = parseCommit("abc", "fix: null pointer in auth", "");
      expect(result).toMatchObject({
        type: "fix",
        message: "null pointer in auth",
        section: "Fixed",
      });
    });

    it("parses refactor into Changed section", () => {
      const result = parseCommit("abc", "refactor: extract helper", "");
      expect(result).toMatchObject({ type: "refactor", section: "Changed" });
    });

    it("parses perf into Changed section", () => {
      const result = parseCommit("abc", "perf: optimize query", "");
      expect(result).toMatchObject({ type: "perf", section: "Changed" });
    });
  });

  describe("filtered types (null section)", () => {
    it.each(["docs", "chore", "ci", "test", "build", "style"])(
      "parses %s into null section (filtered from changelog)",
      (type) => {
        const result = parseCommit("abc", `${type}: some work`, "");
        expect(result.section).toBeNull();
      },
    );
  });

  describe("unknown types", () => {
    it("maps unknown type to Other section", () => {
      const result = parseCommit("abc", "wip: experiment", "");
      expect(result).toMatchObject({ type: "wip", section: "Other" });
    });
  });

  describe("scope handling", () => {
    it("ignores scope parentheses in parsing", () => {
      const result = parseCommit("abc", "fix(core): something", "");
      expect(result).toMatchObject({
        type: "fix",
        message: "something",
        section: "Fixed",
      });
    });

    it("handles multi-word scope", () => {
      const result = parseCommit("abc", "feat(auth-service): add SSO", "");
      expect(result).toMatchObject({
        type: "feat",
        message: "add SSO",
        section: "Added",
      });
    });
  });

  describe("breaking changes", () => {
    it("detects bang syntax", () => {
      const result = parseCommit("abc", "feat!: drop legacy API", "");
      expect(result).toMatchObject({
        breaking: true,
        section: "Breaking Changes",
      });
    });

    it("detects bang with scope", () => {
      const result = parseCommit("abc", "fix(auth)!: remove v1 tokens", "");
      expect(result).toMatchObject({
        breaking: true,
        section: "Breaking Changes",
      });
    });

    it("detects BREAKING CHANGE in body", () => {
      const result = parseCommit(
        "abc",
        "feat: new auth system",
        "BREAKING CHANGE: removed v1 token support",
      );
      expect(result).toMatchObject({
        breaking: true,
        section: "Breaking Changes",
      });
    });

    it("detects BREAKING-CHANGE in body (hyphenated)", () => {
      const result = parseCommit(
        "abc",
        "feat: new auth",
        "BREAKING-CHANGE: removed old API",
      );
      expect(result.breaking).toBe(true);
    });

    it("body breaking overrides section even for fix type", () => {
      const result = parseCommit(
        "abc",
        "fix: remove deprecated endpoint",
        "BREAKING CHANGE: /v1 no longer available",
      );
      expect(result.section).toBe("Breaking Changes");
    });
  });

  describe("non-conventional commits", () => {
    it("handles merge commits", () => {
      const result = parseCommit("abc", "Merge pull request #42", "");
      expect(result).toMatchObject({
        type: "",
        message: "Merge pull request #42",
        section: "Other",
      });
    });

    it("handles plain messages", () => {
      const result = parseCommit("abc", "update readme", "");
      expect(result).toMatchObject({
        type: "",
        section: "Other",
      });
    });

    it("detects breaking in body of non-conventional commit", () => {
      const result = parseCommit(
        "abc",
        "big refactor",
        "BREAKING CHANGE: everything changed",
      );
      expect(result.breaking).toBe(true);
      expect(result.section).toBe("Breaking Changes");
    });
  });

  describe("preserves raw subject and hash", () => {
    it("stores original subject as raw", () => {
      const result = parseCommit("abc123", "feat: new thing", "");
      expect(result.raw).toBe("feat: new thing");
      expect(result.hash).toBe("abc123");
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// suggestBump
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

describe("suggestBump", () => {
  it("suggests major when breaking changes present", () => {
    const commits = [
      makeCommit({ hash: "a", message: "breaking", breaking: true, section: "Breaking Changes" }),
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
      makeCommit({ hash: "a", message: "fix", type: "fix", section: "Fixed" }),
    ];
    expect(suggestBump(commits)).toBe("patch");
  });

  it("suggests patch for filtered-only commits", () => {
    const commits = [
      makeCommit({ hash: "a", message: "chore", type: "chore", section: null }),
    ];
    expect(suggestBump(commits)).toBe("patch");
  });
});
