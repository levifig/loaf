/**
 * Version Command Tests
 *
 * Tests for the loaf version command — version detection, content counting,
 * and target directory detection.
 *
 * These tests exercise the counting and detection logic directly rather than
 * spawning a subprocess, keeping them fast and isolated.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  existsSync,
  mkdirSync,
  rmSync,
  writeFileSync,
} from "fs";
import { join } from "path";

// ─────────────────────────────────────────────────────────────────────────────
// Test Fixtures
// ─────────────────────────────────────────────────────────────────────────────

const TEST_ROOT = join(process.cwd(), ".test-version-command");

// ─────────────────────────────────────────────────────────────────────────────
// Helpers — inline logic mirroring version.ts for isolated testing
// (avoids importing the full module which pulls in build-time path resolution)
// ─────────────────────────────────────────────────────────────────────────────

import { readdirSync, readFileSync } from "fs";
import { parse as parseYaml } from "yaml";

function countSkills(rootDir: string): number {
  const skillsDir = join(rootDir, "content", "skills");
  if (!existsSync(skillsDir)) return 0;
  return readdirSync(skillsDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .length;
}

function countAgents(rootDir: string): number {
  const agentsDir = join(rootDir, "content", "agents");
  if (!existsSync(agentsDir)) return 0;
  return readdirSync(agentsDir)
    .filter((name) => name.endsWith(".md"))
    .length;
}

function countHooks(rootDir: string): number {
  const hooksPath = join(rootDir, "config", "hooks.yaml");
  if (!existsSync(hooksPath)) return 0;
  try {
    const config = parseYaml(readFileSync(hooksPath, "utf-8")) as {
      hooks?: {
        "pre-tool"?: unknown[];
        "post-tool"?: unknown[];
        session?: unknown[];
      };
    };
    const hooks = config?.hooks;
    if (!hooks) return 0;
    return (
      (hooks["pre-tool"]?.length ?? 0) +
      (hooks["post-tool"]?.length ?? 0) +
      (hooks.session?.length ?? 0)
    );
  } catch {
    return 0;
  }
}

const TARGET_OUTPUTS: Record<string, string> = {
  "claude-code": "plugins/loaf/",
  cursor: "dist/cursor/",
  opencode: "dist/opencode/",
  codex: "dist/codex/",
  gemini: "dist/gemini/",
};

function getBuiltTargets(rootDir: string): Array<{ name: string; path: string }> {
  const built: Array<{ name: string; path: string }> = [];
  for (const [name, relPath] of Object.entries(TARGET_OUTPUTS)) {
    if (existsSync(join(rootDir, relPath))) {
      built.push({ name, path: relPath });
    }
  }
  return built;
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

describe("version: skill counting", () => {
  it("returns 0 when skills directory does not exist", () => {
    expect(countSkills(TEST_ROOT)).toBe(0);
  });

  it("counts only directories in content/skills/", () => {
    const skillsDir = join(TEST_ROOT, "content", "skills");
    mkdirSync(join(skillsDir, "python-development"), { recursive: true });
    mkdirSync(join(skillsDir, "typescript-development"), { recursive: true });
    mkdirSync(join(skillsDir, "foundations"), { recursive: true });
    // Add a stray file that should not be counted
    writeFileSync(join(skillsDir, "README.md"), "# Skills\n", "utf-8");

    expect(countSkills(TEST_ROOT)).toBe(3);
  });
});

describe("version: agent counting", () => {
  it("returns 0 when agents directory does not exist", () => {
    expect(countAgents(TEST_ROOT)).toBe(0);
  });

  it("counts only .md files in content/agents/", () => {
    const agentsDir = join(TEST_ROOT, "content", "agents");
    mkdirSync(agentsDir, { recursive: true });
    writeFileSync(join(agentsDir, "backend-dev.md"), "# Backend\n", "utf-8");
    writeFileSync(join(agentsDir, "pm.md"), "# PM\n", "utf-8");
    // Sidecar files should not be counted
    writeFileSync(join(agentsDir, "backend-dev.claude-code.yaml"), "agent: true\n", "utf-8");
    writeFileSync(join(agentsDir, "backend-dev.opencode.yaml"), "agent: true\n", "utf-8");

    expect(countAgents(TEST_ROOT)).toBe(2);
  });
});

describe("version: hook counting", () => {
  it("returns 0 when hooks.yaml does not exist", () => {
    expect(countHooks(TEST_ROOT)).toBe(0);
  });

  it("counts hooks across all categories", () => {
    const configDir = join(TEST_ROOT, "config");
    mkdirSync(configDir, { recursive: true });
    writeFileSync(
      join(configDir, "hooks.yaml"),
      [
        "hooks:",
        "  pre-tool:",
        "    - id: check-a",
        "      script: hooks/pre-tool/a.sh",
        "    - id: check-b",
        "      script: hooks/pre-tool/b.sh",
        "  post-tool:",
        "    - id: post-a",
        "      script: hooks/post-tool/a.sh",
        "  session:",
        "    - id: session-start",
        "      script: hooks/session/start.sh",
        "    - id: session-end",
        "      script: hooks/session/end.sh",
        "    - id: pre-compact",
        "      script: hooks/session/compact.sh",
      ].join("\n"),
      "utf-8",
    );

    expect(countHooks(TEST_ROOT)).toBe(6);
  });

  it("returns 0 for empty hooks config", () => {
    const configDir = join(TEST_ROOT, "config");
    mkdirSync(configDir, { recursive: true });
    writeFileSync(join(configDir, "hooks.yaml"), "hooks:\n", "utf-8");

    expect(countHooks(TEST_ROOT)).toBe(0);
  });
});

describe("version: target detection", () => {
  it("returns empty when no targets are built", () => {
    expect(getBuiltTargets(TEST_ROOT)).toEqual([]);
  });

  it("detects claude-code target at plugins/loaf/", () => {
    mkdirSync(join(TEST_ROOT, "plugins", "loaf"), { recursive: true });

    const targets = getBuiltTargets(TEST_ROOT);
    expect(targets).toEqual([{ name: "claude-code", path: "plugins/loaf/" }]);
  });

  it("detects multiple built targets", () => {
    mkdirSync(join(TEST_ROOT, "plugins", "loaf"), { recursive: true });
    mkdirSync(join(TEST_ROOT, "dist", "cursor"), { recursive: true });
    mkdirSync(join(TEST_ROOT, "dist", "opencode"), { recursive: true });

    const targets = getBuiltTargets(TEST_ROOT);
    const names = targets.map((t) => t.name);
    expect(names).toContain("claude-code");
    expect(names).toContain("cursor");
    expect(names).toContain("opencode");
    expect(names).not.toContain("codex");
    expect(names).not.toContain("gemini");
  });
});

describe("version: against real content tree", () => {
  it("counts actual skills from the repo", () => {
    const repoRoot = process.cwd();
    const skills = countSkills(repoRoot);
    // The repo has 25 skills as of this writing; assert at least some exist
    expect(skills).toBeGreaterThanOrEqual(10);
  });

  it("counts actual agents from the repo", () => {
    const repoRoot = process.cwd();
    const agents = countAgents(repoRoot);
    expect(agents).toBeGreaterThanOrEqual(5);
  });

  it("counts actual hooks from the repo", () => {
    const repoRoot = process.cwd();
    const hooks = countHooks(repoRoot);
    expect(hooks).toBeGreaterThanOrEqual(15);
  });
});
