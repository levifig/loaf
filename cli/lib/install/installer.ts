/**
 * Target Installation Logic
 *
 * Handles copying built output to tool config directories.
 * Ported from install.sh.
 */

import {
  mkdirSync,
  cpSync,
  writeFileSync,
  rmSync,
  existsSync,
  readdirSync,
} from "fs";
import { join, dirname } from "path";
import { readFileSync } from "fs";
import { execFileSync } from "child_process";
import { fileURLToPath } from "url";

const LOAF_MARKER_FILE = ".loaf-version";

function getVersion(): string {
  const __dirname = dirname(fileURLToPath(import.meta.url));
  for (const candidate of [
    join(__dirname, "..", "package.json"),
    join(__dirname, "..", "..", "package.json"),
    join(__dirname, "..", "..", "..", "package.json"),
  ]) {
    try {
      const pkg = JSON.parse(readFileSync(candidate, "utf-8"));
      if (pkg.name === "loaf") return pkg.version;
    } catch {
      continue;
    }
  }
  return "0.0.0";
}

function hasRsync(): boolean {
  try {
    execFileSync("which", ["rsync"], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function syncDir(src: string, dest: string): void {
  mkdirSync(dest, { recursive: true });

  if (hasRsync()) {
    execFileSync("rsync", ["-a", "--delete", `${src}/`, `${dest}/`], {
      stdio: "inherit",
    });
  } else {
    // Fallback: remove and copy
    const entries = readdirSync(dest);
    for (const entry of entries) {
      rmSync(join(dest, entry), { recursive: true, force: true });
    }
    cpSync(src, dest, { recursive: true });
  }
}

function writeMarker(configDir: string): void {
  mkdirSync(configDir, { recursive: true });
  writeFileSync(join(configDir, LOAF_MARKER_FILE), `${getVersion()}\n`);
}

export function installOpencode(distDir: string, configDir: string): void {
  const dirs = ["skills", "agents", "commands", "plugins"];

  for (const dir of dirs) {
    const src = join(distDir, dir);
    const dest = join(configDir, dir);
    if (existsSync(src)) {
      syncDir(src, dest);
    }
  }

  writeMarker(configDir);
}

export function installCursor(distDir: string, configDir: string): void {
  // Remove stale commands from previous installs
  const staleCommands = join(configDir, "commands");
  if (existsSync(staleCommands)) {
    rmSync(staleCommands, { recursive: true });
  }

  // Skills
  const skillsSrc = join(distDir, "skills");
  if (existsSync(skillsSrc)) {
    syncDir(skillsSrc, join(configDir, "skills"));
  }

  // Agents
  const agentsSrc = join(distDir, "agents");
  if (existsSync(agentsSrc)) {
    syncDir(agentsSrc, join(configDir, "agents"));
  }

  // hooks.json
  const hooksSrc = join(distDir, "hooks.json");
  if (existsSync(hooksSrc)) {
    mkdirSync(configDir, { recursive: true });
    cpSync(hooksSrc, join(configDir, "hooks.json"));
  }

  // Hook scripts
  const hooksDir = join(distDir, "hooks");
  if (existsSync(hooksDir)) {
    syncDir(hooksDir, join(configDir, "hooks"));
  }

  writeMarker(configDir);
}

export function installCodex(distDir: string, configDir: string): void {
  const skillsSrc = join(distDir, "skills");
  if (existsSync(skillsSrc)) {
    syncDir(skillsSrc, join(configDir, "skills"));
  }

  writeMarker(configDir);
}

export function installGemini(distDir: string, configDir: string): void {
  const skillsSrc = join(distDir, "skills");
  if (existsSync(skillsSrc)) {
    syncDir(skillsSrc, join(configDir, "skills"));
  }

  writeMarker(configDir);
}

export const INSTALLERS: Record<
  string,
  (distDir: string, configDir: string) => void
> = {
  opencode: installOpencode,
  cursor: installCursor,
  codex: installCodex,
  gemini: installGemini,
};
