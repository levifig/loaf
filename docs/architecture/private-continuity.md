# Private Continuity

## Current Model

Loaf's agreed persistence model is operator-private, project-scoped, append-only typed facts in SQLite. It includes journal entries and wraps, sparks, ideas, explorations, decisions, findings, and handoffs, with supporting project identity, checkpoints, opaque references, and verification evidence. Scratchpad remains deferred until its authority and lifecycle are defined. Temporary reports are not continuity facts by default; [Authority Boundaries](authority-boundaries.md) owns artifact placement and the tracker, Git, and harness boundaries.

The public CLI has not cut over to the complete model. Shipped journal and local records coexist with pre-cutover destination code; their presence does not mean every destination record or operator journey is publicly available.

## Identity and Projection

One stable opaque project identity spans linked worktrees. Branch and worktree identify provenance and query scope, not separate continuity universes. Moving a checkout or rekeying a project must not fork history or silently rederive durable entity identifiers.

Mint an opaque entity identifier once and keep it stable. Human names, historical identifiers, and imported aliases resolve through an explicit identity registry or equivalent lookup owned by the relevant store. Identity divergence and collisions fail explicitly. Compatibility importers may recognize historical deterministic identifiers without making those derivations authoritative for new records.

Current views are deterministic, rebuildable projections of immutable facts. Fact types and versions form a closed catalog with a deterministic total order for replay and convergence; display timestamps are metadata, not ordering authority. Unknown required types or versions, identity conflicts, corrupt payloads, and incomplete writes fail closed rather than being guessed into current truth. Schema evolution must preserve these properties.

## Journal and Resumption

The project journal is the only session-related structure. Entries append independently with opaque harness-conversation correlation metadata. There is no session entity, status, open/close operation, rotation, or lifecycle transition. A wrap is optional synthesis, not a transition. Context is derived at read time and discarded after use, never persisted as a second truth.

Foundations and the generated managed project instructions own ordinary capture: record significant decisions and discoveries as they happen, then synthesize only useful reasoning still missing before substantial completion, handoff, or compaction. A concise `wrap(scope)` checkpoint preserves abandoned approaches and why, unresolved uncertainty, and the next useful step when applicable, without repeating raw events or filling empty fields. Routine interactions need no redundant checkpoint. Existing `wrap` entries, `--since-last-wrap`, and digest behavior remain compatible; there is no standalone Wrap skill or required ceremony.

This is agent instruction, not automatic persistence. The existing PreCompact command emits the same selective guidance and writes nothing itself. Delivery depends on the harness and mode; ordinary capture does not depend on receiving a compaction or end hook. Abrupt endings can still lose unrecorded reasoning. Useful resumption depends on meaningful entries and occasional synthesis; raw history can be less concise when no checkpoint exists. This tradeoff avoids mutable session ownership conflicts across conversations, branches, worktrees, and harnesses.

Project Management owns actionable follow-up capture, Implement and Ship own delivery evidence and Git-state reporting, and Housekeeping owns artifact maintenance. Checkpointing grants none of their action authorities.

Sparks survive independently of archived brainstorm documents. Historical `## Sparks` sections are evidence, not an active capture or promotion authority. The old objection that standalone spark files were too heavy does not constrain typed private continuity or revive a brainstorm-document lifecycle. Capture and promotion behavior belongs to the owning skills.

## Deliberate Transfers

The [Handoff workflow](../../content/skills/handoff/SKILL.md) prepares recipient-focused context for a bounded next action. Ordinary resumption and parked-work synthesis remain Foundations responsibilities. Direct harness delivery needs no persistent artifact when the recipient can consume that context and independent retention has no purpose. Preparation never supplies authorization to spawn, message, create tasks, or execute work; a packet carries native references and requires the recipient to re-read current tracker and Git state.

For transfers that must survive independently, cannot rely on the current harness context, or are explicitly requested to persist, the shipped `loaf handoff new/show/list` commands use the current SQLite store. Verify body and project identity by readback and give a usable retrieval route in the source project. A recipient in another project or environment may not have that store: a local reference alone does not establish delivery. An explicitly requested portable file is a transfer snapshot at the requested destination, not a competing continuity store. Report unavailable persistence honestly without silently inventing a fallback store.

This guidance does not activate the complete destination fact model, change stored IDs or statuses, delete legacy packets, or introduce a deprecation/cleanup lifecycle. Current SQLite persistence is already available; full destination activation and broader resumption layers remain pending below.

## Local Storage and Recovery

The runtime must resolve projects explicitly, reject unsafe symlink, reparse-point, or other path substitution, apply restrictive file and directory permissions where supported, use transactional write boundaries, and provide actionable corruption diagnostics. SQLite is the local canonical private store and needs independent backups. These controls reduce accidental exposure; they do not defend plaintext continuity against hostile processes running as the same user or privileged actors.

Destructive transitions must be previewable, backup-first, verified afterward, represented by a rollback manifest, and safe to rerun within their stated preconditions. Ambiguous or unsupported records remain explicit. Restore is rehearsed in isolated state; activation is a separate, quiesced operator action followed by verification. Running old and new writable stores together is not a restore strategy.

Linked worktree startup gates follow the command's storage dependencies. SQLite journal and private-record operations share the canonical Git project root and do not require reconciliation of worktree-local Markdown. Configuration, instructions, skills, knowledge, and unrelated artifacts can differ between checkouts. Active file-backed report/council readers, Markdown compatibility task operations, and explicit import/recovery operations refuse local-only or divergent files in the storage they consume, whether tracked or untracked. Classification is read-only and never creates `.moved-to`. Relevant invalid markers, symlinks, interrupted staging paths, and unreachable main roots remain explicit recovery failures.

The explicit `loaf migrate worktree-storage` command retains its historical full-tree transfer contract. Its dry-run includes configuration and artifacts, so a startup refusal recommends preview or manual reconciliation of named paths. `--apply` moves files and removes losing conflict copies according to the selected policy; it is not a backup of both versions. Startup never applies that migration implicitly.

Supported legacy Markdown migration remains one-way, never a live mirror. Removal requires an out-of-tree exact-byte backup with a SHA-256 manifest and verification that source bytes still match. Restore uses those original bytes, not rendered approximations. Database transactions do not make Markdown file deletion crash-atomic or transactional. The [migration guide](../knowledge/cli-reference/markdown-migration.md), [recovery guidance](../knowledge/cli-reference/troubleshooting.md), and [migration source and tests](../../internal/state/markdown_migration.go) retain exact compatibility mechanics.

For a registered destination, simulation, apply, retry, and resume use the same decision pipeline while source and destination remain unchanged. A matching known fingerprint can reclaim its registered row; foreign-origin collisions are skipped and reported. Meaningful stored provenance and status win, unknown values may be filled, and archived records never reopen. Inventory-only discovery cannot prove collision or status dispositions against a destination it has not registered and simulated. [Re-import decision tests](../../internal/state/markdown_import_decisions_test.go) and [regression tests](../../internal/state/markdown_reimport_regression_test.go) preserve this boundary.

## Rationale and Rejected Alternatives

Append-only facts preserve history, auditability, deterministic replay, and a common basis for [private synchronization](private-synchronization.md). This costs explicit projection, alias resolution, schema evolution, and failure handling rather than simple row mutation.

- **Branch-local continuity** fragmented resumption and risked conflicting identities or accidental commits of private context.
- **One universal `.agents/` home** addressed worktree fragmentation but incorrectly grouped private continuity, tracker work, ADRs, and Git-authored knowledge under one filesystem rule. Legacy redirects and recovery data remain version-scoped compatibility, not a rule for every new artifact.
- **Private continuity in the native tracker** would use a collaboration authority as a private agent-memory service.
- **Mutable current-state rows or last-writer synchronization** lose durable history and make convergence depend on update timing.
- **IDs derived from project and alias fields** fail when those mutable inputs change; re-import can then mint a different identity for an existing record.
- **Mutable session lifecycle** accumulated unused state, stuck statuses, and routing conflicts. Persisting generated context would add a projection that can drift from its facts.
- **Live Markdown mirrors or deletion without original rollback bytes** introduce competing authorities or make exact recovery impossible.

## Implementation and Gaps

The shipped [journal](../../internal/cli/journal.go), [derived context](../../internal/state/journal_context.go), and [entity registry](../../internal/state/entity_registry.go) implement their current compatibility surfaces. Destination [fact definitions](../../vnext/continuity/facts.go), [identity minting](../../vnext/continuity/identity.go), and [SQLite storage](../../vnext/continuity/sqlite/) implement pre-cutover persistence, with [domain contract tests](../../internal/vnextcontinuitycontract/) and [storage contract tests](../../internal/vnextcontinuitysqlitecontract/).

New destination observations carry an observed harness and version. A writer uses `unknown` when it cannot prove either value. Existing V1 facts retain their canonical bytes and project missing provenance as `unknown` on read; the migration rehearsal marks its historical source as unknown rather than inferring a harness. Legacy journal origins follow the same explicit-unknown rule for new writes. Pre-tool gateway outcomes are separate machine-local `hook_events` rows containing only a hook ID, normalized tool name, allow/warn/block outcome, reason class, harness, version, and timestamp. `loaf telemetry status` reads local aggregates by project, harness, outcome, and reason class for the current UTC month. Gateway recording requires an initialized current-schema local state database and does not alter hook policy when state is unavailable.

The isolated [migration rehearsal](../../vnext/migration/rehearsal/import.go) covers only part of the domain and has no activation surface. Full-domain rehearsal, quiesced manual activation, and post-activation verification remain pending. The [bootstrap command](../../vnext/internal/command/command.go) exposes introspection only.

Exact schemas, canonical encodings, digest domains, migration identifiers, and compatibility literals remain in source and tests. The [kernel](../../vnext/internal/kernel/kernel.go) reports schema `vnext/1`, while the [continuity schema](../../vnext/continuity/sqlite/schema.go) uses line `vnext` at version 11. Resolving that mismatch is pending compatibility work, not an editorial rename. Stored records and archives retain their existing compatibility bytes.
