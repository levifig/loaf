---
description: >-
  Integrates learnings from shipped work into strategic documents. Use after
  completing significant work or when the user asks "what did we learn?"
  Proposes evidence-backed updates to strategic documents and existing
  architecture topics, or concludes no update is needed. Not for
  pre-implementation strategy (use strategy) or choosing and recording an
  architectural commitment (use architecture).
subtask: false
version: 0.5.0
---

# Reflect

Integrate learning from shipped work into the documents that own it. Reflection is not a document-production quota or an automatic decision-record ceremony.

## Contents
- Critical Rules
- Verification
- Quick Reference
- Topics
- Purpose
- When to Reflect
- Process
- Related Skills

**Input:** $ARGUMENTS

---

## Critical Rules

- **Evidence-based** -- separate observed results, inferred lessons, and proposed changes; a merged patch is not proof of successful use or an accepted architectural direction
- **Dogfood before generalization** -- compare the rideable increment's real use and learning sought with what actually happened before proposing more breadth or abstraction
- **Post-implementation only** -- reflect after shipping, not before or during planning
- **Respect scope** -- review and propose by default. Apply updates only with explicit user authorization, including an already explicit request to apply scoped changes; do not ask again for that same permission. This covers architecture topics as well as vision, strategy, and the overview
- **Consolidate** -- batch related learnings into coherent updates, avoid micro-updates
- **Architecture ownership** -- read the [architecture skill](../architecture/SKILL.md) for topic maintenance, rationale, affected-surface review, and exceptional ADR selection. Integrate learning into the owning topic by default; route a candidate consequential choice to Architecture with evidence and why a separate record might help, not an automatic ADR
- **No silent acceptance of drift** -- when shipped behaviour conflicts with an agreed constraint, report the discrepancy and route the unresolved choice to Architecture. Do not rewrite the constraint to bless the implementation or change an ADR's original decision context
- **No forced update** -- retain existing guidance when the evidence does not justify change; useful local observations may remain in code, tests, or the journal
- **Log first** -- log invocation before gathering evidence: `loaf journal log "skill(reflect): <scope>"`
- **Link back** -- always reference the native tracker records, journal entries, reports, and commits that informed each update
- **Log updates** -- log each strategic document update to the project journal: `loaf journal log "decision(scope): updated STRATEGY.md with learning"`

## Verification

- Proposals cite specific native tracker records, journal entries, reports, or commits as evidence
- Reflection states whether the named rider completed the journey, what dogfood taught, which complexity proved necessary, and which deferrals should remain deferred
- Current owning documents and relevant ADRs were read before proposing edits; no strategic or architecture document was modified beyond explicit authorization
- Observations, inferences, agreed direction, and implementation gaps are distinct; unresolved architectural choices remain unresolved
- Topics and ADRs do not duplicate rationale, no ADR was automatically created, and no migration or retirement was inferred from reflection
- Any tracker mutation requested during reflection uses the selected provider skill and is verified by authoritative readback

## Quick Reference

| Learning Type | Document |
|---------------|----------|
| User behavior / market / problem understanding | STRATEGY.md |
| Direction changes | VISION.md |
| Technical constraints / patterns / decision updates | Owning architecture topic or overview, following the architecture skill |
| Potential exceptional ADR or conflict with an agreed constraint | Architecture evaluates the choice and record; reflection supplies evidence |
| Workflow learning / local implementation insight | Owning skill or nearby code/tests/journal; do not force it into architecture |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Update Proposal | [templates/update-proposal.md](templates/update-proposal.md) | Drafting proposals for strategic doc changes |
| Architecture | [architecture skill](../architecture/SKILL.md) | Integrating architectural learning or evaluating a potential exceptional ADR |

---

## Purpose

Strategy evolves through **shipping**, not theorizing.

After completing work, reflect extracts learnings and proposes updates to strategic documents. **Don't update strategy during planning or shaping.** Update after implementation proves (or disproves) assumptions.

---

## When to Reflect

- After completing and shipping a significant native tracker work item
- After shipped work supplies evidence about a surprise encountered during implementation
- After a series of related sessions
- Periodically (monthly/quarterly) to consolidate learnings

---

## Process

### Step 1: Parse Input

`$ARGUMENTS` can be: an issue ref (`LOAF-42`), a topic ("authentication learnings"), or empty (general reflection on recent work).

### Step 2: Gather Evidence

Discover the repository's actual vision, strategy, architecture overview, and topic homes. Read the affected current documents and applicable ADRs before drafting; do not assume every project needs the same files.

Sources:
1. **Completed tracker work** — use the selected `project-management/v1` provider skill and harness-native connection to read completed canonical records and their bodies
2. **Project journal** (`loaf journal recent --json`, `loaf journal search <topic>`) -- insights, surprises, pivots
3. **Recent commits** (`git log --oneline -30`)
4. **Implementation reality** -- what was harder/easier than expected? What assumptions were wrong?
5. **Rideable-increment evidence** -- did the real rider complete the journey, did the safety/integrity proof hold, what was learned, and did any unused machinery slip in?

### Step 3: Interview for Insights

Use existing evidence first and ask only for missing insights that could change the conclusion: what surprised the user, whether the rider completed the journey, what dogfood showed, which complexity earned its place, and which deferrals should remain. Do not repeat a settled decision interview merely because reflection was invoked.

### Step 4: Identify Implications

Use the Quick Reference routing against the repository's actual document homes. Separate a correction to current implementation evidence from a proposal to change the agreed model. If a choice may merit an ADR, apply Architecture's exceptional-record guidance; significance, project maturity, or a release alone does not select that format.

### Step 5: Draft Proposals

For each justified update, use the [update-proposal template](templates/update-proposal.md) as prompts. Present proposals in the response unless a persistent artifact is requested or necessary. Group related edits by owning topic, preserve relevant rationale, and state remaining evidence gaps. “No durable update warranted” is a valid result.

### Step 6: Present and Await Approval

Present proposals grouped by document. If the user has already authorized those scoped edits, proceed; otherwise wait for approval before applying them. Approval of a topic update is not approval to adopt an unresolved choice, create a separate ADR, or migrate the corpus.

User may: approve all, approve some, modify proposals, defer updates, or request more evidence.

### Step 7: Apply Updates

After approval:
1. Update documents with approved changes
2. Reconcile affected architecture topics and any deliberately selected ADR using Architecture's maintenance and decision-history rules; do not infer a corpus migration or retirement
3. Check affected claims against code and tests, verify links, and report any unverified implementation gaps without claiming a fresh quality gate
4. Announce what changed, what remains proposed, and what required no update

---

## Related Skills

- **shape** -- Notes strategic tensions for later reflection
- **strategy** -- Deep discovery (before reflection validates)
- **architecture** -- Owns living topics, exceptional ADR selection and history, rationale, maintenance, and migration
- **research** -- Investigation that may inform reflection
