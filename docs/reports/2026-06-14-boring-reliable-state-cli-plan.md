# Boring Reliable State And CLI Plan

Date: 2026-06-14 12:37
Status: Planning checkpoint
Scope: Single global SQLite state, project identity, migrations, repair UX, backup/export, backend mappings, and human/agent CLI contracts.

This plan pauses opportunistic hardening and turns the remaining work into a measurable reliability contract. The current codebase has made real progress: the native Go runtime is authoritative, state lives in one global SQLite database, project identity is durable and path/name independent, and many state commands already have JSON contracts, human output, and tests. The remaining risk is unevenness: one surface can be excellent while a nearby surface still has unclear output, weak repair guidance, or missing matrix coverage.

## Current Baseline

| Area | Current Evidence | Remaining Concern |
|---|---|---|
| Native runtime | `docs/reports/2026-06-10-native-go-cutover-test-map.md` maps public command coverage and native-only guards. | The map proves migration breadth, not the final reliability contract for state and CLI UX. |
| Global DB location | `README.md` and SPEC-040 document one global SQLite file under XDG data home, partitioned by stable `project_id`. | Need final audit that every state/project command reports this consistently in JSON and human output. |
| Project identity | `loaf project show/list/rename/move` support stable ID, friendly name, current path, and path move safeguards. | Need command-matrix verification that rename/move never creates accidental identities and gives actionable errors. |
| State health | `loaf state status/path/doctor` expose global scope, project details, diagnostics, and repair plans. | Need repair-plan command validation so every suggested command is executable in the diagnosed state mode. |
| Backup/export | `state backup`, `backup verify`, and `export all` verify integrity and include project identity context. | Need restore/import confidence story: backup verification is strong, but restore procedure and test coverage are not yet first-class. |
| Backend/Linear mappings | Doctor validates backend mapping drift, project-level mappings, unknown sync statuses, and Linear unmapped local tasks. | Need a fuller backend consistency policy: what is invalid, warning-only, repairable, exportable, or delegated to future Linear sync. |
| Agent JSON | Many commands include `contract_version`, scope, project identity, machine-readable errors, and stable empty arrays. | Need a command matrix that proves all critical state/project/migration paths return JSON when `--json` is requested, including failures. |
| Human CLI | Human output has improved scope/project/path/safety labels. | Need a deliberate UX pass for consistency, next actions, and terse but useful repair guidance. |

## Reliability Contract

A state/CLI surface is boring reliable only when all applicable checks below are true and verified by current tests or dogfood output.

### Storage And Identity

- The database path is one global SQLite file under XDG data home: `$XDG_DATA_HOME/loaf/loaf.sqlite`, with platform fallback handled by `PathResolver`.
- Project isolation is row-level via durable `project_id`; no new code treats per-project database files as the primary storage model.
- New project IDs are generated and stable; they are not derived from path, friendly name, remote URL, or branch.
- Friendly project name is mutable and independent of path and ID.
- Current path is mutable via `loaf project move`, with historical paths preserved in `project_paths`.
- Read-only project commands do not create databases or project identities.
- Rejected rename/move operations do not create databases, project rows, path rows, sidecars, or repo files.

### Schema And Migration

- Migrations are ordered, checksummed, transactional, and mirrored into schema docs when schema changes.
- Storage-home migration copies legacy DBs to XDG data home without deleting the source or overwriting an existing destination.
- Markdown migration dry-run does not create SQLite state; apply/resume imports without mutating source Markdown.
- Existing state commands reject invalid schema versions or checksum drift with doctor guidance.
- Any future schema change must include data-preservation tests and explicit repair/upgrade UX.

### Doctor And Repair

- `state doctor` is always safe to run and does not mutate unless an explicit safe `--fix` path is selected.
- `state doctor --json` returns a machine-readable status payload even when exiting nonzero for invalid state.
- Repair plans are non-mutating by default and include `code`, `diagnostic_code`, `description`, `safe`, `applied`, and any relevant `command` or `path`.
- Every repair-plan command is valid in the state mode that produced it. Invalid-state diagnostics must not suggest commands that refuse invalid state.
- Unsafe repairs require dry-run/apply separation, clear human labels, and JSON output.

### Backup, Export, And Restore

- `state backup` creates repository-external `.sqlite` backups, verifies them before success, and reports checksum, bytes, schema version, foreign-key check, integrity check, project count, and current project identity.
- `state backup verify` verifies an existing backup without consulting or mutating live state.
- `state export all --format json` and `state export all --json` produce internal project-scoped snapshots with table order, row counts, project identity, scope, and verification manifest.
- Markdown exports are explicit, deterministic, and boundary-validated when external-safe.
- Restore remains incomplete until there is either a documented manual restore procedure with validation commands or a guarded `state restore` command.

### Backend And Linear Consistency

- Backend mappings store only external identity metadata, not tokens or secrets.
- Project-level backend mappings are valid only when they point at the same durable project ID.
- Task/spec/session/etc. mappings reject unknown local entity kinds and missing local entities.
- Ambiguous mappings and unknown sync statuses are visible diagnostics.
- Linear-enabled projects warn about active local tasks without Linear mappings.
- Repair guidance distinguishes local database repairs from backend sync work that requires future Linear integration.

### Agentic JSON

- Any command that accepts `--json` returns JSON for success and validation/runtime failures.
- JSON errors include `contract_version`, `command`, and `error`.
- State/project/migration JSON success payloads include `contract_version` and global database scope when backed by SQLite.
- Critical project-aware JSON payloads include `project_id`, `project_name`, and `project_current_path` when available.
- Empty collections are stable arrays, not `null` or omitted, unless the field is explicitly optional.
- Exit codes are deterministic: success is `0`; invalid state or validation failures are nonzero while preserving JSON when requested.

### Human CLI

- Human output names the command, scope, database path, project name, project ID, and project path when relevant.
- Dry-run output says no changes were written.
- Apply output says exactly what changed and where.
- Repair output labels safe/manual/applied actions.
- Error text points at the next useful command without implying an unsafe mutation.
- Output is concise enough for humans to scan and structured enough for agents to scrape if JSON is not available.

## Command Matrix

This is the focused audit set for the next iteration. Each row should eventually have tests or dogfood evidence for success, JSON success, JSON failure, and no unintended mutation where applicable.

| Surface | Human Success | JSON Success | JSON Failure | No-Mutation Proof | Priority |
|---|---|---|---|---|---|
| `state path` | Known good | Known good | Needs sampled matrix | Does not create DB | Medium |
| `state status` | Known good | Known good | Needs sampled matrix | Does not create DB | High |
| `state init` | Covered | Covered | Needs sampled matrix | Re-run idempotent | Medium |
| `state doctor` | Improving | Improving | Invalid state returns JSON payload | Does not mutate unless `--fix` safe path | Critical |
| `state repair relationship-origin` | Covered | Covered | Needs sampled matrix | Dry-run/apply split | High |
| `state repair legacy-project-database` | Covered | Covered | Needs sampled matrix | Dry-run/apply split | High |
| `state backup` | Covered | Covered | Covered for missing/invalid state | Creates backup only outside repo | Critical |
| `state backup verify` | Covered | Covered | Covered for invalid/missing path | Does not read live state | Critical |
| `state export all` | Covered; `--json` alias added | Covered | Covered for invalid state and markdown alias misuse | Does not mutate DB or repo files | Critical |
| `state export triage` | Covered | Not applicable | Needs JSON-format misuse checks | Does not mutate DB or repo files | High |
| `state export spec/session/release-readiness` | Covered | Not applicable | Needs JSON-format misuse checks | Does not mutate DB or repo files | High |
| `migrate markdown --dry-run` | Covered | Covered | Needs sampled matrix | Does not create DB | Critical |
| `migrate markdown --apply/--resume` | Covered | Covered | Needs interrupted/resume confidence review | Preserves source Markdown | Critical |
| `migrate storage-home --dry-run` | Covered | Covered | Needs sampled matrix | Does not copy | High |
| `migrate storage-home --apply` | Covered | Covered | Covered for overwrite/partial copy | Preserves legacy source DB | High |
| `project show/list` | Covered | Covered | Covered for missing DB and unknown path | Does not create identity | Critical |
| `project rename --dry-run/apply` | Covered | Covered | Covered for missing DB and unknown path | Preserves project ID | Critical |
| `project move --dry-run/apply` | Covered | Covered | Covered for missing DB, unknown from, missing target | Preserves project ID; one current path | Critical |
| Backend mapping diagnostics | Human via doctor | JSON via doctor | Invalid state returns JSON payload | Diagnostics only | Critical |

## Remaining Work Tracks

### Track 1: Prove The Matrix

Add a focused regression harness or table-driven CLI test that runs the critical command matrix against temporary state. It should assert JSON parseability, command names, contract version, scope fields, exit codes, and no repo mutation for read-only commands.

Progress:

- 2026-06-14: `TestRunnerStateControlPlaneJSONFailureMatrix` covers sampled JSON failure contracts across `state`, `state repair`, `state backup verify`, `state export`, `project`, and `migrate` control-plane commands, including contract version, command name, error text, silent exit code, and no state database creation for pre-open/read-only failures.

Go/no-go: the matrix can be re-run with one command and failures identify the exact command contract that regressed.

### Track 2: Restore Confidence

Backup verification is strong, but restore is not yet a first-class story. Decide between:

- documented manual restore: copy verified backup to the global DB path after backing up the current DB, then run `state doctor`;
- or a guarded `loaf state restore <backup> --dry-run|--apply`.

Recommendation: start with documented manual restore plus tests around backup verification and doctor compatibility. Add a command only when the manual procedure proves too clumsy.

Go/no-go: a user can recover from a bad global DB using a verified backup without guessing which files to copy or which checks to run.

### Track 3: Repair UX

Audit every `RepairAction` and human repair-plan line. Repair commands should be executable in the current state mode, dry-run first unless explicitly safe, and clear about whether they inspect, preview, apply, or require external sync.

Go/no-go: no repair plan points to a command that immediately fails for the same diagnostic state.

### Track 4: Backend/Linear Policy

Write and enforce a small policy for backend mappings:

- local DB invariants that make state invalid;
- warning diagnostics that indicate sync drift;
- external sync gaps that are future Linear work;
- fields that must never store secrets.

Go/no-go: doctor diagnostics and repair/export guidance make it obvious whether the next action is local repair, export/audit, or backend sync.

### Track 5: Human Output Pass

Run the human form of the command matrix and normalize output shape. Prefer terse blocks with command name, scope, project, path, status, and next action. Avoid feature explanations and avoid presenting dangerous operations as casual fixes.

Go/no-go: a user can run the state/project/migration commands without reading docs and understand whether anything changed.

### Track 6: Completion Audit

Only after Tracks 1-5, run a requirement-by-requirement completion audit against:

- this reliability contract;
- SPEC-040 acceptance items;
- native cutover test map guardrails;
- current command output from the primary checkout and temporary fixtures.

Go/no-go: every requirement has current evidence, not just a historical changelog entry.

## Next Best Commit

The next implementation commit should continue Track 1 by extending the command matrix from JSON failure contracts into success/no-mutation contracts for the highest-risk read-only surfaces: `state status`, `state doctor`, `state export all`, `project show/list`, and migration dry-runs. This keeps future fixes anchored in the broad reliability contract instead of isolated edge chasing.
