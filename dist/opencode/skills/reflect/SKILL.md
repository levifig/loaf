---
name: reflect
description: >-
  Maintains the Core Docs (VISION, STRATEGY, ARCHITECTURE) from evidence, at any
  time. Use when the user asks "what did we learn?" or "what's our strategy?",
  or wants personas, positioning, or direction revised. Proposes evidence-backed
  updates the owner confirms, or concludes none is needed. Not for problem
  discovery on one piece of work (use pitch) or choosing and recording an
  architectural commitment (use architecture).
subtask: false
version: 0.6.0-rc.3
---

# Reflect

Integrate evidence into the documents that own it: learning from shipped work, and strategic discovery or a premise found wrong before any work ships. Reflection is not a document-production quota or an automatic decision-record ceremony.

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

Write journal entries with the command form in the loaf-core skill's Journal Commands section.

- **Evidence-based** -- separate observed results, inferred lessons, and proposed changes; a merged patch is not proof of successful use or an accepted architectural direction
- **Dogfood before generalization** -- compare the rideable increment's real use and learning sought with what actually happened before proposing more breadth or abstraction
- **Any time, both directions** -- propose a VISION or STRATEGY change whenever evidence supports it: before, during, or after implementation, including that a Core Doc premise is wrong
- **Owner confirms every Core Doc update** -- review and propose by default. Apply an update to a Core Doc or architecture topic only after the owner confirms that specific update; an explicit request to apply a scoped change is that confirmation, so do not ask again for it. However strong the evidence, never edit VISION or STRATEGY without it
- **Surface conflicts between Core Docs** -- when STRATEGY and VISION disagree, report the conflict and propose which to change; never edit one silently to fit the other
- **Interview for missing domain knowledge** -- strategy is extracted from the owner and evidence, not guessed; ask about users, competitors, problem space, and domain language when the evidence is silent
- **Name anti-personas** -- a persona change also says who the project is not building for
- **Consolidate** -- batch related learnings into coherent updates, avoid micro-updates
- **Architecture ownership** -- read the [architecture skill](../architecture/SKILL.md) for topic maintenance, rationale, affected-surface review, and exceptional ADR selection. Integrate learning into the owning topic by default; route a candidate consequential choice to Architecture with evidence and why a separate record might help, not an automatic ADR
- **No silent acceptance of drift** -- when shipped behaviour conflicts with an agreed constraint, report the discrepancy and route the unresolved choice to Architecture. Do not rewrite the constraint to bless the implementation or change an ADR's original decision context
- **No forced update** -- retain existing guidance when the evidence does not justify change; useful local observations may remain in code, tests, or the journal
- **Link back** -- always reference the native tracker records, journal entries, reports, and commits that informed each update
- **Log updates** -- log each Core Doc update to the project journal as a `decision(scope): updated STRATEGY.md with learning` entry

## Verification

- Proposals cite specific native tracker records, journal entries, reports, or commits as evidence
- Reflection states whether the named rider completed the journey, what dogfood taught, which complexity proved necessary, and which deferrals should remain deferred
- Current owning documents and relevant ADRs were read before proposing edits; no Core Doc or architecture topic was modified beyond explicit authorization, and the owner confirmed each VISION or STRATEGY update before it was applied
- Any conflict between VISION and STRATEGY is reported with a proposal, not resolved silently
- STRATEGY persona changes include anti-personas, and its sequencing says what a real operator can do after each milestone
- Observations, inferences, agreed direction, and implementation gaps are distinct; unresolved architectural choices remain unresolved
- Topics and ADRs do not duplicate rationale, no ADR was automatically created, and no migration or retirement was inferred from reflection
- Any tracker mutation requested during reflection uses the selected provider skill and is verified by authoritative readback

## Quick Reference

| Learning Type | Document |
|---------------|----------|
| User behavior / market / problem understanding / personas and anti-personas | STRATEGY.md, following the [strategy template](templates/strategy.md) |
| Direction changes | VISION.md |
| Technical constraints / patterns / decision updates | Owning architecture topic or overview, following the architecture skill |
| Potential exceptional ADR or conflict with an agreed constraint | Architecture evaluates the choice and record; reflection supplies evidence |
| Workflow learning / local implementation insight | Owning skill or nearby code/tests/journal; do not force it into architecture |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Update Proposal | [templates/update-proposal.md](templates/update-proposal.md) | Drafting proposals for Core Doc changes |
| Strategy Structure | [templates/strategy.md](templates/strategy.md) | Creating STRATEGY.md or adding personas, problem space, priorities, landscape, or glossary |
| Architecture | [architecture skill](../architecture/SKILL.md) | Integrating architectural learning or evaluating a potential exceptional ADR |

---

## Purpose

The Core Docs follow evidence. Shipped work proves or disproves assumptions; dogfood, interviews, and research can show a premise is wrong before anything ships. Reflect turns either kind of evidence into a proposal for the owning document, and the owner decides.

- **STRATEGY.md** -- the landscape: who we build for and not, the problem space, priorities, and positioning
- **VISION.md** -- where we're going
- **ARCHITECTURE.md** -- how we build, owned with the architecture skill

---

## When to Reflect

- After completing and shipping a significant native tracker work item
- After shipped work supplies evidence about a surprise encountered during implementation
- Before or during implementation, when evidence shows a VISION or STRATEGY premise is wrong
- When the owner asks for strategy discovery: personas, market landscape, problem space, or glossary
- After a series of related sessions
- Periodically (monthly/quarterly) to consolidate learnings

---

## Process

### Step 1: Parse Input

`$ARGUMENTS` can be: an issue ref (`LOAF-42`), a topic ("authentication learnings"), a strategy area ("personas", "market", "problem space", "glossary"), or empty (general reflection on recent work).

### Step 2: Gather Evidence

Discover the repository's actual Core Docs (`docs/VISION.md`, `docs/STRATEGY.md`, `docs/ARCHITECTURE.md`) and architecture topic homes. Read the affected current documents and applicable ADRs before drafting; do not assume every project needs the same files.

Sources:
1. **Completed tracker work** — use the selected `project-management/v1` provider skill and harness-native connection to read completed canonical records and their bodies
2. **Project journal** (`loaf journal recent --json`, `loaf journal search <topic>`) -- insights, surprises, pivots
3. **Recent commits** (`git log --oneline -30`)
4. **Implementation reality** -- what was harder/easier than expected? What assumptions were wrong?
5. **Rideable-increment evidence** -- did the real rider complete the journey, did the safety/integrity proof hold, what was learned, and did any unused machinery slip in?

### Step 3: Interview for Insights

Use existing evidence first and ask only for missing insights that could change the conclusion: what surprised the user, whether the rider completed the journey, what dogfood showed, which complexity earned its place, and which deferrals should remain. Do not repeat a settled decision interview merely because reflection was invoked.

Strategy discovery needs more interviewing, because the knowledge lives with the owner: ask non-obvious questions about users and anti-personas, competitors, the problem space, and domain language, one at a time, using your harness's structured question tool if it has one.

### Step 4: Identify Implications

Use the Quick Reference routing against the repository's actual document homes. Separate a correction to current implementation evidence from a proposal to change the agreed model. If a choice may merit an ADR, apply Architecture's exceptional-record guidance; significance, project maturity, or a release alone does not select that format.

### Step 5: Draft Proposals

For each justified update, use the [update-proposal template](templates/update-proposal.md) as prompts; shape new STRATEGY.md sections with the [strategy template](templates/strategy.md) and express sequencing as rideable increments, not completed architecture layers. Present proposals in the response unless a persistent artifact is requested or necessary. Group related edits by owning topic, preserve relevant rationale, and state remaining evidence gaps. “No durable update warranted” is a valid result.

### Step 6: Present and Await Approval

Present proposals grouped by document. If the owner has already authorized those scoped edits, proceed; otherwise wait for confirmation before applying them. Each VISION or STRATEGY change needs that confirmation, however strong the evidence. Approval of a topic update is not approval to adopt an unresolved choice, create a separate ADR, or migrate the corpus.

User may: approve all, approve some, modify proposals, defer updates, or request more evidence.

### Step 7: Apply Updates

After approval:
1. Update documents with approved changes
2. Reconcile affected architecture topics and any deliberately selected ADR using Architecture's maintenance and decision-history rules; do not infer a corpus migration or retirement
3. Check affected claims against code and tests, verify links, and report any unverified implementation gaps without claiming a fresh quality gate
4. Announce what changed, what remains proposed, and what required no update

---

## Related Skills

- **shape** -- Notes strategic tensions for reflection
- **pitch** -- Problem discovery for one piece of work; a pitch that finds a Core Doc premise wrong hands that evidence here
- **architecture** -- Owns living topics, exceptional ADR selection and history, rationale, maintenance, and migration
- **research** -- Investigation that may inform reflection
