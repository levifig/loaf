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
} from "fs";
import { join } from "path";
import matter from "gray-matter";

import { findAgentsDir } from "../lib/tasks/resolve.js";

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
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

/** Get current git branch */
function getCurrentBranch(): string {
  try {
    return execSync("git branch --show-current", {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
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

/** Get session file path for a branch (timestamped format) */
function getSessionFilePath(agentsDir: string, branch: string, timestamp?: Date): string {
  const now = timestamp || new Date();
  const datePrefix = now.toISOString().slice(0, 10).replace(/-/g, "");
  const timePrefix = now.toTimeString().slice(0, 8).replace(/:/g, "");
  const slug = branch.replace(/^feat\//, "").replace(/[^a-zA-Z0-9-_]/g, "-").replace(/-+/g, "-");
  return join(agentsDir, "sessions", `${datePrefix}-${timePrefix}-${slug}.md`);
}

/** Find active session for a branch by scanning files */
function findActiveSessionForBranch(agentsDir: string, branch: string): { filePath: string; data: SessionFrontmatter; content: string } | null {
  const sessionsDir = join(agentsDir, "sessions");
  if (!existsSync(sessionsDir)) return null;
  
  const files = readdirSync(sessionsDir).filter(f => f.endsWith(".md") && !f.startsWith("archive"));
  
  for (const file of files) {
    const filePath = join(sessionsDir, file);
    const session = readSessionFile(filePath);
    if (session && session.data.branch === branch && session.data.status !== "archived") {
      return { filePath, data: session.data, content: session.content };
    }
  }
  
  return null;
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
): SessionFrontmatter {
  const now = getTimestamp();
  const frontmatter: SessionFrontmatter = {
    branch,
    status: "active",
    created: now,
    spec: specInfo?.id,
    title: specInfo ? `${specInfo.id}: ${specInfo.title}` : `Session: ${branch}`,
  };

  const body = specInfo
    ? `# ${specInfo.id}: ${specInfo.title}\n\n## Context\n\nSession for ${specInfo.id}: ${specInfo.title}\n\n## Current State\n\nSession started.\n\n## Next Steps\n\n- [ ] First action item\n`
    : `# Session: ${branch}\n\n## Context\n\nAd-hoc session for branch \`${branch}\`.\n\n## Current State\n\nSession started.\n\n## Next Steps\n\n- [ ] First action item\n`;

  const content = matter.stringify(body, frontmatter as unknown as Record<string, unknown>);
  writeFileSync(filePath, content, "utf-8");

  return frontmatter;
}

/** Append entry to session file */
function appendEntry(
  filePath: string,
  header: string,
  entries: string[],
  updateFrontmatter: (data: SessionFrontmatter) => void
): void {
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
  writeFileSync(filePath, newContent, "utf-8");
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
    if (line.match(/^- decide\([^)]+\):/)) {
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
        console.error(`  ${red("error:")} Not in a git repository or no branch detected`);
        process.exit(1);
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

      // Capture timestamp once at session start to ensure consistent filename
      const sessionTimestamp = new Date();

      // Find existing active session or create new one
      let existingSession = findActiveSessionForBranch(agentsDir, branch);
      let sessionFilePath: string;
      let session: { data: SessionFrontmatter; content: string };

      if (existingSession) {
        console.log(`  ${green("✓")} Resuming existing session`);
        sessionFilePath = existingSession.filePath;
        session = existingSession;
      } else {
        console.log(`  ${green("+")} Creating new session file`);
        sessionFilePath = getSessionFilePath(agentsDir, branch, sessionTimestamp);
        createSessionFile(sessionFilePath, branch, specInfo);
        session = readSessionFile(sessionFilePath)!;
      }

      // Compute state
      const lastCommit = getLastCommitSha();
      const commits = getRecentCommits(3);
      const { completed, total } = countTasksInSession(session.content);

      // Build resume entries
      const entries: string[] = [];
      entries.push(`resume(${branch}): session started`);
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

      // Append resume entry
      appendEntry(
        sessionFilePath,
        `${getDateTimeString()} — Start`,
        entries,
        (data) => {
          data.status = "active";
          data.last_updated = getTimestamp();
          data.last_entry = getTimestamp();
        }
      );

      console.log(`  ${green("✓")} Session active: ${gray(sessionFilePath.replace(agentsDir, ".agents"))}`);

      // Output recent entries for context
      const recentEntries = extractRecentEntries(session.content, 15);
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
      appendEntry(
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

      // Find active session by branch lookup
      let existingSession = findActiveSessionForBranch(agentsDir, branch);
      if (!existingSession) {
        console.error(`  ${red("error:")} No active session found for branch ${branch}. Run 'loaf session start' first.`);
        process.exit(1);
      }
      
      const sessionFilePath = existingSession.filePath;
      const session = existingSession;

      let entryText = entry;

      // Handle --from-hook: parse JSON from stdin
      if (options.fromHook) {
        try {
          const stdin = readFileSync(0, "utf-8"); // Read from fd 0 (stdin)
          const hookData = JSON.parse(stdin);
          
          // Extract command from generic tool payload
          const command = hookData.tool_input?.command || hookData.input?.command;
          
          if (command && typeof command === "string") {
            // Parse Bash command to detect entry type
            if (command.includes("git commit")) {
              // Extract commit message if available
              const msgMatch = command.match(/-m\s+['"]([^'"]+)['"]/) || command.match(/-m\s+(\S+)/);
              const message = msgMatch ? msgMatch[1] : "commit";
              const hash = getLastCommitSha();
              entryText = `commit(${hash}): ${message}`;
            } else if (command.includes("gh pr create")) {
              entryText = `pr: PR created`;
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
              console.error(`  ${red("error:")} Could not parse hook data - no command found`);
              process.exit(1);
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
      appendEntry(
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
      const fileName = sessionFilePath.split("/").pop()!;
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
}
