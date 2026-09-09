# Private Synchronization

## Scope and Implementation Status

The agreed model synchronizes only closed, versioned [private-continuity facts](private-continuity.md) between an operator's trusted clients. Shared tracker work, provider credentials, and harness connections never enter the protocol. The relay is an untrusted opaque transport, not a plaintext authority or an authority that can mint accepted client history.

Pre-cutover protocol code, state machines, and vectors exist under [`vnext/sync/`](../../vnext/sync/). The [destination bootstrap](../../vnext/internal/command/command.go) exposes introspection only: no public destination network transport or supported attach, sync, or serve journey exists. The public CLI's similarly named commands remain separate [compatibility surfaces](../cloud/project-environment-attach.md). No independent security certification has been completed.

The [Private Sync Threat Model](../security/private-sync-threat-model.md) owns detailed assets, adversaries, construction, credential schemas, deletion compatibility, verification requirements, and residual risks. Exact envelope bytes, canonical encodings, domain-separated digests, key derivation, limits, vectors, and state-machine transitions remain beside implementation and tests. The shipped protocol retains its separate [source](../../internal/sync/) and [compatibility threat model](../security/substrate-e2e-threat-model.md); neither is silently redefined by this topic.

## Authentication and Convergence

Each project has an operator-controlled synchronization root and channel. Clients encrypt every envelope with the authenticated-encryption construction defined and tested by the implementation. A separate administrator-certified per-environment signature authenticates the producer: possession of a content key alone grants neither membership nor authorship. Each accepted fact binds its environment and key generation.

A fact is sealed once; exact retries reuse the same sealed bytes. Content digests provide idempotence and detect conflicting reuse. Authenticated encryption and certified signatures, not a checksum, provide integrity and attribution.

Local fact insertion, immutable receipt, environment-head update, and advancement of the applied cursor share one transaction. A crash must not acknowledge unapplied continuity or advance one view alone. Clients fail loudly on gaps, incompatible versions, invalid authentication, key-generation rollback, equivocation, suspicious clock or sequence skew, and conflicting identity. There is no silent memoryless, last-writer, or plaintext fallback.

Relay cursors are pagination aids, never completeness authority. Clients retain authenticated environment frontiers; a lower head, missing known object, changed digest, or changed relay generation is a recovery condition. The relay can withhold content or deny availability even when it cannot forge accepted facts.

## Membership and Recovery

Administration and credentials are project-scoped. Credential classes carry different authority:

- **Ephemeral:** expiring certified environment authority, explicit allowed content generations, and bounded relay access; never persisted by Loaf. It carries no project root, administrator key, owner token, or implicit future live-content generation authority. The threat model specifies its stable deletion-bootstrap key and the resulting future deletion-anchor metadata exposure; that key does not grant future live-content access.
- **Trusted:** the project root and its own environment authority, but no administrator signing key or owner token.
- **Recovery:** full project recovery and administrative authority, held as an offline bearer secret.

New-client attachment is staged and explicitly verified before it contributes history. Revocation stops future acceptance but cannot retract plaintext or keys already copied to a former client. Ordinary generation rotation limits new content access for bounded clients; compromise of a trusted client's project root or the recovery credential requires a new root, channel, all derived keys and tokens, and explicit reattachment. Ordinary rotation cannot repair an exposed root.

A brand-new client cannot independently detect every pre-enrollment relay rollback without a trusted checkpoint. The local SQLite replica remains the durable continuity store and needs independent backup; the relay is neither backup authority nor a recovery substitute. Local plaintext and file-backed secrets retain the same-user compromise limitations described in [Private Continuity](private-continuity.md#local-storage-and-recovery).

Scratchpad remains outside current production and projection. Strict historical deletion verification does not authorize new Scratchpad facts or a deletion producer. The threat model preserves exact historical opening/closing anchors, authenticated tombstone/bootstrap verification, and non-resurrection requirements without reviving that deferred domain.

## Rationale and Tradeoffs

The opaque relay boundary prevents relay compromise from disclosing plaintext or forging certified client history. Deterministic facts, sealed-once retries, and atomic fact/receipt/head/cursor application support convergence under interruption and reordering. Project-scoped administration limits membership and credential blast radius, but enrollment, key management, recovery, and rollback detection remain operational responsibilities with unavoidable limits.

Trusting the relay with plaintext or signing authority would expose continuity or permit forged history. Synchronizing mutable projections would discard history and weaken deterministic convergence. Treating checksums as authentication would let attackers recompute them after modification. Silent fallback when keys or history are missing would hide data loss or change the security boundary without operator consent. These alternatives remain rejected.

Transport implementations may vary only while preserving the authenticated envelope and state-machine contracts. Availability still depends on a relay or another transport path, even though durability does not.

## Activation Boundary

Public activation requires complete operator journeys, migration and recovery rehearsals, adversarial tests, and fresh independent review of the effective implementation. The [threat model's verification requirements](../security/private-sync-threat-model.md#verification-and-review) and [residual risks](../security/private-sync-threat-model.md#residual-risks) remain binding review inputs. Moving architecture prose is neither an implementation change nor security approval.
