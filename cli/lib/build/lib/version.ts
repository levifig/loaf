/**
 * Version Injection Utility
 *
 * Reads version from package.json and injects it into frontmatter at build time.
 */

import { readFileSync } from "fs";
import { join } from "path";
import type { SkillFrontmatter } from "../types.js";

export function getVersion(rootDir: string): string {
  const pkgPath = join(rootDir, "package.json");
  const pkg = JSON.parse(readFileSync(pkgPath, "utf-8"));
  return pkg.version;
}

export function injectVersion<T extends Record<string, unknown>>(
  frontmatter: T,
  version: string,
): T & { version: string } {
  return { ...frontmatter, version };
}
