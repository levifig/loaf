/**
 * Release Options Tests
 *
 * Tests for CLI option validation and normalization used by `loaf release`.
 * Covers --bump validation, --base ref validation, and --no-tag/--no-gh
 * flag interaction.
 */

import { describe, it, expect } from "vitest";
import {
  validateBumpType,
  validateBaseRef,
  normalizeSkipFlags,
} from "./options.js";

// ─────────────────────────────────────────────────────────────────────────────
// validateBumpType
// ─────────────────────────────────────────────────────────────────────────────

describe("validateBumpType", () => {
  it.each(["major", "minor", "patch", "prerelease", "release"])(
    "accepts valid bump type: %s",
    (type) => {
      expect(validateBumpType(type)).toBe(type);
    },
  );

  it("rejects invalid bump type", () => {
    expect(() => validateBumpType("bogus")).toThrow(
      /Invalid bump type "bogus"/,
    );
  });

  it("rejects empty string", () => {
    expect(() => validateBumpType("")).toThrow(/Invalid bump type/);
  });

  it("error message lists valid options", () => {
    expect(() => validateBumpType("nope")).toThrow(/major, minor, patch, prerelease, release/);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// validateBaseRef (integration — uses the real git repo we're running in)
// ─────────────────────────────────────────────────────────────────────────────

describe("validateBaseRef", () => {
  const cwd = process.cwd();

  it("returns the ref when it resolves directly", () => {
    expect(validateBaseRef(cwd, "HEAD")).toBe("HEAD");
  });

  it("falls back to origin/<ref> when local ref is missing", () => {
    // origin/main exists in any clone, even without a local main branch
    const result = validateBaseRef(cwd, "main");
    // Should resolve to either "main" (local exists) or "origin/main" (fallback)
    expect(result).toMatch(/^(origin\/)?main$/);
  });

  it("does not double-prefix refs that already contain a slash", () => {
    // origin/main should resolve directly, not try origin/origin/main
    expect(validateBaseRef(cwd, "origin/main")).toBe("origin/main");
  });

  it("rejects a nonexistent ref", () => {
    expect(() => validateBaseRef(cwd, "definitely-not-a-ref-xyz")).toThrow(
      /does not exist or is not reachable/,
    );
  });

  it("error message includes the bad ref name", () => {
    expect(() => validateBaseRef(cwd, "no-such-branch")).toThrow(
      /no-such-branch/,
    );
  });

  it("error message mentions both attempted refs", () => {
    expect(() => validateBaseRef(cwd, "missing-branch")).toThrow(
      /Tried "missing-branch" and "origin\/missing-branch"/,
    );
  });

  it("does not claim an origin/origin fallback for refs with a slash", () => {
    expect(() => validateBaseRef(cwd, "missing/branch")).toThrow(
      /Tried "missing\/branch"\. If this is a remote branch, run: git fetch origin missing\/branch/,
    );
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// normalizeSkipFlags
// ─────────────────────────────────────────────────────────────────────────────

describe("normalizeSkipFlags", () => {
  it("defaults to skipping nothing", () => {
    const { skipTag, skipGh } = normalizeSkipFlags({});
    expect(skipTag).toBe(false);
    expect(skipGh).toBe(false);
  });

  it("--no-tag sets skipTag", () => {
    const { skipTag } = normalizeSkipFlags({ tag: false });
    expect(skipTag).toBe(true);
  });

  it("--no-gh sets skipGh", () => {
    const { skipGh } = normalizeSkipFlags({ gh: false });
    expect(skipGh).toBe(true);
  });

  it("--no-tag implies --no-gh", () => {
    const { skipTag, skipGh } = normalizeSkipFlags({ tag: false });
    expect(skipTag).toBe(true);
    expect(skipGh).toBe(true);
  });

  it("--no-tag implies --no-gh even when gh is explicitly true", () => {
    // Commander won't produce this, but the logic should be robust
    const { skipTag, skipGh } = normalizeSkipFlags({ tag: false, gh: true });
    expect(skipTag).toBe(true);
    expect(skipGh).toBe(true);
  });

  it("explicit --no-gh without --no-tag only skips gh", () => {
    const { skipTag, skipGh } = normalizeSkipFlags({ gh: false });
    expect(skipTag).toBe(false);
    expect(skipGh).toBe(true);
  });

  it("undefined tag/gh (no flags passed) skips nothing", () => {
    const { skipTag, skipGh } = normalizeSkipFlags({ dryRun: true });
    expect(skipTag).toBe(false);
    expect(skipGh).toBe(false);
  });
});
