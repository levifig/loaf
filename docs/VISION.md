# Loaf

_Last updated: 2026-09-05_

Loaf is an opinionated operating framework that makes AI-assisted software work structured, portable, durable, and safe to resume.

## Core Pillars

### Portable knowledge

Author skills once and deliver them to supported harnesses. Skills express shared methods and domain knowledge; profiles, sidecars, hooks, and target builders adapt only where a harness has a real product-specific boundary.

### Tracker-native shared work

The Loaf Flow is pitch → shape → implement → ship → release. Loaf owns the methods and templates. The selected native tracker owns shared work identity, definition, definition of done, out-of-scope, status, hierarchy, dependencies, assignment, and collaboration. Git owns implementation and deliberately promoted artifacts. The harness owns execution and service credentials.

An agent uses the tracker connection already exposed by its harness. Loaf does not proxy provider traffic, store provider credentials, or maintain a parallel local work record.

### Rideable progress

Loaf evolves through narrow, complete operator journeys. Each increment lets a real user do something useful end to end while preserving security, determinism, recovery, honest failure, and data safety. Packages and schemas are evidence toward a journey, not milestones by themselves.

### Bounded autonomy

Profiles constrain mechanical access. Skills provide judgment and method. Native tracker records bound shared scope, and executable checks provide evidence. Human decisions remain explicit where authorization, ambiguity, or irreversible state requires them.

### Private continuity

One operator's journal entries and wraps, sparks, ideas, explorations, decisions, findings, and handoffs form private, project-scoped continuity. It survives context loss, branches, worktrees, harness restarts, and—after explicit attachment—other trusted clients without becoming team memory or tracker state.

Context is derived at read time. There is no session entity or lifecycle. Scratchpad is deferred until its authority and lifecycle are deliberately defined.

Synchronization is private, authenticated, and fail-loud. A relay cannot read plaintext or become a signing authority. Collaboration with other people happens through explicit tracker and Git surfaces, not by sharing the private continuity store.

### Purposeful artifacts

Durable authored knowledge and decisions live in Git. Temporary reports exist only when a skill result needs to outlive the response; the producing skill owns the template, and the user approves cleanup or promotion. Loaf does not turn reports into a universal state subsystem.

## What Success Looks Like

A developer can:

- install a verified native Loaf binary without treating npm as the product distribution channel;
- use the same well-defined skills across supported harnesses;
- move a problem through a tracker-native Flow without duplicate work records;
- resume from private continuity without leaking it into collaboration systems;
- attach another trusted client through explicit, recoverable, authenticated steps;
- understand failures and recover without silent data loss or degraded authority;
- ship useful vertical journeys before broad infrastructure is generalized.

## What Loaf Is Not

Loaf is not a prompt library, a tracker, a provider credential broker, a shared team-memory product, or a second source of truth for Git. It is opinionated about how agent-assisted work is bounded, executed, verified, resumed, and released; it is not opinionated about the product being built.

## Product Boundary

| Authority | Canonical responsibility |
|-----------|--------------------------|
| Loaf | Flow methods, portable skills, templates, profiles, project identity, private continuity, derived context, private synchronization |
| Native tracker | Shared work identity, definition, status, hierarchy, assignment, collaboration |
| Git | Code, tests, configuration, authored documents, promoted artifacts |
| Harness | Execution, model selection, tool boundaries, service connections and credentials |

Each responsibility has one owner. Migration may move supported historical data once; it never creates ongoing synchronization between authorities.
