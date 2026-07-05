---
change: shape-first-change-workflow
created: 2026-07-04
branch: shape-first-change-workflow
---

# Shape-First Hybrid Change Model

## Problem

Loaf's numbered spec/task lifecycle has become heavier than the work it organizes:

- Sequential IDs (`SPEC-NNN`, `TASK-NNN`) require allocation machinery and created
  the cross-worktree collision class that forced SQLite centralization in the
  first place.
- Status lifecycles (SPEC-049 vocabulary) demand coherence between SQLite,
  rendered markdown, and Linear — the render-drift gates exist only to police a
  dual record.
- Task entities duplicate judgment: the agent that decomposes the work is the
  agent that implements it.
- "Done" is an asserted flag, never derived evidence.
- Collaborative shaping has no natural review surface; specs are reviewed
  bespoke rather than PR-native.
- The session entity was removed (SPEC-056) for exactly these reasons. The
  spec/task layer is the same disease one layer up.

Meanwhile the shipped product is still spec-first: README and the shape skill
describe `/shape → SPEC`, and no `loaf change` CLI surface exists yet.

## Hypothesis

A shape-first hybrid model makes each store own what its nature demands:

- **Git owns branch-local work truth**: Change documents, plans, implementation
  units, PR shaping, commits, and review.
- **SQLite owns cross-work process truth**: sparks, ideas, brainstorms, journal
  continuity, source provenance, reports, knowledge routing, and publish
  mappings.
- **External trackers receive projections** of Changes — never a mirrored
  task/status system.

The user-facing verb is `shape`: turn messy input — a journal entry, spark,
idea, brainstorm, Linear issue, review finding, PR conversation, or plain
conversation — into a bounded Change. A Change is temporary, reviewable work
context. Durable artifacts (specs, ADRs, knowledge docs, schemas, changesets,
release notes) are written **after** the work proves what is now true: a final
spec describes reality, not a plan.

If this holds, Loaf keeps its differentiators (intake, journal continuity,
enforced gates) while shedding the dual-record coherence tax — ending stronger
than both its current model and compound-engineering's pure-git model.

## Background

- The project journal is the durable continuity layer; session entities were
  removed (SPEC-056) because lifecycle state cost more than it returned.
- `docs/changes/20260704-worktree-storage-bootstrap/` already piloted the
  branch-local Change artifact successfully.
- The `ce-loaf-analysis` session compared Loaf with compound-engineering and ran
  four Opus-class audits; its executive report recommended this hybrid model,
  found Loaf already mid-pivot post-SPEC-056, and identified verification
  contracts plus gates-derived done as Loaf's strongest differentiator.
- CE usefully collapses brainstorm/requirements/plan/units into one artifact
  moving by document **completeness**, not progress status. Loaf keeps that
  distinction but relocates the signal into PR state and document structure —
  no field at all — while rejecting CE's lack of intake, continuity, and
  enforcement.
- Skills are the portable knowledge layer but are weak at invariants; the CLI is
  better at deterministic, repeatable operations.
- Prior audits found unused lifecycle verbs rot. New machinery exists only where
  an actual ceremony exercises it: **if a command or state cannot name the
  ceremony that uses it, do not build it yet.**
- Older `CR-*`/`CR-000` notes are historical; this direction supersedes numeric
  CR IDs with slug/dated change folders.

## Scope

**In**

- The Change artifact contract: one `change.md` per `docs/changes/YYYYMMDD-slug/`,
  growing toward executability.
- A change template extracted from this pilot, plus the Change-aware PR
  template as distributable content (the PR is the shaping surface, so its
  template is part of the contract; Loaf's `.github/` copy is the installed
  instance).
- A deliberately small `loaf change` CLI: `init` and `check` (carrying the
  structural gate). `archive` waits until a completed Change gives it a ceremony.
- This pilot itself — shaped, branched, and reviewed through the model.

**Out** (deferred to follow-up Changes — see Follow-ups)

- The shape skill rewrite (absorbing breakdown) — its own Change.
- The conversion pass and guidance sweep — their own Change; conversion strictly
  before sweep so no in-flight work is stranded.
- The broad skill-surface tightening — its own Change, the model's second
  dogfood.
- The spike harness and the general Verification Contract format (SPEC-017
  binary-R revival) — after this pilot proves V1/V2.
- Linear publish mechanics beyond the model definition (SPEC-039 direction).
- Changesets integration build-out (no `.changeset/` exists yet; release
  guidance reconciliation pending).

**Cut** (explicitly rejected)

- Numeric CR IDs (`CR-000`) — superseded by slug/date folders.
- A persistent task entity for implementation units.
- Bidirectional tracker sync.
- A Change lifecycle state machine.

## Observable Workflow

```text
SQLite journal / sparks / ideas / brainstorms / Linear / current conversation
        |
        v
/shape   (gather context, interview, compare alternatives, bound scope,
        |  critique gate, choose smallest useful artifact)
        v
docs/changes/YYYYMMDD-slug/
  change.md        canonical Change artifact, grows toward executability
  notes.md         optional working notes, only when useful
  reviews/         optional temporary review packets
        |
        v
draft PR (default at shaping)  →  spike  →  implementation  →  review  →  merge
        |
        v
durable docs:                          post-merge housekeeping:
  docs/specs/<feature>/...               change folder →
  docs/decisions/ADR-...                 docs/changes/archive/YYYYMMDD-slug/
  docs/knowledge/...
  docs/schema/...
  .changeset/...
```

`shape` means: gather context, interview the user, compare alternatives, define
boundaries, identify risks, produce a Product Contract, and optionally mature it
into an implementation-ready Change. The value is shaping ambiguous input, not
artifact creation — `change create` is not the product, and source-specific
input modes (`from idea`, `from journal`, `from Linear`) are conversational, not
CLI surface.

### Readiness (derived, never declared)

Readiness distinguishes **document completeness from implementation progress**
— but it is not a field. It is read from two places that cannot drift:

- **The PR state.** A draft PR *is* shaping — drafting inherently means the
  Change is still being formed. Marking the PR ready for review *is* the
  implementation-ready declaration.
- **Document structure.** `loaf change check` derives executability from the
  required sections themselves: a Change is implementation-ready when its
  planning contract, implementation units, verification contract, and
  definition of done are present and non-empty.

There is no `readiness:` frontmatter to update, police, or let go stale — and
no field for progress words to creep into. `loaf change check` treats any
status-like frontmatter (`readiness`, `status`, `state`) as a violation (see
Verification Contract, V1). Progress remains derived from Git, PR review, and
gates.

## Decisions

Provenance: decisions 1–8 accepted in the 2026-07-05 interactive interview
against the `ce-loaf-analysis` executive report; 9–10 accepted in follow-up
review the same day; 11 accepted after external (Codex) review; 12 accepted
from user direction during dogfooding; 13–14 accepted after a worked-examples
comparison in the same review cycle.

1. **One `change.md`, canonical, dogfooded now.** This document is the pilot of
   its own contract. `notes.md` and `reviews/` remain optional escape hatches.
2. **Archive over delete.** Post-merge housekeeping moves completed change
   folders to `docs/changes/archive/`. Rationale: squash merges collapse branch
   history — deleting the folder pre-merge means the change doc never lands on
   main; deleting post-merge strands it in commit archaeology. Archive keeps it
   retrievable; the temporary-vs-durable line holds because durable knowledge
   still lands in specs/ADRs/knowledge.
3. **Draft PR opens at shaping, by default.** Shaping that produces a Change
   pushes the branch and opens a draft PR (configurable/suggested, solo
   opt-out). `gh pr list` becomes the in-flight change index — the replacement
   for `loaf spec list` and the answer to cross-branch visibility.
4. **Coexistence by conversion.** Genuinely in-flight specs convert to Changes
   by hand (inventory pass required; SPEC-055 and its queued tasks are the first
   candidate); everything else freezes to archive. `loaf spec`/`loaf task`
   surfaces are removed only after conversion completes — no stranded work.
5. **Breakdown folds into shape.** Implementation-unit decomposition is a
   shaping step toward `implementation-ready`. The breakdown skill retires in
   the guidance sweep.
6. **CLI namespace is `loaf change`.** The CLI is noun-oriented (`loaf journal`,
   `loaf kb`); skills are verbs (`/shape`). The split already exists in Loaf's
   product grammar.
7. **The first gate ships with the pilot.** `loaf change check` rejects progress
   words in `readiness` — this change's own Verification Contract criterion V1.
   Gates-derived done is proven inside the pilot, not deferred (SPEC-056's
   convergence-test pattern, applied at birth).
8. **Skill-surface tightening is a related change.** The audit (every skill
   useful, trim unnecessary instructions, kill shadow/never-used skills, no
   silent failures) is shaped as its own change once this template settles.
9. **Change indexing follows the `docs_index` pattern, never `artifact_bodies`.**
   `change.md` is git-canonical; SQLite may hold a derived index (à la
   `docs_index`/`docs_search`, migration 0008) but never a body copy. This
   forecloses recreating the body/render drift class this model exists to
   eliminate. Consequently, SQLite-canonical `loaf plan` prose is retired for
   change-local work: the plan is a section of `change.md`.
10. **Changes cite journal entries by ID.** Journal entries are append-only with
    stable IDs; citations cannot go stale. The same rule extends to
    SQLite-backed reports: cite the report identity, not its render path.
11. **Scope trimmed to core.** This Change ships the artifact contract, the
    template, and `loaf change init/check` with the pilot gate. The shape-skill
    rewrite, conversion pass, guidance sweep, skill tightening, and spike
    harness are follow-up Changes (see Follow-ups) — a smaller reviewable
    pilot, and more Changes shaped through the model.
12. **The readiness field is dropped; readiness is derived.** Drafting a PR
    inherently means shaping — the draft→ready flip is the human
    implementation-ready declaration, and `loaf change check` derives
    structural executability from the required sections. This supersedes the
    *mechanism* in Decision 7 and the original readiness vocabulary while
    keeping their intent: the pilot gate now bans status-like frontmatter
    outright and computes completeness instead of policing a declared value.
    Decision 3 (draft PR at shaping) carries the signal.
13. **Section contract: flat product, contained planning, fixed tail.** The
    Product Contract is the flat opening narrative (Problem, Hypothesis,
    Scope, Observable Workflow, Rabbit Holes and No-Gos, optional Success
    Metrics); the Planning Contract is one container section holding the HOW
    as free-form `###` subsections named by the work; Implementation Units,
    Verification Contract, and Definition of Done are fixed-name tail
    sections. `check` needs only the product set, the container, and three
    tail names — subsection naming inside the container stays free.
14. **`loaf change check` is two-tier with an opt-in gate.** Violations always
    fail: status-like frontmatter, malformed `YYYYMMDD-slug` naming, identity
    mismatch (`change:` vs folder slug, `created:` vs folder date), missing
    Product Contract sections. Executability is derived and reported, never
    failed by default — a shaping-stage Change is valid. `--require-executable`
    turns the report into a gate, for CI on non-draft PRs and implement
    preflight. Output follows the `loaf check` findings shape; required-section
    lists are hardcoded until a real ceremony demands configurability.

## Rabbit Holes and No-Gos

**No-Gos**

- No Change lifecycle state machine.
- No IDs — no `CR-000`, no numbered `SPEC-*` for this direction.
- No new task entity for implementation units.
- No tracker required for local operation; no default task-level sync; no
  bidirectional status sync in v1.
- ADRs are not a planning prerequisite.
- The initial CLI sketch is not final product surface.

**Rabbit holes to avoid**

- Any status-like field creeping back onto the Change — V1 bans the field
  class outright; completeness lives in document structure and PR state.
- SQLite change indexing drifting into body storage — pinned shut by Decision 9.
- Spike overbuild — the spike's stop condition is "enough unknowns discovered to
  revise the Change," never "it works." The discard default is the guarantee.
- Source-specific CLI commands before dogfooding proves them common,
  deterministic, and worth promoting.
- The guidance sweep landing before the conversion pass — stranding in-flight
  spec work in a model whose documentation no longer exists (Decision 4 orders
  them; both live in a follow-up Change).

## Planning Contract

### Section contract

- **Product Contract** — the flat opening sections: Problem, Hypothesis,
  Scope, Observable Workflow, Rabbit Holes and No-Gos, plus Success Metrics
  when validation matters. Product truth reads as narrative; no container.
- **Planning Contract** — this container section, holding the HOW as
  free-form `###` subsections named by the work (approach, placement,
  boundaries, risks, sequencing, spike findings). The container is the
  contract; its subsection names are not.
- **Executable tail** — Implementation Units, Verification Contract, and
  Definition of Done, followed by Durable Outputs. `check` derives
  executability from the container and the tail, so those names are fixed.
- **Decisions** — a cross-cutting provenance log serving both contracts.

### Artifact placement

- **Why `docs/changes/` and not `.agents/changes/`:** ADR-013 deliberately
  routes `.agents/` to the main worktree; Changes are deliberately **branch
  content** and must live where branch context lives. This is the one point
  where the new model intersects that rule, and this placement is the deliberate
  resolution, not an accident.
- Branch names use the human slug only; the folder carries the date:

```text
branch: shape-first-change-workflow
folder: docs/changes/20260704-shape-first-change-workflow/
```

- A small Change maps to one branch and one PR. A larger Change may span
  multiple PRs; the Change remains the shared context and acceptance contract,
  each PR the reviewable implementation slice.

### Skill / CLI boundary

The shape skill owns judgment: collecting context from conversation, journal,
ideas, brainstorms, reports, or Linear; interviewing for ambiguity, constraints,
hidden complexity, and non-goals; running the critique gate; choosing the
smallest useful artifact (none, Product Contract only, implementation-ready
Change, or a post-implementation durable-spec proposal); teaching the harness
when to call the CLI and how to read its output; preserving the line between
interactive shaping and autonomous execution.

The CLI owns invariants: validated `YYYYMMDD-slug` folder init; required-file
and heading checks; the status-like frontmatter ban; branch/change mismatch
detection; listing branch-local Changes; archival moves; machine-readable
check output; verification-gate wiring; generated references; materializing
the PR template into a consumer repo's `.github/` when absent. CLI help must
remain legible to humans, not just agents.

Initial surface, deliberately small:

```bash
loaf change init <slug>
loaf change check
```

`archive` joins this surface when the first completed Change gives post-merge
housekeeping its ceremony (Decision 2) — a command that cannot yet name its
ceremony is not built yet.

### Linear working model

Linear represents the Change, not implementation units:

```text
Project or Initiative: product or implementation family
  Issue: Change: loaf/shape-first-change-workflow
    Sub-issues: only when parallel ownership or team coordination requires
```

Default federation: one Change → one issue, body published as a snapshot,
resulting URL recorded in the publish ledger (`backend_mappings`). No
bidirectional status sync, no sub-issue generation, until dogfooding proves the
need.

### Changesets and releases

Changesets are release input; Change folders are planning and implementation
context. This keeps release conflicts low and versioning mechanics out of both
`CHANGELOG.md` and the Change folder. Not yet implemented — no `.changeset/`
directory exists, and release guidance needs reconciliation.

### Spike step

`spike`, not `dry-run` (`dry-run` already means preview across the Loaf CLI). A
spike is a disposable attempt to learn whether the Change is executable: run in
an isolated worktree, stop when enough unknowns are discovered, discard
implementation output by default, and write findings back into the Change's
planning and verification sections before real implementation begins.

## Implementation Units

In-document work packets — commit-boundary guides and review anchors, not
tracked entities.

- **U1 — Pilot restructure.** This document, restructured to its own contract
  (done in shaping; you are reading the result).
- **U2 — Templates.** Extract a `docs/changes` change template from this
  file's structure, and make the Change-aware PR template distributable
  content under the shape skill (`content/skills/shape/templates/`) — Loaf's
  own `.github/` copy is the installed instance; materializing it into a
  consumer repo's `.github/` when absent belongs to the CLI (`change init`
  or bootstrap).
- **U3 — `loaf change init` and `loaf change check`.** `check` carries V1
  (readiness vocabulary) and V2 (structure) from day one, with machine-readable
  output.

Everything beyond these three units was deliberately spun out (Decision 11) —
see Follow-ups.

## Verification Contract

Executable (machine-checkable):

- **V1.** `loaf change check` exits non-zero on violations: status-like
  frontmatter (`readiness`, `status`, `state`, or progress vocabulary in any
  frontmatter field — the field class is banned, not policed), malformed
  `YYYYMMDD-slug` folder naming, identity mismatch (`change:` vs folder slug,
  `created:` vs folder date), or missing Product Contract sections. *(the
  pilot gate)*
- **V2.** `loaf change check` reports derived executability — Planning
  Contract, Implementation Units, Verification Contract, and Definition of
  Done present and non-empty — without failing a shaping-stage document, in
  machine-readable output following the `loaf check` findings shape.
- **V3.** `loaf change check --require-executable` exits non-zero when the
  document is not executable — the CI gate for non-draft PRs and the
  implement-skill preflight.

Human review:

- **H1.** No status or readiness field exists on a Change; completeness reads
  from document structure and the PR's draft/ready state, and done is derived
  from Git state, PR review, and gates — never a mutable flag.
- **H2.** Cross-change context stays in SQLite/journal surfaces, not duplicated
  across Git folders.
- **H3.** A subsequent Change is shaped from the template without friction —
  the template earns its keep in use, not in review.

The convergence check formerly listed here (no `content/` instruction mints a
numbered `SPEC-*`) moves to the guidance-sweep follow-up Change, which owns the
surface it verifies. The shape-skill criteria (produces a Change without a
spec; explains the CLI boundary) move to the shape-skill-rewrite Change.

## Definition of Done

- The change and PR templates ship as distributable content;
  `loaf change init/check` ships; V1–V3 pass.
- This pilot's PR is reviewed and merged with the change folder on main.
- At least one follow-up Change is shaped from the template (H3) — proof the
  contract works beyond its own pilot.
- H1–H2 confirmed in review.
- Durable outputs (below) landed, or explicitly deferred at merge time.

Follow-up Changes have their own Definitions of Done — this Change does not
gate on them (they are successors, not dependencies).

## Durable Outputs

To create or update after implementation proves the model:

- **ARCHITECTURE.md** — artifact-model section rewritten: the git/SQLite split,
  Changes vs. specs repositioning.
- **ADR** — the shape-first hybrid Change model; annotates ADR-016's trichotomy
  where it concerns work artifacts, and records the Decision 9 indexing pin.
- **Spec repositioning** — specs become post-implementation behavior contracts
  under `docs/specs/<feature>/`: what is now true, not what was planned. ADRs
  stay sparse and retroactive. Reports stay temporary unless the evidence
  snapshot deserves citation.
- **Updated skills and docs** — shape (rewritten), breakdown (retired),
  `cli-reference` (`loaf change`), README.

## Critique Gate

Before implementation, explicitly challenge:

- Is the CLI surface still too large for what we know today?
- Is the skill doing deterministic work that belongs in the CLI, or the CLI
  claiming judgment that belongs in the skill?
- Is the Product Contract enough for this Change, or are Planning Contract
  sections genuinely needed first?
- Are we reintroducing progress states under the word "readiness"? (V1 should
  make this impossible — verify it actually does.)
- What can be deleted from existing skills once the CLI owns the invariant?
- What ceremony exercises every new lifecycle verb? If a command or state
  cannot name its ceremony, do not build it.
- Which acceptance criteria became executable gates, and which remain human
  review — and is that split still right?

## Follow-ups

Spun out of this Change by Decision 11, each to be shaped as its own Change —
more of the model's surface gets dogfooded by construction:

- **shape-skill-rewrite** — rewrite `/shape` around the hybrid model, absorb
  breakdown (Decision 5), teach the CLI boundary, default draft-PR-at-shaping
  (Decision 3). Owns the shape-produces-Change-without-spec and
  explains-the-CLI verification criteria.
- **spec-conversion-and-guidance-sweep** — inventory the 24 active specs,
  convert genuinely in-flight ones (SPEC-055 first), freeze the rest; then
  sweep README and skills so nothing implies numbered specs. Conversion
  strictly before sweep (Decision 4). Owns the convergence check.
- **skill-surface-tightening** — every skill useful, trim unnecessary
  instructions, kill shadow/never-used skills and silent failures (Decision 8).
- **spike-harness** — worktree provisioning, discard guarantee, writeback loop,
  plus the general Verification Contract format (SPEC-017 binary-R revival,
  detached from its task model).
- **`loaf change archive`** — built when the first completed Change gives
  post-merge housekeeping its ceremony (Decision 2).

## Open Questions

- SPEC-017 binary-R revival details — format only, no task model import.
- Smallest useful spike harness (worktree provisioning: harness-native
  `EnterWorktree` vs. a `git worktree add` wrapper).
- Finalize mechanics for proposing durable outputs: manual `/reflect`, an
  automatic nudge, or an explicit `loaf change finalize`.
- Release grouping language ("change bundle"?) and the smallest changesets
  integration that replaces conflict-prone changelog input.
- Conversion inventory specifics: which of the 24 active specs convert vs.
  freeze (owned by the conversion follow-up Change).

## Source Inputs

- The `ce-loaf-analysis` working session (2026-07-04): the compound-engineering
  comparison, four deep codebase audits, and the executive report recommending
  the hybrid Change model, spike step, one-way federation, and gates-derived
  done — plus the 2026-07-05 interactive review that produced the Decisions
  section.
- `docs/changes/20260704-worktree-storage-bootstrap/plan.md`, the first
  branch-local Change pilot.
- Loaf report `report-codex-handoff-journal-first-audit` (SQLite-backed; its
  markdown render is intentionally uncommitted — cited by report identity per
  Decision 10) — drift, lifecycle vocabulary, capture closure, and unused-verb
  lessons.
- External (Codex) review of this Change, 2026-07-05 — branch integrity, source
  integrity, scope trim, and decision provenance.
- Loaf memory notes on brief-first, change-first, and `/ship` versus `/release`.
- `docs/STRATEGY.md` and `docs/ARCHITECTURE.md` — skills describe what to do;
  the CLI executes deterministic behavior.
- Prior `CR-*`/`/change` notes, historical context superseded by the no-ID,
  shape-verb direction.
- Gokul Rajaram's Product Spec post and sample: specs legible to humans and
  executable by agents — Problem, Hypothesis, Scope, User Experience,
  Acceptance Criteria/Evals, Success Metrics.
- NotebookLM notebook "Agile Product Management: From Shaping to
  Implementation" — Shape Up, GitHub Spec Kit, Linear, RAC/Lore, Compound
  Engineering, AI-agent spec writing.
