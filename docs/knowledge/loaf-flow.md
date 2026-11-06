---
topics:
  - workflow
  - pitch
  - tracker-native
  - rideable-increments
covers:
  - content/skills/loaf-reference/references/flow-semantics.md
  - content/skills/project-management/references/record-writing.md
  - content/skills/foundations/references/rideable-increments.md
  - content/skills/ship/SKILL.md
consumers:
  - implementer
  - reviewer
  - researcher
last_reviewed: '2026-11-06'
---

# The Loaf Flow

## Contents

- The pipeline
- Work contracts and selection
- Rideable progress
- Two scales
- Supporting techniques and continuity
- Operating links

Loaf supplies one portable method for taking a problem from discovery through verified release. This guide maps the stages and their operating references. The shipped [Flow semantics](../../content/skills/loaf-reference/references/flow-semantics.md) own the execution rules; [Authority Boundaries](../architecture/authority-boundaries.md) explains the tracker, Git, harness, and Loaf responsibilities.

## The pipeline

```
pitch → shape → implement → ship → release
```

| Stage | Reads | Produces |
|-------|-------|----------|
| **pitch** | Human context and relevant live tracker context | A problem narrative; may fill an existing intent record |
| **shape** | Problem narrative and current native tracker state | A bounded work contract; Todo only when the human selects the work |
| **implement** | Live selected native contract and repository state | Code and observed verification evidence; bare invocation only reports the unblocked Todo frontier |
| **ship** | Live contract, candidate diff, and evidence | A quality verdict and verified native transition when authorized |
| **release** | Already-landed work, Git history, and live native records | A verified repository-native release outcome |

Triage moves private captures to Backlog and decides which existing native candidates advance, defer, close, or return to discovery. Backlog preserves public intent without authorizing execution; Todo is shaped and selected. A richly written record is not automatically selected, and implement never infers a work choice from the branch. Orchestration coordinates bounded execution against the same live contract. Neither owns another queue or work unit.

## Work Contracts and Selection

Shape writes a bounded definition on the canonical native record: the problem, intended outcome, definition of done, verification expectations, explicit out-of-scope boundaries, and supported relationships. A technical design or committed document may support it without becoming another work identity.

Records are written for people. Open with the problem and why it matters, keep the criteria clear, and place detail only agents need in collapsed notes. [Record Writing](../../content/skills/project-management/references/record-writing.md) owns the format; the selected provider owns its tracker's field mappings and syntax. GitHub workflow lanes come from the configured board's Status field, not Issue open/closed.

Before substantive edits, read the native contract and verify both its sufficiency and selection. A concrete implementation request selects the work; verify that selection through the provider without asking for the same approval again. Every later ceremony reads the live record again. A branch name, handoff snapshot, or rich Backlog description cannot replace that readback.

Implementation uses repository-native branches, worktrees, commits, tests, and pull requests. Ship is the sole quality gate: it checks the candidate against the contract and recorded evidence, then performs only authorized landing and native transitions. Its [verification and merge rules](../../content/skills/ship/SKILL.md) govern completion; an agent's claim or a green CI spot check alone does not prove delivery. Release is a separate, retroactive operation over already-landed work. [Runtime and Delivery](../architecture/runtime-and-delivery.md) owns version policy, artifact provenance, and the remaining delivery and rollback gaps.

## Rideable Progress

The unit of progress is a [rideable increment](../../content/skills/foundations/references/rideable-increments.md): a complete, useful operator journey rather than a finished horizontal layer. Shape makes the Rider, complete Journey, Entry point, observable Outcome, real Dogfood, Safety/integrity proof, Learning sought, and explicit Deferrals concrete inside the existing native work definition and criteria.

Breadth may shrink; integrity may not. Foundation work stays beside the immediate journey that exercises it, and real dogfood earns later complexity. This sharpens the existing Flow and work contract—it adds no lifecycle, record, schema, status, score, or approval gate.

## Two scales

| Scale | Pitch output | Shape or bootstrap action |
|-------|--------------|---------------------------|
| Existing project and one direction | Problem narrative, optionally on an existing intent record | Triage moves intent to Backlog; shape defines the canonical work contract and verifies readback; selection is explicit |
| New project | `docs/BRIEF.md` as a frozen intake snapshot | Bootstrap extracts operating documents and offers user-confirmed native backlog records for independently rideable concepts |

At either scale, pitch remains problem-space discovery. It can retain a possible solution as a hypothesis without selecting it for execution. Shape bounds the entry point, implementation, proof, and decomposition and writes the agreed definition once to the tracker.

## Supporting Techniques and Continuity

Explore and brainstorm are agent techniques for uncertainty and divergence; they do not mint shared work. Sparks, ideas, journal entries, wraps, handoffs, and derived context remain private Loaf continuity. Triage owns moving captures to tracker Backlog through the provider skill; an explicit filing request during pitch performs that same path. Moving intent to Backlog does not authorize execution.

The journal has no session entity, lifecycle, open/close operation, or rotation. A wrap is optional synthesis; context is derived at read time. [Private Continuity](../architecture/private-continuity.md) owns that model. Most skill results return through the harness; persist a temporary report only when it must outlive the response. Its producing skill owns the format, and deletion or durable promotion needs the user's approval.

If the configured tracker connection or a required native capability is unavailable, tracker-mutating Flow steps stop clearly. They never fall back to a local issue authority. Scratchpad is deferred from the current Flow and is not an execution or continuity dependency.

For access to historical local records and one-time cutover precautions, use [Command Routing](cli-reference/command-routing.md#legacy-records-and-migration). Those compatibility surfaces do not define new shared work.

## Operating links

| Topic | Where |
|-------|--------|
| Current ceremony and continuity semantics | [Flow semantics](../../content/skills/loaf-reference/references/flow-semantics.md) |
| Work and service authority | [Authority model](../../content/skills/loaf-reference/references/authority-model.md) |
| Stable provider behavior | [Project management](../../content/skills/project-management/SKILL.md) |
| Contributor provider boundary | [Provider modules](../../content/skills/project-management/references/provider-modules.md) |
| Increment doctrine | [Rideable increments](../../content/skills/foundations/references/rideable-increments.md) |
