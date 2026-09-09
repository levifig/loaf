# Strategy

_Last updated: 2026-09-05_

Loaf's strategy is to make trustworthy agent-assisted work feel smaller, clearer, and easier to resume across tools. The unit of progress is a rideable operator journey: an end-to-end capability that is useful before the next layer exists.

## Contents

- Who This Serves
- Strategic Commitments
- Delivery Sequence
- Product Boundaries
- Deliberate Deferrals
- Open Decisions

## Who This Serves

Solo developers need continuity and disciplined execution without maintaining a bespoke orchestration system. Teams need consistent agent behavior, native collaboration surfaces, reviewable decisions, and quality gates that do not hide their evidence. Harness authors and integrators need portable content with narrow, explicit adapter contracts.

## Strategic Commitments

### One authority for each kind of truth

The native tracker owns shared work. Git owns implementation and deliberately promoted artifacts. Loaf owns methods and operator-private continuity. The harness owns execution and provider credentials. Crossing a boundary uses the owner's native surface; it does not create a mirrored database or synchronization loop.

### Portable skill bodies, native adapters

Common methods and knowledge are authored once. Product-specific configuration, capability tokens, and integration behavior stay in labeled sections, sidecars, builders, or tests. Native leverage earns its maintenance cost only when users can observe and verify the improvement.

### Deterministic mechanics in Go

Skills guide judgment; the Go runtime performs deterministic Loaf-owned operations such as project identity, continuity, migrations, diagnostics, builds, and recovery. JavaScript can remain development and packaging tooling without making npm the intended product delivery channel.

### Private continuity without session machinery

The private domain includes journal entries and wraps, sparks, ideas, explorations, decisions, findings, and handoffs. It is project-scoped and available across worktrees. Current context is derived from facts at read time; there is no active-session entity or lifecycle.

Continuity synchronization is local-first, encrypted, authenticated, project-scoped, and fail-loud. Tracker records and provider credentials never enter it.

### Verification proportional to irreversibility

Read-only explanation and audit stay non-mutating. Data transitions require preview, backup, transactional database boundaries, verification, a rollback manifest, and rerunnable decisions. Protocol and cryptographic work require executable vectors, adversarial tests, recovery journeys, and independent review before public activation.

### Reports remain temporary outputs

Most skill results return through the harness. When a report must persist, its skill owns the template and purpose. Housekeeping can recommend retention, extraction, deletion, or deliberate promotion, but the user approves every destructive or durable move. Reports gain no universal database row, synchronization behavior, or lifecycle command.

## Delivery Sequence

The sequence favors journeys that produce immediate dogfood evidence while keeping authority and recovery intact.

1. **Tracker-native local Flow and complete private continuity.** A developer can pitch, shape into the selected tracker, implement through Git, ship after verification, and resume from every supported private record. Complete the one-time migration and activation journey without a local work mirror.
2. **Verified native installation and update.** A developer can install the correct GitHub Release binary through a verified installer, Homebrew, or a harness integration, with one effective upgrade owner and an explicit rollback path. Resolve bootstrap, platform selection, verification, and update mechanics as part of this journey.
3. **Two trusted clients for one project.** A developer can enroll a second client, synchronize private continuity, detect tampering or gaps, revoke it, rotate forward, and recover from interruption while the local replica remains usable.
4. **Hosted-client lifecycle.** A developer can attach an ephemeral or hosted client with scoped credentials, complete useful work, promote collaboration through tracker and Git, and leave no silent memoryless state behind.
5. **Multiple private realms and broader harness coverage.** Generalize only after the single-operator, single-project journeys have produced enough operational evidence.

Each increment must include diagnostics and recovery. A storage layer, transport, or installer component does not count as delivery until an operator can ride the complete path.

## Product Boundaries

Loaf does not become a tracker adapter service. Provider interaction remains agentic through the selected `project-management/v1` skill and the harness-owned authenticated connection. Loaf also does not configure or retain provider credentials.

Historical local issues, tasks, Changes, release cohorts, and receipts are migration or compatibility inputs, not continuing product authority. A verified one-time transition may preserve them, after which work stays tracker-native.

The accepted private-continuity and synchronization architecture has pre-cutover source and tests, but the public CLI is not yet cut over to the complete persistence model and does not expose the destination network attach, sync, or serve journeys. Strategy and release notes must keep that distinction visible.

## Deliberate Deferrals

- Scratchpad remains outside the continuity contract until its intended authority and lifecycle are settled.
- Shared or team memory is outside the private-continuity product boundary; teams collaborate through trackers and Git.
- Broad hosted services and multi-realm abstractions wait for local and two-client journeys to prove their requirements.
- Exact installer ownership, update, and rollback mechanics are not chosen by documentation alone.

## Open Decisions

- Resolve the [configuration proposal](architecture/managed-installation.md#unresolved-configuration-proposal), including input categories, precedence, and identity binding.
- Define the supported installer bootstrap, artifact verification, upgrade-owner, and rollback behavior in [Runtime and Delivery](architecture/runtime-and-delivery.md#native-delivery-direction-and-gaps).
- Refine major-zero minor-versus-patch release judgment and the eventual `1.0.0` readiness bar under [version identity](architecture/runtime-and-delivery.md#version-identity).
- Complete the private-continuity migration scope, isolated restore rehearsal, quiesced activation, and post-activation verification.
- Validate protocol performance limits, recovery ergonomics, key lifecycle behavior, and deletion compatibility without weakening the closed fact model.

The [architecture overview](ARCHITECTURE.md) links owning topics for the current model, rationale, and implementation boundaries, plus deliberately retained narrow decision records.
