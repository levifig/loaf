---
name: architecture
description: >-
  Explains, evaluates, and maintains living, topic-based architecture and its
  rationale. Use when choosing system direction, evaluating a rare ADR,
  auditing documentation drift, or migrating an ADR corpus. Produces current
  architecture docs and deliberately selected decision records, not work plans
  or tracker identity.
---

# Architecture

Maintain the current architectural model with memory of why it exists. The default unit of documentation is a topic that evolves. Rare, narrowly scoped ADRs preserve consequential choices when a separate record adds value.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules

### Select the mode and authority

- As the first action, log the invocation: `loaf journal log "skill(architecture): <intent and scope>"`. If unavailable, report that limitation; do not substitute a tracker write.
- **Explain** — answer from relevant code, documents, and applicable decisions. Distinguish historical rationale from current authority. Do not create an artifact unless requested.
- **Audit** — compare architecture documentation with current evidence and report findings by default. Apply changes only when requested. Read [Auditing and Migration](references/auditing-and-migration.md).
- **Decide/update** — clarify unresolved material choices, compare credible alternatives, and record the agreed model in the smallest suitable architecture home. A request to evaluate options is not approval to adopt one.
- **Migrate** — consolidate existing records into topics only when migration is requested. Read [Auditing and Migration](references/auditing-and-migration.md); do not infer a migration from an ordinary explanation or update.

Discover repository instructions, canonical architecture paths, and relevant code before acting. Read applicable existing ADRs as evidence and respect their still-current constraints; their presence alone does not require creating more ADRs. Follow explicit repository conventions without imposing a migration. Use [Decision Records](references/decision-records.md) when evaluating or authoring an exceptional ADR, and [Legacy ADRs](references/legacy-adrs.md) only when interpreting or maintaining Loaf's historical format.

Reuse explicit user choices and delegated authority. Ask only when an unresolved choice materially affects outcome, scope, safety, compatibility, or a public contract. Do not reopen settled decisions through a mandatory interview. Scale independent review to consequential unresolved risk, not routine explanation or editorial repair.

### Maintain topics, not a decision catalogue

The default home is `docs/architecture/<topic>.md`, with `docs/ARCHITECTURE.md` as a concise system map and navigation entry point. Respect an explicitly established equivalent layout. Use descriptive, stable topic slugs, not decision numbers, work IDs, or timestamps. Prefer updating an existing topic; create or split a file only when the subject needs its own explanation. A small project can remain a single overview until it needs deeper topics.

Keep substantive detail in one owning topic and link to it from the overview and related documents. When a decision has a separate ADR, keep its detailed decision rationale there; the topic states the current constraint and links to the record. Do not maintain duplicate explanations in topics, ADRs, and knowledge files, or convert every old ADR into a separately renamed topic.

Document what future work needs to understand or preserve: system boundaries, quality attributes, broad dependencies, interfaces, construction or delivery choices, and their rationale. Significance and documentation depth require judgment, not a Boolean gate, PR-size test, or document quota. Early exploratory projects usually need less formal documentation; maturity can increase the amount of durable context, not the ceremony.

Use [the topic template](templates/architecture-topic.md) as optional writing prompts. Cover what is useful:

- Current model, scope, and applicability.
- Rationale, important tradeoffs, and credible alternatives actually considered.
- Constraints and invariants that changes must preserve or explicitly revise.
- Evidence of implemented behaviour and any agreed but unimplemented direction.
- Links to the relevant code, tests, and technical references.

Do not invent alternatives, historical dates, approval, or evidence to fill a section. A glossary is supporting evidence, not higher authority; flag conflicts with current decisions rather than blocking on glossary maintenance.

### Keep ADRs exceptional, not obsolete

An ADR records one consequential choice, its original circumstances, the credible alternatives actually weighed, and the tradeoff deliberately accepted. Narrow scope means one decision, not necessarily small impact. A subsystem description, routine implementation update, workflow convention, or release is not itself a reason for an ADR.

Keep rationale in the topic unless separate review and preservation of a particular commitment materially help future work. Consider non-obvious tradeoffs, costly reversal or divergence, and significant authority, security, interoperability, or runtime commitments. These are judgment prompts, not a Boolean gate or quota. An early foundational choice can merit a record; a mature project still makes many changes that do not. Maturity does not prescribe frequency or ceremony.

Propose a record when warranted, explaining why the topic's rationale is insufficient. Create it only when the user selects that proposal, explicitly requests the ADR, or repository instructions authorize that scoped authoring. Do not ask again merely to ratify an already explicit request. Choosing a document format never supplies approval for an unresolved architectural choice.

Start agent retrieval from the current topic, then follow relevant decision links. Easy generation is not evidence that another record is useful. Check applicability and implementation evidence; do not reopen settled debate, resurrect obsolete constraints, or treat an Accepted label as proof of delivery. See [Decision Records](references/decision-records.md) for history and replacement rules.

### Keep current truth and relevant memory

Revise continuing topics in place. Git preserves previous wording; retain relevant rejected alternatives and their reasons beside the current model so readers need not reconstruct them from history. Remove obsolete claims from active guidance without erasing rationale that still explains the design. A topic can contain several related decisions and can be split or merged when the architecture warrants it.

Do not impose ADR IDs, five-state lifecycles, mandatory frontmatter, or per-document revision ledgers on topics. Follow explicit repository metadata requirements; otherwise, dates and change summaries belong in Git and the project journal. Never mark a document reviewed without checking its content.

Distinguish **implemented behaviour**, **agreed direction not yet implemented**, and **unresolved proposals**. Implementation gaps do not invalidate an agreed choice, and agreement does not prove delivery. Keep unchosen ideas in exploration or proposed work, not stated as the current architectural model. Bind constraints only to their actual applicability: a legacy format can remain supported without governing new authoring.

### Review affected surfaces when architecture changes

For a material change, trace affected code, tests, documentation, skills, and compatibility obligations. Reconcile them within the authorized scope, or name precise remaining gaps and follow-up work. Documentation approval does not authorize implementation, tracker mutations, data migration, or a broader codebase rewrite. Do not quietly weaken the documented model to match accidental drift, or claim implementation follows merely because the document changed.

This skill owns the convention, topic format, and migration rules. Other workflows consume them:

- **Shape** identifies architectural implications and unresolved choices without automatically creating documents.
- **Implement** maintains affected topics within the authorized change.
- **Ship** checks architectural claims and remaining gaps against the actual diff and evidence.
- **Reflect** incorporates durable learning into existing topics and routes exceptional ADR candidates here rather than automatically producing records.

No new runtime, registry, or automated drift checker is implied. Perform a scoped evidence review using existing tools.

### Route other material to its owner

| Material | Destination |
|----------|-------------|
| Current system map | Architecture overview, linking to owning topics |
| Architectural model, constraint, or significant choice with lasting rationale | Existing architecture topic; create one only when needed |
| Shared implementation scope, sequencing, completion criteria, hierarchy, or status | Provider-neutral shaping workflow and the selected native tracker through its harness-owned connection |
| Technical design useful to implementers | Existing design/reference home, linked from the native work record when useful; never a second work identity |
| Product purpose, principle, or product-scope boundary | Vision document |
| Strategic bets, journey priorities, product deferrals, or learning sought | Strategy document |
| Workflow convention or reusable method | Owning skill or nearby guidance |
| Local implementation detail or evidence | Code, types, tests, comments, or the project journal |

### Preserve invariants while relocating detail

Keep current invariants and applicability in the owning topic, with durable rationale, tradeoffs, and consequences there or in its linked exceptional ADR. Put exact schemas, codecs, algorithms, limits, fixtures, incident narratives, and migration steps beside their implementation or in an existing technical reference. Before removing detail, identify its destination, preserve a truthful link, and verify that no security, storage, recovery, or compatibility guarantee disappears.

Use product or domain names in maintained prose. Treat identifiers embedded in code paths, stored schemas, serialized bytes, key derivation, signatures, authentication transcripts, or supported archives as compatibility contracts. An editorial rename never authorizes changing them.

## Verification

- The requested mode and authority were respected without unnecessary artifacts, migration, or questioning.
- Current code, applicable decisions, and the repository's actual canonical documentation paths were inspected.
- Topics describe the current model and relevant rationale, tradeoffs, alternatives, constraints, and applicability without duplicated authority.
- Implemented behaviour, agreed pending direction, and unresolved proposals are distinguished; no approval or implementation evidence was invented.
- A material change includes an affected-surface review and explicit remaining gaps; follow-up work did not exceed authorization.
- Any new ADR was deliberately authorized, captures one consequential choice, and adds value beyond the topic's rationale; topics gained no mandatory ADR numbering, metadata, or revision ledger.
- Current topic applicability and linked ADR history agree; original decision context was not silently rewritten into a different choice.
- Shared work routes only to the selected native tracker through the provider-neutral workflow; no local mirror or synchronization record was created.
- Schema, wire, cryptographic, credential, archive, and source-path identifiers were preserved unless separately inventoried and migrated.
- Navigation and links resolve, legacy references are handled honestly, and relevant repository checks pass.

## Quick Reference

| Situation | Action |
|-----------|--------|
| Explain a choice | Read its topic and evidence; answer without writing |
| Same topic, model evolves | Revise in place, retain relevant rationale, review affected surfaces |
| Useful architectural learning | Integrate into the owning topic; do not create a decision event |
| Agreed direction is not implemented | State the gap separately from shipped behaviour |
| Existing ADR corpus | Read applicable constraints; migrate only when requested |
| Consequential choice may need a separate record | Evaluate the exceptional ADR mode; propose only when it adds value |
| Explicit ADR convention or request | Use scoped decision-record guidance and the repository's actual format |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Topic template | [templates/architecture-topic.md](templates/architecture-topic.md) | Writing or revising a topic using optional prompts |
| Decision records | [references/decision-records.md](references/decision-records.md) | Evaluating, authoring, or replacing a rare, single-choice ADR |
| Decision record template | [templates/decision-record.md](templates/decision-record.md) | Writing an authorized ADR when the repository has no template |
| Auditing and migration | [references/auditing-and-migration.md](references/auditing-and-migration.md) | Auditing current truth or consolidating an existing corpus into topics |
| Legacy ADRs | [references/legacy-adrs.md](references/legacy-adrs.md) | Maintaining Loaf's historical living-record convention or interpreting its status during migration |
| Legacy ADR template | [templates/legacy-adr.md](templates/legacy-adr.md) | Preserving the historical Loaf format when the repository explicitly retains it |
| Documentation | `documentation-standards/references/documentation.md` | Applying repository documentation conventions |
