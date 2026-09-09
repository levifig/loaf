# Managed Installation

## Current Model

Loaf installs shared methods into harness-owned environments without taking ownership of unrelated user or third-party content. Prefer one shared discoverable user skill home where a supported harness has verified discovery behavior; use native plugin or product-specific packaging where sharing is unavailable or insufficient. Project-local skills and harness runtime plugins remain separate concerns.

The repository-root `AGENTS.md` is the canonical real project instruction file. Harness adapters may expose verified compatibility paths, but not a second independently maintained instruction body. Loaf preserves user-authored text and manages only its own fenced section. Layout repairs must preserve content and use the tested backup and repair paths rather than treating an old location as disposable.

Ownership must be proven before updating or cleaning an installed entry. A shared directory is not wholly Loaf-owned merely because it contains Loaf skills. Exact paths, precedence, supported compatibility links, plugin exceptions, repair prompts, and migration details belong to version-tested adapters. A new harness capability or path change needs current evidence, not extrapolation from another product.

## Rationale and Tradeoffs

One managed skill copy reduces duplicate content and stale leftovers when discovery rules align. Independent copies avoid shared-path assumptions but multiply drift and upgrade work. Clearing a shared directory during upgrade is rejected because it would delete content Loaf does not own. Shared discovery still costs adapter maintenance and needs refreshed capability evidence as harness behavior changes.

One real instruction file avoids the ambiguous ownership and drift of duplicate per-harness files. The earlier `.agents/AGENTS.md` canonical layout left the standard root path indirect and mixed project instructions with Loaf state. Root `AGENTS.md` makes that discovery path authoritative; `.agents/` may still hold project configuration or supported compatibility state without becoming the instruction document.

## Implementation and Limits

The [target installer](../../internal/cli/install_target.go) groups shared-skill destinations and chooses a compatible source. A single shared store can carry only one frontmatter shape; the [canonical-store tests](../../internal/cli/install_canonical_skills_test.go) reject competing sidecar owners and cover foreign-content preservation, conflicts, retry, and path safety. The installer treats built distributions as trusted, stable input: its comparison detects body divergence, not tampering with authorized metadata values or concurrent distribution changes. Build-time authorization and shared authoring are governed by [Skill Portability](skill-portability.md).

The [fenced writer](../../internal/cli/install_fenced.go) checks managed-section structure and fingerprints before replacing owned content. Its [tests](../../internal/cli/install_fenced_test.go) cover preservation, tampering, shared canonical paths, and retries. [Doctor and its layout-repair tests](../../internal/cli/doctor_test.go) cover canonical-root migration, backup preservation, and refusing invalid layouts. These are shipped implementation checks, not a claim that every integration has been proven in every live harness version.

Runtime acquisition is separate from managed content. The [native delivery journey](runtime-and-delivery.md#native-delivery-direction-and-gaps) now includes archive installation and Claude onboarding, but the user remains responsible for choosing the PATH runtime and upgrading it. Managing skill ownership does not authorize replacing a binary or introducing a private executable pin.

## Unresolved Configuration Proposal

The former project-configuration proposal remains unresolved; moving it here does not accept it. It proposes distinguishing optional shareable repository defaults, automation-friendly environment inputs, protected credential inputs, and stable project identity, allowing operation without a mandatory project file when inputs can be safely derived. Provider credentials and connections remain harness-owned under [Authority Boundaries](authority-boundaries.md).

The exact repository filename, supported fields, precedence, identity binding, and mutation rules still need design against real operator journeys before loaders, schemas, or credential inputs change. The current [config command](../../internal/cli/config.go) and [project-resolution package](../../internal/project/) are compatibility evidence, not approval of a new precedence model.

The alternatives already considered remain useful: a mandatory `.agents/loaf.json` offers discoverability but conflates defaults, identity, and secret-adjacent inputs; a package-manifest location couples configuration to Node; environment-only configuration supports automation but lacks a reviewable home for shareable defaults. No replacement is chosen by this refactor. Shared work to resolve the proposal belongs in the native tracker, not another local specification identity.
