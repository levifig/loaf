# Loaf Architecture

Loaf is a portable operating framework for agent-assisted software work. It packages methods, skills, templates, profiles, hooks, and deterministic local services for several harnesses while leaving shared work and service credentials with their proper authorities.

This document is the current system map and entry point to owning architecture topics. It distinguishes the shipped system from agreed architecture still being integrated; agreement is not proof of public or complete implementation. Topics own the evolving model and link to deliberately retained narrow decisions where their separate context is useful.

## Contents

- Authority Model
- System Map
- Shipped and Pending Boundaries
- Loaf Flow
- Private Continuity
- Private Synchronization
- Migration and Recovery
- Build and Distribution
- Managed Installation
- Compatibility Names
- Decision Records

## Authority Model

The tracker owns shared work; Git owns implementation and authored material; the harness owns execution and service connections; Loaf owns methods and private continuity. [Authority Boundaries](architecture/authority-boundaries.md) owns the detailed model, artifact placement, rationale, and compatibility limits. No second writable tracker authority is introduced across these boundaries.

## System Map

### Public runtime

The shipped CLI enters at [`cmd/loaf/main.go`](../cmd/loaf/main.go) and dispatches through [`internal/cli/`](../internal/cli/). The Go runtime currently includes installation, build, project identity, SQLite state, compatibility migrations, search, journal and other local record commands, and the shipped synchronization surface.

The shipped CLI still exposes historical local work and record commands. Those commands are frozen migration compatibility, not the authority for new shared work. The Loaf Flow installed into supported targets routes new shared work to the native tracker.

### Accepted destination under integration

The isolated source tree at [`vnext/`](../vnext/) contains machine-checked destination contracts for authority boundaries, private continuity, migration rehearsal, private synchronization, and tracker-native Flow content. Its product is Loaf; the directory name is a temporary source boundary, not a product edition or permanent architecture term.

The bootstrap command at [`vnext/cmd/loaf`](../vnext/cmd/loaf/) intentionally supports only identity and ownership introspection. Stateful destination journeys are not publicly cut over merely because their packages and tests exist.

### Portable content and generated targets

The [Skill Portability](architecture/skill-portability.md) topic owns shared authoring, target adaptation, rationale, and validation boundaries. Current target builders produce:

- `plugins/loaf/` for Claude Code;
- `dist/opencode/`;
- `dist/cursor/`;
- `dist/codex/`;
- `dist/amp/`.

## Shipped and Pending Boundaries

| Area | Verified current state | Accepted direction or remaining gap |
|------|------------------------|-------------------------------------|
| Runtime | Public CLI, development launcher, build, and packaging tooling are Go; Node remains for harness capability runners | Go remains the sole product runtime |
| Distribution | Native archive/checksum packaging, curl installer, Homebrew tooling, and Claude onboarding are implemented; native binaries are untracked | Cross-platform live delivery, signature verification, runtime compatibility, and rollback require operator evidence; harness runtime invocations must use user-managed PATH |
| Shared work | Generated installed Flow content is tracker-native; shipped local work commands still exist | Native tracker is the sole shared-work authority; local commands remain one-time migration compatibility only |
| Private continuity | Shipped journal and local record stores exist; pre-cutover fact persistence and tests exist under `vnext/` | Cut over the full private-continuity domain without changing stored compatibility identifiers |
| Migration | Compatibility importers and a partial, non-activating destination rehearsal exist | Cover every supported private record, rehearse restore in isolation, activate only while quiesced, and verify after activation |
| Private sync | Shipped sync code exists; destination protocol, crypto, relay, coordinator, and vectors exist under `vnext/` | No public destination attach, network sync, or serve journey yet; adversarial review and operator recovery journeys remain required |

This table prevents architecture prose from treating accepted decisions as completed delivery.

## Loaf Flow

The Loaf Flow is:

1. **Pitch** discovers the problem and desired outcome.
2. **Shape** creates or updates the canonical native tracker record through the selected provider skill, including the supported work definition, definition of done, out-of-scope, and hierarchy.
3. **Implement** reads that record and changes the repository through ordinary Git mechanics.
4. **Ship** is the sole quality gate and updates tracker status only after verification and provider readback.
5. **Release** describes a coherent set of already-landed work.

The methods are Loaf-owned; shared work identity and state are tracker-owned. A one-time migration may move supported historical local work into the tracker, but no ongoing synchronization or parallel work record follows.

Progress should be delivered as rideable increments: end-to-end operator journeys that are useful and testable on their own. A layer, schema, or package is evidence toward a journey, not a substitute for one.

## Private Continuity

[Private Continuity](architecture/private-continuity.md) owns the private record domain, project and entity identity, append-only facts, deterministic projections, journal semantics, and local storage guarantees. The complete model remains pre-cutover; shipped journal behavior and destination persistence are distinguished there.

## Private Synchronization

[Private Synchronization](architecture/private-synchronization.md) owns the opaque relay boundary, authenticated facts, atomic application, membership, and recovery limits. Exact security construction and activation requirements remain in the [Private Sync Threat Model](security/private-sync-threat-model.md). Destination protocol code is not a public attach, network sync, or serve journey.

## Migration and Recovery

[Private Continuity's recovery boundary](architecture/private-continuity.md#local-storage-and-recovery) preserves preview, backup, transactional database writes, exact legacy source bytes, rerunnable decisions, isolated rehearsal, and quiesced activation. [Authority Boundaries](architecture/authority-boundaries.md#migration-and-compatibility) owns one-time tracker migration without ongoing dual authority.

## Build and Distribution

[Runtime and Delivery](architecture/runtime-and-delivery.md) owns the Go runtime boundary, verified native delivery direction, version identity, and current gaps. It links the deliberately retained Go-language and verify-only CI decisions. [Skill Portability](architecture/skill-portability.md) owns content packaging across harness targets.

## Managed Installation

[Managed Installation](architecture/managed-installation.md) owns shared skill discovery, proven ownership before updates, canonical root project instructions, and the still-unresolved configuration proposal. Native runtime delivery remains separate from installing content into harness environments.

## Compatibility Names

The label `vnext` still appears in source paths, package paths, schema lines, archive or digest constants, tests, and generated compatibility content. Those literals may be stored or signed protocol inputs. Product-facing prose calls the product Loaf, but implementation cleanup must not rename compatibility bytes without an explicit migration and vector update.

Migrated or retired ADR files need not remain as pointers: Git preserves their history, while maintained references point to the current owning topic. Historical links can be resolved against the Git revision that contained the record.

## Decision Records

Read current topics first, then follow relevant decision links. The [decision index](decisions/README.md) lists the retained Go-runtime and verify-only CI choices. Their detailed alternatives and decision context remain there; the owning topic carries current implementation and delivery context. New records remain rare and deliberate, not an automatic result of architectural work. Exact schemas, protocol constants, command syntax, and test vectors live beside code; shared work scope and status belong in the tracker.
