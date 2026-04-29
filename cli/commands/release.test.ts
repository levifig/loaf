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
import { execFileSync, spawnSync } from "child_process";
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
  // spawnSync captures stdout AND stderr regardless of exit code, which
  // matters for warnings emitted on stderr alongside a 0 exit.
  const result = spawnSync("node", [CLI_PATH, "release", ...args], {
    cwd: process.cwd(),
    encoding: "utf-8",
    stdio: ["ignore", "pipe", "pipe"],
    timeout: 15_000,
  });
  return {
    stdout: result.stdout ?? "",
    stderr: result.stderr ?? "",
    exitCode: result.status ?? 1,
  };
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
// --yes flag wiring
// ─────────────────────────────────────────────────────────────────────────────

describe("--yes flag", () => {
  it("is accepted as a valid option", () => {
    // --yes with --dry-run: dry-run exits before confirmation, so --yes has
    // no visible effect, but the flag must parse without error
    const result = runRelease("--yes", "--dry-run");
    expect(result.exitCode).toBe(0);
  });

  it("short form -y is accepted", () => {
    const result = runRelease("-y", "--dry-run");
    expect(result.exitCode).toBe(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// --version-file flag wiring (SPEC-031 / TASK-143)
// ─────────────────────────────────────────────────────────────────────────────

describe("--version-file flag", () => {
  it("aborts cleanly when a declared path does not exist", () => {
    const result = runRelease(
      "--version-file",
      "definitely/missing/path.json",
      "--dry-run",
    );
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain(
      "version file definitely/missing/path.json not found",
    );
  });

  it("uses the overridden file (ignoring root package.json) when path exists", () => {
    // package.json at project root has the real loaf version. Pointing
    // --version-file at it explicitly is a no-op for content but proves
    // the override is wired and the dry-run preview shows that file.
    const result = runRelease("--version-file", "package.json", "--dry-run");
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("package.json");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// --pre-merge flag wiring (SPEC-031 / TASK-144)
// ─────────────────────────────────────────────────────────────────────────────

describe("--pre-merge flag", () => {
  it("bundles --no-tag --no-gh and prints the auto-detected base", () => {
    // Running against the loaf repo itself: no PR for the test branch, no
    // git config override → step 4 wins. We only assert that *some* base
    // is auto-detected (the value depends on the local clone), and that
    // tag + gh are skipped.
    const result = runRelease("--pre-merge", "--dry-run");
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Auto-detected base:");
    expect(result.stdout).toContain("--no-tag — skipped");
    expect(result.stdout).toContain("--no-gh — skipped");
  });

  it("explicit --base short-circuits auto-detection (no 'Auto-detected base' line)", () => {
    // Use HEAD as the explicit base — it always resolves and produces a clean
    // "no commits" exit before the action list. We assert that auto-detection
    // did NOT run (no banner) and that the explicit ref was used.
    const result = runRelease("--pre-merge", "--base", "HEAD", "--dry-run");
    expect(result.exitCode).toBe(0);
    // Auto-detection is skipped when --base is explicit.
    expect(result.stdout).not.toContain("Auto-detected base:");
    // Explicit base is still used.
    expect(result.stdout).toContain("via --base flag");
  });

  it("emits a warning when --pre-merge is combined with --tag (and tags anyway)", () => {
    // Using --base origin/main forces a commit-list with content so the
    // action list is displayed (and we can verify the tag step is NOT
    // greyed out as skipped).
    const result = runRelease(
      "--pre-merge",
      "--tag",
      "--base",
      "origin/main",
      "--dry-run",
    );
    expect(result.exitCode).toBe(0);
    expect(result.stderr).toContain(
      "--tag overrides --pre-merge default",
    );
    // Tag is NOT skipped in the action list.
    expect(result.stdout).not.toContain("--no-tag — skipped");
    // GH release is still bundled-skipped.
    expect(result.stdout).toContain("--no-gh — skipped");
  });

  it("emits a warning when --pre-merge is combined with --gh", () => {
    // Note: --pre-merge bundles --no-tag, and (per normalizeSkipFlags)
    // --no-tag implies --no-gh because `gh release create` would auto-push
    // the missing tag. So --gh cannot fully override the bundled --no-gh
    // when --no-tag is also active. The warning still fires for transparency,
    // but the action list reflects the implication.
    const result = runRelease(
      "--pre-merge",
      "--gh",
      "--base",
      "origin/main",
      "--dry-run",
    );
    expect(result.exitCode).toBe(0);
    expect(result.stderr).toContain(
      "--gh overrides --pre-merge default",
    );
  });

  it("--gh + --tag override (--pre-merge with both) creates a real release", () => {
    // Combining --pre-merge with --tag AND --gh restores the full default
    // pipeline (tag + gh), with two warnings on stderr.
    const result = runRelease(
      "--pre-merge",
      "--tag",
      "--gh",
      "--base",
      "origin/main",
      "--dry-run",
    );
    expect(result.exitCode).toBe(0);
    expect(result.stderr).toContain("--tag overrides --pre-merge default");
    expect(result.stderr).toContain("--gh overrides --pre-merge default");
    // Neither tag nor gh should be skipped in the action list.
    expect(result.stdout).not.toContain("--no-tag — skipped");
    expect(result.stdout).not.toContain("--no-gh — skipped");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// --post-merge flag wiring (SPEC-031 / TASK-142)
// ─────────────────────────────────────────────────────────────────────────────

describe("--post-merge flag", () => {
  it("rejects combination with --bump", () => {
    const result = runRelease("--post-merge", "--bump", "patch");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain(
      "--post-merge is incompatible with --bump",
    );
  });

  it("rejects combination with --dry-run", () => {
    const result = runRelease("--post-merge", "--dry-run");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain(
      "--post-merge is incompatible with --dry-run",
    );
  });

  it("rejects combination with --no-tag", () => {
    const result = runRelease("--post-merge", "--no-tag");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain("--post-merge is incompatible with --no-tag");
  });

  it("rejects combination with --no-gh", () => {
    const result = runRelease("--post-merge", "--no-gh");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain("--post-merge is incompatible with --no-gh");
  });

  it("rejects combination with --base", () => {
    const result = runRelease("--post-merge", "--base", "main");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain("--post-merge is incompatible with --base");
  });

  it("rejects combination with --pre-merge", () => {
    const result = runRelease("--post-merge", "--pre-merge");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain(
      "--post-merge is incompatible with --pre-merge",
    );
  });

  it("aborts with guardrail failure on the loaf repo (no chore: release HEAD)", () => {
    // Running on the loaf repo itself: HEAD is not a chore: release commit,
    // so guardrail 3 (subject shape) should fail. The test asserts that the
    // guardrail layer runs and surfaces a non-zero exit with a helpful
    // message — the exact guardrail that trips depends on local git state
    // (branch, worktree cleanliness), so we accept any guardrail failure.
    const result = runRelease("--post-merge");
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toMatch(/guardrail \d+ failed/);
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
