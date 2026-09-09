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

Useful resumption still depends on meaningful entries and occasional synthesis; raw history can be less concise when no wrap exists. This is a deliberate tradeoff for avoiding mutable session ownership conflicts across concurrent conversations, branches, worktrees, and harnesses.

Sparks survive independently of archived brainstorm documents. Historical `## Sparks` sections are evidence, not an active capture or promotion authority. The old objection that standalone spark files were too heavy does not constrain typed private continuity or revive a brainstorm-document lifecycle. Capture and promotion behavior belongs to the owning skills.

## Local Storage and Recovery

The runtime must resolve projects explicitly, reject unsafe symlink, reparse-point, or other path substitution, apply restrictive file and directory permissions where supported, use transactional write boundaries, and provide actionable corruption diagnostics. SQLite is the local canonical private store and needs independent backups. These controls reduce accidental exposure; they do not defend plaintext continuity against hostile processes running as the same user or privileged actors.

Destructive transitions must be previewable, backup-first, verified afterward, represented by a rollback manifest, and safe to rerun within their stated preconditions. Ambiguous or unsupported records remain explicit. Restore is rehearsed in isolated state; activation is a separate, quiesced operator action followed by verification. Running old and new writable stores together is not a restore strategy.

Supported legacy Markdown migration remains one-way, never a live mirror. Removal requires an out-of-tree exact-byte backup with a SHA-256 manifest and verification that source bytes still match. Restore uses those original bytes, not rendered approximations. Database transactions do not make Markdown file deletion crash-atomic or transactional. The [migration guide](../../content/skills/loaf-reference/references/markdown-migration.md), [recovery guidance](../../content/skills/loaf-reference/references/troubleshooting.md), and [migration source and tests](../../internal/state/markdown_migration.go) retain exact compatibility mechanics.

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

The isolated [migration rehearsal](../../vnext/migration/rehearsal/import.go) covers only part of the domain and has no activation surface. Full-domain rehearsal, quiesced manual activation, and post-activation verification remain pending. The [bootstrap command](../../vnext/internal/command/command.go) exposes introspection only.

Exact schemas, canonical encodings, digest domains, migration identifiers, and compatibility literals remain in source and tests. The [kernel](../../vnext/internal/kernel/kernel.go) reports schema `vnext/1`, while the [continuity schema](../../vnext/continuity/sqlite/schema.go) uses line `vnext` at version 11. Resolving that mismatch is pending compatibility work, not an editorial rename. No stored records, archives, commands, or compatibility bytes change as part of this documentation refactor.
