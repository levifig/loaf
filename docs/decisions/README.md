# Architecture Decision Records

Start with the [architecture overview](../ARCHITECTURE.md). Living topics own the current system model; the records below preserve two narrow choices whose original circumstances and alternatives remain useful across later implementation changes.

| Decision | Why Keep a Separate Record | Owning Topic |
|----------|----------------------------|--------------|
| [Use Go for the Loaf Runtime](ADR-014-go-for-stateful-runtime.md) | Preserves the runtime-language commitment and the TypeScript/Rust tradeoffs separately from delivery mechanics | [Runtime and Delivery](../architecture/runtime-and-delivery.md) |
| [CI Verifies Source Artifacts Instead of Repairing Branches](ADR-012-ci-verify-only-build-artifacts.md) | Preserves the branch-protection failure and rejected auto-repair approaches across changing publication paths | [Runtime and Delivery](../architecture/runtime-and-delivery.md#reviewed-source-and-release-evidence) |

New standalone ADRs are rare, deliberately selected records of one consequential choice, not a catalogue of subsystems or routine changes. Acceptance records agreed direction, not proof of implementation.

Migrated and retired records have been removed after preserving their useful content in topics or owning guidance. Their original text and prior applicability remain in Git history; no redirect or tombstone catalogue is maintained.
