# Authority Boundaries

Loaf assigns authority by purpose, not by file format or whether something can be queried. Shared work, Git-authored material, operator-private continuity, and temporary output have different owners. These boundaries govern new work; shipped compatibility commands do not extend their authority.

## Current Model

| Authority | Owns | Boundary |
|-----------|------|----------|
| Native tracker | Shared work identity, title, problem definition, definition of done, out-of-scope, status, hierarchy, dependencies, assignment, and collaboration | No private agent-memory store or second local work definition |
| Loaf | Flow methods, skills, templates, profiles, stable project identity, private continuity, derived context, and private synchronization | No tracker credential broker, provider API proxy, or writable tracker mirror |
| Git | Code, tests, configuration, executable generators, deliberately authored documents, and promoted artifacts; implementation history | Not collaborative workflow state or high-churn operator-private continuity |
| Harness | Execution, model selection, tool boundaries, exposed service connections, and their credentials | Does not redefine the tracker, Git, or Loaf data model |

The Flow methods are Loaf-owned, but their shared work record is tracker-owned. The agent uses the selected `project-management/v1` provider skill through a connection the harness already exposes and authenticates. Loaf neither configures provider authentication nor calls a provider API on the agent's behalf. Provider-specific field and relationship mappings stay in the provider skill.

Public intent and selected work share the tracker but carry different authority. Backlog preserves intent without authorizing execution; Todo means shaped and explicitly selected. A complete definition does not itself select work. The [Flow semantics](../../vnext/content/skills/loaf-reference/references/flow-semantics.md) own the ceremony rules. For GitHub, the configured Projects v2 Status field supplies these lanes alongside Issue state; Issue open/closed is not a substitute. Missing lane access is a configuration or connection gap, never permission to invent a local queue.

When the necessary connection or native capability is unavailable, the tracker-mutating operation stops and reports the limitation. It does not invent a local fallback or report approximate fidelity as exact. Material writes require authoritative native readback, and an ambiguous result must be resolved before retrying a non-idempotent operation.

## Artifact Placement

- **Shared work** lives in the native tracker, including its definition and workflow state. A design document can support work without becoming another work identity.
- **Durable authored material** lives in Git, including architecture topics, deliberately selected ADRs, maintained knowledge, and executable generators. Authored Markdown is not presumed to be a generated projection merely because it is text.
- **Private continuity** includes journal entries and wraps, sparks, ideas, explorations, decisions, findings, handoffs, and supporting opaque references or verification evidence. It may retain a native reference for resumption, but never a mirror of the tracker definition or workflow state. Context is derived at read time. [Private continuity](private-continuity.md) owns the persistence model; Scratchpad remains deferred.
- **Temporary reports** belong to their producing skill and may remain harness output or temporary files. They have no universal database entity, status vocabulary, synchronization contract, or lifecycle CLI. The skill owns any template. Housekeeping may recommend leaving a report, extracting its durable conclusions, deleting it, or promoting it to `docs/reports/`; deletion and promotion require user approval. Deliberate promotion makes the authored result Git-owned, not a report subsystem.

Private synchronization does not transfer tracker work or harness credentials. Its independent trust and recovery constraints remain in [Private Synchronization](private-synchronization.md).

## Knowledge and Tool Boundaries

Loaf-owned maintained knowledge and private continuity have explicit authorities; harness tools and user-managed memory do not become Loaf stores. Loaf does not require a separate retrieval product as an architectural dependency. QMD may remain an optional integration, but cannot define knowledge authority or portability. Reusing its collection and search features could avoid implementation work; making it mandatory would impose a portability constraint without demonstrated need. Removing the existing integration was not selected or authorized. Optional integrations still need explicit capability and degradation behavior.

A Serena-specific rule reserving it for code intelligence and forbidding its memory surface was rejected as portable architecture: it addressed one drift risk by prescribing a particular harness tool. Users remain responsible for clarifying authority when they deliberately add memory tools. Loaf does not uninstall, erase, or repurpose that memory. [Knowledge Management Design](../knowledge/knowledge-management-design.md) carries repository naming, metadata, and current retrieval mechanics rather than another architecture policy.

Agents may draft through skills and deterministic CLI operations; the boundary is human authority over acceptance, change, publication, and deletion, not human-only authorship or a ban on CLI state writes. The [vision's bounded-autonomy principle](../VISION.md#bounded-autonomy) and owning skills control those actions.

## Rationale and Tradeoffs

One authority per kind of truth avoids reconstructing a shared work definition from competing stores. It also keeps private continuity outside collaboration systems and leaves credential handling with the harness that already provides it. The cost is explicit coordination across tracker, Git, continuity, and temporary outputs instead of a universal query or mutation surface.

The earlier models explain why these boundaries matter:

- **Local specs with tracker tasks** split one definition between Git and the tracker, leaving agents to reconstruct which part controlled the work.
- **Loaf-owned issue identity with tracker reconciliation** retained local bodies, completion criteria, claims, or status. This created a second authority and a synchronization burden.
- **Loaf as a provider API client** would centralize transport but duplicate credential, authentication, ambiguous-retry, and provider API maintenance already handled by the harness.
- **Every queryable entity in SQLite** made local querying uniform but displaced tracker collaboration, authored documents, and lightweight report ownership. Keeping generators in Git was useful; classifying every noun as a database entity was not.
- **Everything as files in Git** would make private, high-churn continuity branch-bound and poorly suited to cross-environment resumption, while failing to provide native tracker collaboration.

These are retained reasons for the current model, not new product choices made by this documentation refactor.

Local Change files, release cohorts, and canonical pitch briefs were retired because they duplicated native shared-work scope and workflow state. Pitch remains problem discovery, with method and templates owned by its skill. Triage moves private intent to Backlog; an explicit filing request during pitch uses that same path. Shape creates or updates the work definition, and execution selection remains explicit. Historical artifacts are evidence, not material to automatically promote back into current authority. Git remains the implementation witness, ship is the sole quality gate before a verified native transition, and releases describe already-landed work. [Runtime and Delivery](runtime-and-delivery.md#reviewed-source-and-release-evidence) preserves the lasting content-bound evidence rule without retaining cohort receipts or their schema as a second release authority.

## Migration and Compatibility

Supported historical local work may move once to the selected native tracker through an explicit, previewable, agentic migration. Preserve a rollback archive, verify material native writes by readback, and make unsupported or ambiguous dispositions explicit. After cutover, the local representation stops being writable work authority.

There is no ongoing local-to-tracker synchronization, reconciliation loop, dual read, or dual write in the new-work model. Historical local issue, task, and report commands may remain shipped for compatibility; their existence is not authorization to use them as current Flow infrastructure. This documentation move does not migrate stored records, delete archives, or retire commands.

## Implementation and Gaps

The [provider contract and skills](../../vnext/content/skills/project-management/SKILL.md) implement tracker-native Flow guidance. The [shared build overlay](../../internal/cli/build_codex.go) promotes that guidance into every generated target. The isolated [kernel ownership contract](../../vnext/internal/kernel/kernel.go) and [Flow contract tests](../../internal/vnextflowcontract/) encode the intended authority split.

The public [Go CLI](../../internal/cli/cli.go) still exposes historical local work and report commands, and the [state package](../../internal/state/) still contains their records and compatibility integrations. Removing them safely remains migration and cutover work, not an already completed consequence of adopting this topic.

Pre-cutover [private continuity code](../../vnext/continuity/) exists, but its complete public activation remains pending as described in the [system map](../ARCHITECTURE.md#shipped-and-pending-boundaries). A built skill or passing package test is not evidence that every operator journey has shipped.

For workflow operation and supported compatibility procedures, see [Loaf Flow](../knowledge/loaf-flow.md), [Shared Work Model](../knowledge/work-model.md), and [Work Records and Compatibility](../knowledge/task-system.md). Those guides consume this authority model rather than defining another one.
