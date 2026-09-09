# Flow Semantics

The vNext Loaf Flow is `pitch → shape → implement → ship → release`. Triage moves private captures onto tracker Backlog and disposes existing native candidates. Orchestration coordinates bounded execution without acquiring work authority.

Loaf optimizes for coherent outcomes and reliable continuation across people, agents, sessions, and tools. Preserve intent in shared, durable context without turning every idea into a commitment. Before execution, resolve the uncertainties that materially determine value and scope, then give the executor enough context and autonomy to deliver a small, end-to-end useful result. Pull work deliberately, limit concurrent commitments, and judge progress by working outcomes and evidence rather than activity or ticket counts. When discoveries invalidate the direction, reconsider it instead of extending execution by default. The issue tracker is the shared memory and coordination surface — not a transcript dump, and not a command queue merely because an item exists.

A native record may hold public intent without being selected work. Backlog (or the provider's uncommitted lane) is shared memory: thin or rich, with no promise of execution. Todo is shaped and selected. `/implement` starts only from selected work. Sparks and ideas stay private until triage moves them to Backlog.

## Rideable Progress

Every bounded milestone is a complete, useful operator journey: a real rider starts from a named entry point, reaches an observable outcome, and can use the result safely today. Narrow breadth before weakening integrity. Security, determinism, recovery, honest failure, and data safety still apply to the smallest slice; real dogfood and the learning it produces earn later complexity.

Rideable increments sharpen the existing problem narrative, native tracker work contract, criteria, evidence, and review. They introduce no additional lifecycle, work authority, score, status, storage schema, or approval gate.

## Ceremonies

| Ceremony | Reads | Produces |
|----------|-------|----------|
| Pitch | Human context and relevant live tracker context | A problem narrative; may fill an existing intent record |
| Triage | Private captures and candidate native tracker records | Keep, archive, move to Backlog, hand to pitch, or hand to shape |
| Shape | Problem narrative plus current native state | A complete work contract on the tracker record; optional same-turn move to Todo when the human selects the work |
| Implement | Live selected work contract and repository state | Code, verification evidence, and tracker updates; bare invocation reports the unblocked Todo frontier and does not start |
| Ship | Live contract, candidate change, and evidence | A quality verdict and verified tracker transition |
| Release | Already-landed work and release evidence | A release outcome recorded on native work |

## Continuity Rules

Every ceremony begins by reading the native record again. Handoffs carry its native reference, not a copied snapshot treated as authority. Mutations use the configured provider mapping, then re-read the same native record and report the observed result.

The main agent executes the common contract through the selected provider skill. Dedicated provider profiles remain deferred until target packaging and connector-only enforcement are proven; no current Flow surface links to an unavailable profile.

If the connection is unavailable or a required provider capability is absent, preserve the narrative as conversation output or private continuity and report the gap. Do not manufacture shared work, relationships, or status outside the tracker.
