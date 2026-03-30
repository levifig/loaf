/**
 * Release Version Tests
 *
 * Tests for semver parsing, bumping, and version file detection.
 */

import { describe, it, expect } from "vitest";
import { parseSemVer, formatSemVer, bumpVersion } from "./version.js";

// ─────────────────────────────────────────────────────────────────────────────
// parseSemVer
// ─────────────────────────────────────────────────────────────────────────────

describe("parseSemVer", () => {
  it("parses a stable version", () => {
    expect(parseSemVer("2.0.0")).toEqual({ major: 2, minor: 0, patch: 0 });
  });

  it("parses a pre-release version", () => {
    expect(parseSemVer("2.0.0-dev.6")).toEqual({
      major: 2,
      minor: 0,
      patch: 0,
      prerelease: "dev.6",
    });
  });

  it("parses a pre-release with no numeric suffix", () => {
    expect(parseSemVer("1.0.0-alpha")).toEqual({
      major: 1,
      minor: 0,
      patch: 0,
      prerelease: "alpha",
    });
  });

  it("returns null for invalid versions", () => {
    expect(parseSemVer("not-a-version")).toBeNull();
    expect(parseSemVer("1.2")).toBeNull();
    expect(parseSemVer("1.2.3-")).toBeNull();
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// formatSemVer
// ─────────────────────────────────────────────────────────────────────────────

describe("formatSemVer", () => {
  it("formats a stable version", () => {
    expect(formatSemVer({ major: 2, minor: 1, patch: 0 })).toBe("2.1.0");
  });

  it("formats a pre-release version", () => {
    expect(
      formatSemVer({ major: 2, minor: 0, patch: 0, prerelease: "dev.6" }),
    ).toBe("2.0.0-dev.6");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// bumpVersion
// ─────────────────────────────────────────────────────────────────────────────

describe("bumpVersion", () => {
  describe("stable versions", () => {
    it("bumps major", () => {
      expect(bumpVersion("1.2.3", "major")).toBe("2.0.0");
    });

    it("bumps minor", () => {
      expect(bumpVersion("1.2.3", "minor")).toBe("1.3.0");
    });

    it("bumps patch", () => {
      expect(bumpVersion("1.2.3", "patch")).toBe("1.2.4");
    });

    it("returns null for prerelease bump on stable", () => {
      expect(bumpVersion("1.2.3", "prerelease")).toBeNull();
    });

    it("returns null for release bump on stable", () => {
      expect(bumpVersion("1.2.3", "release")).toBeNull();
    });
  });

  describe("pre-release versions", () => {
    it("increments dev counter", () => {
      expect(bumpVersion("2.0.0-dev.6", "prerelease")).toBe("2.0.0-dev.7");
    });

    it("promotes to stable release", () => {
      expect(bumpVersion("2.0.0-dev.6", "release")).toBe("2.0.0");
    });

    it("bumps major from pre-release", () => {
      expect(bumpVersion("2.0.0-dev.6", "major")).toBe("3.0.0");
    });

    it("bumps minor from pre-release", () => {
      expect(bumpVersion("2.0.0-dev.6", "minor")).toBe("2.1.0");
    });

    it("appends .1 to prerelease without numeric suffix", () => {
      expect(bumpVersion("1.0.0-alpha", "prerelease")).toBe("1.0.0-alpha.1");
    });

    it("increments existing numeric suffix", () => {
      expect(bumpVersion("1.0.0-alpha.3", "prerelease")).toBe("1.0.0-alpha.4");
    });

    it("appends .1 to non-numeric suffix segment", () => {
      expect(bumpVersion("1.0.0-alpha.beta", "prerelease")).toBe("1.0.0-alpha.beta.1");
    });
  });

  describe("invalid input", () => {
    it("returns null for unparseable version", () => {
      expect(bumpVersion("garbage", "major")).toBeNull();
    });

    it("returns null for incomplete version", () => {
      expect(bumpVersion("1.2", "patch")).toBeNull();
    });
  });
});
