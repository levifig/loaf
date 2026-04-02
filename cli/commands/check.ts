/**
 * loaf check command
 *
 * Run enforcement hook checks for quality gates and security.
 * Reads hook context via stdin (JSON from harness) and returns appropriate exit codes.
 *
 * Exit codes:
 *   0 = pass (including warnings)
 *   1 = internal error only
 *   2 = block (check failed)
 *
 * Usage:
 *   echo '{"tool":{"name":"Bash"},"input":{"command":"git push"}}' | loaf check --hook validate-push
 *   loaf check --hook check-secrets --json < context.json
 */

import { Command } from "commander";
import { readFileSync, existsSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

interface HookContext {
  tool?: {
    name?: string;
  };
  tool_name?: string;
  tool_input?: {
    command?: string;
    file_path?: string;
    content?: string;
    new_string?: string;
    [key: string]: unknown;
  };
  input?: {
    command?: string;
    file_path?: string;
    content?: string;
    new_string?: string;
    [key: string]: unknown;
  };
  agent_type?: string;
  validation_level?: string;
}

interface CheckResult {
  passed: boolean;
  blocked: boolean;
  warnings: string[];
  errors: string[];
  findings?: string[];
}

interface JsonOutput {
  hook: string;
  passed: boolean;
  blocked: boolean;
  exitCode: number;
  warnings: string[];
  errors: string[];
  findings?: string[];
}

// ─────────────────────────────────────────────────────────────────────────────
// ANSI color helpers
// ─────────────────────────────────────────────────────────────────────────────

const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;

// ─────────────────────────────────────────────────────────────────────────────
// Utility Functions
// ─────────────────────────────────────────────────────────────────────────────

async function readStdin(): Promise<string> {
  return new Promise((resolve, reject) => {
    let data = "";
    process.stdin.setEncoding("utf8");
    
    process.stdin.on("data", (chunk) => {
      data += chunk;
    });
    
    process.stdin.on("end", () => {
      resolve(data);
    });
    
    process.stdin.on("error", (err) => {
      reject(err);
    });
    
    // Handle case where stdin is already closed or not available
    if (process.stdin.isTTY) {
      resolve("");
    }
  });
}

function parseContext(stdinData: string): HookContext {
  if (!stdinData.trim()) {
    return {};
  }
  
  try {
    return JSON.parse(stdinData) as HookContext;
  } catch {
    return {};
  }
}

function getToolName(context: HookContext): string {
  return context.tool?.name || context.tool_name || "";
}

function getCommand(context: HookContext): string {
  return context.tool_input?.command || context.input?.command || "";
}

function getFilePath(context: HookContext): string {
  return context.tool_input?.file_path || context.input?.file_path || "";
}

function getContent(context: HookContext): string {
  return context.tool_input?.content || 
         context.tool_input?.new_string || 
         context.input?.content || 
         context.input?.new_string || 
         "";
}

function outputJson(result: CheckResult, hookId: string): void {
  const output: JsonOutput = {
    hook: hookId,
    passed: result.passed && !result.blocked,
    blocked: result.blocked,
    exitCode: result.blocked ? 2 : 0,
    warnings: result.warnings,
    errors: result.errors,
    findings: result.findings,
  };
  console.log(JSON.stringify(output, null, 2));
}

function outputText(result: CheckResult, hookId: string): void {
  if (result.blocked) {
    console.error(`\n${red("✗")} ${bold(hookId)}: blocked`);
    for (const error of result.errors) {
      console.error(`   ${red("•")} ${error}`);
    }
    if (result.findings && result.findings.length > 0) {
      console.error(`\n   ${bold("Findings:")}`);
      for (const finding of result.findings) {
        console.error(`   ${gray("-")} ${finding}`);
      }
    }
  } else if (result.warnings.length > 0) {
    console.log(`\n${yellow("⚠")} ${bold(hookId)}: passed with warnings`);
    for (const warning of result.warnings) {
      console.log(`   WARN: ${warning}`);
    }
  } else {
    console.log(`${green("✓")} ${bold(hookId)}: passed`);
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Check: check-secrets
// Scan for hardcoded secrets, API keys, credentials
// ─────────────────────────────────────────────────────────────────────────────

async function checkSecrets(context: HookContext): Promise<CheckResult> {
  const result: CheckResult = {
    passed: true,
    blocked: false,
    warnings: [],
    errors: [],
    findings: [],
  };

  const filePath = getFilePath(context);
  const content = getContent(context);

  // Skip if no file path or content
  if (!filePath && !content) {
    return result;
  }

  // Skip known safe files
  const safePatterns = [
    /\.md$/,
    /\.txt$/,
    /\.lock$/,
    /package-lock\.json$/,
    /yarn\.lock$/,
    /poetry\.lock$/,
    /\.example$/,
    /\.template$/,
    /\.sample$/,
  ];

  for (const pattern of safePatterns) {
    if (pattern.test(filePath)) {
      return result;
    }
  }

  // Secret patterns to detect
  const secretPatterns: Array<{ name: string; regex: RegExp }> = [
    { name: "AWS Access Key ID", regex: /AKIA[0-9A-Z]{16}/ },
    { name: "AWS Secret Key", regex: /aws_secret_access_key\s*=\s*["']?[A-Za-z0-9/+=]{40}["']?/i },
    { name: "OpenAI API Key", regex: /sk-[a-zA-Z0-9]{20,}/ },
    { name: "Stripe Live Key", regex: /sk_live_[a-zA-Z0-9]{10,}/ },
    { name: "Stripe Test Key", regex: /sk_test_[a-zA-Z0-9]{10,}/ },
    { name: "Private Key", regex: /-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----/ },
    { name: "Database Connection", regex: /(postgres|mysql|mongodb):\/\/[^:]+:[^@]+@/ },
    { name: "Password Assignment", regex: /password\s*=\s*["'][^"']{8,}["']/i },
    { name: "Secret Assignment", regex: /secret\s*=\s*["'][^"']{8,}["']/i },
    { name: "API Key Assignment", regex: /api_key\s*=\s*["'][^"']{16,}["']/i },
    { name: "JWT Token", regex: /eyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_.+/=]*/ },
    { name: "GitHub Token", regex: /gh[pousr]_[A-Za-z0-9_]{36}/ },
  ];

  const targetContent = content || "";
  const foundSecrets: string[] = [];

  for (const { name, regex } of secretPatterns) {
    const matches = targetContent.match(regex);
    if (matches) {
      const matchPreview = matches[0].substring(0, 40);
      foundSecrets.push(`${name}: ${matchPreview}...`);
    }
  }

  if (foundSecrets.length > 0) {
    result.passed = false;
    result.blocked = true;
    result.errors.push(`Potential secrets detected in ${filePath || "input"}`);
    result.findings = foundSecrets;
  }

  return result;
}

// ─────────────────────────────────────────────────────────────────────────────
// Check: validate-push
// Verify version bump, CHANGELOG entry, and successful build
// ─────────────────────────────────────────────────────────────────────────────

async function validatePush(context: HookContext): Promise<CheckResult> {
  const result: CheckResult = {
    passed: true,
    blocked: false,
    warnings: [],
    errors: [],
  };

  const toolName = getToolName(context);
  const command = getCommand(context);

  // Only process Bash tool with git push
  if (toolName !== "Bash" || !command.match(/^git\s+push/)) {
    return result;
  }

  const errors: string[] = [];

  // Detect project type
  const hasPackageJson = existsSync("package.json");
  let hasBuildScript = false;
  
  if (hasPackageJson) {
    try {
      const pkg = JSON.parse(readFileSync("package.json", "utf-8"));
      hasBuildScript = !!pkg.scripts?.build;
    } catch {
      // ignore
    }
  }

  // Check 1: Version bump since last tag
  try {
    const lastTag = execSync("git describe --tags --abbrev=0 2>/dev/null", {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "ignore"],
    }).trim();

    if (lastTag && hasPackageJson) {
      try {
        const pkg = JSON.parse(readFileSync("package.json", "utf-8"));
        const currentVersion = pkg.version;
        
        const tagPkgContent = execSync(
          `git show ${lastTag}:package.json 2>/dev/null`,
          { encoding: "utf-8", stdio: ["pipe", "pipe", "ignore"] }
        );
        const tagPkg = JSON.parse(tagPkgContent);
        const tagVersion = tagPkg.version;

        if (currentVersion === tagVersion) {
          errors.push(`Version not bumped since ${lastTag} (still ${currentVersion})`);
        }
      } catch {
        // ignore errors reading tag version
      }
    }
  } catch {
    // No tags yet, skip version check
  }

  // Check 2: CHANGELOG updated since last tag
  try {
    const lastTag = execSync("git describe --tags --abbrev=0 2>/dev/null", {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "ignore"],
    }).trim();

    if (lastTag && existsSync("CHANGELOG.md")) {
      try {
        const changedFiles = execSync(
          `git diff ${lastTag} --name-only -- CHANGELOG.md 2>/dev/null`,
          { encoding: "utf-8", stdio: ["pipe", "pipe", "ignore"] }
        ).trim();

        if (!changedFiles) {
          errors.push(`CHANGELOG.md not updated since ${lastTag}`);
        }
      } catch {
        // ignore
      }
    }
  } catch {
    // No tags yet, skip changelog check
  }

  // Check 3: Build succeeds
  if (hasBuildScript) {
    try {
      execSync("npm run build --silent 2>/dev/null", {
        stdio: "ignore",
        timeout: 60000,
      });
    } catch {
      errors.push("Build failed (npm run build)");
    }
  } else if (existsSync("Makefile")) {
    try {
      const makefile = readFileSync("Makefile", "utf-8");
      if (makefile.includes("build:") || makefile.match(/^build:/m)) {
        try {
          execSync("make build 2>/dev/null", {
            stdio: "ignore",
            timeout: 60000,
          });
        } catch {
          errors.push("Build failed (make build)");
        }
      }
    } catch {
      // ignore
    }
  } else if (existsSync("Cargo.toml")) {
    try {
      execSync("cargo build 2>/dev/null", {
        stdio: "ignore",
        timeout: 120000,
      });
    } catch {
      errors.push("Build failed (cargo build)");
    }
  } else if (existsSync("go.mod")) {
    try {
      execSync("go build ./... 2>/dev/null", {
        stdio: "ignore",
        timeout: 60000,
      });
    } catch {
      errors.push("Build failed (go build ./...)");
    }
  }

  if (errors.length > 0) {
    result.passed = false;
    result.blocked = true;
    result.errors = errors;
  }

  return result;
}

// ─────────────────────────────────────────────────────────────────────────────
// Check: workflow-pre-pr
// Enforce PR title format and CHANGELOG entry
// ─────────────────────────────────────────────────────────────────────────────

async function workflowPrePr(context: HookContext): Promise<CheckResult> {
  const result: CheckResult = {
    passed: true,
    blocked: false,
    warnings: [],
    errors: [],
  };

  const toolName = getToolName(context);
  const command = getCommand(context);

  // Only match gh pr create
  if (toolName !== "Bash" || !command.includes("gh pr create")) {
    return result;
  }

  // Check if CHANGELOG.md has actual entries under [Unreleased]
  if (existsSync("CHANGELOG.md")) {
    try {
      const changelog = readFileSync("CHANGELOG.md", "utf-8");
      
      // Find the Unreleased section
      const unreleasedMatch = changelog.match(/##\s*\[Unreleased\]([\s\S]*?)(?=##\s*\[|$)/);
      
      if (unreleasedMatch) {
        const unreleasedSection = unreleasedMatch[1];
        // Check for list items (lines starting with - or *)
        const hasEntries = /^[-*]\s/m.test(unreleasedSection);
        
        if (!hasEntries) {
          result.passed = false;
          result.blocked = true;
          result.errors.push("CHANGELOG.md [Unreleased] section is empty (no entries found)");
          result.errors.push("Add changelog entries before creating a PR");
        }
      } else {
        result.passed = false;
        result.blocked = true;
        result.errors.push("CHANGELOG.md missing [Unreleased] section");
      }
    } catch {
      result.passed = false;
      result.blocked = true;
      result.errors.push("Could not read CHANGELOG.md");
    }
  } else {
    result.passed = false;
    result.blocked = true;
    result.errors.push("CHANGELOG.md not found");
  }

  return result;
}

// ─────────────────────────────────────────────────────────────────────────────
// Check: validate-commit
// Validate Conventional Commits conventions
// ─────────────────────────────────────────────────────────────────────────────

async function validateCommit(context: HookContext): Promise<CheckResult> {
  const result: CheckResult = {
    passed: true,
    blocked: false,
    warnings: [],
    errors: [],
  };

  const toolName = getToolName(context);
  const command = getCommand(context);

  // Only check Bash commands that are git commits
  if (toolName !== "Bash" || !command.includes("git commit")) {
    return result;
  }

  // Skip if --amend without -m (uses existing message)
  if (command.includes("--amend") && !command.includes("-m")) {
    return result;
  }

  // Skip merge commits
  if (command.includes("--no-edit") || command.includes("git merge")) {
    return result;
  }

  // Extract commit message from command
  let message: string | null = null;

  // Match -m "message" or -m 'message'
  const doubleQuoteMatch = command.match(/-m\s+"([^"]+)"/);
  const singleQuoteMatch = command.match(/-m\s+'([^']+)'/);
  
  if (doubleQuoteMatch) {
    message = doubleQuoteMatch[1];
  } else if (singleQuoteMatch) {
    message = singleQuoteMatch[1];
  } else {
    // Match HEREDOC: -m "$(cat <<'EOF' ... EOF )"
    const heredocMatch = command.match(/<<'?EOF'?\s*\n(.+?)\n\s*EOF/s);
    if (heredocMatch) {
      message = heredocMatch[1].trim();
    }
  }

  if (!message) {
    // Can't extract message (might be interactive), allow
    return result;
  }

  // Validate Conventional Commits format
  // Format: <type>[optional scope]: <description>
  const conventionalCommitRegex = /^(feat|fix|docs|style|refactor|perf|test|chore|ci|build|revert)(\(.+\))?!?: .+/;
  
  if (!conventionalCommitRegex.test(message)) {
    result.passed = false;
    result.blocked = true;
    result.errors.push("Commit message does not follow Conventional Commits format");
    result.errors.push("Expected format: <type>[optional scope]: <description>");
    result.errors.push("Valid types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert");
    result.errors.push(`Your message: "${message}"`);
  }

  // Check message length (should be under 72 chars for the subject)
  const subjectLine = message.split("\n")[0];
  if (subjectLine.length > 72) {
    result.warnings.push(`Subject line is ${subjectLine.length} characters (recommended: ≤72)`);
  }

  return result;
}

// ─────────────────────────────────────────────────────────────────────────────
// Check: security-audit
// Detect dangerous Bash command patterns
// ─────────────────────────────────────────────────────────────────────────────

async function securityAudit(context: HookContext): Promise<CheckResult> {
  const result: CheckResult = {
    passed: true,
    blocked: false,
    warnings: [],
    errors: [],
    findings: [],
  };

  const toolName = getToolName(context);
  const command = getCommand(context);

  // Only run on Bash tool
  if (toolName !== "Bash") {
    return result;
  }

  // Dangerous patterns to detect
  const dangerousPatterns: Array<{ name: string; regex: RegExp; critical: boolean }> = [
    { 
      name: "Dangerous rm -rf", 
      regex: /rm\s+-rf?\s+\/|rm\s+-rf?\s+\*+/, 
      critical: true 
    },
    { 
      name: "chmod 777 (world-writable)", 
      regex: /chmod\s+.*777/, 
      critical: true 
    },
    { 
      name: "eval of untrusted input", 
      regex: /eval\s*\$|eval\s*\`/, 
      critical: true 
    },
    { 
      name: "Unsafe curl to bash", 
      regex: /curl.*\|\s*(ba)?sh/, 
      critical: true 
    },
    { 
      name: "wget to shell", 
      regex: /wget.*\|\s*(ba)?sh/, 
      critical: true 
    },
    { 
      name: "sudo without validation", 
      regex: /sudo\s+(rm|dd|mkfs|fdisk|format)/, 
      critical: false 
    },
    { 
      name: "Hardcoded sudo password", 
      regex: /echo\s+['"].*['"]\s*\|\s*sudo/, 
      critical: true 
    },
    { 
      name: "Unsafe find exec", 
      regex: /find\s+.*-exec\s+(rm|mv|cp)\s*\{\}/, 
      critical: false 
    },
    { 
      name: "SQL injection risk in command", 
      regex: /mysql.*-e\s*.*['"]\s*\$/, 
      critical: false 
    },
  ];

  const targetCommand = command || "";
  const criticalFindings: string[] = [];
  const warningFindings: string[] = [];

  for (const { name, regex, critical } of dangerousPatterns) {
    if (regex.test(targetCommand)) {
      const finding = `${name}: matched pattern in command`;
      if (critical) {
        criticalFindings.push(finding);
      } else {
        warningFindings.push(finding);
      }
    }
  }

  if (criticalFindings.length > 0) {
    result.passed = false;
    result.blocked = true;
    result.errors.push("Critical security issues detected in command");
    result.findings = [...criticalFindings, ...warningFindings];
  } else if (warningFindings.length > 0) {
    result.warnings.push(...warningFindings);
  }

  return result;
}

// ─────────────────────────────────────────────────────────────────────────────
// Command Registration
// ─────────────────────────────────────────────────────────────────────────────

export function registerCheckCommand(program: Command): void {
  program
    .command("check")
    .description("Run enforcement hook checks")
    .option("--hook <id>", "Hook ID to run (check-secrets, validate-push, workflow-pre-pr, validate-commit, security-audit)")
    .option("--json", "Output JSON format")
    .action(async (options: { hook?: string; json?: boolean }) => {
      // Validate hook ID
      const validHooks = [
        "check-secrets",
        "validate-push", 
        "workflow-pre-pr",
        "validate-commit",
        "security-audit",
      ];

      if (!options.hook) {
        console.error(`${red("error:")} --hook <id> is required`);
        console.error(`${gray("Valid hooks:")} ${validHooks.join(", ")}`);
        process.exit(1);
      }

      if (!validHooks.includes(options.hook)) {
        console.error(`${red("error:")} Unknown hook: ${bold(options.hook)}`);
        console.error(`${gray("Valid hooks:")} ${validHooks.join(", ")}`);
        process.exit(1);
      }

      // Read stdin
      let context: HookContext;
      try {
        const stdinData = await readStdin();
        context = parseContext(stdinData);
      } catch (err) {
        console.error(`${red("error:")} Failed to read stdin: ${err}`);
        process.exit(1);
      }

      // Run appropriate check
      let result: CheckResult;
      try {
        switch (options.hook) {
          case "check-secrets":
            result = await checkSecrets(context);
            break;
          case "validate-push":
            result = await validatePush(context);
            break;
          case "workflow-pre-pr":
            result = await workflowPrePr(context);
            break;
          case "validate-commit":
            result = await validateCommit(context);
            break;
          case "security-audit":
            result = await securityAudit(context);
            break;
          default:
            // This should never happen due to validation above
            console.error(`${red("error:")} Unknown hook: ${options.hook}`);
            process.exit(1);
        }
      } catch (err) {
        console.error(`${red("error:")} Check failed with exception: ${err}`);
        process.exit(1);
      }

      // Output results
      if (options.json) {
        outputJson(result, options.hook);
      } else {
        outputText(result, options.hook);
      }

      // Exit with appropriate code
      // 0 = pass (including warnings), 2 = block, 1 = internal error only
      if (result.blocked) {
        process.exit(2);
      } else {
        process.exit(0);
      }
    });
}
