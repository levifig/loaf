/**
 * SPEC-042 Track A — `.agents/loaf.json` routing across linked worktrees.
 *
 * Verifies that `readLoafConfig`, `loafConfigPath`, and `mergeLoafConfigIntegrations`
 * resolve to the main worktree's `.agents/loaf.json` when invoked from a linked
 * git worktree that has been migrated under SPEC-036. Without this, a linked
 * worktree's `loaf.json` is invisible to release/kb/install code paths that
 * resolve `.agents/` via `findAgentsDir`, and writes silently create a stray
 * config file next to the `.moved-to` back-pointer.
 *
 * Fixture pattern mirrors `cli/lib/migrate/worktree-storage.test.ts` — real
 * `git init` + `git worktree add`, no git mocking.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from "fs";
import { execFileSync } from "child_process";
import { join } from "path";
import { tmpdir } from "os";

import {
  loafConfigPath,
  mergeLoafConfigIntegrations,
  readLoafConfig,
} from "./agents-config.js";

// ─────────────────────────────────────────────────────────────────────────────
// Scaffolding (mirrors cli/lib/migrate/worktree-storage.test.ts)
// ─────────────────────────────────────────────────────────────────────────────

let TEST_ROOT: string;

const TEST_IDENTITY = [
  "-c",
  "user.name=Test User",
  "-c",
  "user.email=test@test.com",
] as const;

function git(args: readonly string[], cwd: string): string {
  return execFileSync("git", args as string[], {
    cwd,
    stdio: ["ignore", "pipe", "pipe"],
  }).toString();
}

function createMainRepo(name: string): string {
  const repoPath = realpathSync(mkdtempSync(join(TEST_ROOT, `${name}-`)));
  git(["init", "--initial-branch=main"], repoPath);
  writeFileSync(join(repoPath, "README.md"), "# Test\n", "utf-8");
  git(["add", "."], repoPath);
  git([...TEST_IDENTITY, "commit", "-m", "Initial commit"], repoPath);
  mkdirSync(join(repoPath, ".agents"), { recursive: true });
  return repoPath;
}

function addWorktree(repoPath: string, branch: string): string {
  const wtPath = `${repoPath}-wt-${branch.replace(/\//g, "-")}`;
  git(["worktree", "add", "-b", branch, wtPath], repoPath);
  return realpathSync(wtPath);
}

/**
 * Convert a linked worktree into the post-migration shape: empty `.agents/`
 * containing only a `.moved-to` back-pointer to the main worktree.
 */
function migrateLinkedWorktree(linked: string, mainRoot: string): void {
  const agents = join(linked, ".agents");
  mkdirSync(agents, { recursive: true });
  writeFileSync(join(agents, ".moved-to"), `${mainRoot}\n`, "utf-8");
}

beforeEach(() => {
  TEST_ROOT = realpathSync(mkdtempSync(join(tmpdir(), "loaf-agents-config-")));
});

afterEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// Single-checkout (regression)
// ─────────────────────────────────────────────────────────────────────────────

describe("agents-config — single-checkout regression", () => {
  it("loafConfigPath resolves to <root>/.agents/loaf.json for a plain repo", () => {
    const main = createMainRepo("sc-path");
    expect(loafConfigPath(main)).toBe(join(main, ".agents", "loaf.json"));
  });

  it("readLoafConfig reads from <root>/.agents/loaf.json", () => {
    const main = createMainRepo("sc-read");
    writeFileSync(
      join(main, ".agents", "loaf.json"),
      JSON.stringify({ release: { versionFiles: ["pyproject.toml"] } }, null, 2) + "\n",
      "utf-8",
    );
    const cfg = readLoafConfig(main);
    expect(cfg.release?.versionFiles).toEqual(["pyproject.toml"]);
  });

  it("mergeLoafConfigIntegrations writes to <root>/.agents/loaf.json", () => {
    const main = createMainRepo("sc-write");
    mergeLoafConfigIntegrations(main, { linear: { enabled: true } });

    const written = JSON.parse(
      readFileSync(join(main, ".agents", "loaf.json"), "utf-8"),
    );
    expect(written.integrations?.linear).toEqual({ enabled: true });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Migrated linked worktree (SPEC-036)
// ─────────────────────────────────────────────────────────────────────────────

describe("agents-config — migrated linked worktree", () => {
  it("readLoafConfig follows the worktree resolution to the main worktree's loaf.json", () => {
    const main = createMainRepo("wt-read");
    const linked = addWorktree(main, "feat/read");
    migrateLinkedWorktree(linked, main);

    writeFileSync(
      join(main, ".agents", "loaf.json"),
      JSON.stringify(
        { release: { versionFiles: ["pyproject.toml"] } },
        null,
        2,
      ) + "\n",
      "utf-8",
    );

    const cfg = readLoafConfig(linked);
    expect(cfg.release?.versionFiles).toEqual(["pyproject.toml"]);
  });

  it("loafConfigPath returns the main worktree's path when called from a linked worktree", () => {
    const main = createMainRepo("wt-path");
    const linked = addWorktree(main, "feat/path");
    migrateLinkedWorktree(linked, main);

    expect(loafConfigPath(linked)).toBe(join(main, ".agents", "loaf.json"));
  });

  it("mergeLoafConfigIntegrations writes to the main worktree's loaf.json (not the linked worktree's)", () => {
    const main = createMainRepo("wt-write");
    const linked = addWorktree(main, "feat/write");
    migrateLinkedWorktree(linked, main);

    // Sanity: linked has only the back-pointer.
    expect(existsSync(join(linked, ".agents", "loaf.json"))).toBe(false);
    expect(existsSync(join(linked, ".agents", ".moved-to"))).toBe(true);

    mergeLoafConfigIntegrations(linked, { linear: { enabled: true } });

    // Write must land in main, not in the linked worktree.
    const mainConfigPath = join(main, ".agents", "loaf.json");
    expect(existsSync(mainConfigPath)).toBe(true);
    const written = JSON.parse(readFileSync(mainConfigPath, "utf-8"));
    expect(written.integrations?.linear).toEqual({ enabled: true });

    // No stray loaf.json next to the back-pointer.
    expect(existsSync(join(linked, ".agents", "loaf.json"))).toBe(false);
  });

  it("read preserves existing main config and merges integrations without clobbering siblings", () => {
    const main = createMainRepo("wt-merge");
    const linked = addWorktree(main, "feat/merge");
    migrateLinkedWorktree(linked, main);

    writeFileSync(
      join(main, ".agents", "loaf.json"),
      JSON.stringify(
        {
          release: { versionFiles: ["pyproject.toml"] },
          integrations: { serena: { enabled: false } },
        },
        null,
        2,
      ) + "\n",
      "utf-8",
    );

    mergeLoafConfigIntegrations(linked, { linear: { enabled: true } });

    const written = JSON.parse(
      readFileSync(join(main, ".agents", "loaf.json"), "utf-8"),
    );
    expect(written.release?.versionFiles).toEqual(["pyproject.toml"]);
    expect(written.integrations?.linear).toEqual({ enabled: true });
    expect(written.integrations?.serena).toEqual({ enabled: false });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Defensive fallback
// ─────────────────────────────────────────────────────────────────────────────

describe("agents-config — defensive fallback", () => {
  it("falls through to projectRoot when not in a git worktree at all", () => {
    // Plain tmp dir, no git init — `findMainWorktreeRoot` returns null and the
    // helper must keep the legacy behavior verbatim.
    const nonGit = realpathSync(mkdtempSync(join(TEST_ROOT, "no-git-")));
    expect(loafConfigPath(nonGit)).toBe(join(nonGit, ".agents", "loaf.json"));

    mkdirSync(join(nonGit, ".agents"), { recursive: true });
    writeFileSync(
      join(nonGit, ".agents", "loaf.json"),
      JSON.stringify({ release: { versionFiles: ["pyproject.toml"] } }, null, 2) + "\n",
      "utf-8",
    );
    const cfg = readLoafConfig(nonGit);
    expect(cfg.release?.versionFiles).toEqual(["pyproject.toml"]);
  });
});
