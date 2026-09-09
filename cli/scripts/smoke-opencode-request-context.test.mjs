import test from "node:test";
import assert from "node:assert/strict";
import { delimiter, join } from "node:path";
import { buildCandidate, collectTextValues, modelVisibleProof, opencodeVersionMatches, parseOpenCodeJSONL, sanitizeError, sanitizedStderr } from "./smoke-opencode-request-context.mjs";

test("candidate build uses native Go and isolated non-linking host runtime", () => {
  const calls = [];
  const env = buildCandidate("/isolated/loaf.sqlite", (...args) => { calls.push(args); return { status: 0 }; });
  assert.deepEqual(calls.map(([command, args]) => [command, args]), [["go", ["run", "./cmd/loafdev", "build-go"]], ["loaf", ["build", "--target", "opencode"]]]);
  assert.equal(env.LOAF_DB, "/isolated/loaf.sqlite");
  assert.equal(env.LOAF_DEV_LINK, "0");
  assert.equal(env.LOAF_BUILD_TARGETS, `${process.platform}-${process.arch}`);
  assert.equal(env.LOAF_NATIVE_ARTIFACT_DRY_RUN, "0");
  assert.ok(env.PATH.split(delimiter)[0].endsWith(join("bin", "native", `${process.platform}-${process.arch}`)));
  for (const call of calls) assert.equal(call[3], env);
});

test("failed native build prevents target publication", () => {
  const calls = [];
  assert.throws(() => buildCandidate("/isolated/loaf.sqlite", (...args) => { calls.push(args); return { status: 1 }; }), /candidate Go build failed/);
  assert.equal(calls.length, 1);
});

test("requires the exact OpenCode version token", () => {
  assert.equal(opencodeVersionMatches("9.8.7\n", "9.8.7"), true);
  assert.equal(opencodeVersionMatches("9.8.70", "9.8.7"), false);
  assert.equal(opencodeVersionMatches("v9.8.7", "9.8.7"), false);
  assert.equal(opencodeVersionMatches("9.8.7-dev", "9.8.7"), false);
});

test("recursively extracts only string values at text keys", () => {
  const texts = collectTextValues({
    text: "top",
    nested: [{ text: "nested" }, { value: { text: "deep" } }],
    notText: "ignored",
  });
  assert.deepEqual(texts, ["top", "nested", "deep"]);
});

test("parses JSONL recursively and matches an exact trimmed marker", () => {
  const marker = "LOAF_OPENCODE_REQUEST_SMOKE_ABCDEF123456";
  const raw = [
    JSON.stringify({ type: "message", parts: [{ type: "text", text: `context ${marker}` }] }),
    JSON.stringify({ type: "result", data: { nested: { text: ` ${marker} ` } } }),
  ].join("\n");
  assert.equal(parseOpenCodeJSONL(raw, marker).assistantMarkerMatch, true);
  assert.equal(parseOpenCodeJSONL(JSON.stringify({ type: "message", text: "wrong" }), marker).assistantMarkerMatch, false);
});

test("rejects malformed and non-object JSONL events", () => {
  assert.throws(() => parseOpenCodeJSONL("{not-json}", "marker"), SyntaxError);
  assert.throws(() => parseOpenCodeJSONL("null", "marker"), SyntaxError);
  assert.throws(() => parseOpenCodeJSONL("[]", "marker"), SyntaxError);
});

test("assistant text alone is not model-visible proof without hook observation", () => {
  assert.equal(modelVisibleProof(true, false), false);
  assert.equal(modelVisibleProof(true, true), true);
});

test("sanitizes paths and bounds failure text", () => {
  const failure = sanitizeError(new Error("failed at /Users/test/.cache/loaf/tmp/repo with\nsecret details"), [
    ["/Users/test/.cache/loaf", "<home>"],
    ["/tmp/repo", "<disposable>"],
  ]);
  assert.equal(failure, "failed at <home><disposable> with secret details");
  assert.ok(failure.length <= 400);
  assert.equal(sanitizeError(new Error("\n\t"), []), "smoke failed");
});

test("sanitizes stderr to the receipt contract", () => {
  assert.equal(sanitizedStderr("\n"), "");
  assert.equal(sanitizedStderr("credential=secret\n"), "unexpected stderr");
});
