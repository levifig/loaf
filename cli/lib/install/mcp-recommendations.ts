/**
 * Post-install MCP recommendation flow (TASK-088).
 */

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { createInterface } from "readline";
import { execFileSync } from "child_process";
import { join } from "path";

import { mergeAgentsConfigIntegrations } from "../config/agents-config.js";
import {
  buildMcpStatuses,
  getMcpDefinition,
  linearCursorMcpEntry,
  linearMcpRemoteArgv,
  type McpDefinition,
  type McpStatus,
} from "../detect/mcp.js";

const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;

export interface McpRecommendationOptions {
  projectRoot: string;
  upgrade: boolean;
  hasClaudeCode: boolean;
  /** True only when Cursor was selected for this install (not merely detected on disk). */
  cursorTargetThisRun: boolean;
  hasAnyDetectedTool: boolean;
  installedTargets?: string[];
}

function hasClaudeCli(): boolean {
  try {
    execFileSync("which", ["claude"], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function askGpn(question: string): Promise<"global" | "project" | "no"> {
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      const a = answer.trim().toLowerCase();
      if (a.startsWith("g")) resolve("global");
      else if (a.startsWith("p")) resolve("project");
      else resolve("no");
    });
  });
}

function claudeScopeFlag(scope: "global" | "project"): string {
  return scope === "global" ? "user" : "project";
}

function installViaClaude(
  def: McpDefinition,
  scope: "global" | "project",
): void {
  const scopeFlag = claudeScopeFlag(scope);
  const argv =
    def.id === "linear" ? linearMcpRemoteArgv() : def.claudeArgs;
  execFileSync(
    "claude",
    ["mcp", "add", "--scope", scopeFlag, def.id, "--", ...argv],
    { stdio: "inherit" },
  );
}

const white = (s: string) => `\x1b[97m${s}\x1b[0m`;

function mergeCursorMcpServer(
  projectRoot: string,
  serverKey: string,
  spec: Record<string, unknown>,
  scope: "global" | "project",
): void {
  const home = process.env.HOME || process.env.USERPROFILE || "";
  const dir =
    scope === "global" ? join(home, ".cursor") : join(projectRoot, ".cursor");
  const p = join(dir, "mcp.json");
  let data: Record<string, unknown> = {};
  if (existsSync(p)) {
    try {
      data = JSON.parse(readFileSync(p, "utf-8")) as Record<string, unknown>;
    } catch {
      data = {};
    }
  }
  if (!data.mcpServers || typeof data.mcpServers !== "object") {
    data.mcpServers = {};
  }
  (data.mcpServers as Record<string, unknown>)[serverKey] = spec;
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  writeFileSync(p, `${JSON.stringify(data, null, 2)}\n`, "utf-8");
}

function installLinearCursor(
  projectRoot: string,
  scope: "global" | "project",
): void {
  const { command, args } = linearCursorMcpEntry();
  mergeCursorMcpServer(projectRoot, "linear", { command, args }, scope);
}

function installSerenaCursor(
  projectRoot: string,
  def: McpDefinition,
  scope: "global" | "project",
): void {
  mergeCursorMcpServer(
    projectRoot,
    "serena",
    {
      command: def.claudeArgs[0],
      args: def.claudeArgs.slice(1),
    },
    scope,
  );
}

/**
 * Whether this MCP is fully satisfied for the current install run (exported for tests).
 * Cursor is only required when `cursorTargetThisRun` is true.
 *
 * When neither Claude nor Cursor is part of this run (Codex/OpenCode/Gemini/Amp only),
 * we do **not** treat on-disk Claude/Cursor MCP config as "done" — those clients are
 * unrelated to the tool the user is installing for, so we must not set
 * `integrations.*.enabled` from them or skip manual setup instructions.
 */
export function mcpIntegrationDoneForSession(
  st: McpStatus,
  canClaude: boolean,
  cursorTargetThisRun: boolean,
): boolean {
  const hasAutoPath = canClaude || cursorTargetThisRun;
  if (!hasAutoPath) {
    return false;
  }
  return (
    (!canClaude || st.claude.configured) &&
    (!cursorTargetThisRun || st.cursor.configured)
  );
}

function formatStackLine(
  label: string,
  st: { configured: boolean; scope: string | null },
): string {
  if (st.configured) {
    const where =
      st.scope === "global"
        ? "global"
        : st.scope === "project"
          ? "project"
          : "";
    return `    ${green("✓")} ${label}${where ? ` (${where})` : ""}`;
  }
  return `    ${yellow("⚡")} ${label}: not configured`;
}

/**
 * Run after target installation. Skipped when --upgrade or non-interactive stdin.
 */
export async function runMcpRecommendations(
  opts: McpRecommendationOptions,
): Promise<void> {
  if (opts.upgrade) return;
  if (!process.stdin.isTTY) return;

  const { projectRoot, hasClaudeCode, cursorTargetThisRun, hasAnyDetectedTool } =
    opts;
  const installedTargets = opts.installedTargets ?? [];
  if (!hasAnyDetectedTool && !hasClaudeCode) {
    return;
  }

  const canClaude = hasClaudeCode && hasClaudeCli();
  const hasAutoPath = canClaude || cursorTargetThisRun;

  console.log(`\n${bold("Recommended MCP servers")}\n`);

  const statuses = buildMcpStatuses(projectRoot);

  for (const st of statuses) {
    const def = getMcpDefinition(st.id);
    if (!def) continue;

    const tierLabel =
      def.tier === "optional" ? `${gray("Optional — ")}` : "";
    console.log(`  ${tierLabel}${bold(def.displayName)}`);

    if (canClaude) {
      console.log(formatStackLine("Claude Code", st.claude));
    }
    if (cursorTargetThisRun) {
      console.log(formatStackLine("Cursor", st.cursor));
    }

    const doneForSession = mcpIntegrationDoneForSession(
      st,
      canClaude,
      cursorTargetThisRun,
    );

    if (doneForSession) {
      mergeAgentsConfigIntegrations(projectRoot, {
        [def.id]: { enabled: true },
      });
      console.log();
      continue;
    }

    const needsClaudeInstall = canClaude && !st.claude.configured;
    const needsCursorInstall = cursorTargetThisRun && !st.cursor.configured;

    const choice = await askGpn(
      `    Install missing MCP(s)? [g]lobal / [p]roject / [n]o: `,
    );

    if (choice === "no") {
      mergeAgentsConfigIntegrations(projectRoot, {
        [def.id]: { enabled: false },
      });
      console.log(`    ${gray("○")} Skipped — recorded in .agents/config.json`);
      console.log();
      continue;
    }

    let claudeSuccess = false;
    let cursorSuccess = false;

    if (needsClaudeInstall && canClaude) {
      try {
        installViaClaude(def, choice);
        claudeSuccess = true;
        console.log(
          `    ${green("✓")} ${def.displayName} added for Claude Code (${claudeScopeFlag(choice)} scope)`,
        );
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        console.log(`    ${yellow("⚠")} Claude CLI failed: ${msg}`);
        console.log(`    ${gray("Manual:")} ${def.manualHint}`);
      }
    }

    if (needsCursorInstall) {
      try {
        if (def.id === "linear") {
          installLinearCursor(projectRoot, choice);
        } else {
          installSerenaCursor(projectRoot, def, choice);
        }
        cursorSuccess = true;
        const loc =
          choice === "global" ? "~/.cursor/mcp.json" : ".cursor/mcp.json";
        console.log(
          `    ${green("✓")} ${def.displayName} merged into ${gray(loc)}`,
        );
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        console.log(`    ${yellow("⚠")} Cursor mcp.json merge failed: ${msg}`);
      }
    }

    if (!hasAutoPath) {
      console.log(`    ${gray("Manual:")} ${white(def.manualHint)}`);
      mergeAgentsConfigIntegrations(projectRoot, {
        [def.id]: { enabled: false },
      });
    } else {
      const stacksOk =
        (!canClaude || st.claude.configured || claudeSuccess) &&
        (!cursorTargetThisRun || st.cursor.configured || cursorSuccess);
      mergeAgentsConfigIntegrations(projectRoot, {
        [def.id]: { enabled: stacksOk },
      });
      if (!stacksOk) {
        console.log(
          `    ${yellow("⚠")} ${gray(`integrations.${def.id}.enabled remains false until every required stack succeeds.`)}`,
        );
      }
    }

    const otherHint = installedTargets.filter((t) =>
      ["opencode", "codex", "gemini", "amp"].includes(t),
    );
    if (otherHint.length > 0) {
      console.log(
        `    ${gray(`Also configure this MCP in ${otherHint.join(", ")} via each tool's MCP settings.`)}`,
      );
      console.log(`    ${gray("Reference:")} ${white(def.manualHint)}`);
    }

    console.log();
  }

  console.log(
    `${green("✓")} Integration choices recorded under ${gray("integrations")} in ${gray(".agents/config.json")}`,
  );
  console.log();
}

/** Exported for tests: install one MCP via Cursor file merge. */
export function installLinearCursorForTest(projectRoot: string): void {
  installLinearCursor(projectRoot, "project");
}
