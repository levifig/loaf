import test from "node:test";
import assert from "node:assert/strict";
import { delimiter, join } from "node:path";
import { buildCandidate, classifyCursorPreflight } from "./preflight-cursor-agent-context.mjs";

test("candidate build uses native Go and isolated non-linking host runtime", () => {
  const calls = [];
  const env = buildCandidate("/isolated/loaf.sqlite", (...args) => { calls.push(args); return { status: 0 }; });
  assert.deepEqual(calls.map(([command, args]) => [command, args]), [["go", ["run", "./cmd/loafdev", "build-go"]], ["loaf", ["build", "--target", "cursor"]]]);
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

test("classifies an unsafe Cursor preflight without claiming execution", () => {
  const preflight = classifyCursorPreflight({ status: 0, stdout: "2026.09.02-c22c1a3\n" }, "Usage: agent\n");
  assert.equal(preflight.observedVersion, "2026.09.02-c22c1a3");
  assert.equal(preflight.noSessionPersistence, false);
  assert.equal(preflight.smokeExecuted, false);
  assert.match(preflight.blocker, /no-session-persistence/);
});

test("an unknown version does not mask capability observations or claim a smoke", () => {
  const preflight = classifyCursorPreflight({ status: 127 }, "--no-session-persistence\n");
  assert.equal(preflight.observedVersion, "unknown");
  assert.equal(preflight.noSessionPersistence, true);
  assert.equal(preflight.smokeExecuted, false);
  assert.match(preflight.blocker, /smoke implementation is not enabled/);
  assert.match(classifyCursorPreflight({ status: 0, stdout: "unfamiliar" }, "").blocker, /does not expose --no-session-persistence/);
});
