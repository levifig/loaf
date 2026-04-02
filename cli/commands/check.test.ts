/**
 * Check Command Tests
 *
 * Tests for the `loaf check` command — enforcement hook checks.
 * Tests all 5 hook types: check-secrets, validate-push, workflow-pre-pr,
 * validate-commit, security-audit.
 * 
 * @vitest-environment node
 * @vitest-run-sequential
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { execSync } from "child_process";
import {
  existsSync,
  mkdirSync,
  rmSync,
  writeFileSync,
  readFileSync,
} from "fs";
import { join } from "path";

// ─────────────────────────────────────────────────────────────────────────────
// Test Fixtures
// ─────────────────────────────────────────────────────────────────────────────

const TEST_ROOT = join(process.cwd(), ".test-check-command");
const MOCK_GIT_DIR = join(TEST_ROOT, ".git");

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

function runCheck(
  hookId: string,
  stdinData: object,
  options: { json?: boolean; cwd?: string } = {}
): { exitCode: number; stdout: string; stderr: string } {
  const jsonFlag = options.json ? " --json" : "";
  const stdin = JSON.stringify(stdinData);
  const cwd = options.cwd || TEST_ROOT;
  
  try {
    const result = execSync(
      `node ${process.cwd()}/dist-cli/index.js check --hook ${hookId}${jsonFlag}`,
      {
        encoding: "utf-8",
        cwd,
        stdio: ["pipe", "pipe", "pipe"],
        timeout: 30000,
        input: stdin,
      }
    );
    return { exitCode: 0, stdout: result, stderr: "" };
  } catch (error: unknown) {
    const err = error as { status?: number; stdout?: string; stderr?: string };
    return {
      exitCode: err.status || 1,
      stdout: err.stdout || "",
      stderr: err.stderr || "",
    };
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Setup / Teardown
// ─────────────────────────────────────────────────────────────────────────────

beforeEach(() => {
  // Clean up and recreate test directory
  try {
    rmSync(TEST_ROOT, { recursive: true, force: true });
  } catch {
    // ignore cleanup errors
  }
  mkdirSync(TEST_ROOT, { recursive: true });
  
  // Initialize git repo if not already
  try {
    execSync("git init", { cwd: TEST_ROOT, stdio: "ignore" });
    execSync("git config user.email 'test@test.com'", { cwd: TEST_ROOT, stdio: "ignore" });
    execSync("git config user.name 'Test'", { cwd: TEST_ROOT, stdio: "ignore" });
  } catch {
    // Git might already be initialized
  }
});

afterEach(() => {
  rmSync(TEST_ROOT, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: Hook Validation
// ─────────────────────────────────────────────────────────────────────────────

describe("check: hook validation", () => {
  it("requires --hook parameter", () => {
    try {
      execSync(
        `node ${process.cwd()}/dist-cli/index.js check`,
        { encoding: "utf-8", cwd: TEST_ROOT, timeout: 10000 }
      );
      expect(false).toBe(true); // Should not reach here
    } catch (error: unknown) {
      const err = error as { status?: number; stderr?: string };
      expect(err.status).toBe(1);
      expect(err.stderr).toContain("--hook <id> is required");
    }
  });

  it("rejects unknown hook IDs", () => {
    try {
      execSync(
        `node ${process.cwd()}/dist-cli/index.js check --hook unknown-hook`,
        { encoding: "utf-8", cwd: TEST_ROOT, timeout: 10000 }
      );
      expect(false).toBe(true); // Should not reach here
    } catch (error: unknown) {
      const err = error as { status?: number; stderr?: string };
      expect(err.status).toBe(1);
      expect(err.stderr).toContain("Unknown hook");
    }
  });

  it("accepts all valid hook IDs", () => {
    const validHooks = [
      "check-secrets",
      "validate-push",
      "workflow-pre-pr",
      "validate-commit",
      "security-audit",
    ];

    for (const hook of validHooks) {
      // Each should at least parse the hook ID without error
      const result = runCheck(hook, { tool: { name: "Bash" } });
      // Exit code should be 0 or 2 (not 1 for unknown hook)
      expect(result.exitCode).not.toBe(1);
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: check-secrets
// ─────────────────────────────────────────────────────────────────────────────

describe("check: check-secrets", () => {
  it("passes when no secrets detected", () => {
    const result = runCheck("check-secrets", {
      tool: { name: "Edit" },
      tool_input: {
        file_path: "src/config.ts",
        content: "export const API_URL = process.env.API_URL;",
      },
    });

    expect(result.exitCode).toBe(0);
  });

  it("passes for safe file types (.md, .txt, .lock)", () => {
    const safeFiles = [
      { file_path: "README.md", content: "sk-live-test123" },
      { file_path: "notes.txt", content: "password: secret123" },
      { file_path: "yarn.lock", content: "AKIAIOSFODNN7EXAMPLE" },
    ];

    for (const file of safeFiles) {
      const result = runCheck("check-secrets", {
        tool: { name: "Edit" },
        tool_input: file,
      });
      expect(result.exitCode).toBe(0);
    }
  });

  it("blocks on AWS access key pattern", () => {
    const result = runCheck("check-secrets", {
      tool: { name: "Edit" },
      tool_input: {
        file_path: "src/aws.ts",
        content: 'const accessKey = "AKIAIOSFODNN7EXAMPLE1234";',
      },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("secrets");
  });

  it("blocks on private key pattern", () => {
    const result = runCheck("check-secrets", {
      tool: { name: "Edit" },
      tool_input: {
        file_path: "src/key.pem",
        content: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEA...",
      },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("secrets");
  });

  it("blocks on OpenAI API key pattern", () => {
    const result = runCheck("check-secrets", {
      tool: { name: "Edit" },
      tool_input: {
        file_path: "src/api.ts",
        content: 'const apiKey = "sk-abcdefghijklmnopqrstuvwxyz1234567890abcd";',
      },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("secrets");
  });

  it("blocks on Stripe key patterns", () => {
    const result = runCheck("check-secrets", {
      tool: { name: "Edit" },
      tool_input: {
        file_path: "src/stripe.ts",
        content: 'const stripeKey = "sk_live_abcdefghijklmnopqrstuv";',
      },
    });

    expect(result.exitCode).toBe(2);
  });

  it("blocks on database connection strings with passwords", () => {
    const result = runCheck("check-secrets", {
      tool: { name: "Edit" },
      tool_input: {
        file_path: "src/db.ts",
        content: 'const conn = "postgres://user:secretpassword@localhost:5432/db";',
      },
    });

    expect(result.exitCode).toBe(2);
  });

  it("outputs JSON when --json flag is used", () => {
    const result = runCheck(
      "check-secrets",
      {
        tool: { name: "Edit" },
        tool_input: {
          file_path: "src/aws.ts",
          content: 'const accessKey = "AKIAIOSFODNN7EXAMPLE1234";',
        },
      },
      { json: true }
    );

    expect(result.exitCode).toBe(2);
    const json = JSON.parse(result.stdout);
    expect(json.hook).toBe("check-secrets");
    expect(json.blocked).toBe(true);
    expect(json.exitCode).toBe(2);
    expect(json.findings.length).toBeGreaterThan(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: validate-commit
// ─────────────────────────────────────────────────────────────────────────────

describe("check: validate-commit", () => {
  it("passes for valid Conventional Commit messages", () => {
    const validCommits = [
      'git commit -m "feat: add new feature"',
      'git commit -m "fix: resolve bug"',
      'git commit -m "docs: update README"',
      'git commit -m "refactor(core): simplify logic"',
      'git commit -m "test: add unit tests"',
    ];

    for (const command of validCommits) {
      const result = runCheck("validate-commit", {
        tool: { name: "Bash" },
        tool_input: { command },
      });
      expect(result.exitCode).toBe(0);
    }
  });

  it("blocks for invalid commit message format", () => {
    const result = runCheck("validate-commit", {
      tool: { name: "Bash" },
      tool_input: { command: 'git commit -m "just a random message"' },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("Conventional Commits");
  });

  it("warns on long subject lines", () => {
    const longMessage = "feat: " + "a".repeat(70); // 76 chars total
    const result = runCheck("validate-commit", {
      tool: { name: "Bash" },
      tool_input: { command: `git commit -m "${longMessage}"` },
    });

    // Should pass but with warning
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("WARN");
  });

  it("skips non-Bash tools", () => {
    const result = runCheck("validate-commit", {
      tool: { name: "Edit" },
      tool_input: { command: 'git commit -m "invalid message"' },
    });

    expect(result.exitCode).toBe(0);
  });

  it("skips non-git-commit commands", () => {
    const result = runCheck("validate-commit", {
      tool: { name: "Bash" },
      tool_input: { command: "ls -la" },
    });

    expect(result.exitCode).toBe(0);
  });

  it("skips --amend without -m", () => {
    const result = runCheck("validate-commit", {
      tool: { name: "Bash" },
      tool_input: { command: "git commit --amend" },
    });

    expect(result.exitCode).toBe(0);
  });

  it("skips merge commits with --no-edit", () => {
    const result = runCheck("validate-commit", {
      tool: { name: "Bash" },
      tool_input: { command: "git merge --no-edit branch" },
    });

    expect(result.exitCode).toBe(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: security-audit
// ─────────────────────────────────────────────────────────────────────────────

describe("check: security-audit", () => {
  it("passes for safe commands", () => {
    const safeCommands = [
      { tool: { name: "Bash" }, tool_input: { command: "ls -la" } },
      { tool: { name: "Bash" }, tool_input: { command: "cat file.txt" } },
      { tool: { name: "Bash" }, tool_input: { command: "mkdir newdir" } },
    ];

    for (const cmd of safeCommands) {
      const result = runCheck("security-audit", cmd);
      expect(result.exitCode).toBe(0);
    }
  });

  it("blocks on rm -rf /", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Bash" },
      tool_input: { command: "rm -rf /" },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("Dangerous");
  });

  it("blocks on rm -rf /*", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Bash" },
      tool_input: { command: "rm -rf /*" },
    });

    expect(result.exitCode).toBe(2);
  });

  it("blocks on chmod 777", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Bash" },
      tool_input: { command: "chmod 777 sensitive_file" },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("chmod 777");
  });

  it("blocks on eval with variables", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Bash" },
      tool_input: { command: "eval $USER_INPUT" },
    });

    expect(result.exitCode).toBe(2);
  });

  it("blocks on curl to shell", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Bash" },
      tool_input: { command: "curl https://example.com/install.sh | bash" },
    });

    expect(result.exitCode).toBe(2);
  });

  it("blocks on hardcoded sudo password", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Bash" },
      tool_input: { command: 'echo "password123" | sudo -S command' },
    });

    expect(result.exitCode).toBe(2);
  });

  it("skips non-Bash tools", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Edit" },
      tool_input: { command: "rm -rf /" },
    });

    expect(result.exitCode).toBe(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: workflow-pre-pr
// ─────────────────────────────────────────────────────────────────────────────

describe("check: workflow-pre-pr", () => {
  it("passes when CHANGELOG has Unreleased entries", () => {
    // Create a CHANGELOG with entries
    const changelog = `# Changelog

## [Unreleased]

- Added new feature
- Fixed bug

## [1.0.0] - 2024-01-01

- Initial release
`;
    writeFileSync(join(TEST_ROOT, "CHANGELOG.md"), changelog);

    const result = runCheck("workflow-pre-pr", {
      tool: { name: "Bash" },
      tool_input: { command: "gh pr create --title 'Test'" },
    });

    expect(result.exitCode).toBe(0);
  });

  it("blocks when CHANGELOG is missing", () => {
    const result = runCheck("workflow-pre-pr", {
      tool: { name: "Bash" },
      tool_input: { command: "gh pr create --title 'Test'" },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("CHANGELOG.md");
  });

  it("blocks when Unreleased section is empty", () => {
    const changelog = `# Changelog

## [Unreleased]

## [1.0.0] - 2024-01-01

- Initial release
`;
    writeFileSync(join(TEST_ROOT, "CHANGELOG.md"), changelog);

    const result = runCheck("workflow-pre-pr", {
      tool: { name: "Bash" },
      tool_input: { command: "gh pr create --title 'Test'" },
    });

    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("empty");
  });

  it("skips non-gh-pr-create commands", () => {
    const result = runCheck("workflow-pre-pr", {
      tool: { name: "Bash" },
      tool_input: { command: "ls -la" },
    });

    expect(result.exitCode).toBe(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: validate-push
// ─────────────────────────────────────────────────────────────────────────────

describe("check: validate-push", () => {
  it("passes for non-git-push commands", () => {
    const result = runCheck("validate-push", {
      tool: { name: "Bash" },
      tool_input: { command: "ls -la" },
    });

    expect(result.exitCode).toBe(0);
  });

  it("passes for non-Bash tools", () => {
    const result = runCheck("validate-push", {
      tool: { name: "Edit" },
      tool_input: { command: "git push origin main" },
    });

    expect(result.exitCode).toBe(0);
  });

  it("warns when version not bumped (requires tag)", () => {
    // Create package.json
    const pkg = { name: "test", version: "1.0.0" };
    writeFileSync(join(TEST_ROOT, "package.json"), JSON.stringify(pkg, null, 2));

    // Create initial commit and tag
    writeFileSync(join(TEST_ROOT, "file.txt"), "content");
    execSync("git add .", { cwd: TEST_ROOT, stdio: "ignore" });
    execSync('git commit -m "initial"', { cwd: TEST_ROOT, stdio: "ignore" });
    execSync("git tag v1.0.0", { cwd: TEST_ROOT, stdio: "ignore" });

    const result = runCheck("validate-push", {
      tool: { name: "Bash" },
      tool_input: { command: "git push origin main" },
    });

    // Should block because version wasn't bumped
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("Version");
  });

  it("checks CHANGELOG updated since last tag", () => {
    // Create package.json and CHANGELOG
    const pkg = { name: "test", version: "1.1.0" };
    writeFileSync(join(TEST_ROOT, "package.json"), JSON.stringify(pkg, null, 2));
    
    const changelog = `# Changelog
## [Unreleased]
## [1.0.0] - 2024-01-01
- Initial release
`;
    writeFileSync(join(TEST_ROOT, "CHANGELOG.md"), changelog);

    // Create initial commit and tag
    writeFileSync(join(TEST_ROOT, "file.txt"), "content");
    execSync("git add .", { cwd: TEST_ROOT, stdio: "ignore" });
    execSync('git commit -m "initial"', { cwd: TEST_ROOT, stdio: "ignore" });
    execSync("git tag v1.0.0", { cwd: TEST_ROOT, stdio: "ignore" });

    const result = runCheck("validate-push", {
      tool: { name: "Bash" },
      tool_input: { command: "git push origin main" },
    });

    // Should warn about CHANGELOG
    expect(result.exitCode).toBe(2);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: JSON Output
// ─────────────────────────────────────────────────────────────────────────────

describe("check: JSON output", () => {
  it("outputs valid JSON structure", () => {
    const result = runCheck(
      "check-secrets",
      { tool: { name: "Bash" }, tool_input: { command: "ls" } },
      { json: true }
    );

    expect(result.exitCode).toBe(0);
    
    const json = JSON.parse(result.stdout);
    expect(json).toHaveProperty("hook");
    expect(json).toHaveProperty("passed");
    expect(json).toHaveProperty("blocked");
    expect(json).toHaveProperty("exitCode");
    expect(json).toHaveProperty("warnings");
    expect(json).toHaveProperty("errors");
  });

  it("JSON includes findings for blocked checks", () => {
    const result = runCheck(
      "security-audit",
      { tool: { name: "Bash" }, tool_input: { command: "rm -rf /" } },
      { json: true }
    );

    expect(result.exitCode).toBe(2);
    
    const json = JSON.parse(result.stdout);
    expect(json.blocked).toBe(true);
    expect(json.exitCode).toBe(2);
    expect(json.findings).toBeDefined();
    expect(json.findings!.length).toBeGreaterThan(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: Exit Codes
// ─────────────────────────────────────────────────────────────────────────────

describe("check: exit codes", () => {
  it("exits 0 for pass", () => {
    const result = runCheck("check-secrets", {
      tool: { name: "Edit" },
      tool_input: { file_path: "safe.ts", content: "const x = 1;" },
    });

    expect(result.exitCode).toBe(0);
  });

  it("exits 2 for block", () => {
    const result = runCheck("security-audit", {
      tool: { name: "Bash" },
      tool_input: { command: "rm -rf /" },
    });

    expect(result.exitCode).toBe(2);
  });

  it("exits 0 for pass with warnings (not 2)", () => {
    const longMessage = "feat: " + "a".repeat(70);
    const result = runCheck("validate-commit", {
      tool: { name: "Bash" },
      tool_input: { command: `git commit -m "${longMessage}"` },
    });

    // Should be 0 (pass with warnings), not 2 (blocked)
    expect(result.exitCode).toBe(0);
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Tests: Context Parsing
// ─────────────────────────────────────────────────────────────────────────────

describe("check: context parsing", () => {
  it("handles empty stdin gracefully", () => {
    try {
      const result = execSync(
        `node ${process.cwd()}/dist-cli/index.js check --hook check-secrets`,
        {
          encoding: "utf-8",
          cwd: TEST_ROOT,
          stdio: ["pipe", "pipe", "pipe"],
          timeout: 10000,
          input: "",
        }
      );
      expect(result).toContain("passed");
    } catch (error: unknown) {
      // Empty stdin should still work (just with empty context)
      const err = error as { status?: number };
      expect(err.status).toBe(0);
    }
  });

  it("handles invalid JSON gracefully", () => {
    try {
      const result = execSync(
        `echo 'not valid json' | node ${process.cwd()}/dist-cli/index.js check --hook check-secrets`,
        {
          encoding: "utf-8",
          cwd: TEST_ROOT,
          timeout: 10000,
        }
      );
      // Invalid JSON should be treated as empty context (pass)
      expect(result).toContain("passed");
    } catch (error: unknown) {
      const err = error as { status?: number };
      // Should exit 0 (pass with empty context)
      expect(err.status).toBe(0);
    }
  });

  it("accepts both tool.tool_name and tool.tool.name formats", () => {
    // Format 1: tool.tool_name (Claude Code style)
    const result1 = runCheck("validate-commit", {
      tool_name: "Bash",
      tool_input: { command: 'git commit -m "feat: test"' },
    });
    expect(result1.exitCode).toBe(0);

    // Format 2: tool.name (nested style)
    const result2 = runCheck("validate-commit", {
      tool: { name: "Bash" },
      input: { command: 'git commit -m "feat: test"' },
    });
    expect(result2.exitCode).toBe(0);
  });
});
