# Flow Semantics

The Loaf Flow is `pitch → shape → implement → ship → release`. Triage moves private captures onto tracker Backlog and disposes existing native candidates. Orchestration coordinates bounded execution without acquiring work authority.

Loaf optimizes for coherent outcomes and reliable continuation across people, agents, sessions, and tools. Preserve intent in shared, durable context without turning every idea into a commitment. Before execution, resolve the uncertainties that materially determine value and scope, then give the executor enough context and autonomy to deliver a small, end-to-end useful result. Pull work deliberately, limit concurrent commitments, and judge progress by working outcomes and evidence rather than activity or ticket counts. When discoveries invalidate the direction, reconsider it instead of extending execution by default. The issue tracker is the shared memory and coordination surface — not a transcript dump, and not a command queue merely because an item exists.

A native record may hold public intent without being selected work. Backlog (or the provider's uncommitted lane) is shared memory: thin or rich, with no promise of execution. Substantive sufficiency is whether an implementer can deliver and a reviewer can evaluate without inventing intent. Native establishment and selection are separate: writing a sufficient definition does not select the work. Todo is shaped and selected. `/implement` starts only from selected work. Sparks and ideas stay private until triage moves them to Backlog.

## Before Entering Implementation

Before announcing implementation, invoking Implement for execution, dispatching an implementation worker, or making substantive edits (including regression tests), establish a selected native issue through the configured project-management provider. This applies to ordinary requests such as "fix this bug" and "Let's implement it" after discussion or context transfer. A stale summary, branch, task/thread identifier, local note, or issue existence alone cannot establish authority.

Read the named native issue and confirm it covers the request; without a native reference, search for a matching issue before creating one. Reuse a complete selected issue without repeating Shape. Shape only missing or incomplete contract details, then authoritatively read back the scope, observable completion criteria, and selected workflow state. Before starting, identify the verified native reference and completion criteria proportionately so the operator can see what will be implemented. An existing issue covers incidental related work; do not create a ticket for every edit.

An explicit user instruction to implement a concrete change selects that work, including "Let's implement it" when the discussion makes the change concrete. Reflect that existing selection in the native workflow and verify it without another approval. Inspection-only, review-only, idea capture, Backlog presence, and an unqualified bare workflow invocation do not select implementation. A bare Implement invocation reports the unblocked native Todo frontier and stops; do not infer its work from a branch or open editor.

The main agent owns this check whether it implements directly or delegates. Before optional implementation delegation, pass the verified native reference, scope, completion criteria, selected-state readback, allowed paths/actions and required verification. Resolve missing context before dispatch; include an explicit scoped override honestly when it applies. The packet is transient caller-supplied evidence, not independent tracker attestation or a mechanical write barrier. Review-only workers remain read-only and do not authorize fixes.

Read-only discovery, inspection, and running existing checks are allowed before an issue is established; diagnosing a problem does not authorize editing its fix. A purely incidental spelling, whitespace, or formatting correction may proceed without an issue only when it changes no behavior, requirement, meaning, or scope. Changing installation instructions, policy, expected test behavior, or semantic formatting is substantive. A one-line bug or security fix is substantive regardless of its size.

If the connection, required workflow capability, or authoritative readback is unavailable or indeterminate, report the blocker and remain in permitted discovery. Entering implementation requires an explicit scoped override of this prerequisite from the user; record its scope honestly and never claim an issue or selection was verified. Reading guidance to route a bare or incomplete request is permitted inspection, not permission to announce or begin implementation. Do not configure authentication or introduce a local shadow tracker. These exceptions waive only this prerequisite, not other authority or safety requirements.

## Rideable Progress

Every bounded milestone is a complete, useful operator journey: a real rider starts from a named entry point, reaches an observable outcome, and can use the result safely today. Narrow breadth before weakening integrity. Security, determinism, recovery, honest failure, and data safety still apply to the smallest slice; real dogfood and the learning it produces earn later complexity.

Rideable increments sharpen the existing problem narrative, native tracker work contract, criteria, evidence, and review. They introduce no additional lifecycle, work authority, score, status, storage schema, or approval gate.

## Ceremonies

| Ceremony | Reads | Produces |
|----------|-------|----------|
| Pitch | Human context and relevant live tracker context | A problem narrative; may fill an existing intent record |
| Triage | Private captures and candidate native tracker records | Keep, archive, move to Backlog, hand to pitch, or hand to shape |
| Shape | Existing pitch, conversation, or native issue plus current native state | A proportionate, independently reviewable work contract on the tracker record; optional same-turn move to Todo when the human selects the work |
| Implement | Live selected work contract and repository state | Code, verification evidence, and tracker updates; a clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence stays with implement; unclear or changed contract substance or a material scope or commitment shift returns to shape; bare invocation reports the unblocked Todo frontier and does not start |
| Ship | Live contract, actual candidate, and existing evidence | A quality verdict mapped to the contract's specified proof, a durable disposition discoverable from the issue, and a verified tracker transition; acceptance does not authorize landing and does not accept material contract drift by recording it |
| Release | Already-landed work and release evidence | A release outcome recorded on native work |

## Continuity Rules

Every ceremony begins by reading the native record again. Handoffs carry its native reference, not a copied snapshot treated as authority. Mutations use the configured provider mapping, then re-read the same native record and report the observed result.

The main agent executes the common contract through the selected provider skill. Dedicated provider profiles remain deferred until target packaging and connector-only enforcement are proven; no current Flow surface links to an unavailable profile.

If the connection is unavailable or a required provider capability is absent, preserve the narrative as conversation output or private continuity and report the gap. Do not manufacture shared work, relationships, or status outside the tracker.
