/**
 * Path resolution for the souls catalog.
 *
 * The CLI is bundled by tsup into `dist-cli/`; at runtime `__dirname` points
 * inside the bundle, so we walk up looking for `package.json` with `name`
 * `loaf` to locate the loaf source root (mirrors the pattern in
 * `cli/commands/{build,install,version,setup}.ts`).
 */

import { readFileSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

/**
 * Walk up from `__dirname` looking for the `loaf` package root.
 * Throws if not found within 10 levels.
 */
export function findLoafRoot(): string {
  let dir = __dirname;
  for (let i = 0; i < 10; i++) {
    const pkgPath = join(dir, "package.json");
    try {
      const pkg = JSON.parse(readFileSync(pkgPath, "utf-8")) as { name?: string };
      if (pkg.name === "loaf") return dir;
    } catch {
      // not found, walk up
    }
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error("Could not find loaf root directory (no package.json with name 'loaf')");
}

/**
 * Absolute path to the souls catalog (`content/souls/`) inside the loaf
 * package root. Optional `loafRoot` lets callers (mainly tests) override the
 * root resolution.
 */
export function getCatalogDir(loafRoot?: string): string {
  return join(loafRoot ?? findLoafRoot(), "content", "souls");
}
