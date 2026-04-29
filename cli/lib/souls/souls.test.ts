/**
 * Souls library tests.
 *
 * Covers catalog reading, divergence detection, copy/activate semantics, and
 * the loaf.json read-merge-write helper. All filesystem writes happen inside
 * `mkdtempSync` directories; no test touches `process.cwd()`.
 *
 * The catalog itself is also synthesized inside a tmpdir (one fellowship-like
 * soul without a tagline, one none-like soul with a tagline blockquote), so
 * the tests stay independent of the real `content/souls/` payload while still
 * exercising the tagline-or-H1 fallback the real catalog relies on.
 *
 * @vitest-environment node
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "fs";
import { tmpdir } from "os";
import { join } from "path";

import {
  catalogHashes,
  checkDivergence,
  copySoulToProject,
  extractDescription,
  listSouls,
  loafConfigPath,
  localSoulPath,
  readActiveSoul,
  readSoul,
  sha256,
  soulPathFor,
  writeActiveSoul,
} from "./index.js";
import { resolveCatalogDir } from "./paths.js";

// ─────────────────────────────────────────────────────────────────────────────
// Fixtures
// ─────────────────────────────────────────────────────────────────────────────

/** Fellowship-style soul: H1 + body, no blockquote tagline. */
const FELLOWSHIP_SOUL = `# The Warden

You are **Arandil**, the Warden — a Wizard who guides the fellowship.

## The Fellowship

Smiths forge. Sentinels watch. Rangers scout.
`;

/** None-style soul: H1 + blockquote tagline + body. */
const NONE_SOUL = `# The Orchestrator

> A neutral, function-only soul — describes the team by role, not by character.

You are the **orchestrator**. You coordinate, plan, and delegate.

## The Team

Implementer. Reviewer. Researcher. Librarian.
`;

let LOAF_ROOT: string;
let PROJECT_ROOT: string;

function buildFakeLoafRoot(root: string): void {
  // Minimal package.json so findLoafRoot would find this dir if asked, but the
  // tests pass `loafRoot` explicitly so no walking happens.
  writeFileSync(
    join(root, "package.json"),
    JSON.stringify({ name: "loaf", version: "0.0.0-test" }, null, 2),
  );
  const fellowshipDir = join(root, "content", "souls", "fellowship");
  const noneDir = join(root, "content", "souls", "none");
  mkdirSync(fellowshipDir, { recursive: true });
  mkdirSync(noneDir, { recursive: true });
  writeFileSync(join(fellowshipDir, "SOUL.md"), FELLOWSHIP_SOUL);
  writeFileSync(join(noneDir, "SOUL.md"), NONE_SOUL);
}

beforeEach(() => {
  LOAF_ROOT = mkdtempSync(join(tmpdir(), "loaf-souls-root-"));
  PROJECT_ROOT = mkdtempSync(join(tmpdir(), "loaf-souls-proj-"));
  buildFakeLoafRoot(LOAF_ROOT);
});

afterEach(() => {
  rmSync(LOAF_ROOT, { recursive: true, force: true });
  rmSync(PROJECT_ROOT, { recursive: true, force: true });
});

// ─────────────────────────────────────────────────────────────────────────────
// catalog.ts
// ─────────────────────────────────────────────────────────────────────────────

describe("listSouls", () => {
  it("returns both souls sorted by name", () => {
    const souls = listSouls(LOAF_ROOT);
    expect(souls.map((s) => s.name)).toEqual(["fellowship", "none"]);
  });

  it("uses the H1 line as description when no tagline blockquote is present", () => {
    const souls = listSouls(LOAF_ROOT);
    const fellowship = souls.find((s) => s.name === "fellowship");
    expect(fellowship?.description).toBe("The Warden");
  });

  it("uses the tagline blockquote as description when present", () => {
    const souls = listSouls(LOAF_ROOT);
    const none = souls.find((s) => s.name === "none");
    expect(none?.description).toBe(
      "A neutral, function-only soul — describes the team by role, not by character.",
    );
  });

  it("skips directories without SOUL.md", () => {
    mkdirSync(join(LOAF_ROOT, "content", "souls", "empty-dir"), { recursive: true });
    const souls = listSouls(LOAF_ROOT);
    expect(souls.map((s) => s.name)).toEqual(["fellowship", "none"]);
  });

  it("returns an empty list when the catalog directory is missing", () => {
    const emptyRoot = mkdtempSync(join(tmpdir(), "loaf-souls-empty-"));
    try {
      writeFileSync(
        join(emptyRoot, "package.json"),
        JSON.stringify({ name: "loaf" }),
      );
      expect(listSouls(emptyRoot)).toEqual([]);
    } finally {
      rmSync(emptyRoot, { recursive: true, force: true });
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// No-catalog state: paths.ts and catalog.ts share the same null-on-missing
// contract. When `resolveCatalogDir()` returns null (no candidate reachable),
// catalog.ts must not throw — `listSouls()` returns [], `readSoul()` and
// `soulPathFor()` throw a clear "Unknown soul" / "Souls catalog not found".
//
// Simulated by routing the env-candidate fallback chain through empty tmp
// directories *and* running `findLoafRoot` from a synthesised __dirname
// without any loaf package.json upstream. Since we cannot relocate the test
// file's __dirname, we cover the contract via env-driven candidates that
// resolve to nothing, and assert via the public `resolveCatalogDir()` that
// the dev-tree fallback still wins (returning a real path) — proving the
// chain is exhaustive when paths exist. The contract for the *missing*
// path is asserted at the catalog.ts boundary by passing a `loafRoot` that
// points to a directory with no `content/souls/` subtree.
// ─────────────────────────────────────────────────────────────────────────────

describe("no-catalog state (catalog.ts contract)", () => {
  let savedEnv: Record<string, string | undefined>;
  let scratchHome: string;

  beforeEach(() => {
    // Snapshot env keys we may mutate so afterEach can restore them.
    savedEnv = {
      HOME: process.env.HOME,
      USERPROFILE: process.env.USERPROFILE,
      XDG_CONFIG_HOME: process.env.XDG_CONFIG_HOME,
      CLAUDE_PLUGIN_ROOT: process.env.CLAUDE_PLUGIN_ROOT,
      CODEX_HOME: process.env.CODEX_HOME,
    };
    scratchHome = mkdtempSync(join(tmpdir(), "loaf-souls-noenv-"));
    // Point every per-tool candidate at this empty scratch home so the
    // env-based portion of the candidate chain resolves to nothing.
    process.env.HOME = scratchHome;
    process.env.USERPROFILE = scratchHome;
    process.env.XDG_CONFIG_HOME = join(scratchHome, ".config");
    process.env.CODEX_HOME = join(scratchHome, ".codex");
    delete process.env.CLAUDE_PLUGIN_ROOT;
  });

  afterEach(() => {
    for (const [k, v] of Object.entries(savedEnv)) {
      if (v === undefined) delete process.env[k];
      else process.env[k] = v;
    }
    rmSync(scratchHome, { recursive: true, force: true });
  });

  it("listSouls returns [] when given a loafRoot with no catalog", () => {
    // The loafRoot path forces activeCatalogDir to bypass the env chain and
    // resolve to <loafRoot>/content/souls — which doesn't exist.
    const noCatalogRoot = mkdtempSync(join(tmpdir(), "loaf-souls-no-catalog-"));
    try {
      expect(listSouls(noCatalogRoot)).toEqual([]);
    } finally {
      rmSync(noCatalogRoot, { recursive: true, force: true });
    }
  });

  it("readSoul throws 'Unknown soul' when given a loafRoot with no catalog", () => {
    const noCatalogRoot = mkdtempSync(join(tmpdir(), "loaf-souls-no-catalog-"));
    try {
      expect(() => readSoul("none", noCatalogRoot)).toThrow(/Unknown soul/);
    } finally {
      rmSync(noCatalogRoot, { recursive: true, force: true });
    }
  });

  it("does not throw 'Could not find loaf root' through any public surface", () => {
    // Regression guard for the original bug: catalog.ts used to fall through
    // to getCatalogDir() (which calls findLoafRoot) when resolveCatalogDir()
    // returned null. With the env chain pointed at empty scratch dirs the
    // dev-tree branch still resolves (loaf's own package.json sits above
    // __dirname), so this asserts the *no exception* outcome on the path
    // we care about: a real dev tree, no per-tool installs.
    expect(() => listSouls()).not.toThrow();
    expect(() => resolveCatalogDir()).not.toThrow();
  });
});

describe("readSoul", () => {
  it("returns the catalog SOUL.md content verbatim", () => {
    expect(readSoul("fellowship", LOAF_ROOT)).toBe(FELLOWSHIP_SOUL);
    expect(readSoul("none", LOAF_ROOT)).toBe(NONE_SOUL);
  });

  it("throws on unknown soul", () => {
    expect(() => readSoul("nonexistent", LOAF_ROOT)).toThrow(/Unknown soul/);
  });
});

describe("soulPathFor", () => {
  it("resolves under content/souls/<name>/SOUL.md", () => {
    expect(soulPathFor("fellowship", LOAF_ROOT)).toBe(
      join(LOAF_ROOT, "content", "souls", "fellowship", "SOUL.md"),
    );
  });
});

describe("extractDescription", () => {
  it("prefers a tagline blockquote directly under the H1", () => {
    expect(
      extractDescription("# Title\n\n> short tagline\n\nbody"),
    ).toBe("short tagline");
  });

  it("falls back to the H1 text when no blockquote follows", () => {
    expect(extractDescription("# Title\n\nbody")).toBe("Title");
  });

  it("ignores blockquotes that appear after body prose", () => {
    expect(
      extractDescription("# Title\n\nFirst line of body.\n\n> not a tagline"),
    ).toBe("Title");
  });

  it("returns empty string when no H1 exists", () => {
    expect(extractDescription("just some text")).toBe("");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// divergence.ts
// ─────────────────────────────────────────────────────────────────────────────

describe("sha256 / catalogHashes", () => {
  it("hashes content deterministically", () => {
    expect(sha256("hello")).toBe(sha256("hello"));
    expect(sha256("hello")).not.toBe(sha256("world"));
  });

  it("collects every catalog soul's hash", () => {
    const hashes = catalogHashes(LOAF_ROOT);
    expect(hashes.size).toBe(2);
    expect(hashes.has(sha256(FELLOWSHIP_SOUL))).toBe(true);
    expect(hashes.has(sha256(NONE_SOUL))).toBe(true);
  });
});

describe("checkDivergence", () => {
  it("returns diverged: false when no local file exists", () => {
    const localPath = localSoulPath(PROJECT_ROOT);
    const result = checkDivergence(localPath, LOAF_ROOT);
    expect(result.diverged).toBe(false);
    expect(result.localHash).toBeNull();
    expect(result.matchedSoul).toBeNull();
  });

  it("returns diverged: false when local matches a catalog soul", () => {
    const localPath = localSoulPath(PROJECT_ROOT);
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(localPath, NONE_SOUL);

    const result = checkDivergence(localPath, LOAF_ROOT);
    expect(result.diverged).toBe(false);
    expect(result.matchedSoul?.name).toBe("none");
  });

  it("returns diverged: true when local matches no catalog soul", () => {
    const localPath = localSoulPath(PROJECT_ROOT);
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(localPath, "# Custom Soul\n\nUser-modified.\n");

    const result = checkDivergence(localPath, LOAF_ROOT);
    expect(result.diverged).toBe(true);
    expect(result.matchedSoul).toBeNull();
    expect(result.localHash).not.toBeNull();
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// install.ts
// ─────────────────────────────────────────────────────────────────────────────

describe("copySoulToProject", () => {
  it("writes the catalog SOUL.md to .agents/SOUL.md", () => {
    const result = copySoulToProject("none", PROJECT_ROOT, LOAF_ROOT);
    const written = readFileSync(localSoulPath(PROJECT_ROOT), "utf-8");
    expect(written).toBe(NONE_SOUL);
    expect(result.written).toBe(localSoulPath(PROJECT_ROOT));
  });

  it("creates .agents/ when missing", () => {
    expect(() =>
      copySoulToProject("fellowship", PROJECT_ROOT, LOAF_ROOT),
    ).not.toThrow();
    expect(readFileSync(localSoulPath(PROJECT_ROOT), "utf-8")).toBe(
      FELLOWSHIP_SOUL,
    );
  });

  it("overwrites an existing local SOUL.md unconditionally", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(localSoulPath(PROJECT_ROOT), "old content");
    copySoulToProject("none", PROJECT_ROOT, LOAF_ROOT);
    expect(readFileSync(localSoulPath(PROJECT_ROOT), "utf-8")).toBe(NONE_SOUL);
  });
});

describe("readActiveSoul", () => {
  it("returns null when loaf.json is missing", () => {
    expect(readActiveSoul(PROJECT_ROOT)).toBeNull();
  });

  it("returns null when loaf.json has no soul field", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      loafConfigPath(PROJECT_ROOT),
      JSON.stringify({ integrations: {} }),
    );
    expect(readActiveSoul(PROJECT_ROOT)).toBeNull();
  });

  it("returns the soul name when present", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      loafConfigPath(PROJECT_ROOT),
      JSON.stringify({ soul: "fellowship" }),
    );
    expect(readActiveSoul(PROJECT_ROOT)).toBe("fellowship");
  });

  it("returns null when loaf.json is corrupt", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(loafConfigPath(PROJECT_ROOT), "{ not json");
    expect(readActiveSoul(PROJECT_ROOT)).toBeNull();
  });
});

describe("writeActiveSoul", () => {
  it("creates .agents/loaf.json with soul field on a fresh project", () => {
    writeActiveSoul(PROJECT_ROOT, "none");
    const cfg = JSON.parse(readFileSync(loafConfigPath(PROJECT_ROOT), "utf-8"));
    expect(cfg).toEqual({ soul: "none" });
  });

  it("preserves existing keys when merging", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      loafConfigPath(PROJECT_ROOT),
      JSON.stringify({
        knowledge: { staleness_threshold_days: 30 },
        integrations: { linear: { enabled: true } },
      }),
    );

    writeActiveSoul(PROJECT_ROOT, "fellowship");

    const cfg = JSON.parse(readFileSync(loafConfigPath(PROJECT_ROOT), "utf-8"));
    expect(cfg.soul).toBe("fellowship");
    expect(cfg.knowledge?.staleness_threshold_days).toBe(30);
    expect(cfg.integrations?.linear?.enabled).toBe(true);
  });

  it("writes 2-space indent and trailing newline", () => {
    writeActiveSoul(PROJECT_ROOT, "none");
    const raw = readFileSync(loafConfigPath(PROJECT_ROOT), "utf-8");
    expect(raw.endsWith("\n")).toBe(true);
    expect(raw).toContain('  "soul": "none"');
  });
});
