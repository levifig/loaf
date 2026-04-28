/**
 * Install-time soul-selection prompt tests.
 *
 * Covers the gating logic of `promptAndApplySoul` (interactive on/off,
 * fresh vs. legacy, empty catalog) and the helper that orders the catalog
 * for the prompt. Interactive TTY input itself is *not* exercised here —
 * `askChoice` falls through to its default when stdin is not a TTY, which
 * is the same semantics this helper depends on. A full TTY round-trip would
 * require pty plumbing the rest of the suite avoids, so we trust the
 * `askChoice` contract (covered in `cli/lib/prompts.test.ts`) and verify
 * the wiring around it.
 *
 * @vitest-environment node
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "fs";
import { tmpdir } from "os";
import { join } from "path";

import {
  orderSoulsForPrompt,
  promptAndApplySoul,
} from "./install-soul-prompt.js";
import type { SoulEntry } from "../souls/index.js";

// ---------------------------------------------------------------------------
// Fixtures — synthesise a tiny catalog inside a tmpdir so these tests stay
// independent of the real `content/souls/` payload.
// ---------------------------------------------------------------------------

const FELLOWSHIP_SOUL = `# The Warden\n\nFellowship soul body.\n`;
const NONE_SOUL = `# The Orchestrator\n\n> A neutral, function-only soul.\n\nNone soul body.\n`;

let LOAF_ROOT: string;
let PROJECT_ROOT: string;

function buildFakeLoafRoot(root: string): void {
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
  LOAF_ROOT = mkdtempSync(join(tmpdir(), "loaf-soul-prompt-root-"));
  PROJECT_ROOT = mkdtempSync(join(tmpdir(), "loaf-soul-prompt-proj-"));
  buildFakeLoafRoot(LOAF_ROOT);
});

afterEach(() => {
  rmSync(LOAF_ROOT, { recursive: true, force: true });
  rmSync(PROJECT_ROOT, { recursive: true, force: true });
});

// ---------------------------------------------------------------------------
// orderSoulsForPrompt
// ---------------------------------------------------------------------------

describe("orderSoulsForPrompt", () => {
  const sample: SoulEntry[] = [
    { name: "fellowship", description: "warden", path: "/fake/fellowship" },
    { name: "none", description: "neutral", path: "/fake/none" },
    { name: "studio", description: "studio", path: "/fake/studio" },
  ];

  it("hoists the default soul to the first position", () => {
    const ordered = orderSoulsForPrompt(sample, "none");
    expect(ordered.map((s) => s.name)).toEqual([
      "none",
      "fellowship",
      "studio",
    ]);
  });

  it("leaves order alphabetical when default is missing from catalog", () => {
    const ordered = orderSoulsForPrompt(sample, "nonexistent");
    expect(ordered.map((s) => s.name)).toEqual([
      "fellowship",
      "none",
      "studio",
    ]);
  });

  it("does not duplicate when default is already first after sort", () => {
    const partial: SoulEntry[] = [
      { name: "alpha", description: "a", path: "/fake/alpha" },
      { name: "beta", description: "b", path: "/fake/beta" },
    ];
    const ordered = orderSoulsForPrompt(partial, "alpha");
    expect(ordered.map((s) => s.name)).toEqual(["alpha", "beta"]);
  });
});

// ---------------------------------------------------------------------------
// promptAndApplySoul — gating logic
// ---------------------------------------------------------------------------

describe("promptAndApplySoul", () => {
  it("skips when interactive=false (CI / --yes / non-TTY)", async () => {
    const outcome = await promptAndApplySoul({
      projectRoot: PROJECT_ROOT,
      interactive: false,
      loafRoot: LOAF_ROOT,
    });
    expect(outcome).toEqual({ action: "skipped-non-interactive" });
    // No filesystem writes should have happened.
    expect(existsSync(join(PROJECT_ROOT, ".agents"))).toBe(false);
  });

  it("skips when an existing .agents/SOUL.md is present (legacy upgrade)", async () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      "# Existing user soul\n",
    );

    const outcome = await promptAndApplySoul({
      projectRoot: PROJECT_ROOT,
      interactive: true,
      loafRoot: LOAF_ROOT,
    });
    expect(outcome).toEqual({ action: "skipped-not-fresh" });
    // The pre-existing SOUL.md must be untouched.
    expect(
      readFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "utf-8"),
    ).toBe("# Existing user soul\n");
    // And we must not have written `soul:` into loaf.json — that's the
    // legacy-upgrade path's job (installSoul), not the prompt's.
    expect(existsSync(join(PROJECT_ROOT, ".agents", "loaf.json"))).toBe(false);
  });

  it("skips when soul: is already configured in loaf.json", async () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "loaf.json"),
      JSON.stringify({ soul: "fellowship" }),
    );

    const outcome = await promptAndApplySoul({
      projectRoot: PROJECT_ROOT,
      interactive: true,
      loafRoot: LOAF_ROOT,
    });
    expect(outcome).toEqual({ action: "skipped-not-fresh" });
  });

  it("on a fresh project with non-TTY stdin, applies the default soul (none)", async () => {
    // `askChoice` falls through to its `defaultChoice` argument when stdin
    // is not a TTY — vitest runs without a TTY, so this exercises the
    // pre-selected `none` default we ship for the prompt.
    const outcome = await promptAndApplySoul({
      projectRoot: PROJECT_ROOT,
      interactive: true,
      loafRoot: LOAF_ROOT,
    });
    expect(outcome).toEqual({ action: "prompted", soul: "none" });

    // Side effects: SOUL.md copied + loaf.json written.
    const soulPath = join(PROJECT_ROOT, ".agents", "SOUL.md");
    expect(readFileSync(soulPath, "utf-8")).toBe(NONE_SOUL);
    const cfg = JSON.parse(
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    );
    expect(cfg.soul).toBe("none");
  });

  it("returns skipped-empty-catalog when the catalog has no souls", async () => {
    const emptyRoot = mkdtempSync(join(tmpdir(), "loaf-soul-prompt-empty-"));
    try {
      writeFileSync(
        join(emptyRoot, "package.json"),
        JSON.stringify({ name: "loaf" }),
      );
      // No content/souls/ directory at all.
      const outcome = await promptAndApplySoul({
        projectRoot: PROJECT_ROOT,
        interactive: true,
        loafRoot: emptyRoot,
      });
      expect(outcome).toEqual({ action: "skipped-empty-catalog" });
      expect(existsSync(join(PROJECT_ROOT, ".agents"))).toBe(false);
    } finally {
      rmSync(emptyRoot, { recursive: true, force: true });
    }
  });
});
