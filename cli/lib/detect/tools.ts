/**
 * Tool Detection Module
 *
 * Detects installed AI coding tools and their config directories.
 * Ported from install.sh.
 */

import { existsSync } from "fs";
import { join } from "path";
import { execFileSync } from "child_process";
import { platform } from "os";

const HOME = process.env.HOME || process.env.USERPROFILE || "";
const XDG_CONFIG_HOME = process.env.XDG_CONFIG_HOME || join(HOME, ".config");

export interface DetectedTool {
  key: string;
  name: string;
  configDir: string;
  installed: boolean;
  detectedVia: string;
}

const LOAF_MARKER_FILE = ".loaf-version";

function hasCmd(cmd: string): boolean {
  try {
    execFileSync("which", [cmd], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function isMacOS(): boolean {
  return platform() === "darwin";
}

function isLoafInstalled(configDir: string): boolean {
  if (existsSync(join(configDir, LOAF_MARKER_FILE))) {
    return true;
  }

  const skillsDir = join(configDir, "skills");
  for (const skill of ["foundations", "python-development", "python"]) {
    if (existsSync(join(skillsDir, skill))) {
      return true;
    }
  }

  return false;
}

export function detectClaudeCode(): boolean {
  return hasCmd("claude");
}

export function detectTools(): DetectedTool[] {
  const tools: DetectedTool[] = [];

  // OpenCode (uses XDG)
  const opencodeConfig = join(XDG_CONFIG_HOME, "opencode");
  if (existsSync(opencodeConfig)) {
    tools.push({
      key: "opencode",
      name: "OpenCode",
      configDir: opencodeConfig,
      installed: isLoafInstalled(opencodeConfig),
      detectedVia: "config",
    });
  }

  // Cursor
  const cursorConfig = join(HOME, ".cursor");
  let cursorDetected = false;
  let cursorVia = "";

  if (hasCmd("cursor")) {
    cursorDetected = true;
    cursorVia = "cli";
  } else if (
    isMacOS() &&
    (existsSync("/Applications/Cursor.app") ||
      existsSync(join(HOME, "Applications/Cursor.app")))
  ) {
    cursorDetected = true;
    cursorVia = "app";
  } else if (existsSync(cursorConfig)) {
    cursorDetected = true;
    cursorVia = "config";
  }

  if (cursorDetected) {
    tools.push({
      key: "cursor",
      name: "Cursor",
      configDir: cursorConfig,
      installed: isLoafInstalled(cursorConfig),
      detectedVia: cursorVia,
    });
  }

  // Codex
  const codexConfig = (process.env.CODEX_HOME || join(HOME, ".codex")).replace(
    /\/$/,
    "",
  );
  let codexDetected = false;
  let codexVia = "";

  if (hasCmd("codex")) {
    codexDetected = true;
    codexVia = "cli";
  } else if (existsSync(codexConfig) || existsSync(join(HOME, ".codex"))) {
    codexDetected = true;
    codexVia = "config";
  }

  if (codexDetected) {
    tools.push({
      key: "codex",
      name: "Codex",
      configDir: codexConfig,
      installed: isLoafInstalled(codexConfig),
      detectedVia: codexVia,
    });
  }

  // Gemini
  const geminiConfig = join(HOME, ".gemini");
  let geminiDetected = false;
  let geminiVia = "";

  if (hasCmd("gemini")) {
    geminiDetected = true;
    geminiVia = "cli";
  } else if (existsSync(geminiConfig)) {
    geminiDetected = true;
    geminiVia = "config";
  }

  if (geminiDetected) {
    tools.push({
      key: "gemini",
      name: "Gemini",
      configDir: geminiConfig,
      installed: isLoafInstalled(geminiConfig),
      detectedVia: geminiVia,
    });
  }

  return tools;
}

/** Default config directories for targets (used when --to overrides auto-detection) */
export const DEFAULT_CONFIG_DIRS: Record<string, string> = {
  opencode: join(XDG_CONFIG_HOME, "opencode"),
  cursor: join(HOME, ".cursor"),
  codex: process.env.CODEX_HOME || join(HOME, ".codex"),
  gemini: join(HOME, ".gemini"),
};

/**
 * Detect if running from a local dev repo (not from npm global).
 */
export function isDevMode(rootDir: string): boolean {
  return (
    existsSync(join(rootDir, ".git")) &&
    existsSync(join(rootDir, "package.json")) &&
    existsSync(join(rootDir, "content", "skills"))
  );
}
