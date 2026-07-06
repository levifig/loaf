# Handoff — shape-first-change-workflow

In-flight baton (Decision 18): the state of this Change for the next agent or
session on this branch. Ephemeral by rule — deleted at ship.

## Where this stands (2026-07-06)

- The contract is complete: 21 decisions with provenance, externally reviewed
  three times — Codex rounds 1–2 and the mattpocock/skills adversarial round
  (summaries live as PR #91 comments, per Decision 15).
- U1 (pilot) and U2 (change + PR templates, distributed to all targets) are
  done. **U3 (`loaf change init/check`) is implemented, pending review.**
  Native Go under `internal/cli/change.go` with V1–V3 written as tests first
  (`internal/cli/change_test.go`); the template is embedded and drift-gated
  byte-identical to `content/skills/shape/templates/change.md`. `go test ./...`,
  `gofmt`, and `CGO_ENABLED=0 go build ./...` pass; the pilot itself checks
  clean (no violations, executable: yes). Commit `938e9d3c`.
- PR #91 is a **draft**. Flipping to ready is the Decision 12 declaration;
  pending after review of U3.

## In flight (2026-07-06)

- The mattpocock/skills adversarial review is **complete and adjudicated**:
  Decision 21 (eight sub-amendments), raw reports in
  `research/mattpocock-review/`, round summary on PR #91. Contract now stands
  at 21 decisions; the draft PR is opt-in (21a), harvest mechanics belong to
  the sweep follow-up (21b), and a new beyond-one-context follow-up exists
  (21g).
- The originating session's executive report (CE-vs-Loaf comparison) exists
  only in that session's JSONL — recoverable if needed; its substance is
  absorbed into change.md and research/external-inputs.md.

## Next actions, in order

1. Review U3 (`internal/cli/change.go` + `change_test.go`, commit `938e9d3c`).
   Focus: the finding below (fresh-init reads executable:yes), exit-code and
   JSON-shape conventions, folder-resolution edge cases.
2. External review round 3 on the PR. Suggested challenge focus: Decision 13's
   free-form H3s (checkable enough?), Decision 14's identity-mismatch
   strictness, Decision 17's harvest mechanics, Decision 19's delta-merge.
3. Apply surviving findings; push the branch (confirm first) and flip #91 to
   ready — the Decision 12 implementation-ready declaration. (Push and the
   draft→ready flip are the orchestrator's calls, not the implementer's.)

## Open finding from U3 (for spec review)

A freshly `init`'d Change reads `executable: yes` because the shipped template
carries all tail sections (Planning Contract, Implementation Units,
Verification Contract, Definition of Done) with placeholder prose — content
under the heading, which V2 counts as non-empty. This is literal to the
Verification Contract ("present and non-empty") and the shaping-stage case
(product sections only) still reads non-executable. If a fresh Change should
read non-executable until authored, either V2 must discount bracket-placeholder
content or the template must ship the tail empty/omitted. Left as-is because the
draft PR state is the real readiness signal (D12/D21a) and stripping
placeholders was not in the U3 contract.

## Warnings

- **The global `loaf` binary is stale relative to this branch.** `loaf build`
  regenerates pre-#89 hook artifacts (dropped `--advisory` flags, mangled
  ephemeral-provenance command) — it clobbered two files once, restored in
  `348c56d4`. Rebuild from source before trusting build outputs, and never
  bulk-add `dist/`/`plugins/` without reading the diff.
- The untracked `.agents/reports/*` renders on this branch are intentional
  (SQLite-backed reports; cited by identity per Decision 10).

## Pointers

- Contract: `change.md` — Decisions 1–21, Verification Contract V1–V3.
- Evidence: `research/` — the decision aid that settled D13/D14, and the
  external-inputs evaluation feeding D17–D20.
- Review history: PR #91 round-summary comments.
