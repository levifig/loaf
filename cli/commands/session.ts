/**
 * loaf session command
 *
 * Session journal management for tracking work state and continuity.
 */

import { Command } from "commander";
import { execSync } from "child_process";
import {
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  writeFileSync,
  renameSync,
  unlinkSync,
  openSync,
  closeSync,
} from "fs";
import { join, dirname, basename } from "path";
import matter from "gray-matter";

import { findAgentsDir } from "../lib/tasks/resolve.js";
import { readStdin } from "./check.js";

// ANSI color helpers
const bold = (s: string) => `\x1b[1m${s}\x1b[0m`;
const green = (s: string) => `\x1b[32m${s}\x1b[0m`;
const red = (s: string) => `\x1b[31m${s}\x1b[0m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[0m`;
const cyan = (s: string) => `\x1b[36m${s}\x1b[0m`;
const gray = (s: string) => `\x1b[90m${s}\x1b[0m`;

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

interface SessionFrontmatter {
  branch: string;
  status: "active" | "paused" | "blocked" | "complete" | "archived";
  created: string;
  last_updated?: string;
  last_entry?: string;
  archived_at?: string;
  archived_by?: string;
  linear_issue?: string;
  linear_url?: string;
  task?: string;
  spec?: string;
  title?: string;
}

type EntryType =
  | "resume"
  | "pause"
  | "progress"
  | "commit"
  | "pr"
  | "merge"
  | "decide"
  | "discover"
  | "conclude"
  | "block"
  | "unblock"
  | "spark"
  | "todo"
  | "assume"
  // New types from SPEC-020
  | "branch"
  | "task"
  | "linear"
  | "hypothesis"
  | "try"
  | "reject"
  | "compact";

interface JournalEntry {
  type: EntryType;
  scope?: string;
  description: string;
}

interface SpecFrontmatterWithBranch {
  id: string;
  title: string;
  branch?: string;
  status?: string;
  [key: string]: unknown;
}

// ─────────────────────────────────────────────────────────────────────────────
// File Locking for Atomic Operations
// ─────────────────────────────────────────────────────────────────────────────

/** Staleness threshold in milliseconds (30 seconds) */
const LOCK_STALENESS_THRESHOLD = 30000;

/** Lock file content format: JSON with PID and timestamp */
interface LockFileContent {
  pid: number;
  timestamp: number;
}

/** Check if lock file is stale (older than threshold) */
function isLockStale(lockPath: string): boolean {
  try {
    const content = readFileSync(lockPath, 'utf-8');
    const data: LockFileContent = JSON.parse(content);
    const age = Date.now() - data.timestamp;
    return age > LOCK_STALENESS_THRESHOLD;
  } catch {
    // If we can't read/parse the lock file, assume it's stale
    return true;
  }
}

/** Force-release a stale lock */
function forceReleaseLock(lockPath: string): void {
  try {
    unlinkSync(lockPath);
    console.log(`  Released stale lock: ${lockPath}`);
  } catch {
    // Ignore errors
  }
}

/** Simple file lock using lock file with retry logic and staleness detection */
async function acquireLock(lockPath: string, maxRetries = 50, retryDelay = 100): Promise<void> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      // Check if existing lock is stale
      if (existsSync(lockPath) && isLockStale(lockPath)) {
        forceReleaseLock(lockPath);
      }
      
      // Try to create lock file atomically with O_EXCL
      const fd = openSync(lockPath, 'wx');
      // Write PID and timestamp to lock file
      const lockContent: LockFileContent = {
        pid: process.pid,
        timestamp: Date.now()
      };
      writeFileSync(fd, JSON.stringify(lockContent), 'utf-8');
      closeSync(fd);
      return; // Lock acquired
    } catch {
      // Lock file exists (and is not stale), wait and retry
      await new Promise(resolve => setTimeout(resolve, retryDelay));
    }
  }
  throw new Error(`Could not acquire lock after ${maxRetries} retries: ${lockPath}`);
}

function releaseLock(lockPath: string): void {
  try {
    unlinkSync(lockPath);
  } catch {
    // Ignore errors on unlock
  }
}

/** Atomic write using temp file + rename */
function writeFileAtomic(filePath: string, content: string): void {
  const tempPath = `${filePath}.tmp.${Date.now()}.${Math.random().toString(36).slice(2, 11)}`;
  try {
    writeFileSync(tempPath, content, 'utf-8');
    renameSync(tempPath, filePath);
  } catch (err) {
    // Cleanup temp file on failure
    try { unlinkSync(tempPath); } catch { /* ignore */ }
    throw err;
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

/** Get current git branch, or commit SHA if detached HEAD */
function getCurrentBranch(): string {
  try {
    const branch = execSync("git branch --show-current", {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
    
    // Empty branch indicates detached HEAD
    if (!branch) {
      // Get short commit SHA for detached HEAD
      const commitSha = execSync("git rev-parse --short HEAD", {
        encoding: "utf-8",
        stdio: ["pipe", "pipe", "pipe"],
      }).trim();
      return `detached-${commitSha}`;
    }
    
    return branch;
  } catch {
    return "unknown";
  }
}

/** Get recent commits with structured data */
interface Commit {
  hash: string;
  message: string;
}

function getRecentCommits(count = 5): Commit[] {
  try {
    const output = execSync(`git log --pretty=format:"%H %s" -${count}`, {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();

    if (!output) return [];

    return output.split("\n").map((line) => {
      const spaceIndex = line.indexOf(" ");
      return {
        hash: line.substring(0, 7), // Short hash
        message: line.substring(spaceIndex + 1),
      };
    });
  } catch {
    return [];
  }
}

/** Get last commit SHA */
function getLastCommitSha(): string {
  try {
    return execSync("git rev-parse --short HEAD", {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
  } catch {
    return "unknown";
  }
}

/** Find spec ID linked to a branch */
function findSpecForBranch(
  agentsDir: string,
  branch: string
): { id: string; title: string } | null {
  const specsDir = join(agentsDir, "specs");
  if (!existsSync(specsDir)) return null;

  const files = readdirSync(specsDir).filter((f) => f.endsWith(".md"));

  for (const file of files) {
    try {
      const content = readFileSync(join(specsDir, file), "utf-8");
      const parsed = matter(content);
      const fm = parsed.data as SpecFrontmatterWithBranch;

      if (fm.branch === branch) {
        return { id: fm.id || file.replace(/\.md$/, ""), title: fm.title || "Untitled" };
      }
    } catch {
      continue;
    }
  }

  return null;
}

/** Get session file path for a branch (deterministic based on branch name) */
function getSessionFilePath(agentsDir: string, branch: string): string {
  // Use deterministic path based on branch name only
  // This prevents duplicate session files when concurrent processes race
  const slug = branch.replace(/^feat\//, "").replace(/[^a-zA-Z0-9-_]/g, "-").replace(/-+/g, "-");
  return join(agentsDir, "sessions", `${slug}.md`);
}

/** Find active session for a branch by scanning files */
function findActiveSessionForBranch(agentsDir: string, branch: string): { filePath: string; data: SessionFrontmatter; content: string } | null {
  const sessionsDir = join(agentsDir, "sessions");
  if (!existsSync(sessionsDir)) return null;
  
  const files = readdirSync(sessionsDir).filter(f => f.endsWith(".md") && !f.startsWith("archive"));
  
  // Collect all candidate sessions for this branch
  const candidates: Array<{ filePath: string; data: SessionFrontmatter; content: string }> = [];
  
  for (const file of files) {
    const filePath = join(sessionsDir, file);
    const session = readSessionFile(filePath);
    if (session && session.data.branch === branch && session.data.status !== "archived") {
      candidates.push({ filePath, data: session.data, content: session.content });
    }
  }
  
  if (candidates.length === 0) return null;
  
  // Prioritize: active > paused/blocked/complete > others
  // Sort by status priority (lower number = higher priority)
  const statusPriority: Record<string, number> = {
    active: 1,
    paused: 2,
    blocked: 2,
    complete: 2,
  };
  
  candidates.sort((a, b) => {
    // First: sort by status priority
    const priorityA = statusPriority[a.data.status] ?? 3;
    const priorityB = statusPriority[b.data.status] ?? 3;
    if (priorityA !== priorityB) {
      return priorityA - priorityB;
    }
    
    // Second: tie-break by recency (newer wins)
    // Use last_updated from frontmatter, or fall back to last_entry, or 0
    const timeA = a.data.last_updated || a.data.last_entry || "0";
    const timeB = b.data.last_updated || b.data.last_entry || "0";
    return timeB.localeCompare(timeA); // descending (newer first)
  });
  
  return candidates[0];
}

/** Atomically get existing session or create new one (prevents concurrent creation) */
async function getOrCreateSession(
  agentsDir: string,
  branch: string,
  specInfo: { id: string; title: string } | null
): Promise<{ filePath: string; data: SessionFrontmatter; content: string; isNew: boolean }> {
  const sessionsDir = join(agentsDir, "sessions");
  // Use a branch-level lock for session creation
  const lockPath = join(sessionsDir, `.create-${branch.replace(/[^a-zA-Z0-9-]/g, '-')}.lock`);
  let lockAcquired = false;
  
  try {
    // Acquire creation lock
    await acquireLock(lockPath, 100, 50);
    lockAcquired = true;
    
    // Re-check for existing session under lock with small delay for filesystem sync
    // This handles the race condition where another process just created the file
    let existing = findActiveSessionForBranch(agentsDir, branch);
    
    // Also check deterministic path directly for faster detection
    const deterministicPath = getSessionFilePath(agentsDir, branch);
    if (!existing && existsSync(deterministicPath)) {
      // Delay to ensure file is fully written and visible
      await new Promise(resolve => setTimeout(resolve, 50));
      const directSession = readSessionFile(deterministicPath);
      if (directSession) {
        existing = { ...directSession, filePath: deterministicPath };
      }
    }
    
    if (existing) {
      return { ...existing, isNew: false };
    }
    
    // Create new session file (or detect if another process created it)
    const filePath = getSessionFilePath(agentsDir, branch);
    const created = createSessionFile(filePath, branch, specInfo);
    
    if (!created) {
      // Another process created the file while we were waiting for the lock
      // Read the existing session
      const existingSession = readSessionFile(filePath);
      if (existingSession) {
        return { filePath, data: existingSession.data, content: existingSession.content, isNew: false };
      }
      // File was created but we couldn't read it - try to create anyway
      const fallbackSession = readSessionFile(filePath);
      if (fallbackSession) {
        return { filePath, data: fallbackSession.data, content: fallbackSession.content, isNew: false };
      }
      // This shouldn't happen, but fall back to treating as new
    }
    
    const session = readSessionFile(filePath)!;
    
    return { filePath, data: session.data, content: session.content, isNew: true };
  } finally {
    if (lockAcquired) {
      releaseLock(lockPath);
    }
  }
}

/** Read session file or return null */
function readSessionFile(filePath: string): { data: SessionFrontmatter; content: string } | null {
  if (!existsSync(filePath)) return null;

  try {
    const raw = readFileSync(filePath, "utf-8");
    const parsed = matter(raw);
    const rawData = parsed.data as unknown as Record<string, unknown>;
    
    // Handle legacy nested frontmatter migration (SPEC-020 format change)
    // Old format: { session: { branch, status, created, ... }, otherFields... }
    // New format: { branch, status, created, ... }
    // Migration strategy: nested fields take precedence over top-level fields on collision
    if (rawData.session && typeof rawData.session === "object") {
      const nested = rawData.session as Record<string, unknown>;
      // Start with all existing top-level fields
      const migratedData: Record<string, unknown> = { ...rawData };
      // Remove the nested session object itself
      delete migratedData.session;
      // Add/replace fields from nested session data
      for (const [key, value] of Object.entries(nested)) {
        migratedData[key] = value;
      }
      return {
        data: migratedData as unknown as SessionFrontmatter,
        content: parsed.content,
      };
    }
    
    return { 
      data: rawData as unknown as SessionFrontmatter, 
      content: parsed.content 
    };
  } catch {
    return null;
  }
}

/** Parse entry string into structured format */
function parseEntry(entry: string): JournalEntry | null {
  // Match type(scope): description or type: description
  const match = entry.match(/^([a-z]+)(?:\(([^)]+)\))?:\s*(.+)$/);
  if (!match) return null;

  const [, type, scope, description] = match;
  const validTypes: EntryType[] = [
    "resume", "pause", "progress", "commit", "pr", "merge",
    "decide", "discover", "conclude", "block", "unblock",
    "spark", "todo", "assume",
    // New types
    "branch", "task", "linear", "hypothesis", "try", "reject", "compact"
  ];

  if (!validTypes.includes(type as EntryType)) return null;

  return {
    type: type as EntryType,
    scope,
    description,
  };
}

/** Format entry for journal */
function formatEntry(entry: JournalEntry): string {
  const scope = entry.scope ? `(${entry.scope})` : "";
  return `- ${entry.type}${scope}: ${entry.description}`;
}

/** Get current timestamp in ISO format */
function getTimestamp(): string {
  return new Date().toISOString();
}

/** Get date-time string for journal header */
function getDateTimeString(): string {
  const now = new Date();
  const date = now.toISOString().split("T")[0];
  const time = now.toTimeString().split(":")[0] + ":" + now.toTimeString().split(":")[1];
  return `${date} ${time}`;
}

/** Create new session file */
function createSessionFile(
  filePath: string,
  branch: string,
  specInfo: { id: string; title: string } | null
): SessionFrontmatter | null {
  // Check if file already exists (from concurrent process)
  if (existsSync(filePath)) {
    return null;
  }
  
  const now = getTimestamp();
  const frontmatter: SessionFrontmatter = {
    branch,
    status: "active",
    created: now,
    title: specInfo ? `${specInfo.id}: ${specInfo.title}` : `Session: ${branch}`,
  };

  // Only add spec field when there's a linked spec
  if (specInfo?.id) {
    frontmatter.spec = specInfo.id;
  }

  const body = specInfo
    ? `# ${specInfo.id}: ${specInfo.title}\n\n## Context\n\nSession for ${specInfo.id}: ${specInfo.title}\n\n## Current State\n\nSession started.\n\n## Next Steps\n\n- [ ] First action item\n`
    : `# Session: ${branch}\n\n## Context\n\nAd-hoc session for branch \`${branch}\`.\n\n## Current State\n\nSession started.\n\n## Next Steps\n\n- [ ] First action item\n`;

  const content = matter.stringify(body, frontmatter as unknown as Record<string, unknown>);
  
  // Use atomic write with 'wx' flag to prevent race conditions
  try {
    const fd = openSync(filePath, 'wx');
    writeFileSync(fd, content, 'utf-8');
    closeSync(fd);
  } catch {
    // File already exists (another process created it)
    return null;
  }

  return frontmatter;
}

/** Append entry to session file (atomic with file locking) */
async function appendEntry(
  filePath: string,
  header: string,
  entries: string[],
  updateFrontmatter: (data: SessionFrontmatter) => void
): Promise<void> {
  const lockPath = `${filePath}.lock`;
  let lockAcquired = false;
  
  try {
    // Acquire exclusive lock
    await acquireLock(lockPath);
    lockAcquired = true;
    
    // Read fresh state under lock
    const session = readSessionFile(filePath);
    if (!session) {
      throw new Error("Session file not found");
    }

    // Update frontmatter
    updateFrontmatter(session.data);

    // Build new content
    const newSection = `\n## ${header}\n${entries.map((e) => e).join("\n")}\n`;
    
    const newContent = matter.stringify(
      session.content.trim() + newSection,
      session.data as unknown as Record<string, unknown>
    );

    // Atomic write
    writeFileAtomic(filePath, newContent);
  } finally {
    // Only release lock if we acquired it
    if (lockAcquired) {
      releaseLock(lockPath);
    }
  }
}

/** Extract recent journal entries for context display */
function extractRecentEntries(content: string, count = 15): string[] {
  const lines = content.split("\n");
  const entries: string[] = [];
  let collecting = false;
  let currentEntry: string[] = [];

  for (let i = lines.length - 1; i >= 0 && entries.length < count; i--) {
    const line = lines[i];

    if (line.match(/^## \d{4}-\d{2}-\d{2}/)) {
      if (currentEntry.length > 0) {
        entries.unshift(currentEntry.join("\n"));
        currentEntry = [];
      }
      collecting = true;
    }

    if (collecting && line.trim()) {
      currentEntry.unshift(line);
    }
  }

  if (currentEntry.length > 0) {
    entries.unshift(currentEntry.join("\n"));
  }

  return entries.slice(-count);
}

/** Extract decide entries from session */
function extractDecideEntries(content: string): string[] {
  const decideEntries: string[] = [];
  const lines = content.split("\n");

  for (const line of lines) {
    if (line.match(/^- decide(?:\([^)]+\))?:/)) {
      decideEntries.push(line.trim());
    }
  }

  return decideEntries;
}

/** Count tasks by status from session content */
function countTasksInSession(content: string): { completed: number; total: number } {
  const completed = (content.match(/- \[x\]/gi) || []).length;
  const total = (content.match(/- \[[ x]\]/gi) || []).length;
  return { completed, total };
}

// ─────────────────────────────────────────────────────────────────────────────
// Command
// ─────────────────────────────────────────────────────────────────────────────

export function registerSessionCommand(program: Command): void {
  const session = program
    .command("session")
    .description("Manage session journals");

  // ── loaf session start ─────────────────────────────────────────────────────

  session
    .command("start")
    .description("Start/resume session for current branch")
    .action(async () => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      // Ensure sessions directory exists
      const sessionsDir = join(agentsDir, "sessions");
      if (!existsSync(sessionsDir)) {
        mkdirSync(sessionsDir, { recursive: true });
      }

      // Detect current branch
      const branch = getCurrentBranch();
      if (branch === "unknown") {
        console.error(`  ${red("error:")} Not in a git repository`);
        process.exit(1);
      }
      
      if (branch.startsWith("detached-")) {
        console.error(`  ${yellow("⚠")} Warning: In detached HEAD state (${branch})`);
        console.error(`  ${gray("Sessions in detached HEAD state are isolated to the specific commit.")}`);
      }

      console.log(`\n  ${bold("loaf session start")}\n`);
      console.log(`  Branch: ${cyan(branch)}`);

      // Find linked spec
      const specInfo = findSpecForBranch(agentsDir, branch);
      if (specInfo) {
        console.log(`  Linked to: ${bold(specInfo.id)} — ${specInfo.title}`);
      } else {
        console.log(`  ${yellow("!")} No linked spec found — creating ad-hoc session`);
      }

      // Atomically get existing session or create new one
      const { filePath: sessionFilePath, data: sessionData, content: sessionContent, isNew } = 
        await getOrCreateSession(agentsDir, branch, specInfo);
      
      if (isNew) {
        console.log(`  ${green("+")} Creating new session file`);
      } else {
        console.log(`  ${green("✓")} Resuming existing session`);
      }

      // Compute state
      const lastCommit = getLastCommitSha();
      const commits = getRecentCommits(3);
      const { completed, total } = countTasksInSession(sessionContent);

      // Build entries based on whether this is a new or resumed session
      const entries: string[] = [];
      if (isNew) {
        entries.push(`resume(${branch}): session started`);
      } else {
        entries.push(`resume(${branch}): session resumed`);
      }
      if (lastCommit !== "unknown") {
        entries.push(`context: last commit ${lastCommit}`);
      }
      if (completed > 0 || total > 0) {
        entries.push(`progress: ${completed}/${total} tasks completed`);
      }
      if (commits.length > 0) {
        entries.push("recent commits:");
        for (const commit of commits.slice(0, 3)) {
          entries.push(`  - ${commit.hash} ${commit.message}`);
        }
      }

      // Append entry with appropriate header
      await appendEntry(
        sessionFilePath,
        isNew ? `${getDateTimeString()} — Start` : `${getDateTimeString()} — Resume`,
        entries,
        (data) => {
          data.status = "active";
          data.last_updated = getTimestamp();
          data.last_entry = getTimestamp();
        }
      );

      console.log(`  ${green("✓")} Session active: ${gray(sessionFilePath.replace(agentsDir, ".agents"))}`);

      // Output recent entries for context
      const recentEntries = extractRecentEntries(sessionContent, 15);
      if (recentEntries.length > 0) {
        console.log(`\n  ${bold("Recent journal entries:")}\n`);
        for (const entry of recentEntries.slice(0, 20)) {
          const lines = entry.split("\n");
          for (const line of lines.slice(0, 5)) {
            console.log(`    ${line}`);
          }
          if (lines.length > 5) {
            console.log(`    ${gray("...")}`);
          }
        }
      }

      console.log();
    });

  // ── loaf session end ───────────────────────────────────────────────────────

  session
    .command("end")
    .description("End session with progress summary")
    .action(async () => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const branch = getCurrentBranch();
      if (branch === "unknown") {
        console.error(`  ${red("error:")} Not in a git repository`);
        process.exit(1);
      }

      // Find active session by branch lookup
      let existingSession = findActiveSessionForBranch(agentsDir, branch);
      if (!existingSession) {
        console.error(`  ${red("error:")} No active session found for branch ${branch}`);
        process.exit(1);
      }
      
      const sessionFilePath = existingSession.filePath;
      const session = existingSession;

      console.log(`\n  ${bold("loaf session end")}\n`);

      // Compute progress
      const lastCommit = getLastCommitSha();
      const commits = getRecentCommits(5);
      const { completed, total } = countTasksInSession(session.content);
      const commitCount = commits.length;

      // Build pause entries
      const entries: string[] = [];
      entries.push(`pause(${branch}): session paused`);
      entries.push(`progress: ${completed}/${total} tasks completed, ${commitCount} commits`);
      if (lastCommit !== "unknown") {
        entries.push(`last commit: ${lastCommit}`);
      }

      // Prompt for final entries
      console.log(`  ${yellow("?")} Consider adding final entries:`);
      console.log(`    ${gray("loaf session log \"decide(scope): key decision\"")}`);
      console.log(`    ${gray("loaf session log \"conclude(scope): final notes\"")}`);
      console.log(`    ${gray("loaf session log \"todo(next): follow-up task\"")}`);
      console.log();

      // Append pause entry
      await appendEntry(
        sessionFilePath,
        `${getDateTimeString()} — Pause`,
        entries,
        (data) => {
          data.status = "paused";
          data.last_updated = getTimestamp();
          data.last_entry = getTimestamp();
        }
      );

      console.log(`  ${green("✓")} Session paused: ${gray(sessionFilePath.replace(agentsDir, ".agents"))}`);
      console.log();
    });

  // ── loaf session log ───────────────────────────────────────────────────────

  session
    .command("log [entry]")
    .description("Log entry to session journal")
    .option("--from-hook", "Parse entry from hook stdin")
    .option("--detect-linear", "Detect Linear magic words in recent commits")
    .action(async (entry: string, options: { fromHook?: boolean; detectLinear?: boolean }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const branch = getCurrentBranch();
      if (branch === "unknown") {
        console.error(`  ${red("error:")} Not in a git repository`);
        process.exit(1);
      }
      
      if (branch.startsWith("detached-")) {
        console.error(`  ${yellow("⚠")} Warning: In detached HEAD state (${branch})`);
      }

      // Find active session by branch lookup
      let existingSession = findActiveSessionForBranch(agentsDir, branch);
      if (!existingSession) {
        // For hook calls, no-op gracefully instead of erroring
        // This prevents hooks from failing when no session exists
        if (options.fromHook) {
          process.exit(0);
        }
        console.error(`  ${red("error:")} No active session found for branch ${branch}. Run 'loaf session start' first.`);
        process.exit(1);
      }
      
      const sessionFilePath = existingSession.filePath;
      const session = existingSession;

      let entryText = entry;

      // Handle --from-hook: parse JSON from stdin
      if (options.fromHook) {
        try {
          const stdin = await readStdin(); // Read from fd 0 (stdin)
          const hookData = JSON.parse(stdin);
          
          // Extract command from generic tool payload
          // Support both flat (tool_input) and nested (tool.input) formats for cross-harness compatibility
          const command = hookData.tool_input?.command || 
                          hookData.tool?.input?.command || 
                          hookData.input?.command;
          
          if (command && typeof command === "string") {
            // Parse Bash command to detect entry type
            if (command.includes("git commit")) {
              // Extract commit message if available
              const msgMatch = command.match(/-m\s+['"]([^'"]+)['"]/) || command.match(/-m\s+(\S+)/);
              const message = msgMatch ? msgMatch[1] : "commit";
              const hash = getLastCommitSha();
              entryText = `commit(${hash}): ${message}`;
            } else if (command.includes("gh pr create")) {
              // Extract PR title from command
              const titleMatch = command.match(/--title\s+["']([^"']+)["']/);
              const title = titleMatch ? titleMatch[1] : "";
              
              // Try to extract PR number from hook output (if available)
              // The raw field may contain tool output with PR URL
              const raw = hookData.raw;
              let prNum = "";
              if (raw && typeof raw === "string") {
                // Match PR URL patterns like https://github.com/owner/repo/pull/123
                const prUrlMatch = raw.match(/https:\/\/github\.com\/[^\/]+\/[^\/]+\/pull\/(\d+)/);
                if (prUrlMatch) {
                  prNum = prUrlMatch[1];
                }
              }
              
              if (prNum && title) {
                entryText = `pr(#${prNum}): ${title}`;
              } else if (title) {
                entryText = `pr: ${title}`;
              } else {
                entryText = `pr: PR created`;
              }
            } else if (command.includes("gh pr merge")) {
              // Try to extract PR number from URL or direct argument
              const prMatch = command.match(/merge\s+https:\/\/github\.com\/[^\/]+\/[^\/]+\/pull\/(\d+)/) ||
                               command.match(/pr\s+merge\s+(\d+)/);
              const prNum = prMatch ? prMatch[1] : "unknown";
              entryText = `merge(#${prNum}): merged`;
            } else {
              // Generic entry for unrecognized command
              entryText = `progress(hook): ${command.slice(0, 100)}${command.length > 100 ? "..." : ""}`;
            }
          } else {
            // Fallback: try legacy fields for backward compatibility
            if (hookData.commit) {
              entryText = `commit(${hookData.commit}): ${hookData.message || "commit"}`;
            } else if (hookData.pr) {
              entryText = `pr(#${hookData.pr}): ${hookData.title || "PR created"}`;
            } else if (hookData.merge) {
              entryText = `merge(#${hookData.merge}): merged`;
            } else {
              // No command found - this is a hook-safe no-op for when session doesn't exist
              // Exit 0 so hooks don't fail, but don't log anything
              process.exit(0);
            }
          }
        } catch (err) {
          console.error(`  ${red("error:")} Failed to parse stdin JSON: ${err}`);
          process.exit(1);
        }
      }

      // Handle --detect-linear: scan commits for magic words
      if (options.detectLinear) {
        try {
          const commits = getRecentCommits(3);
          const detections: Array<{ issueId: string; action: string; commit: string }> = [];

          // Magic word patterns: Fixes/Closes/Resolves + ISSUE-ID
          const magicPattern = /\b(fixe?s?d?|close?s?d?|resolve?s?d?)\s+([A-Z]+-\d+)/gi;

          for (const commit of commits) {
            const matches = commit.message.matchAll(magicPattern);
            for (const match of matches) {
              detections.push({
                issueId: match[2],
                action: match[1].toLowerCase(),
                commit: commit.hash,
              });
            }
          }

          if (detections.length === 0) {
            console.log(`  ${gray("ℹ")} No Linear magic words detected in recent commits`);
            process.exit(0);
          }

          // Create entry text summarizing detections
          const uniqueIssues = [...new Set(detections.map((d) => d.issueId))];
          entryText = `discover(linear): found magic words for ${uniqueIssues.join(", ")}`;
        } catch (err) {
          console.error(`  ${red("error:")} Failed to scan commits: ${err}`);
          process.exit(1);
        }
      }

      // Validate entry is provided
      if (!entryText) {
        console.error(`  ${red("error:")} Entry text is required. Use: loaf session log "type(scope): description"`);
        console.error(`  ${gray("Examples:")}`);
        console.error(`    ${gray("loaf session log \"decide(hooks): remove bash wrappers\"")}`);
        console.error(`    ${gray("loaf session log \"todo(next): implement tests\"")}`);
        process.exit(1);
      }

      // Validate entry format
      const parsedEntry = parseEntry(entryText);
      if (!parsedEntry) {
        console.error(`  ${red("error:")} Invalid entry format. Use: type(scope): description`);
        console.error(`  ${gray("Examples:")}`);
        console.error(`    ${gray("loaf session log \"decide(hooks): remove bash wrappers\"")}`);
        console.error(`    ${gray("loaf session log \"todo(next): implement tests\"")}`);
        process.exit(1);
      }

      // Append entry
      await appendEntry(
        sessionFilePath,
        `${getDateTimeString()}`,
        [formatEntry(parsedEntry)],
        (data) => {
          data.last_updated = getTimestamp();
          data.last_entry = getTimestamp();
        }
      );

      console.log(`  ${green("✓")} Logged: ${cyan(entryText)}`);
    });

  // ── loaf session archive ───────────────────────────────────────────────────

  session
    .command("archive")
    .description("Archive completed session")
    .option("--branch <branch>", "Archive session for specific branch (default: current)")
    .action(async (options: { branch?: string }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const branch = options.branch || getCurrentBranch();
      if (branch === "unknown" && !options.branch) {
        console.error(`  ${red("error:")} Not in a git repository. Use --branch <branch>`);
        process.exit(1);
      }

      // Find active session by branch lookup
      let existingSession = findActiveSessionForBranch(agentsDir, branch);
      if (!existingSession) {
        console.error(`  ${red("error:")} No active session found for branch ${branch}`);
        process.exit(1);
      }
      
      const sessionFilePath = existingSession.filePath;
      const session = existingSession;

      console.log(`\n  ${bold("loaf session archive")}\n`);

      // Ensure archive directory exists
      const archiveDir = join(agentsDir, "sessions", "archive");
      if (!existsSync(archiveDir)) {
        mkdirSync(archiveDir, { recursive: true });
      }

      // Extract key decisions
      const decideEntries = extractDecideEntries(session.content);
      if (decideEntries.length > 0) {
        console.log(`  ${bold("Key decisions extracted:")}`);
        for (const entry of decideEntries.slice(0, 10)) {
          console.log(`    ${entry}`);
        }
        if (decideEntries.length > 10) {
          console.log(`    ${gray(`... and ${decideEntries.length - 10} more`)}`);
        }
        console.log();
      }

      // Update frontmatter
      const now = getTimestamp();
      session.data.status = "archived";
      session.data.archived_at = now;
      session.data.last_updated = now;

      // Write updated content
      const newContent = matter.stringify(
        session.content,
        session.data as unknown as Record<string, unknown>
      );
      writeFileSync(sessionFilePath, newContent, "utf-8");

      // Move to archive
      const fileName = basename(sessionFilePath);
      const archivePath = join(archiveDir, fileName);

      try {
        renameSync(sessionFilePath, archivePath);
        console.log(`  ${green("✓")} Archived: ${gray(archivePath.replace(agentsDir, ".agents"))}`);
      } catch (err) {
        console.error(`  ${red("error:")} Failed to move file: ${err}`);
        process.exit(1);
      }

      console.log();
    });

  // ── loaf session list ────────────────────────────────────────────────────

  session
    .command("list")
    .description("List all active and archived sessions")
    .option("--all", "Include archived sessions")
    .action(async (options: { all?: boolean }) => {
      const agentsDir = findAgentsDir();
      if (!agentsDir) {
        console.error(`  ${red("error:")} Could not find .agents/ directory`);
        process.exit(1);
      }

      const sessionsDir = join(agentsDir, "sessions");
      if (!existsSync(sessionsDir)) {
        console.log(`\n  ${gray("No sessions directory found.")}`);
        console.log(`  Run ${cyan("loaf session start")} to create your first session.\n`);
        return;
      }

      console.log(`\n  ${bold("loaf session list")}\n`);

      // Collect active sessions
      const activeSessions: Array<{
        filePath: string;
        data: SessionFrontmatter;
        content: string;
        isArchived: boolean;
      }> = [];

      const files = readdirSync(sessionsDir).filter(f => f.endsWith(".md"));
      for (const file of files) {
        const filePath = join(sessionsDir, file);
        const session = readSessionFile(filePath);
        if (session && session.data.status !== "archived") {
          activeSessions.push({ ...session, filePath, isArchived: false });
        }
      }

      // Collect archived sessions if requested
      const archivedSessions: typeof activeSessions = [];
      if (options.all) {
        const archiveDir = join(sessionsDir, "archive");
        if (existsSync(archiveDir)) {
          const archiveFiles = readdirSync(archiveDir).filter(f => f.endsWith(".md"));
          for (const file of archiveFiles) {
            const filePath = join(archiveDir, file);
            const session = readSessionFile(filePath);
            if (session) {
              archivedSessions.push({ ...session, filePath, isArchived: true });
            }
          }
        }
      }

      // Display active sessions
      if (activeSessions.length === 0) {
        console.log(`  ${gray("No active sessions found.")}`);
      } else {
        console.log(`  ${bold("Active Sessions")}:`);
        for (const s of activeSessions) {
          const statusColor = s.data.status === "active" ? green : yellow;
          const specInfo = s.data.spec ? gray(` (${s.data.spec})`) : "";
          const lastUpdated = s.data.last_updated 
            ? gray(` — updated ${new Date(s.data.last_updated).toLocaleDateString()}`)
            : "";
          console.log(`    ${statusColor("●")} ${s.data.branch}${specInfo}${lastUpdated}`);
          console.log(`      ${gray(s.filePath.replace(agentsDir, ".agents"))}`);
        }
      }

      // Display archived sessions
      if (options.all && archivedSessions.length > 0) {
        console.log();
        console.log(`  ${bold("Archived Sessions")}:`);
        for (const s of archivedSessions) {
          const archivedDate = s.data.archived_at 
            ? gray(` — archived ${new Date(s.data.archived_at).toLocaleDateString()}`)
            : "";
          console.log(`    ${gray("○")} ${s.data.branch}${archivedDate}`);
          console.log(`      ${gray(s.filePath.replace(agentsDir, ".agents"))}`);
        }
      }

      // Summary
      console.log();
      console.log(`  ${activeSessions.length} active${options.all ? `, ${archivedSessions.length} archived` : ""}`);
      console.log();
    });
}
