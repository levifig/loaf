import { existsSync, mkdirSync, rmSync } from "fs";
import { join } from "path";

export function resetTargetOutput(distDir: string, subdirs: string[] = []): void {
  if (existsSync(distDir)) {
    rmSync(distDir, { recursive: true });
  }

  mkdirSync(distDir, { recursive: true });

  for (const subdir of subdirs) {
    mkdirSync(join(distDir, subdir), { recursive: true });
  }
}
