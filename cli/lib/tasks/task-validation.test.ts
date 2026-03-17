/**
 * Task Validation Tests
 *
 * Tests for slug generation and type constant validation.
 */

import { describe, it, expect } from "vitest";
import {
  TASK_STATUSES,
  TASK_PRIORITIES,
  SPEC_STATUSES,
} from "./types.js";
import { generateSlug } from "./slug.js";

// ─────────────────────────────────────────────────────────────────────────────
// generateSlug
// ─────────────────────────────────────────────────────────────────────────────

describe("generateSlug", () => {
  it("converts basic title to kebab-case", () => {
    expect(generateSlug("Add test infrastructure")).toBe("add-test-infrastructure");
  });

  it("strips single quotes", () => {
    expect(generateSlug("It's a 'test'")).toBe("its-a-test");
  });

  it("strips double quotes", () => {
    expect(generateSlug('Handle "edge" cases')).toBe("handle-edge-cases");
  });

  it("strips backticks", () => {
    expect(generateSlug("Use `vitest`")).toBe("use-vitest");
  });

  it("replaces special characters with hyphens", () => {
    expect(generateSlug("foo@bar#baz")).toBe("foo-bar-baz");
  });

  it("trims leading hyphens", () => {
    expect(generateSlug("---hello")).toBe("hello");
  });

  it("trims trailing hyphens", () => {
    expect(generateSlug("hello---")).toBe("hello");
  });

  it("trims both leading and trailing hyphens", () => {
    expect(generateSlug("---hello---")).toBe("hello");
  });

  it("truncates to max 50 characters", () => {
    const longTitle = "This is a very long task title that definitely exceeds the fifty character limit we set";
    const slug = generateSlug(longTitle);
    expect(slug.length).toBeLessThanOrEqual(50);
    expect(slug).toBe("this-is-a-very-long-task-title-that-definitely-exc");
  });

  it("collapses multiple consecutive special characters into a single hyphen", () => {
    expect(generateSlug("hello   world")).toBe("hello-world");
    expect(generateSlug("hello---world")).toBe("hello-world");
    expect(generateSlug("hello@@@world")).toBe("hello-world");
  });

  it("handles empty string", () => {
    expect(generateSlug("")).toBe("");
  });

  it("handles string with only special characters", () => {
    expect(generateSlug("@#$%^&*")).toBe("");
  });

  it("handles mixed quotes and special chars", () => {
    expect(generateSlug("Fix `parser's` \"edge-case\" handling")).toBe(
      "fix-parsers-edge-case-handling",
    );
  });

  it("handles numbers in title", () => {
    expect(generateSlug("TASK 42 is important")).toBe("task-42-is-important");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Type Constants
// ─────────────────────────────────────────────────────────────────────────────

describe("TASK_STATUSES", () => {
  it("contains exactly the expected statuses in order", () => {
    expect(TASK_STATUSES).toEqual(["todo", "in_progress", "blocked", "review", "done"]);
  });
});

describe("TASK_PRIORITIES", () => {
  it("contains exactly the expected priorities in order", () => {
    expect(TASK_PRIORITIES).toEqual(["P0", "P1", "P2", "P3"]);
  });
});

describe("SPEC_STATUSES", () => {
  it("contains exactly the expected statuses in order", () => {
    expect(SPEC_STATUSES).toEqual(["drafting", "approved", "implementing", "complete"]);
  });
});
