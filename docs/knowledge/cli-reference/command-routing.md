---
topics:
  - command-routing
  - compatibility
  - tracker-native
covers:
  - internal/cli/cli.go
  - internal/cli/cli_reference.go
  - internal/cli/frozen_work_model.go
  - content/skills/loaf-reference/references/artifact-routing.md
  - content/skills/migrate-project/SKILL.md
last_reviewed: '2026-11-06'
---

# Command Routing

Choose the authority before choosing a command. The generated command index records implemented CLI help, including compatibility commands; its presence does not authorize those commands as the current work flow.

| Intent | Route |
|--------|-------|
| Shape new bounded work | Shape writes the canonical native tracker record through the configured Project Management provider |
| Implement selected work | Implement reads the selected native contract and uses repository-native Git mechanics |
| See or change shared work state | Read or mutate the native tracker through the selected provider, then verify readback |
| Continue after a restart | `loaf journal context`; find a specific checkpoint with `loaf journal search <query>` and `loaf journal show <id>` |
| Load a named saved Handoff | In its source project, `loaf handoff list` or `loaf search <query>`, then `loaf handoff show <id-or-alias>` |
| Read a council | `loaf council list/show <ref>` reads its authored file and currently requires initialized SQLite state; an explicit file path can be read directly |
| Review a temporary report or proposal | Read the authored file; no database row, universal lifecycle, or compatibility report command is required |
| Read an explicit snapshot | Read the supplied path and revalidate current authoritative sources before acting |
| Skills or content changed | `loaf build`; installation or upgrade is a separate explicitly scoped action |
| Check knowledge freshness | `loaf kb check` |
| Inspect an existing local task, plan, issue, or report lifecycle record | Use exact command help for historical compatibility; do not create a second shared work authority |
| Import legacy `.agents` Markdown into SQLite | Explicit migration only: `loaf migrate markdown --dry-run`, then authorized `--apply` |

The shipped Loaf Reference's [Artifact Routing](../../../content/skills/loaf-reference/references/artifact-routing.md) topic supplies the family-specific storage and maintenance boundaries. Handoff owns deliberate transfers; Foundations owns ordinary capture. Missing state or a missing CLI is an access limitation, not a reason to invent another store or initialize a fallback for an existing record.

## Legacy Records and Migration

New shared work lives only in the selected native tracker. Historical local `issue`, `task`, `plan`, `intent`, and report-lifecycle commands remain shipped, and their exact help describes available access. Some legacy help still recommends local issues for new work; that recommendation conflicts with the current [Flow](../loaf-flow.md). Use the configured Project Management provider for shared work rather than following that compatibility recommendation.

Historical data may include Change folders, spec Markdown, SQLite work rows, task packets, cohorts, verification receipts, and rendered projections. These surfaces preserve access to existing data and support deliberate migration. Their presence does not make them current work authority. There are no public `loaf change` or `loaf spec` command namespaces; retained data and state exports are separate from command availability.

Moving local work into the tracker is a one-time, agentic, verified cutover through the [migration workflow](../../../content/skills/migrate-project/SKILL.md):

1. Resolve the exact source records and destination tracker project.
2. Preview the mapping before mutation.
3. Preserve source bytes, stable identifiers, provenance, relationships, and status semantics needed for recovery.
4. Write through the selected provider and harness-owned connection, then read back every material write.
5. Resolve ambiguous or unsupported dispositions explicitly, record the cutover, and stop writing the local work representation.

Migration never introduces background push/pull, reconciliation queues, credential storage, ongoing provider mappings, or a writable local twin. Historical Linear compatibility integration is not the provider-neutral current workflow.

Legacy Markdown import into SQLite is a different operation from tracker cutover. Its [migration reference](markdown-migration.md) preserves simulation versus inventory, origin and status authority, and retry limits. The [recovery reference](troubleshooting.md#state-backup-and-restore) preserves exact-byte backups, checksums, isolated restore rehearsal, and quiesced activation. Database transactions do not make file deletion or restoration transactional; restoring source files is a distinct operator-controlled action.

## JSON diagnosis surfaces

Prefer structured output when available:

- `loaf config check --json` — project config and installed hook validity
- `loaf state doctor --json` / `loaf state status --json` — SQLite health and readiness
- `loaf check --hook <id> --json` — one enforcement hook's result
- `loaf check commit-msg|secrets|changelog|compliance|test-naming|dockerfile|k8s-manifest|bounds|units|standard-refs --json` — standalone validators
- `loaf kb check --json` — knowledge staleness against Git history
- `loaf journal recent --json` — private timeline; current shared work comes from the provider
- `loaf migrate markdown --dry-run --json` — migration simulation/inventory, not ongoing synchronization

The [troubleshooting reference](troubleshooting.md) covers diagnostic selection and isolated state. Compatibility readiness checks describe legacy local records; they do not verify a native work contract.
