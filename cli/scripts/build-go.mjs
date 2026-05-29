#!/usr/bin/env node
/**
 * Build the Go front controller used as the public loaf command.
 */

import { spawnSync } from "node:child_process";
import { chmodSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";

const rootDir = process.cwd();
const output = join(rootDir, "bin", "loaf");

mkdirSync(dirname(output), { recursive: true });

const env = {
  ...process.env,
  CGO_ENABLED: "0",
};

const result = spawnSync("go", ["build", "-trimpath", "-o", output, "./cmd/loaf"], {
  cwd: rootDir,
  env,
  stdio: "inherit",
});

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

chmodSync(output, 0o755);
console.log(`✓ Built Go front controller: ${output}`);
