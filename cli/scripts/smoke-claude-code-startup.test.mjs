import test from "node:test";
import assert from "node:assert/strict";
import { chmodSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";
import { claudeVersion, parseClaudeStreamOutput, candidateRuntimeEnvironment, verifyCandidatePATH } from "./smoke-claude-code-startup.mjs";
import { observedClientVersion, parseRunnerArgs, publishReceiptIfSuccessful } from "./capability-runner-utils.mjs";

test("candidate runtime uses PATH, isolates state and never selects a private override", () => {
  const env = candidateRuntimeEnvironment("/candidate/bin/loaf", "/scratch/loaf.sqlite", "/user/bin");
  assert.equal(env.PATH, `/candidate/bin${delimiter}/user/bin`);
  assert.equal(env.LOAF_DB, "/scratch/loaf.sqlite");
  assert.equal(env.LOAF_DEV_LINK, "0");
  assert.equal(env.LOAF_BUILD_TARGETS, `${process.platform}-${process.arch}`);
  assert.equal(env.LOAF_NATIVE_ARTIFACT_DRY_RUN, "0");
  assert.equal(env.LOAF_BIN, undefined);
});

test("PATH proof refuses shadowed, nonexecutable and absent candidates", () => {
  const root = mkdtempSync(join(tmpdir(), "loaf-path-proof-"));
  const candidateDir = join(root, "candidate");
  const otherDir = join(root, "other");
  const filename = process.platform === "win32" ? "loaf.exe" : "loaf";
  const binary = join(candidateDir, filename);
  try {
    mkdirSync(candidateDir);
    mkdirSync(otherDir);
    writeFileSync(binary, "candidate", { mode: 0o755 });
    writeFileSync(join(otherDir, filename), "other", { mode: 0o755 });
    assert.equal(verifyCandidatePATH(binary, `${candidateDir}${delimiter}${otherDir}`), binary);
    assert.throws(() => verifyCandidatePATH(binary, `${otherDir}${delimiter}${candidateDir}`), /does not resolve the candidate/);
    assert.throws(() => verifyCandidatePATH(binary, ""), /not executable on PATH/);
    if (process.platform !== "win32") {
      chmodSync(binary, 0o644);
      assert.throws(() => verifyCandidatePATH(binary, candidateDir), /not executable on PATH/);
    }
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("parses native SessionStart output and exact assistant marker", () => {
  const marker = "LOAF_CLAUDE_STARTUP_SMOKE_ABCDEF123456";
  const raw = [
    JSON.stringify({ type: "system", subtype: "hook_response", hook_event: "SessionStart", output: JSON.stringify({ hookSpecificOutput: { hookEventName: "SessionStart", additionalContext: `digest ${marker}` } }) }),
    JSON.stringify({ type: "assistant", message: { content: [{ type: "text", text: marker }] } }),
    JSON.stringify({ type: "result", result: marker }),
  ].join("\n");
  const parsed = parseClaudeStreamOutput(raw, marker);
  assert.equal(parsed.hookObservation.native_json, true);
  assert.equal(parsed.hookObservation.additional_context_marker, true);
  assert.equal(parsed.assistantMarkerMatch, true);
});

test("rejects a stream without SessionStart hook response", () => {
  assert.throws(() => parseClaudeStreamOutput(JSON.stringify({ type: "result", result: "no hook" }), "marker"), /SessionStart hook response/);
});

test("records the observed Claude version without a version allowlist", () => {
  for (const version of ["9.8.7", "9.8.70", "10.0.0-preview.1+build.2"]) {
    assert.equal(claudeVersion({ status: 0, stdout: `${version} (Claude Code)\n` }), version);
  }
  assert.equal(claudeVersion({ status: 0, stdout: "unfamiliar identity" }), "unknown");
  assert.equal(claudeVersion({ status: 1, stdout: "9.8.7 (Claude Code)" }), "unknown");
});

test("unavailable or malformed version output is unknown, not a capability refusal", () => {
  for (const result of [
    { status: 127, stdout: "9.8.7" }, { status: null }, { status: 0 },
    { status: 0, stdout: "" }, { status: 0, stdout: "2.x" },
    { status: 0, stdout: "unfamiliar build\nwith diagnostics" },
  ]) assert.equal(observedClientVersion(result), "unknown");
});

test("requires one safe value for every runner option", () => {
  const observed = parseRunnerArgs(["--client", "claude", "--receipt", "proof.json"]);
  assert.equal(observed.client, "claude");
  assert.equal(observed.expectedVersion, undefined);
  const parsed = parseRunnerArgs(["--client", "/opt/claude", "--receipt", "proof.json"]);
  assert.equal(parsed.client, "/opt/claude");
  assert.ok(parsed.receiptPath.endsWith("proof.json"));
  assert.throws(() => parseRunnerArgs(["--client", "claude"]), /missing required option/);
  assert.throws(() => parseRunnerArgs(["--client", "claude", "--client", "other", "--receipt", "proof.json"]), /duplicate option/);
  assert.throws(() => parseRunnerArgs(["--unknown", "value", "--client", "claude", "--receipt", "proof.json"]), /unknown option/);
  assert.throws(() => parseRunnerArgs(["--client", "claude\nunsafe", "--receipt", "proof.json"]), /safe executable/);
  assert.throws(() => parseRunnerArgs(["--client", "claude", "--expected-version", "9.8.7", "--receipt", "proof.json"]), /unknown option --expected-version/);
  assert.throws(() => parseRunnerArgs(["--client", "claude", "--receipt", "proof.txt"]), /safe JSON path/);
});

test("publishes success atomically and preserves an existing receipt on failure", () => {
  const root = mkdtempSync(join(tmpdir(), "loaf-capability-receipt-test-"));
  const receiptPath = join(root, "receipt.json");
  try {
    writeFileSync(receiptPath, "existing\n");
    assert.equal(publishReceiptIfSuccessful(receiptPath, { status: "failed" }, false), false);
    assert.equal(readFileSync(receiptPath, "utf8"), "existing\n");
    assert.equal(publishReceiptIfSuccessful(receiptPath, { status: "passed" }, true), true);
    assert.deepEqual(JSON.parse(readFileSync(receiptPath, "utf8")), { status: "passed" });
    assert.deepEqual(readdirSync(root), ["receipt.json"]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
