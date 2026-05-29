#!/usr/bin/env node
/**
 * Verify generated Go command artifacts are present and synchronized.
 */

import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

const rootDir = process.cwd();
const packageJSON = JSON.parse(readFileSync(join(rootDir, "package.json"), "utf8"));

const requiredFiles = [
  "bin/loaf",
  "dist-cli/index.js",
  "plugins/loaf/bin/loaf",
  "plugins/loaf/dist-cli/index.js",
];

for (const file of requiredFiles) {
  if (!existsSync(join(rootDir, file))) {
    fail(`missing required artifact: ${file}`);
  }
}

assertSinglePublicCommand(packageJSON);
assertSame("bin/loaf", "plugins/loaf/bin/loaf");
assertSame("dist-cli/index.js", "plugins/loaf/dist-cli/index.js");

console.log("Go command artifacts are present and synchronized.");

function assertSinglePublicCommand(manifest) {
  const bin = manifest.bin;
  if (!bin || typeof bin !== "object" || Array.isArray(bin)) {
    fail("package.json must expose a bin object with one public command");
  }
  const entries = Object.entries(bin);
  if (entries.length !== 1 || entries[0][0] !== "loaf" || entries[0][1] !== "bin/loaf") {
    fail("package.json must expose exactly one public command: loaf -> bin/loaf");
  }
}

function assertSame(left, right) {
  const leftBytes = readFileSync(join(rootDir, left));
  const rightBytes = readFileSync(join(rootDir, right));
  if (!leftBytes.equals(rightBytes)) {
    fail(`${right} is stale; run npm run build`);
  }
}

function fail(message) {
  console.error(`ERROR: ${message}`);
  process.exit(1);
}
