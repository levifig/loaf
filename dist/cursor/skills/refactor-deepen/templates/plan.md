# Deepening Proposal Template

Use this structure for analysis in the conversation or an explicitly saved temporary report. It defines no work identity, status, or lifecycle. The native tracker owns actionable work; the proposal supplies evidence. Existing PLAN files remain historical records.

Include the scope and source evidence before the five sections below. When saving, follow the owning skill's Analysis Output rules and use `.agents/reports/<YYYYMMDDHHMMSS>-<descriptive-slug>.md`. Do not infer implementation authority from the report.

## Required Sections

The five sections below are the *minimal shape* for a deepening proposal. Order is load-bearing: each section assumes the previous one is settled. Do not reorder.

1. **Candidate** — what's being considered for deepening. Name the **module**
   verbatim (use the canonical glossary term — run `loaf kb glossary check`
   before writing this section). One paragraph; describe the current shape and
   why it reads as shallow (interface surface ≈ implementation surface,
   leaky callers, scattered locality, etc.).

2. **Dependency Category** — exactly one of the four categories from
   [references/deepening.md](../references/deepening.md):
   - `in-process` — code inside the same runtime, no external surface
   - `local-substitutable` — local dependency with a swappable implementation
   - `ports-and-adapters` — true seam with multiple realizable adapters
   - `true-external` — dependency on a system Loaf does not own

   The category determines the seam discipline that applies. State the
   category explicitly, then briefly justify the classification (one or two
   sentences). Proposals that span categories should be split.

3. **Proposed Deepened Module** — the new shape. Describe the **interface**
   the module would expose, what its **implementation** would hide, and where
   the **seam** would land. This is the section the parallel
   INTERFACE-DESIGN sub-agents feed; if their three designs converged on one,
   record that one and reference the rejected variants in *Rejected
   Alternatives*. Use the eight-term vocabulary verbatim (Module, Interface,
   Implementation, Depth, Seam, Adapter, Leverage, Locality).

4. **What Survives in Tests** — which test characteristics the deepening
   preserves and which it invalidates. Be concrete: name the tests (or test
   types) that keep passing across the seam unchanged, and the tests that
   must move, be rewritten, or be deleted. This section exists because the
   deletion test (see `references/language.md`) only earns its keep when the
   test surface survives the deepening — if every test breaks, the seam is
   the wrong one.

5. **Rejected Alternatives** — designs explored and ruled out. Include the
   two non-chosen INTERFACE-DESIGN sub-agent outputs, plus any other shape
   the grilling loop surfaced and discarded. Each entry: a one-line
   description and a one-paragraph rationale for rejection. "Rejected
   because the user picked another" is not a rationale — name the load-bearing
   tradeoff.
