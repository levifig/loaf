/**
 * Conventional Commit Parser
 *
 * Shells out to git to get commits since the last tag, then parses them
 * into typed structures for changelog generation and version bumping.
 */

import { execFileSync } from "child_process";

export type ChangelogSection =
  | "Added"
  | "Fixed"
  | "Changed"
  | "Breaking Changes"
  | "Other";

export interface ParsedCommit {
  hash: string;
  type: string;
  message: string;
  breaking: boolean;
  section: ChangelogSection | null;
  raw: string;
}

const CONVENTIONAL_RE = /^(\w+)(\(.+?\))?(!)?:\s*(.+)$/;

const SECTION_MAP: Record<string, ChangelogSection | null> = {
  feat: "Added",
  fix: "Fixed",
  refactor: "Changed",
  perf: "Changed",
  docs: null,
  chore: null,
  ci: null,
  test: null,
  build: null,
  style: null,
};

const BREAKING_BODY_RE = /^BREAKING[ -]CHANGE:/m;

/** Get the most recent git tag, or null if none exist. */
export function getLastTag(cwd: string): string | null {
  try {
    const tag = execFileSync("git", ["describe", "--tags", "--abbrev=0"], {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
    return tag || null;
  } catch {
    return null;
  }
}

/** Get all commits since the given tag (or all commits if tag is null). Returns newest first. */
export function getCommitsSince(
  cwd: string,
  tag: string | null,
): ParsedCommit[] {
  const format = "%h%x00%s%x00%B%x00";
  const args = tag
    ? ["log", `${tag}..HEAD`, `--format=${format}`]
    : ["log", `--format=${format}`];

  let output: string;
  try {
    output = execFileSync("git", args, {
      cwd,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
      maxBuffer: 10 * 1024 * 1024,
    });
  } catch {
    return [];
  }

  if (!output.trim()) {
    return [];
  }

  // Each commit produces: hash\0subject\0body\0
  // Split on the trailing null byte to get per-commit chunks.
  const chunks = output.split("\0\n").filter((c) => c.trim());

  const commits: ParsedCommit[] = [];

  for (const chunk of chunks) {
    const parts = chunk.split("\0");
    if (parts.length < 2) continue;

    const hash = parts[0].trim();
    const subject = parts[1].trim();
    const body = (parts[2] || "").trim();

    if (!hash || !subject) continue;

    commits.push(parseCommit(hash, subject, body));
  }

  return commits;
}

function parseCommit(hash: string, subject: string, body: string): ParsedCommit {
  const match = subject.match(CONVENTIONAL_RE);

  if (!match) {
    return {
      hash,
      type: "",
      message: subject,
      breaking: BREAKING_BODY_RE.test(body),
      section: BREAKING_BODY_RE.test(body) ? "Breaking Changes" : "Other",
      raw: subject,
    };
  }

  const type = match[1];
  const bangIndicator = !!match[3];
  const message = match[4];
  const breakingFromBody = BREAKING_BODY_RE.test(body);
  const breaking = bangIndicator || breakingFromBody;

  let section: ChangelogSection | null;
  if (breaking) {
    section = "Breaking Changes";
  } else if (type in SECTION_MAP) {
    section = SECTION_MAP[type];
  } else {
    section = "Other";
  }

  return {
    hash,
    type,
    message,
    breaking,
    section,
    raw: subject,
  };
}

/** Suggest a version bump based on parsed commits. */
export function suggestBump(
  commits: ParsedCommit[],
): "major" | "minor" | "patch" {
  if (commits.some((c) => c.breaking)) {
    return "major";
  }
  if (commits.some((c) => c.section === "Added")) {
    return "minor";
  }
  return "patch";
}
