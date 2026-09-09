---
topics:
  - workflow
  - pitch
  - tracker-native
  - rideable-increments
covers:
  - vnext/content/skills/pitch/**/*
  - vnext/content/skills/shape/**/*
  - vnext/content/skills/triage/**/*
  - vnext/content/skills/implement/**/*
  - vnext/content/skills/ship/**/*
  - vnext/content/skills/release/**/*
  - vnext/content/skills/orchestration/**/*
consumers:
  - implementer
  - reviewer
  - researcher
last_reviewed: '2026-09-05'
---

# The Loaf Flow

## Contents

- The pipeline
- Authority boundary
- Rideable progress
- Two scales
- Problem-space vs solution-space
- Supporting techniques and continuity
- Operating links

Loaf supplies one portable method for taking a problem from discovery through verified release. The selected external tracker owns shared work; Loaf never creates or synchronizes a second work authority.

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

## Authority Boundary

| Authority | Owns |
|-----------|------|
| Tracker | Shared work identity, definition, definition of done, status, hierarchy, assignment, and collaboration |
| Loaf | Flow instructions, templates, profiles, private continuity, derived context, and deterministic local mechanics |
| Harness | Tracker connection, authentication, model execution, and native tools |
| Git | Code and deliberately promoted artifacts |

Provider skills implement the stable `project-management/v1` behavior through the current harness's authenticated connection. They preserve provider-native fidelity and verify writes by readback. Loaf never asks for provider API keys, installs or authenticates connectors, proxies provider traffic, mirrors tracker fields, or stores ongoing local-to-remote mappings.

Legacy local work records are migration evidence only. Moving them into a tracker is a one-time, agentic, verified migration, never synchronization, reconciliation, push/pull, or dual authority.

## Rideable Progress

The unit of progress is a [rideable increment](../../content/skills/foundations/references/rideable-increments.md): a complete, useful operator journey rather than a finished horizontal layer. Shape makes the Rider, complete Journey, Entry point, observable Outcome, real Dogfood, Safety/integrity proof, Learning sought, and explicit Deferrals concrete inside the existing native work definition and criteria.

Breadth may shrink; integrity may not. Foundation work stays beside the immediate journey that exercises it, and real dogfood earns later complexity. This sharpens the existing Flow and work contract—it adds no lifecycle, record, schema, status, score, or approval gate.

## Two scales

| Scale | Pitch output | Shape or bootstrap action |
|-------|--------------|---------------------------|
| Existing project and one direction | Problem narrative, optionally on an existing intent record | Triage moves intent to Backlog; shape defines the canonical work contract and verifies readback; selection is explicit |
| New project | `docs/BRIEF.md` as a frozen intake snapshot | Bootstrap extracts operating documents and offers user-confirmed native backlog records for independently rideable concepts |

At either scale, pitch remains problem-space discovery. It can retain a possible solution as a hypothesis without selecting it for execution. Shape bounds the entry point, implementation, proof, and decomposition and writes the agreed definition once to the tracker.

## Problem-space vs solution-space

- **Pitch** asks who experiences the problem, what happens now, why it matters, what observable outcome is desired, and which first complete journey is worth shaping.
- **Shape** bounds the solution, proof, risks, deferrals, hierarchy, and dependencies on the canonical native record.
- **Implement** preserves that live contract while choosing the smallest coherent code change and rejecting machinery the journey does not exercise.
- **Ship** independently proves criteria, end-to-end usability, dogfood, and integrity before any authorized landing or native transition.
- **Release** describes the larger real journey made possible by already-landed work; it does not repeat ship's quality gate.

## Supporting Techniques and Continuity

Explore and brainstorm are agent techniques for uncertainty and divergence; they do not mint shared work. Sparks, ideas, journal entries, wraps, handoffs, and derived context remain private Loaf continuity. Triage owns moving captures to tracker Backlog through the provider skill; an explicit filing request during pitch performs that same path. Moving intent to Backlog does not authorize execution.

If the configured tracker connection or a required native capability is unavailable, tracker-mutating Flow steps stop clearly. They never fall back to a local issue authority. Scratchpad is deferred from the current Flow and is not an execution or continuity dependency.

## Operating links

| Topic | Where |
|-------|--------|
| Current ceremony and continuity semantics | [Flow semantics](../../vnext/content/skills/loaf-reference/references/flow-semantics.md) |
| Work and service authority | [Authority model](../../vnext/content/skills/loaf-reference/references/authority-model.md) |
| Stable provider behavior | [Project management](../../vnext/content/skills/project-management/SKILL.md) |
| Contributor provider boundary | [Provider modules](../../vnext/content/skills/project-management/references/provider-modules.md) |
| Increment doctrine | [Rideable increments](../../content/skills/foundations/references/rideable-increments.md) |
