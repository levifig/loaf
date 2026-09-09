#!/usr/bin/env node

import { createHash, randomBytes } from "node:crypto";
import { accessSync, constants, existsSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, rmSync, statSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { delimiter, dirname, join, relative, resolve, sep } from "node:path";
import { spawnSync } from "node:child_process";
import { tmpdir } from "node:os";
import { observedClientVersion, parseRunnerArgs, publishReceiptIfSuccessful } from "./capability-runner-utils.mjs";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "../..");
const platform = `${process.platform}-${process.arch}`;
const candidatePluginPath = "plugins/loaf";
const markerPattern = /^LOAF_CLAUDE_STARTUP_SMOKE_[A-F0-9]{12}$/;

export function parseClaudeStreamOutput(raw, marker) {
  const events = raw.split(/\r?\n/).filter(Boolean).map((line) => JSON.parse(line));
  const hook = events.find((event) => event.type === "system" && event.subtype === "hook_response" && event.hook_event === "SessionStart");
  if (!hook || typeof hook.output !== "string") throw new Error("SessionStart hook response was not observed");
  const native = JSON.parse(hook.output);
  const hookSpecific = native.hookSpecificOutput;
  const additionalContext = hookSpecific?.additionalContext;
  const assistantTexts = [];
  for (const event of events) {
    if (event.type === "assistant") {
      for (const block of event.message?.content ?? []) if (typeof block.text === "string") assistantTexts.push(block.text);
    }
    if (event.type === "result" && typeof event.result === "string") assistantTexts.push(event.result);
  }
  return {
    native,
    hookObservation: {
      event_name: "SessionStart:startup",
      native_json: hookSpecific?.hookEventName === "SessionStart" && typeof additionalContext === "string",
      hook_event_name: hookSpecific?.hookEventName ?? "",
      additional_context_marker: typeof additionalContext === "string" && additionalContext.includes(marker),
    },
    assistantMarkerMatch: assistantTexts.some((text) => text.trim() === marker),
  };
}

export function claudeVersion(result) {
  return observedClientVersion(result, /^(\S+) \(Claude Code\)$/);
}

export function candidateRuntimeEnvironment(candidateBinary, dbPath, inheritedPATH = process.env.PATH ?? "") {
  return {
    LOAF_DB: dbPath,
    LOAF_DEV_LINK: "0",
    LOAF_BUILD_TARGETS: platform,
    LOAF_NATIVE_ARTIFACT_DRY_RUN: "0",
    LOAF_BIN: undefined,
    PATH: [dirname(candidateBinary), inheritedPATH].filter(Boolean).join(delimiter),
  };
}

export function verifyCandidatePATH(candidateBinary, envPATH) {
  const filename = process.platform === "win32" ? "loaf.exe" : "loaf";
  for (const directory of envPATH.split(delimiter).filter(Boolean)) {
    const executable = resolve(directory, filename);
    try {
      accessSync(executable, constants.X_OK);
      if (!statSync(executable).isFile()) continue;
    } catch {
      continue;
    }
    if (realpathSync(executable) !== realpathSync(candidateBinary)) throw new Error("PATH does not resolve the candidate loaf binary");
    return candidateBinary;
  }
  throw new Error("candidate loaf is not executable on PATH");
}

function run(command, args, cwd, env = {}, timeout = 120000) {
  const result = spawnSync(command, args, {
    cwd,
    env: { ...process.env, LOAF_DB: undefined, ...env },
    encoding: "utf8",
    timeout,
    maxBuffer: 16 * 1024 * 1024,
  });
  return {
    status: result.status ?? 1,
    stdout: result.stdout ?? "",
    stderr: result.stderr ?? "",
    error: result.error,
  };
}

function sha256(path) {
  return createHash("sha256").update(readFileSync(path)).digest("hex");
}

function main(argv = process.argv.slice(2)) {
  const { client, receiptPath } = parseRunnerArgs(argv);
  const marker = `LOAF_CLAUDE_STARTUP_SMOKE_${randomBytes(6).toString("hex").toUpperCase()}`;
  if (!markerPattern.test(marker)) throw new Error("generated marker does not match the required format");
  const timestamp = new Date().toISOString();
  const tempRoot = mkdtempSync(join(tmpdir(), "loaf-claude-code-startup-smoke-"));
  if (resolve(tempRoot).startsWith(`${repoRoot}${sep}`)) throw new Error("disposable Claude Code smoke root must be outside the repository");
  const cleanup = () => {
    try {
      rmSync(tempRoot, { recursive: true, force: true, maxRetries: 3, retryDelay: 50 });
    } catch {
      return false;
    }
    return !existsSync(tempRoot);
  };
  for (const signal of ["SIGINT", "SIGTERM", "SIGHUP"]) {
    process.once(signal, () => {
      cleanup();
      process.exit(128 + ({ SIGINT: 2, SIGTERM: 15, SIGHUP: 1 }[signal] ?? 1));
    });
  }
  const disposableRepo = join(tempRoot, "repo");
  const dbDir = join(tempRoot, "state");
  mkdirSync(disposableRepo, { recursive: true });
  mkdirSync(dbDir, { recursive: true });
  const dbPath = join(dbDir, "loaf.sqlite");
  const candidatePlugin = join(repoRoot, candidatePluginPath);
  const candidateBinaryPath = join("bin", "native", platform, process.platform === "win32" ? "loaf.exe" : "loaf");
  const candidateBinary = join(repoRoot, candidateBinaryPath);
  const candidateEnv = candidateRuntimeEnvironment(candidateBinary, dbPath);
  let smoke;
  let failure;
  let cleanupSucceeded = false;
  try {
    const buildGo = run("go", ["run", "./cmd/loafdev", "build-go"], repoRoot, candidateEnv);
    if (buildGo.status !== 0) throw new Error("candidate Go build failed");
    if (!existsSync(candidateBinary)) throw new Error("candidate native binary is missing");
    verifyCandidatePATH(candidateBinary, candidateEnv.PATH);
    const buildClaude = run("loaf", ["build", "--target", "claude-code"], repoRoot, candidateEnv);
    if (buildClaude.status !== 0) throw new Error("candidate Claude build failed");
    if (existsSync(join(candidatePlugin, "bin"))) throw new Error("candidate Claude plugin must not ship an executable");
    const candidateDigests = {
      hooks_path: "plugins/loaf/hooks/hooks.json",
      hooks_sha256: sha256(join(candidatePlugin, "hooks", "hooks.json")),
      native_binary_path: candidateBinaryPath.split(sep).join("/"),
      native_binary_sha256: sha256(candidateBinary),
    };
    const version = run(client, ["--version"], repoRoot);
    if (run("git", ["init", "-q"], disposableRepo).status !== 0) throw new Error("disposable Git initialization failed");
    if (run("loaf", ["state", "init", "--json"], disposableRepo, candidateEnv).status !== 0) throw new Error("isolated Loaf state initialization failed");
    if (run("loaf", ["journal", "log", `discover(smoke): ${marker}`], disposableRepo, candidateEnv).status !== 0) throw new Error("isolated journal marker write failed");
    const claudeArgs = [
      "--plugin-dir", "<repo>/plugins/loaf",
      "--strict-mcp-config", "--mcp-config", '{"mcpServers":{}}',
      "--no-session-persistence", "--setting-sources", "", "--tools", "",
      "--include-hook-events", "--output-format", "stream-json", "-p",
      "Reply with exactly the unique marker present in Loaf continuity context, and nothing else.",
    ];
    const claude = run(client, [
      "--plugin-dir", candidatePlugin, "--strict-mcp-config", "--mcp-config", '{"mcpServers":{}}',
      "--no-session-persistence", "--setting-sources", "", "--tools", "", "--include-hook-events",
      "--output-format", "stream-json", "-p", "Reply with exactly the unique marker present in Loaf continuity context, and nothing else.",
    ], disposableRepo, candidateEnv);
    const parsed = parseClaudeStreamOutput(claude.stdout, marker);
    const resolvedBinary = verifyCandidatePATH(candidateBinary, candidateEnv.PATH);
    if (sha256(candidateBinary) !== candidateDigests.native_binary_sha256 || sha256(join(candidatePlugin, "hooks", "hooks.json")) !== candidateDigests.hooks_sha256) throw new Error("candidate artifacts changed during the smoke");
    smoke = {
      evidence_version: 4,
      timestamp,
      target: "claude-code",
      surface: "cli",
      version: claudeVersion(version),
      platform,
      installed_mode: "plugin-dir",
      context_mode: "startup",
      adapter: "claude-session-start-v1",
      mode: "explicit-plugin-dir",
      invocation: { command: "claude", args: claudeArgs, cwd: "<disposable-repo>" },
      setup: ["build candidate with Go and LOAF_DEV_LINK=0", "prepend candidate directory to PATH and verify resolution", "create disposable Git repository", "initialize absolute disposable LOAF_DB", "write random marker to isolated journal"],
      candidate_plugin_path: candidatePluginPath,
      runtime_lookup: "PATH",
      resolved_binary_path: relative(repoRoot, resolvedBinary).split(sep).join("/"),
      exit_code: claude.status,
      stderr_empty: claude.stderr.length === 0,
      model_visible_marker_observed: parsed.hookObservation.additional_context_marker,
      assistant_marker_match: parsed.assistantMarkerMatch,
      marker,
      hook_observation: parsed.hookObservation,
      candidate_artifacts: candidateDigests,
    };
    if (claude.status !== 0 || claude.stderr.length !== 0 || !parsed.hookObservation.additional_context_marker || !parsed.assistantMarkerMatch) throw new Error("model-visible marker smoke did not pass");
  } catch (error) {
    failure = error;
  } finally {
    cleanupSucceeded = cleanup();
  }
  if (failure) throw failure;
  if (!cleanupSucceeded) throw new Error("disposable Claude Code smoke cleanup failed");
  publishReceiptIfSuccessful(receiptPath, smoke, true);
  process.stdout.write(`${JSON.stringify({ receipt: receiptPath, exit_code: smoke.exit_code, assistant_marker_match: smoke.assistant_marker_match, cleanup_succeeded: true }, null, 2)}\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : "Claude Code startup smoke failed"}\n`);
    process.exitCode = 1;
  }
}
