/**
 * Project Detection Module
 *
 * Scans the current working directory to detect languages, frameworks,
 * and existing documentation patterns.
 */

import { existsSync, readFileSync } from "fs";
import { join } from "path";

export interface DetectedLanguage {
  name: string;
  confidence: "high" | "medium";
  indicator: string;
}

export interface DetectedFramework {
  name: string;
  language: string;
  indicator: string;
}

export interface ExistingStructure {
  hasAgentsDir: boolean;
  hasAgentsMd: boolean;
  hasDocsDir: boolean;
  hasChangelog: boolean;
  hasClaudeDir: boolean;
  hasLoafJson: boolean;
  existingDocs: string[];
}

export interface ProjectInfo {
  languages: DetectedLanguage[];
  frameworks: DetectedFramework[];
  existing: ExistingStructure;
}

function safeReadFile(path: string): string {
  try {
    return readFileSync(path, "utf-8");
  } catch {
    return "";
  }
}

function hasLanguage(languages: DetectedLanguage[], name: string): boolean {
  return languages.some((l) => l.name === name);
}

function hasJsFamily(languages: DetectedLanguage[]): boolean {
  return hasLanguage(languages, "TypeScript") || hasLanguage(languages, "JavaScript");
}

function detectLanguages(cwd: string): DetectedLanguage[] {
  const languages: DetectedLanguage[] = [];

  // TypeScript
  if (existsSync(join(cwd, "tsconfig.json"))) {
    languages.push({ name: "TypeScript", confidence: "high", indicator: "tsconfig.json" });
  } else if (existsSync(join(cwd, "package.json"))) {
    const content = safeReadFile(join(cwd, "package.json"));
    if (content.includes('"typescript"') || content.includes('"ts-node"')) {
      languages.push({ name: "TypeScript", confidence: "medium", indicator: "package.json (typescript in deps)" });
    }
  }

  // JavaScript (only when no TS indicators)
  if (!hasLanguage(languages, "TypeScript")) {
    if (existsSync(join(cwd, "package.json"))) {
      languages.push({ name: "JavaScript", confidence: "high", indicator: "package.json" });
    }
  }

  // Python
  if (existsSync(join(cwd, "pyproject.toml"))) {
    languages.push({ name: "Python", confidence: "high", indicator: "pyproject.toml" });
  } else if (existsSync(join(cwd, "setup.py"))) {
    languages.push({ name: "Python", confidence: "high", indicator: "setup.py" });
  } else if (existsSync(join(cwd, "requirements.txt"))) {
    languages.push({ name: "Python", confidence: "medium", indicator: "requirements.txt" });
  } else if (existsSync(join(cwd, "uv.lock"))) {
    languages.push({ name: "Python", confidence: "medium", indicator: "uv.lock" });
  } else if (existsSync(join(cwd, "Pipfile"))) {
    languages.push({ name: "Python", confidence: "medium", indicator: "Pipfile" });
  }

  // Ruby
  if (existsSync(join(cwd, "Gemfile"))) {
    languages.push({ name: "Ruby", confidence: "high", indicator: "Gemfile" });
  } else if (existsSync(join(cwd, ".ruby-version"))) {
    languages.push({ name: "Ruby", confidence: "medium", indicator: ".ruby-version" });
  } else if (existsSync(join(cwd, ".ruby-gemset"))) {
    languages.push({ name: "Ruby", confidence: "medium", indicator: ".ruby-gemset" });
  }

  // Go
  if (existsSync(join(cwd, "go.mod"))) {
    languages.push({ name: "Go", confidence: "high", indicator: "go.mod" });
  }

  // Rust
  if (existsSync(join(cwd, "Cargo.toml"))) {
    languages.push({ name: "Rust", confidence: "high", indicator: "Cargo.toml" });
  }

  return languages;
}

function detectFrameworks(cwd: string, languages: DetectedLanguage[]): DetectedFramework[] {
  const frameworks: DetectedFramework[] = [];

  // JS/TS frameworks
  if (hasJsFamily(languages)) {
    const lang = hasLanguage(languages, "TypeScript") ? "TypeScript" : "JavaScript";

    // Next.js
    const nextConfigs = ["next.config.js", "next.config.mjs", "next.config.ts"];
    const nextIndicator = nextConfigs.find((f) => existsSync(join(cwd, f)));
    if (nextIndicator) {
      frameworks.push({ name: "Next.js", language: lang, indicator: nextIndicator });
    }

    // React (only if not Next.js)
    if (!nextIndicator && existsSync(join(cwd, "package.json"))) {
      const content = safeReadFile(join(cwd, "package.json"));
      if (content.includes('"react"')) {
        frameworks.push({ name: "React", language: lang, indicator: "package.json (react in deps)" });
      }
    }
  }

  // Python frameworks
  if (hasLanguage(languages, "Python")) {
    // Read dependency sources
    const pyprojectContent = safeReadFile(join(cwd, "pyproject.toml"));
    const requirementsContent = safeReadFile(join(cwd, "requirements.txt"));
    const pyDeps = pyprojectContent + "\n" + requirementsContent;

    // FastAPI
    if (pyDeps.includes("fastapi")) {
      frameworks.push({ name: "FastAPI", language: "Python", indicator: "fastapi in deps" });
    }

    // Django
    if (existsSync(join(cwd, "manage.py")) || pyDeps.includes("django")) {
      const indicator = existsSync(join(cwd, "manage.py")) ? "manage.py" : "django in deps";
      frameworks.push({ name: "Django", language: "Python", indicator });
    }

    // Flask
    if (pyDeps.includes("flask")) {
      frameworks.push({ name: "Flask", language: "Python", indicator: "flask in deps" });
    }
  }

  // Ruby frameworks
  if (hasLanguage(languages, "Ruby")) {
    if (existsSync(join(cwd, "config", "routes.rb")) || existsSync(join(cwd, "bin", "rails"))) {
      const indicator = existsSync(join(cwd, "config", "routes.rb")) ? "config/routes.rb" : "bin/rails";
      frameworks.push({ name: "Rails", language: "Ruby", indicator });
    }
  }

  return frameworks;
}

function detectExistingStructure(cwd: string): ExistingStructure {
  const docFiles = [
    "docs/VISION.md",
    "docs/STRATEGY.md",
    "docs/ARCHITECTURE.md",
    "README.md",
  ];

  const existingDocs = docFiles.filter((f) => existsSync(join(cwd, f)));

  return {
    hasAgentsDir: existsSync(join(cwd, ".agents")),
    hasAgentsMd: existsSync(join(cwd, ".agents", "AGENTS.md")),
    hasDocsDir: existsSync(join(cwd, "docs")),
    hasChangelog: existsSync(join(cwd, "CHANGELOG.md")),
    hasClaudeDir: existsSync(join(cwd, ".claude")),
    hasLoafJson: existsSync(join(cwd, ".agents", "loaf.json")),
    existingDocs,
  };
}

export function detectProject(cwd: string): ProjectInfo {
  const languages = detectLanguages(cwd);
  const frameworks = detectFrameworks(cwd, languages);
  const existing = detectExistingStructure(cwd);

  return { languages, frameworks, existing };
}
