# Handoff — shape-first-change-workflow

In-flight baton (Decision 18): the state of this Change for the next agent or session on this branch. Ephemeral by rule — deleted at ship.

## Where this stands (2026-07-06, post-round-3)

- The contract is complete: **22 decisions with provenance**, externally reviewed four rounds — Codex rounds 1–3 and the mattpocock/skills three-lens adversarial round (all summaries live as PR #91 comments, per Decision 15).
- **All implementation units are done.** U1 (pilot), U2 (change + PR templates, distributed to all targets), U3 (`loaf change init/check`, native Go, V1–V3 as tests first, embedded template drift-gated). The V2 placeholder fix landed (`29d9a6b6`): a fresh init reads `executable: no` — and after the template-tail rework (`6fb0ca09`) it shows **all four** gaps. The pilot itself checks clean: no violations, `executable: yes`, exit 0.
- The package was reflowed to the no-hard-wrap rule (`49e51e4e`, word-diff verified whitespace-only); Decision 22 (the operating model: derived states, cached projections, partitioned/transferable authority, `todo` column, released as project-level join, local-first parity, tracker-native task lists) is pinned (`2a377bb0`).
- PR #91 is a **draft**. Flipping to ready is the Decision 12 declaration; the endgame sequence is agreed and below.

## Next actions, in order

1. **External review round 4** (user-run, in flight). Fresh surfaces since round 3: Decision 22, the reflow, the template-tail rework (four-gap fresh init), the V2 placeholder semantics, U3's full implementation.
2. Apply surviving round-4 findings.
3. **Endgame** (orchestrator's calls): generate the pilot's architectural walkthrough (Delba pattern per Decision 20 — intended-vs-actual against the Planning Contract, V→test map, anchored mermaid) and post it as the flip-accompanying PR comment; delete this file (Decision 18 / Definition of Done); flip #91 to ready; merge.

## Open proposals (not yet pinned)

- The walkthrough wiring rides the sweep follow-up's ship amendments (Decision 20's "explainer guidance") — v1 skill-taught, delivered as a PR comment (mermaid renders natively), promoted to `research/` only if it becomes evidence. The pilot's own walkthrough at flip is the first instance and reference example.

## Warnings

- **The global `loaf` binary is stale relative to this branch.** `loaf build` regenerates pre-#89 hook artifacts — it clobbered two files once, restored in `348c56d4`. Rebuild from source before trusting build outputs; never bulk-add `dist/`/`plugins/` without reading the diff. (A source-built scratch binary is fine: `go build -o /tmp/loaf-scratch ./cmd/loaf`.)
- The untracked `.agents/reports/*` renders on this branch are intentional (SQLite-backed reports; cited by identity per Decision 10).
- Two sparks captured via lane 2 late in shaping: hooks-surface deep re-evaluation (cross-harness focus: Claude Code, Codex, Cursor, OpenCode) and a Pi-shaped harness (separate endeavour). In intake, not this Change's scope.

## Pointers

- Contract: `change.md` — Decisions 1–22, Verification Contract V1–V3.
- Evidence: `research/` — the decision aid that settled D13/D14, the external-inputs evaluation feeding D17–D20, and the three mattpocock-review reports behind Decision 21.
- Review history: PR #91 round-summary comments (rounds 1–3 + U3 landing + finding resolution).
