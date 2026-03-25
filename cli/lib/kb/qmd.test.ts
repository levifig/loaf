/**
 * QMD Integration Tests
 *
 * Tests for the QMD soft dependency module. All child_process calls are
 * mocked to avoid requiring QMD to be installed.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { execFileSync } from "child_process";
import {
  isQmdAvailable,
  registerCollection,
  removeCollection,
  listCollections,
} from "./qmd.js";

vi.mock("child_process", () => ({
  execFileSync: vi.fn(),
}));

const mockExecFileSync = vi.mocked(execFileSync);

beforeEach(() => {
  mockExecFileSync.mockReset();
});

// ─────────────────────────────────────────────────────────────────────────────
// isQmdAvailable
// ─────────────────────────────────────────────────────────────────────────────

describe("isQmdAvailable", () => {
  it("returns true when which qmd succeeds", () => {
    mockExecFileSync.mockReturnValueOnce(Buffer.from("/usr/local/bin/qmd"));

    expect(isQmdAvailable()).toBe(true);
    expect(mockExecFileSync).toHaveBeenCalledWith("which", ["qmd"], {
      stdio: "pipe",
    });
  });

  it("returns false when which qmd throws", () => {
    mockExecFileSync.mockImplementationOnce(() => {
      throw new Error("not found");
    });

    expect(isQmdAvailable()).toBe(false);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// registerCollection
// ─────────────────────────────────────────────────────────────────────────────

describe("registerCollection", () => {
  it("calls qmd with correct args", () => {
    mockExecFileSync.mockReturnValueOnce("");

    registerCollection("my-knowledge", "/path/to/docs");

    expect(mockExecFileSync).toHaveBeenCalledWith(
      "qmd",
      ["collection", "add", "/path/to/docs", "--name", "my-knowledge"],
      { encoding: "utf-8", stdio: ["pipe", "pipe", "pipe"] },
    );
  });

  it("throws with helpful message on failure", () => {
    mockExecFileSync.mockImplementationOnce(() => {
      throw new Error("command failed");
    });

    expect(() => registerCollection("bad-col", "/nope")).toThrow(
      /Failed to register QMD collection "bad-col"/,
    );
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// removeCollection
// ─────────────────────────────────────────────────────────────────────────────

describe("removeCollection", () => {
  it("calls qmd collection remove with correct args", () => {
    mockExecFileSync.mockReturnValueOnce("");

    removeCollection("my-knowledge");

    expect(mockExecFileSync).toHaveBeenCalledWith(
      "qmd",
      ["collection", "remove", "my-knowledge"],
      { encoding: "utf-8", stdio: ["pipe", "pipe", "pipe"] },
    );
  });

  it("throws with helpful message on failure", () => {
    mockExecFileSync.mockImplementationOnce(() => {
      throw new Error("not found");
    });

    expect(() => removeCollection("missing")).toThrow(
      /Failed to remove QMD collection "missing"/,
    );
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// listCollections
// ─────────────────────────────────────────────────────────────────────────────

describe("listCollections", () => {
  it("parses output correctly", () => {
    mockExecFileSync.mockReturnValueOnce("loaf-knowledge\nloaf-decisions\n");

    const result = listCollections();

    expect(result).toEqual(["loaf-knowledge", "loaf-decisions"]);
    expect(mockExecFileSync).toHaveBeenCalledWith(
      "qmd",
      ["collection", "list"],
      { encoding: "utf-8", stdio: ["pipe", "pipe", "pipe"] },
    );
  });

  it("returns empty array on failure", () => {
    mockExecFileSync.mockImplementationOnce(() => {
      throw new Error("qmd not installed");
    });

    expect(listCollections()).toEqual([]);
  });

  it("filters out empty lines", () => {
    mockExecFileSync.mockReturnValueOnce("col-a\n\ncol-b\n");

    expect(listCollections()).toEqual(["col-a", "col-b"]);
  });
});
