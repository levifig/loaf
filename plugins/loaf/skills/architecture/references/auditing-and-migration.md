# Auditing and Migrating Architecture Documentation

Use this reference for corpus audits and authorized migrations to topic-based architecture. An audit is read-only unless edits are requested. Updating the skill or adopting its convention does not authorize relocating or deleting an existing repository's documents.

## Contents

- Establish the evidence set
- Consolidate by meaning
- Retire old homes deliberately
- Reconcile authority and implementation
- Preserve compatibility and guarantees
- Verify the corpus

## Establish the evidence set

1. Discover the repository's architecture, vision, strategy, glossary, technical references, and legacy decision-record locations.
2. Inventory documents by path, subject, applicability, current authority, and active citations. Record legacy IDs, statuses, dates, and replacement links when present; do not impose them on topics.
3. Read relevant implementation, tests, technical references, and current workflow guidance. An old `Accepted` label is evidence of a past choice, not proof that its assumptions remain current.
4. Separate explicit user decisions from audit recommendations and unresolved questions. Do not turn a reviewer's proposal into an accepted decision without authority.

## Consolidate by meaning

Propose the smallest coherent topic map. The mapping can be many-to-one or one-to-many: several ADRs may describe one domain, while one oversized record may contain architecture, a protocol specification, and an incident history. A one-file-per-ADR rename leaves the organising model unchanged.

- **Retain/revise** current models, constraints, and rationale in their owning topics.
- **Retain a narrow ADR** when its separate decision context still earns that home under Architecture's decision-record rules. Link it from the current topic rather than duplicating its rationale. Do not create a new ADR for every extracted choice.
- **Consolidate/split** when the subject boundaries need to change, preserving relevant reasons and alternatives in each destination.
- **Rehome** exact technical detail or non-architectural material to its real owner, with verified links.
- **Retire** obsolete guidance from the active model after checking applicability and preservation of relevant rationale.

Use a temporary mapping for a substantial migration so each source's useful content and active citations are accounted for. It is verification evidence, not a new permanent decision registry. Do not invent a lost rationale; label reconstruction as inference and resolve consequential uncertainty with the user.

## Retire old homes deliberately

Before removing or archiving an old file:

1. Verify that current constraints, relevant rejected alternatives, and guarantees have working destinations. Retiring an ADR as a document does not retire the constraint it described.
2. Inspect inbound references in code, tests, documentation, skills, and known external consumers. Update maintained references to the owning topic; for version-specific claims, use an appropriate historical revision instead of implying unchanged meaning.
3. Confirm the original text is recoverable in Git or an approved archive. Do not rely on Git for uncommitted material or silently delete useful evidence.
4. Obtain authorization for the retirement operation. If old public links need compatibility, retain a clearly historical pointer at the old location or an agreed redirect. It must name the current home without restating the active model or becoming another maintained authority.
5. Remove obsolete ADR-creation routes from applicable maintained guidance within the migration's scope. Keep historical IDs and supported compatibility references truthful.

An existing corpus can remain while migration is deferred. Do not create parallel conflicting topics or declare migration complete until ownership and citations are reconciled. Do not require a new ADR to record the change of documentation convention.

## Reconcile authority and implementation

For each topic, establish:

- its domain and applicability;
- the agreed architectural invariant;
- the important tradeoffs and credible alternatives;
- whether it is implemented, partially implemented, or accepted but pending;
- the current code, test, technical-reference, workflow, or tracker surfaces that carry it.

Shared work identity, definition, completion criteria, hierarchy, status, assignment, and collaboration belong in the selected native tracker when that is the project's authority model. Use the provider-neutral shaping workflow and harness-owned connection. A technical design may be committed when it has independent value, but it does not become another work identity or synchronization surface.

The architecture overview describes the system that exists now and accepted constraints on its evolution. Strategy describes current bets, operator journeys, deferrals, and learning sought. The native tracker owns execution status and sequencing. Avoid branch names, temporary construction labels, stale milestone checklists, and completed migration plans in current product vocabulary.

## Preserve compatibility and guarantees

Before changing a name or moving detail, classify each occurrence:

- **Editorial identity** — maintained prose, title, navigation, or a new filename. Use the product or domain name.
- **Source compatibility** — existing import path, package directory, command, or linked file. Keep the truthful name until code moves with compatibility coverage.
- **Stored compatibility** — schema line, column, persisted kind, archive member, or serialized value. Inventory supported data and design migration/recovery before changing it.
- **Cryptographic binding** — AAD, key-derivation label, signature transcript, credential prefix, protocol version, or test vector. Treat byte changes as protocol changes and require new vectors plus supported-data handling.

When consolidating documents, preserve security and storage guarantees such as authority boundaries, encryption and authentication properties, atomicity, retry behavior, recovery limits, revocation limits, and explicit residual risks. Move exact algorithms, field layouts, SQL, limits, and vectors only to an implementation-adjacent source that exists and is linked. If the destination does not exist, keep the detail or create a separately authorized technical reference; do not discard it.

## Verify the corpus

- Every source's meaningful content has a verified destination or an explicit, authorized retirement disposition.
- The overview links to owning topics without duplicating their substantive content. Deliberately retained narrow ADRs are reachable from those topics; the chronological catalogue is not a competing current-state model.
- Every Markdown link resolves or intentionally targets an external authoritative source.
- Maintained citations point to current topics or intentionally to historical revisions; required compatibility pointers identify the new home clearly.
- Retained ADRs preserve truthful decision status and history under their actual convention; topics do not inherit that lifecycle.
- No record invents an acceptance date, rejected alternative, implementation result, or security property.
- Current guidance contains no active dependency on retired work models.
- Agreed pending direction has precise implementation gaps instead of false shipped-state claims. Follow-up work remains within the user's authority and the selected native tracker model.
- Compatibility-sensitive identifiers remain unchanged unless their migration is part of the authorized scope and is verified.
