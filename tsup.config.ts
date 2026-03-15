import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["cli/index.ts"],
  format: ["esm"],
  target: "node18",
  outDir: "dist-cli",
  clean: true,
  sourcemap: true,
  dts: false,
  shims: false,
  splitting: false,
  banner: {
    js: "#!/usr/bin/env node",
  },
});
