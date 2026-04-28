/**
 * `installSoul` tests (SPEC-033, T6/T7).
 *
 * Covers the three install paths — fresh, legacy upgrade, already-configured —
 * using `mkdtempSync` project roots. The catalog is synthesized inside a
 * second tmpdir so the tests do not depend on the live `content/souls/`
 * payload (mirrors the pattern in `cli/lib/souls/souls.test.ts`).
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

import { installSoul, LEGACY_SOUL } from "./install-soul.js";
import { DEFAULT_SOUL } from "../config/agents-config.js";

const FELLOWSHIP_SOUL = `# The Warden

You are the Warden — a Wizard who guides the fellowship.
`;

const NONE_SOUL = `# The Orchestrator

> A neutral, function-only soul.

You are the orchestrator.
`;

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
  LOAF_ROOT = mkdtempSync(join(tmpdir(), "loaf-install-soul-root-"));
  PROJECT_ROOT = mkdtempSync(join(tmpdir(), "loaf-install-soul-proj-"));
  buildFakeLoafRoot(LOAF_ROOT);
});

afterEach(() => {
  rmSync(LOAF_ROOT, { recursive: true, force: true });
  rmSync(PROJECT_ROOT, { recursive: true, force: true });
});

describe("installSoul: fresh install", () => {
  it("writes the none SOUL.md and sets soul: none in loaf.json", () => {
    const result = installSoul(PROJECT_ROOT, LOAF_ROOT);

    expect(result.action).toBe("fresh");
    expect(result.soul).toBe(DEFAULT_SOUL);
    expect(result.soul).toBe("none");

    const soulPath = join(PROJECT_ROOT, ".agents", "SOUL.md");
    expect(existsSync(soulPath)).toBe(true);
    expect(readFileSync(soulPath, "utf-8")).toBe(NONE_SOUL);

    const cfgPath = join(PROJECT_ROOT, ".agents", "loaf.json");
    const cfg = JSON.parse(readFileSync(cfgPath, "utf-8"));
    expect(cfg.soul).toBe("none");
  });

  it("creates .agents/ when missing", () => {
    expect(existsSync(join(PROJECT_ROOT, ".agents"))).toBe(false);
    installSoul(PROJECT_ROOT, LOAF_ROOT);
    expect(existsSync(join(PROJECT_ROOT, ".agents"))).toBe(true);
  });
});

describe("installSoul: legacy upgrade", () => {
  it("pins soul: fellowship when SOUL.md exists but loaf.json has no soul field", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    const existingSoul = "# Custom Warden\n\nUser-edited content.\n";
    writeFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), existingSoul);

    const result = installSoul(PROJECT_ROOT, LOAF_ROOT);

    expect(result.action).toBe("legacy-upgrade");
    expect(result.soul).toBe(LEGACY_SOUL);
    expect(result.soul).toBe("fellowship");

    // SOUL.md untouched.
    const soulPath = join(PROJECT_ROOT, ".agents", "SOUL.md");
    expect(readFileSync(soulPath, "utf-8")).toBe(existingSoul);

    // loaf.json now has soul: fellowship.
    const cfg = JSON.parse(
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    );
    expect(cfg.soul).toBe("fellowship");
  });

  it("preserves existing keys in loaf.json when adding soul: fellowship", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      "# Existing Soul\n",
    );
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "loaf.json"),
      JSON.stringify({
        knowledge: { staleness_threshold_days: 14 },
        integrations: { linear: { enabled: true } },
      }),
    );

    installSoul(PROJECT_ROOT, LOAF_ROOT);

    const cfg = JSON.parse(
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    );
    expect(cfg.soul).toBe("fellowship");
    expect(cfg.knowledge?.staleness_threshold_days).toBe(14);
    expect(cfg.integrations?.linear?.enabled).toBe(true);
  });

  it("treats a corrupt loaf.json as no soul: field (legacy upgrade path)", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "# Existing\n");
    writeFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "{ not json");

    const result = installSoul(PROJECT_ROOT, LOAF_ROOT);

    expect(result.action).toBe("legacy-upgrade");
    const cfg = JSON.parse(
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    );
    expect(cfg.soul).toBe("fellowship");
  });
});

describe("installSoul: already configured", () => {
  it("is a no-op when both SOUL.md exists and soul: field is set", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    const existingSoul = "# Pre-existing Soul\n\nUntouched.\n";
    writeFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), existingSoul);
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "loaf.json"),
      JSON.stringify({ soul: "none", integrations: {} }),
    );

    const result = installSoul(PROJECT_ROOT, LOAF_ROOT);

    expect(result.action).toBe("noop");
    expect(result.soul).toBe("none");

    // Neither file changed.
    expect(readFileSync(join(PROJECT_ROOT, ".agents", "SOUL.md"), "utf-8")).toBe(
      existingSoul,
    );
    const cfg = JSON.parse(
      readFileSync(join(PROJECT_ROOT, ".agents", "loaf.json"), "utf-8"),
    );
    expect(cfg.soul).toBe("none");
    expect(cfg.integrations).toEqual({});
  });

  it("is a no-op when soul: fellowship is already configured (legacy already migrated)", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      FELLOWSHIP_SOUL,
    );
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "loaf.json"),
      JSON.stringify({ soul: "fellowship" }),
    );

    const result = installSoul(PROJECT_ROOT, LOAF_ROOT);
    expect(result.action).toBe("noop");
    expect(result.soul).toBe("fellowship");
  });
});

describe("installSoul: idempotence", () => {
  it("running twice on a fresh project produces a noop on the second call", () => {
    const first = installSoul(PROJECT_ROOT, LOAF_ROOT);
    expect(first.action).toBe("fresh");

    const second = installSoul(PROJECT_ROOT, LOAF_ROOT);
    expect(second.action).toBe("noop");
    expect(second.soul).toBe("none");
  });

  it("running twice on a legacy project produces a noop on the second call", () => {
    mkdirSync(join(PROJECT_ROOT, ".agents"));
    writeFileSync(
      join(PROJECT_ROOT, ".agents", "SOUL.md"),
      "# Legacy Warden\n",
    );

    const first = installSoul(PROJECT_ROOT, LOAF_ROOT);
    expect(first.action).toBe("legacy-upgrade");

    const second = installSoul(PROJECT_ROOT, LOAF_ROOT);
    expect(second.action).toBe("noop");
    expect(second.soul).toBe("fellowship");
  });
});
