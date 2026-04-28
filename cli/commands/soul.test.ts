/**
 * Soul Command Tests
 *
 * End-to-end tests for `loaf soul` subcommands, exercising the built CLI
 * binary against per-test mkdtemp project roots. These complement the
 * library-level tests in `cli/lib/souls/souls.test.ts` — these tests verify
 * that the command surface (Commander wiring, exit codes, project-root
 * resolution) plumbs through correctly to the souls library.
 *
 * The tests use the real `content/souls/` catalog shipped with the repo
 * (fellowship + none). All inputs are static literals — no untrusted data
 * flows into the spawned subprocess.
 *
 * @vitest-environment node
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { execFileSync, execSync } from "child_process";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from "fs";
import { tmpdir } from "os";
import { join } from "path";

let PROJECT_ROOT: string;
const CLI_PATH = join(process.cwd(), "dist-cli", "index.js");
const FELLOWSHIP_SOURCE = join(
  process.cwd(),
  "content",
  "souls",
  "fellowship",
  "SOUL.md",
);
const NONE_SOURCE = join(
  process.cwd(),
  "content",
  "souls",
  "none",
  "SOUL.md",
);

function run(
  args: string[],
  opts: { cwd?: string } = {},
): { exitCode: number; stdout: string; stderr: string } {
  try {
    const stdout = execFileSync("node", [CLI_PATH, ...args], {
      encoding: "utf-8",
      cwd: opts.cwd ?? PROJECT_ROOT,
      stdio: ["pipe", "pipe", "pipe"],
      timeout: 15000,
    });
    return { exitCode: 0, stdout, stderr: "" };
  } catch (error: unknown) {
    const err = error as { status?: number; stdout?: string; stderr?: string };
    return {
      exitCode: err.status ?? 1,
      stdout: err.stdout ?? "",
      stderr: err.stderr ?? "",
    };
  }
}

function strip(s: string): string {
  return s.replace(/\x1b\[\d+m/g, "");
}

beforeEach(() => {
  PROJECT_ROOT = realpathSync(mkdtempSync(join(tmpdir(), "loaf-soul-cmd-")));
  // Initialize as a git repo so `git rev-parse --show-toplevel` resolves to
  // PROJECT_ROOT (otherwise it'd resolve to the loaf repo and pollute the
  // real `.agents/loaf.json`).
  execSync("git init", { cwd: PROJECT_ROOT, stdio: "ignore" });
  execSync("git config user.email test@test", { cwd: PROJECT_ROOT, stdio: "ignore" });
  execSync("git config user.name Test", { cwd: PROJECT_ROOT, stdio: "ignore" });
});

afterEach(() => {
  rmSync(PROJECT_ROOT, { recursive: true, force: true });
});

// ---------------------------------------------------------------------------

describe("loaf soul list", () => {
  it("prints both catalog souls with descriptions", () => {
    const { exitCode, stdout } = run(["soul", "list"]);
    expect(exitCode).toBe(0);
    const out = strip(stdout);
    expect(out).toMatch(/^fellowship — /m);
    expect(out).toMatch(/^none — /m);
  });
});

describe("loaf soul show <name>", () => {
  it("prints catalog SOUL.md content", () => {
    const { exitCode, stdout } = run(["soul", "show", "none"]);
    expect(exitCode).toBe(0);
    expect(stdout).toBe(readFileSync(NONE_SOURCE, "utf-8"));
  });

  it("does not write .agents/SOUL.md", () => {
    run(["soul", "show", "fellowship"]);
    const { exitCode } = run(["soul", "current"]);
    expect(exitCode).toBe(0);
    expect(() =>
      readFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "utf-8"),
    ).toThrow();
  });

  it("errors on unknown soul name", () => {
    const { exitCode, stderr } = run(["soul", "show", "nonexistent"]);
    expect(exitCode).toBe(1);
    expect(strip(stderr)).toContain("Unknown soul");
  });
});

describe("loaf soul current", () => {
  it("defaults to 'none' when loaf.json is missing", () => {
    const { exitCode, stdout } = run(["soul", "current"]);
    expect(exitCode).toBe(0);
    expect(stdout.trim()).toBe("none");
  });

  it("defaults to 'none' when soul field is missing from loaf.json", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "loaf.json"),
      JSON.stringify({ integrations: {} }),
    );
    const { exitCode, stdout } = run(["soul", "current"]);
    expect(exitCode).toBe(0);
    expect(stdout.trim()).toBe("none");
  });

  it("prints the configured soul", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "loaf.json"),
      JSON.stringify({ soul: "fellowship" }),
    );
    const { exitCode, stdout } = run(["soul", "current"]);
    expect(exitCode).toBe(0);
    expect(stdout.trim()).toBe("fellowship");
  });
});

describe("loaf soul use <name>", () => {
  it("copies catalog SOUL.md and writes soul: <name> to loaf.json on a fresh project", () => {
    const { exitCode } = run(["soul", "use", "none"]);
    expect(exitCode).toBe(0);

    const written = readFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      "utf-8",
    );
    expect(written).toBe(readFileSync(NONE_SOURCE, "utf-8"));

    const cfg = JSON.parse(
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    );
    expect(cfg.soul).toBe("none");
  });

  it("succeeds when local SOUL.md already matches a catalog hash", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      readFileSync(NONE_SOURCE, "utf-8"),
    );
    const { exitCode } = run(["soul", "use", "fellowship"]);
    expect(exitCode).toBe(0);
    expect(
      readFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "utf-8"),
    ).toBe(readFileSync(FELLOWSHIP_SOURCE, "utf-8"));
  });

  it("refuses to overwrite a diverged local SOUL.md without --force", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      "# Custom Soul\n\nUser-modified.\n",
    );
    const { exitCode, stderr } = run(["soul", "use", "none"]);
    expect(exitCode).toBe(1);
    expect(strip(stderr)).toContain("local SOUL.md diverges from catalog");
    expect(strip(stderr)).toContain("--force");

    expect(
      readFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "utf-8"),
    ).toBe("# Custom Soul\n\nUser-modified.\n");
    expect(() =>
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    ).toThrow();
  });

  it("overwrites a diverged local SOUL.md when --force is set", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      "# Custom Soul\n\nUser-modified.\n",
    );
    const { exitCode } = run(["soul", "use", "none", "--force"]);
    expect(exitCode).toBe(0);
    expect(
      readFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "utf-8"),
    ).toBe(readFileSync(NONE_SOURCE, "utf-8"));
  });

  it("preserves existing keys in loaf.json when writing the soul field", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "loaf.json"),
      JSON.stringify({ integrations: { linear: { enabled: true } } }),
    );
    const { exitCode } = run(["soul", "use", "fellowship"]);
    expect(exitCode).toBe(0);

    const cfg = JSON.parse(
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    );
    expect(cfg.soul).toBe("fellowship");
    expect(cfg.integrations?.linear?.enabled).toBe(true);
  });

  it("errors on unknown soul name", () => {
    const { exitCode, stderr } = run(["soul", "use", "nonexistent"]);
    expect(exitCode).toBe(1);
    expect(strip(stderr)).toContain("Unknown soul");
    expect(() =>
      readFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "utf-8"),
    ).toThrow();
  });
});
