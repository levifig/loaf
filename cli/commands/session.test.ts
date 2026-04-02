/**
 * Session Command Integration Tests
 *
 * Black-box tests for the loaf session command using temp git repos.
 * Tests CLI behavior via child_process.spawn/execFile.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { execFile, spawn } from "child_process";
import { promisify } from "util";
import {
  existsSync,
  mkdirSync,
  readdirSync,
  rmSync,
  writeFileSync,
  readFileSync,
} from "fs";
import { join } from "path";

const execFileAsync = promisify(execFile);

// ─────────────────────────────────────────────────────────────────────────────
// Test Fixtures
// ─────────────────────────────────────────────────────────────────────────────

const TEST_ROOT = join(process.cwd(), ".test-session-command");
const CLI_PATH = join(process.cwd(), "dist-cli/index.js");

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

async function runLoaf(
  args: string[],
  options: { cwd: string; input?: string } = { cwd: TEST_ROOT }
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    const child = spawn("node", [CLI_PATH, "session", ...args], {
      cwd: options.cwd,
      stdio: ["pipe", "pipe", "pipe"],
    });

    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (data) => {
      stdout += data.toString();
    });

    child.stderr.on("data", (data) => {
      stderr += data.toString();
    });

    if (options.input) {
      child.stdin.write(options.input);
      child.stdin.end();
    }

    child.on("close", (exitCode) => {
      resolve({ stdout, stderr, exitCode: exitCode ?? 0 });
    });
  });
}

function createTempRepo(name: string): string {
  const repoPath = join(TEST_ROOT, name);
  mkdirSync(repoPath, { recursive: true });
  
  // Initialize git repo
  execFileSync("git", ["init"], { cwd: repoPath });
  execFileSync("git", ["config", "user.email", "test@test.com"], { cwd: repoPath });
  execFileSync("git", ["config", "user.name", "Test User"], { cwd: repoPath });
  
  // Create initial commit
  writeFileSync(join(repoPath, "README.md"), "# Test\n", "utf-8");
  execFileSync("git", ["add", "."], { cwd: repoPath });
  execFileSync("git", ["commit", "-m", "Initial commit"], { cwd: repoPath });
  
  // Create .agents directory
  mkdirSync(join(repoPath, ".agents"), { recursive: true });
  writeFileSync(
    join(repoPath, ".agents/AGENTS.md"),
    "# Project Instructions\n",
    "utf-8"
  );
  
  return repoPath;
}

function execFileSync(cmd: string, args: string[], options: { cwd: string }) {
  require("child_process").execFileSync(cmd, args, { ...options, stdio: "pipe" });
}

function getSessionFiles(repoPath: string): string[] {
  const sessionsDir = join(repoPath, ".agents/sessions");
  if (!existsSync(sessionsDir)) return [];
  return readdirSync(sessionsDir).filter(f => f.endsWith(".md") && !f.startsWith("."));
}

// ─────────────────────────────────────────────────────────────────────────────
// Setup / Teardown
// ─────────────────────────────────────────────────────────────────────────────

beforeEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
  mkdirSync(TEST_ROOT, { recursive: true });
});

afterEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

describe("session: start", () => {
  it("creates an ad-hoc session in a fresh repo with .agents/", async () => {
    const repoPath = createTempRepo("adhoc-test");
    
    const result = await runLoaf(["start"], { cwd: repoPath });
    
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Creating new session file");
    
    const sessionFiles = getSessionFiles(repoPath);
    expect(sessionFiles.length).toBe(1);
    
    const content = readFileSync(
      join(repoPath, ".agents/sessions", sessionFiles[0]),
      "utf-8"
    );
    expect(content).toContain("Ad-hoc session for branch");
    expect(content).toContain("status: active");
  });

  it("uses detached-<sha> branch key when in detached HEAD", async () => {
    const repoPath = createTempRepo("detached-test");
    
    // Checkout detached state
    const commitSha = require("child_process")
      .execFileSync("git", ["rev-parse", "--short", "HEAD"], { cwd: repoPath, encoding: "utf-8" })
      .trim();
    require("child_process").execFileSync("git", ["checkout", "--detach"], { cwd: repoPath });
    
    const result = await runLoaf(["start"], { cwd: repoPath });
    
    expect(result.exitCode).toBe(0);
    expect(result.stderr).toContain(`detached-${commitSha}`);
    
    const sessionFiles = getSessionFiles(repoPath);
    expect(sessionFiles.length).toBe(1);
    
    const content = readFileSync(
      join(repoPath, ".agents/sessions", sessionFiles[0]),
      "utf-8"
    );
    expect(content).toContain(`branch: detached-${commitSha}`);
  });

  it("two concurrent session start processes leave exactly one session file", async () => {
    const repoPath = createTempRepo("concurrent-test");
    
    // Start two session start processes simultaneously
    const [result1, result2] = await Promise.all([
      runLoaf(["start"], { cwd: repoPath }),
      runLoaf(["start"], { cwd: repoPath }),
    ]);
    
    // Both should exit 0
    expect(result1.exitCode).toBe(0);
    expect(result2.exitCode).toBe(0);
    
    // Only one session file should exist
    const sessionFiles = getSessionFiles(repoPath);
    expect(sessionFiles.length).toBe(1);
    
    // Verify there's only ONE "Start" section (not duplicate starts)
    const content = readFileSync(
      join(repoPath, ".agents/sessions", sessionFiles[0]),
      "utf-8"
    );
    const startMatches = content.match(/## \d{4}-\d{2}-\d{2}.*— Start/g);
    expect(startMatches?.length ?? 0).toBe(1);
  }, 30000); // Higher timeout for concurrent operations

  it("resuming a paused session adds resume entry and updates status", async () => {
    const repoPath = createTempRepo("resume-test");
    
    // Start session
    const startResult = await runLoaf(["start"], { cwd: repoPath });
    expect(startResult.exitCode).toBe(0);
    
    // End session (pauses it)
    const endResult = await runLoaf(["end"], { cwd: repoPath });
    expect(endResult.exitCode).toBe(0);
    
    // Resume session
    const resumeResult = await runLoaf(["start"], { cwd: repoPath });
    expect(resumeResult.exitCode).toBe(0);
    expect(resumeResult.stdout).toContain("Resuming existing session");
    
    // Verify session file
    const sessionFiles = getSessionFiles(repoPath);
    expect(sessionFiles.length).toBe(1);
    
    const content = readFileSync(
      join(repoPath, ".agents/sessions", sessionFiles[0]),
      "utf-8"
    );
    
    // Should have both Start and Resume sections
    expect(content).toContain("— Start");
    expect(content).toContain("— Resume");
    
    // Should have a resume entry
    expect(content).toContain("session resumed");
    
    // Status should be active (updated from paused)
    expect(content).toContain("status: active");
  });
});

describe("session: log", () => {
  it("hook-safe exit when no active session exists", async () => {
    const repoPath = createTempRepo("no-session-hook-test");
    
    // Try to log from hook without starting a session first
    // Should exit 0 (hook-safe) not 1 (error)
    const result = await runLoaf(["log", "--from-hook"], { 
      cwd: repoPath,
      input: JSON.stringify({ tool_input: { command: "git commit -m 'test commit'" } })
    });
    
    // Should exit 0 (no-op for hooks) rather than failing
    expect(result.exitCode).toBe(0);
  }, 10000);

  it("accepts nested tool.input format from hooks", async () => {
    const repoPath = createTempRepo("nested-payload-test");
    
    // Start a session
    await runLoaf(["start"], { cwd: repoPath });
    
    // Log with nested format (tool.input instead of tool_input)
    const result = await runLoaf(["log", "--from-hook"], { 
      cwd: repoPath,
      input: JSON.stringify({ 
        tool: { 
          name: "Bash",
          input: { command: "git commit -m 'feat: add feature'" } 
        } 
      })
    });
    
    expect(result.exitCode).toBe(0);
    
    // Verify the commit was logged
    const sessionFiles = getSessionFiles(repoPath);
    const content = readFileSync(
      join(repoPath, ".agents/sessions", sessionFiles[0]),
      "utf-8"
    );
    
    expect(content).toContain("commit(");
    expect(content).toContain("feat: add feature");
  }, 10000);

  it("two concurrent session log processes both preserve their entries", async () => {
    const repoPath = createTempRepo("concurrent-log-test");
    
    // Start a session first
    await runLoaf(["start"], { cwd: repoPath });
    
    // Two concurrent log entries
    const [result1, result2] = await Promise.all([
      runLoaf(["log", "discover(test): finding one"], { cwd: repoPath }),
      runLoaf(["log", "discover(test): finding two"], { cwd: repoPath }),
    ]);
    
    // Both should succeed
    expect(result1.exitCode).toBe(0);
    expect(result2.exitCode).toBe(0);
    
    // Both entries should be in the session file
    const sessionFiles = getSessionFiles(repoPath);
    expect(sessionFiles.length).toBe(1);
    
    const content = readFileSync(
      join(repoPath, ".agents/sessions", sessionFiles[0]),
      "utf-8"
    );
    
    expect(content).toContain("discover(test): finding one");
    expect(content).toContain("discover(test): finding two");
  }, 30000);
});

describe("session: list", () => {
  it("shows active sessions", async () => {
    const repoPath = createTempRepo("list-test");
    
    await runLoaf(["start"], { cwd: repoPath });
    
    const result = await runLoaf(["list"], { cwd: repoPath });
    
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Active Sessions");
    expect(result.stdout).toContain("1 active");
  });

  it("shows active and archived sessions with --all", async () => {
    const repoPath = createTempRepo("list-all-test");
    
    // Create and archive a session
    await runLoaf(["start"], { cwd: repoPath });
    await runLoaf(["archive"], { cwd: repoPath });
    
    // Create another active session
    // Switch to new branch
    require("child_process").execFileSync("git", ["checkout", "-b", "feature-branch"], { cwd: repoPath });
    await runLoaf(["start"], { cwd: repoPath });
    
    const result = await runLoaf(["list", "--all"], { cwd: repoPath });
    
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Active Sessions");
    expect(result.stdout).toContain("Archived Sessions");
    expect(result.stdout).toContain("1 active, 1 archived");
  });
});
