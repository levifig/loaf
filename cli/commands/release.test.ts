/**
 * Release Command Integration Tests
 *
 * Tests that the `loaf release` command correctly wires option parsing
 * to the underlying helpers. These are subprocess tests that invoke the
 * CLI, so they verify the full Commander → handler → helper chain,
 * not just the helpers in isolation.
 *
 * The CLI is built from source in beforeAll, so these tests always
 * exercise current code regardless of dist-cli/ state.
 *
 * All tests use --dry-run, so no files or git state are modified.
 */

import { describe, it, expect, beforeAll } from "vitest";
import { execFileSync } from "child_process";
import { join } from "path";

// ─────────────────────────────────────────────────────────────────────────────
// Setup — build the CLI from source before running integration tests
// ─────────────────────────────────────────────────────────────────────────────

const CLI_PATH = join(process.cwd(), "dist-cli", "index.js");

beforeAll(() => {
  execFileSync("npm", ["run", "build:cli"], {
    cwd: process.cwd(),
    stdio: "ignore",
    timeout: 30_000,
  });
}, 30_000);

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

interface RunResult {
  stdout: string;
  stderr: string;
  exitCode: number;
}

/** Run the release command with given args. Never throws — captures exit code. */
function runRelease(...args: string[]): RunResult {
  try {
    const stdout = execFileSync("node", [CLI_PATH, "release", ...args], {
      cwd: process.cwd(),
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "pipe"],
      timeout: 15_000,
    });
    return { stdout, stderr: "", exitCode: 0 };
  } catch (error: unknown) {
    const err = error as { stdout?: string; stderr?: string; status?: number };
    return {
      stdout: err.stdout ?? "",
      stderr: err.stderr ?? "",
      exitCode: err.status ?? 1,
    };
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// --bump validation wiring
// ─────────────────────────────────────────────────────────────────────────────

describe("--bump flag", () => {
  it("rejects invalid bump type with exit code 1", () => {
    const result = runRelease("--bump", "bogus", "--dry-run");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain('Invalid bump type "bogus"');
  });

  it("accepts valid bump type and skips interactive prompt", () => {
    const result = runRelease("--bump", "prerelease", "--dry-run");
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("via --bump flag");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// --base validation wiring
// ─────────────────────────────────────────────────────────────────────────────

describe("--base flag", () => {
  it("rejects nonexistent ref with exit code 1", () => {
    const result = runRelease("--base", "definitely-not-a-ref", "--dry-run");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain("does not exist or is not reachable");
  });

  it("accepts valid ref and scopes commits", () => {
    // HEAD..HEAD = 0 commits, but the ref validation should pass
    const result = runRelease("--base", "HEAD", "--dry-run");
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("via --base flag");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// --no-tag / --no-gh wiring
// ─────────────────────────────────────────────────────────────────────────────

describe("--no-tag implies --no-gh", () => {
  it("shows both tag and gh as skipped when only --no-tag is passed", () => {
    const result = runRelease("--no-tag", "--dry-run");
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("--no-tag — skipped");
    expect(result.stdout).toContain("--no-gh — skipped");
  });

  it("skips only gh when only --no-gh is passed", () => {
    const result = runRelease("--no-gh", "--dry-run");
    expect(result.exitCode).toBe(0);
    // Tag step should NOT be skipped
    expect(result.stdout).not.toContain("--no-tag — skipped");
    // GH step should be skipped
    expect(result.stdout).toContain("--no-gh — skipped");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Regression guards
// ─────────────────────────────────────────────────────────────────────────────

describe("existing behavior preserved", () => {
  it("--dry-run alone exits 0 without modifying anything", () => {
    const result = runRelease("--dry-run");
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("No changes made");
  });
});
