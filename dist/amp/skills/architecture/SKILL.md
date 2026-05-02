---
name: architecture
description: >-
  Creates Architecture Decision Records for architecturally significant
  decisions — choices that affect the system's structure, key quality
  attributes, dependencies, interfaces, or construction techniques, or that are
  difficult to reverse. Captures the rationale for a single such choice and its
  alternatives so future debate has the why preserved. Not for development
  patterns or implementation evidence (use /shape for SPECs), guiding principles
  (update ARCHITECTURE.md or VISION.md), workflow conventions (document in the
  owning skill), or local choices changeable in a single PR (session-log
  decision() instead). The ADR log is append-only — when circumstances change,
  write a new ADR that supersedes the old one.
version: 2.0.0-dev.37
---

# Architecture

Guides decision-making for **architecturally significant** choices through structured interviews, options analysis, and Architecture Decision Records. ADRs are **rare yet binding** — they record the rationale for choices that shape the system's structure, quality attributes, dependencies, interfaces, or construction techniques, and that the team agrees to honor until explicitly superseded. Most technical decisions do not warrant an ADR; the skill routes those to their proper destination (session log, SPEC, ARCHITECTURE.md, or owning skill) and stops.

Stabilizes canonical vocabulary in `docs/knowledge/glossary.md` when load-bearing terms surface mid-interview — additive to ADR creation, never a gate on it.

## Critical Rules

### The Bar

ADRs are reserved for **architecturally significant** decisions — those affecting:

- **Structure** (system boundaries, modules, layering)
- **Quality attributes** (performance, security, reliability, scalability)
- **Dependencies** (external services, libraries with broad reach, runtime/language commitments)
- **Interfaces** (public APIs, contracts between teams, cross-system protocols)
- **Construction techniques** (build system, deployment model, testing strategy at the system level)

…**and** are **difficult to reverse** in the project's current state (would require coordination beyond a single PR — schema migration, contract change, multi-skill update, external-tool retraining).

The bar is constant. **The number of decisions clearing it scales with project maturity.** Early/exploratory projects clear the bar rarely — usually only foundational shape commitments (language/runtime/standard adoption, primary architectural shape). Mature projects clear it more often as cost-of-change rises across the codebase. **When in doubt during early phases, prefer SPEC over ADR** — SPECs evolve, ADRs supersede.

### Triage Gate

Before grilling, confirm with the user that the decision passes all three:

1. **Architectural significance** — does it affect *structure, quality attributes, dependencies, interfaces, or construction techniques*?
2. **Cost of change** — would reversing it require coordination beyond a single PR (schema migration, contract change, multi-skill update, external-tool retraining)?
3. **Rationale durability** — when this debate returns in 18 months, would the team need the *why* reconstructed, or is it self-evident from the code?

**All three "yes" → proceed to grilling and ADR.** Any "no" → route per the table below and stop.

| Decision shape | Destination |
|---|---|
| Architecturally significant + rationale needs preservation | **ADR** (`docs/decisions/`) |
| Development pattern, direction, implementation evidence | SPEC via `/shape` |
| Guiding principle, philosophy, operating model | `ARCHITECTURE.md` / `VISION.md` (mutable, `/reflect`-revisable) |
| Workflow convention, skill-specific lore | Owning skill's `SKILL.md` or references |
| Local choice, single-PR scope, no downstream coordination | Session-log `decision(scope)` + code comment if needed |

### Skip ADR When

- Decision is a convention or naming preference — no measurable effect on architecture (fails the architectural-significance test)
- Decision is workflow lore belonging to a specific skill — document in that skill, not in `docs/decisions/`
- Decision is a guiding principle or operating philosophy — update `ARCHITECTURE.md` (or `VISION.md` if strategic); principles evolve, ADRs are immutable post-acceptance
- Decision is exploration of alternatives without a chosen direction — that's a SPEC via `/shape`; the ADR comes after if the chosen direction is architecturally significant
- Decision can be changed in a single PR with no downstream coordination — session-log `decision(scope)` and a code comment if needed
- Rationale is aesthetic ("looks/feels better", "scans nicer", "more consistent visually") — never an ADR

### Lifecycle

ADRs are **append-only post-acceptance**. Don't edit accepted records.

When circumstances change and the team's answer changes, write a **new ADR that supersedes the old one** — set `supersedes: ADR-NNN` on the new one and `superseded_by: ADR-MMM` on the old one. Both stay in `docs/decisions/`. The old one preserves the historical "_was_ the decision, _no longer_ the decision" record (Nygard).

Supersession is healthy. The bar for *writing* an ADR is high; once written, the bar for *quietly diverging from* it is also high (write a superseding ADR instead).

**Always**
- Run the Triage Gate before grilling. If any of the three questions answers "no", route to the destination in the table and stop — do not produce an ADR.
- When a decision is architecturally significant but routes elsewhere (SPEC, ARCHITECTURE.md, owning skill), help the user place it in the right destination. The skill's job is correct routing, not just ADR creation.
- Read existing VISION.md, ARCHITECTURE.md, and ADRs before proposing changes
- Read `docs/knowledge/glossary.md` at interview start (via `loaf kb glossary list`); use canonical terms throughout
- When fuzzy/drifted language surfaces, challenge inline; if a load-bearing term emerges, offer `loaf kb glossary upsert` or `stabilize`
- Follow the shared interview protocol in [templates/grilling.md](templates/grilling.md)
- Present multiple options with pros/cons and "fits when" context
- Wait for explicit user decision before proceeding with documentation
- Log decision to session journal: `loaf session log "decision(architecture): ADR-NNN adopted for X"`

**Never**
- Make architectural decisions without user input
- Contradict existing decisions without explicitly superseding them
- Proceed past the Triage Gate when any of the three questions answers "no"
- Create ADRs without user approval, even when the user requests one — if the decision fails the Triage Gate, propose the correct destination and decline the ADR
- Use the word "irreversible" — software decisions can always be reversed via supersession; the operative criterion is "difficult to reverse"
- ADR-ify aesthetic preferences, naming conventions, workflow lore, or guiding principles — those have other homes (see Skip ADR When)
- Edit accepted ADRs to reflect a change of mind — write a new superseding ADR instead
- Block ADR creation on glossary state — glossary mutations are additive and opt-in
- Call `loaf kb glossary propose` (reserved for upstream ambiguity-resolving skills)

## Verification

After work completes, verify:
- Triage Gate ran before any grilling; all three questions affirmatively answered before proceeding
- Decision falls within the five canonical domains (structure, quality attributes, dependencies, interfaces, construction techniques) OR clears the difficult-to-reverse bar
- ADR captures rationale, not exploration (exploration belongs in a SPEC)
- If the decision supersedes a prior ADR, the new ADR has `supersedes:` and the old one has `superseded_by:`
- ADR created using template at [templates/adr.md](templates/adr.md)
- ARCHITECTURE.md updated with new constraints and ADR reference
- ADR number assigned sequentially (ls docs/decisions/ADR-*.md for next number)
- Council convened if decision affects multiple domains
- Glossary read at interview start; any load-bearing term surfaced was offered for `stabilize` or `upsert`

## Quick Reference

### Routing Cheat Sheet

| Decision shape | Destination |
|---|---|
| Architecturally significant + rationale needs preservation | **ADR** (`docs/decisions/`) |
| Development pattern, direction, implementation evidence | SPEC via `/shape` |
| Guiding principle, philosophy, operating model | `ARCHITECTURE.md` / `VISION.md` (mutable, `/reflect`-revisable) |
| Workflow convention, skill-specific lore | Owning skill's `SKILL.md` or references |
| Local choice, single-PR scope, no downstream coordination | Session-log `decision(scope)` + code comment if needed |

### ADR Numbering

```bash
ls docs/decisions/ADR-*.md 2>/dev/null | \
  grep -oE 'ADR-[0-9]+' | sort -t- -k2 -n | tail -1 | awk -F- '{print $2 + 1}'
```

Start with `ADR-001` if none exist.

### Council Triggers

Convene when: multiple domains affected, conflicting team opinions, high reversal cost, novel problem, or user requests deliberation.

### Evaluation Criteria

For each option: alignment with VISION/ARCHITECTURE, complexity added, reversibility, team capability, maintenance cost.

### Glossary Mutation Policy

This skill *stabilizes* terms — promote a previously-proposed candidate, or write a canonical term directly when one emerges mid-interview.

| Verb | When |
|------|------|
| `loaf kb glossary list` | At interview start (via grilling protocol) |
| `loaf kb glossary check <term>` | When a term's status is in question during the interview |
| `loaf kb glossary stabilize <term>` | A previously-proposed candidate has firmed up into a load-bearing decision |
| `loaf kb glossary upsert <term> --definition <d> --avoid <list>` | A load-bearing term emerges fresh and is canonical from the outset |

`propose` is reserved for upstream skills that resolve ambiguity (e.g., a future `/shape` evolution) — do not call it here.

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| ADR Template | [templates/adr.md](templates/adr.md) | Creating new architecture decision records |
| Grilling Protocol | [templates/grilling.md](templates/grilling.md) | Running the structured interview, including glossary discipline |
| Council Workflow | `orchestration/references/councils.md` | Multi-agent deliberation for complex decisions |
| Documentation | `documentation-standards/references/documentation.md` | ADR formatting and standards |
| Canonical ADR sources | [https://adr.github.io/](https://adr.github.io/) | Reference for ADR practice; format hub |
| AWS ADR Process | [AWS prescriptive guidance](https://docs.aws.amazon.com/prescriptive-guidance/latest/architectural-decision-records/adr-process.html) | "Architecturally significant" framing, separate-design-from-decision principle |
| Microsoft Well-Architected ADR | [Azure docs](https://learn.microsoft.com/en-us/azure/well-architected/architect-role/architecture-decision-record) | "Difficult to reverse" criterion, append-only log discipline |
| Nygard original | [Documenting Architecture Decisions (2011)](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions.html) | Foundational article; supersession lifecycle |
| Architecturally Significant Requirements | [Wikipedia](https://en.wikipedia.org/wiki/Architecturally_significant_requirements) | ASR test — measurable effect on architecture |
