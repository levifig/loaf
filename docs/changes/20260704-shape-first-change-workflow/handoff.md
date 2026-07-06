# Handoff — shape-first-change-workflow

In-flight baton (Decision 18): the state of this Change for the next agent or
session on this branch. Ephemeral by rule — deleted at ship.

## Where this stands (2026-07-06)

- The contract is complete: 21 decisions with provenance, externally reviewed
  three times — Codex rounds 1–2 and the mattpocock/skills adversarial round
  (summaries live as PR #91 comments, per Decision 15).
- U1 (pilot) and U2 (change + PR templates, distributed to all targets) are
  done. **U3 (`loaf change init/check`) is the only open unit and the merge
  gate** — V1–V3 in change.md define its contract; implement Go-side under
  `internal/cli/` with V1–V3 written as tests first.
- PR #91 is a **draft**. Flipping to ready is the Decision 12 declaration;
  pending after the next review round.

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

1. External review round 3 on the PR. Suggested challenge focus: Decision 13's
   free-form H3s (checkable enough?), Decision 14's identity-mismatch
   strictness, Decision 17's harvest mechanics, Decision 19's delta-merge.
2. Apply surviving findings; flip #91 to ready.
3. Implement U3 on this branch (delegate to an implementer agent; TDD).

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
