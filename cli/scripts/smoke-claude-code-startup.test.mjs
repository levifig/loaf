import test from "node:test";
import assert from "node:assert/strict";
import { chmodSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";
import { claudeVersionMatches, parseClaudeStreamOutput, candidateRuntimeEnvironment, verifyCandidatePATH } from "./smoke-claude-code-startup.mjs";
import { parseRunnerArgs, publishReceiptIfSuccessful } from "./capability-runner-utils.mjs";

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

test("requires the exact Claude Code version token", () => {
  assert.equal(claudeVersionMatches("9.8.7 (Claude Code)\n", "9.8.7"), true);
  assert.equal(claudeVersionMatches("9.8.70 (Claude Code)", "9.8.7"), false);
  assert.equal(claudeVersionMatches("Claude Code 9.8.7", "9.8.7"), false);
});

test("requires one safe value for every runner option", () => {
  const parsed = parseRunnerArgs(["--client", "/opt/claude", "--expected-version", "9.8.7", "--receipt", "proof.json"]);
  assert.equal(parsed.client, "/opt/claude");
  assert.equal(parsed.expectedVersion, "9.8.7");
  assert.ok(parsed.receiptPath.endsWith("proof.json"));
  assert.throws(() => parseRunnerArgs(["--client", "claude"]), /missing required option/);
  assert.throws(() => parseRunnerArgs(["--client", "claude", "--client", "other", "--expected-version", "9.8.7", "--receipt", "proof.json"]), /duplicate option/);
  assert.throws(() => parseRunnerArgs(["--unknown", "value", "--client", "claude", "--expected-version", "9.8.7", "--receipt", "proof.json"]), /unknown option/);
  assert.throws(() => parseRunnerArgs(["--client", "claude\nunsafe", "--expected-version", "9.8.7", "--receipt", "proof.json"]), /safe executable/);
  assert.throws(() => parseRunnerArgs(["--client", "claude", "--expected-version", "9.8.7 unsafe", "--receipt", "proof.json"]), /exact safe identity/);
  assert.throws(() => parseRunnerArgs(["--client", "claude", "--expected-version", "9.8.7", "--receipt", "proof.txt"]), /safe JSON path/);
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
