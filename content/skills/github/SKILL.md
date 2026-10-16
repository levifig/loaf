---
name: github
description: Maps project-management/v1 operations to GitHub repository Issue semantics through an exposed harness-native connection. Use when the selected canonical tracker is GitHub. Produces verified native outcomes without configuring credentials or a provider client.
---

# GitHub

Apply [`project-management/v1`](../project-management/contract.json) through a GitHub-capable connection already exposed by the current harness. The [capability mapping](capabilities.json) describes semantic ceilings; the runtime connection decides which operations are actually available for the selected repository.

## Contents

- Critical Rules
- Verification
- Destination Diagnosis
- Version Milestones
- Record Syntax
- Quick Reference
- Topics

## Critical Rules

- GitHub is elected as the canonical tracker only by `integrations.github.tracker: true`. Its destination is `integrations.github.repository` (`owner/name`) plus `integrations.github.project_id` (the exact opaque Projects v2 global node ID). Account-only settings, `enabled`, a legacy board title, historical local issue-authority settings, and the local Git remote do not elect GitHub or identify its board. Discover exposed connections and verify this exact destination before mutation. Read the current board title live for display; if it matches the repository or Loaf project name, call it the project board. Never route by title, `Project #N`, or a `users/<login>/projects/<n>` path. Do not decode global IDs or infer object type or ownership from their spelling. A working `gh` authenticated as the project account that can read the destination repository and configured board is a GitHub connection. Another provider's MCP does not hide GitHub.
- Prefer an already configured, usable GitHub MCP that matches the intended account. If none is available, an installed authenticated `gh` in the main agent's permitted shell is a valid exposed GitHub capability, including `gh api`, before asking the user to configure MCP. Unrelated Linear MCP does not determine GitHub authority or hide GitHub. Select the connection matching the intended account before any resource access. Authenticated `gh` fallback does not bypass the account requirement or observed permissions.
- Anchor the expected GitHub account to `integrations.github.account`. Resolve the host from explicit project configuration or user context, never from the repository owner. Inspect only nonsecret account metadata.
- Verify actor identity through the selected connection's own transport before resource access. For MCP, use that connector's trustworthy principal identity or metadata capability; a `gh` identity is not evidence of the MCP actor. If the connector identity is missing, ambiguous, or mismatched, that MCP connection is unusable. The main agent may then try authenticated `gh` fallback, and must separately verify the `gh` actor in the same shell context with `gh api --hostname <host> user --jq .login`, accounting for environment overrides. Stop on a mismatched or unverifiable identity. Repeat actor verification through the actual chosen transport immediately before mutations, after known context changes, and on same-connection readback, naming the explicit repository and host.
- Do not authenticate, refresh, or globally switch accounts, extract credentials, or add routing wrappers. The installed `github-account` hook may still switch the global active account; this guidance does not change, bypass, or isolate that hook. Preflight and readback cannot eliminate a concurrent switching race and cannot prove the mutation actor. If concurrent switching makes identity uncertain, stop.
- Inspect runtime capabilities and permissions for the selected repository. `connection.describe` is only the first signal because GitHub has no reliable non-mutating write-capability probe; use successful safe native reads as additional operation-level evidence. Downgrade an unavailable mapping to `advisory`, `manual`, or `unsupported` without inventing a fallback representation.
- Accept the connection's observed permission evidence without requiring one header family: classic OAuth, fine-grained credentials, and GitHub Apps can expose different permission metadata. Never inspect, print, or infer the credential itself.
- Use repository Issues as canonical work records and the Issue body as the canonical definition. Search with GitHub's `is:issue` qualifier, read candidate matches, and reject every issue-shaped object carrying a `pull_request` property before reading or mutating it as an Issue. Apply the same check to relationship results.
- Use GitHub's native sub-issue relationships for hierarchy and native issue dependencies for directed blocked-by/blocking edges. Arbitrary related edges are unsupported. Never emulate either relationship with labels, task-list prose, links, or comments.
- Orient dependency writes from the blocked Issue: when the target blocks another Issue, write the target as a `blocked_by` edge on that other Issue, then read back both the target's `blocking` collection and the other's `blocked_by` collection.
- GitHub as the canonical tracker requires an active Projects v2 board whose Status field includes at least Backlog, Todo, In Progress, and Done; report In Review when present. Use the [destination diagnosis](#destination-diagnosis) to distinguish a confirmed configuration defect from unavailable or unverified live evidence. Stop affected board-dependent work when the board or Status evidence is incomplete. Do not treat Issue open/closed as Todo or Backlog, and never encode lanes in labels or comments.
- Map Loaf workflow through two native objects together: Issue `state` and `state_reason` (`completed`, `not_planned`, `duplicate` with `duplicate_issue_id`, `reopened`, or `null`) for open/closed, and the board item's Status single-select for lanes. Backlog is public intent with no promise of execution. Todo is shaped and selected. In Progress, In Review, and Done follow the building track.
- Read Issue state, state_reason, board membership, and Status options before a transition. After a lane change, re-read the board item Status. After a close or duplicate, re-read Issue state, state_reason, and native `duplicate_of` when relevant. Adding a newly created Issue to the named board and setting Status Backlog is moving intent onto the board, not a substitute for `work.create` identity.
- Before a `duplicate` transition, read the proposed canonical Issue and reject it if it is missing, ambiguous, or carries a `pull_request` property. After the write, confirm both `state_reason: duplicate` and GitHub's native `duplicate_of` relationship against the exact requested Issue identity; the reason alone is insufficient. If the selected connection cannot read that native relationship, downgrade the duplicate transition rather than claiming exact confirmation.
- Treat a reason-only change on an already-closed Issue as manual or unsupported unless the user authorized a two-step reopen and reclose. Preserve partial evidence if only one of those transitions can be confirmed.
- Respect GitHub's native hierarchy bounds: at most 100 direct sub-issues, at most eight hierarchy levels, and cross-repository sub-issues only when parent and child repositories have the same owner. A connection that cannot safely represent the requested relationship must refuse the change rather than flatten it.
- Do not infer missing capability or permission from a `404` alone; GitHub can conceal authorization failures that way. Return the observed manual, failed, or indeterminate outcome with evidence rather than exposing or probing secrets.
- Keep the title and other Issue fields, body, sub-issues, dependencies, Issue state, board Status, and comments as distinct native semantics.
- Read the current Issue and relevant native relationships before writing; re-read the same Issue and relationship after writing.
- Never install or authenticate a connector, request or store credentials, construct a GitHub client, invoke a direct provider transport, proxy traffic through Loaf, or store a local work record or ongoing mapping. Installed authenticated `gh` is an exposed harness capability, not a Loaf-owned provider client, auth, proxy, or local mirror.
- Never blindly repeat Issue creation or comment append after an ambiguous response. Search or re-read native state first and return `indeterminate` when duplication cannot be excluded.
- For `project-management/v1` operations, return the common result envelope instead of GitHub-shaped success claims. Native milestone operations use their own native evidence without inventing a common-contract operation.

## Verification

- The explicit tracker election and exact repository plus opaque board ID were checked locally, and the [destination diagnosis](#destination-diagnosis) reports the connected live evidence separately. A complete diagnosis confirms the exact repository, active Projects v2 object, current title, observed access, and required Status lanes; unavailable evidence is explicitly unverified.
- The expected account was taken from `integrations.github.account` and the host from explicit configuration or context, never the repository owner. The selected connection matched that account before resource access. Actor identity was verified through that connection's own transport; `gh` identity was not treated as MCP evidence. Missing, ambiguous, mismatched, or unverifiable identity stopped work before resource operations.
- Every requested `project-management/v1` operation maps to all visible runtime capabilities named by the `before`, `execute`, and `after` phases in `capabilities.json`.
- Every targeted Issue and related Issue was confirmed to be an Issue rather than a pull request before mutation, and Issue creation duplicate detection used an `is:issue` search followed by candidate reads.
- A common-contract mutation is `confirmed` only when the final native read shows the intended Issue field, body, relationship, state and state reason, board Status, or comment. A duplicate transition additionally requires native `duplicate_of` readback matching the requested canonical Issue identity. A lane transition is confirmed only when the board Status matches the requested option. Native milestone mutations require the milestone and assignment readbacks described below.
- Missing sub-issue, dependency, state-reason, board, or comment capabilities are reported at their observed fidelity rather than represented through labels, prose, or another field. Confirmed missing routing or Status lanes are configuration gaps; an inaccessible board or absent connection capability is unverified, not proof that the board was deleted. Prompt for the specific unresolved choice or access, and do not invent workflow.
- No connection configuration, credential material, provider transport, local work copy, synchronization path, or persistent mapping was created.
- Written Issue bodies and comments follow the [Record Syntax](#record-syntax) below; a read-back body shows agent notes collapsed below the divider and references as links.

## Destination Diagnosis

Use this procedure for bootstrap and config-aware maintenance when GitHub is explicitly selected. It is read-only: diagnosis does not authorize changing configuration, board fields, permissions, or Issues.

1. Read `loaf config check --json` for local configuration validity and `loaf version` for the running distribution and configured project destination. The CLI validates local syntax only: `github_destination.live_verification: "not-performed"` and the human message `Live board verification: not performed (local configuration only).` do not prove that the node is a board, that it exists, or that the account can use it. Report local errors first; ask for missing project-owned election or destination values rather than guessing or changing them through `--fix`.
2. Select an already exposed connection and verify its actor against `integrations.github.account` using the transport and host rules above, before accessing resources. If no permitted, identity-verified connection can read the board, report live verification as unavailable or unverified and name the missing capability or access. Do not install a client, configure authentication, or make Loaf perform provider calls.
3. Through that connection, read the exact configured repository and resolve the exact `project_id` as a global node. Confirm that the returned object is an active Projects v2 board and read its current title and Status field options. Treat the ID as opaque: never decode it, require a prefix, search by a cached title, substitute a same-named board, or infer its owner from the repository. A null result, `404`, or permission error does not prove deletion or a bad ID; preserve the observed response and report that evidence as unverified unless the connection establishes a specific defect.
4. Return one combined report: configured repository and exact board ID; local validity; verified actor and transport; observed repository and board readability; current live board title and active state when available; Status options with Backlog, Todo, In Progress, and Done each marked present, confirmed missing, or unverified; and In Review when present. Separate confirmed configuration defects from unavailable capability or authorization. Successful reads prove the observed read access only, never write permission. Do not persist title, lanes, access snapshots, credentials, or work mirrors in project config. If required lanes are confirmed missing, name the missing lanes and stop affected work; if they cannot be read, leave them unverified. Never replace board Status with Issue open/closed, labels, or prose.

## Version Milestones

Use the selected connection's native milestone operations and the Issue's milestone field for version planning. If the configured MCP lacks those operations, the main agent may use installed authenticated `gh` after verifying its actor through that transport. Milestones add no Loaf command, configuration, capability entry, or `project-management/v1` operation; do not emulate them with labels, bodies, a board field, or a local ledger.

Read repository milestones and the current Issue before creation or assignment. Reuse a matching title, create a bucket only when shaped work needs it, and read back both the milestone and the Issue's assigned milestone. Preserve partial or indeterminate results; never blindly repeat an ambiguous create. A milestone's native number is distinct from an Issue number or the opaque project-board ID.

Titles equal the full release-tag string from the project's instructions, including any prefix or prerelease suffix. Compare version order for Now / Next / Later; use a `Horizon:` line in the milestone description only when that order is ambiguous. Horizon belongs to the milestone, never an Issue or another board axis. Existing assignments and names are not permission to rename buckets or rebalance the backlog. Milestone membership does not change board Status, select implementation, or prove a release.

At release time, read the exact matching open milestone and its remaining open Issues before asking about closure and rollover. Publication, Issue reassignment, and milestone closure are distinct authorized effects; read back each effect that actually occurs. Releases without a matching milestone keep their ordinary path.

## Record Syntax

Write Issue bodies and comments as the project-management skill's [Record Writing](../project-management/references/record-writing.md) reference describes. GitHub renders them this way:

- **Agent notes:** after the divider, `<details>`, then `<summary>Agent notes</summary>`, a blank line, the Markdown bullets, and `</details>`. Without the blank line after `</summary>`, GitHub shows the bullets as raw text.
- **Links:** write `#123` for an Issue or PR in the same repository and `owner/repo#123` for another; GitHub renders both as links, and a full Issue or PR URL also links. Code spans do not link.
- **Checkboxes:** `- [ ]` is open and `- [x]` is ticked.
- **Line breaks:** in Issue bodies and comments a single newline renders as a line break, so write each paragraph or bullet on one line and separate paragraphs with a blank line. A line directly above `---` renders as a heading unless a blank line separates them.
- **Source:** record it with the native field when one exists: a linked PR, the Issue type, a label, or a sub-issue or dependency relationship that is actually true. GitHub has no general related-issue edge; mentioning a related Issue in the prose where it matters is context, not a recorded relationship.

## Quick Reference

| Common semantic | GitHub semantic | Required runtime capability phases |
|-----------------|-----------------|------------------------------------|
| Work | Repository Issue | Scope repository and reject pull-request records; search/read before create, mutate, then read back |
| Definition | Issue body | Issue and kind read, body write, then Issue and kind readback |
| Hierarchy | Native sub-issues | Parent and sub-issue reads around native sub-issue mutation |
| Dependency | Native issue dependencies | Blocking and blocked-by reads around native dependency mutation |
| Status | Issue `state`/`state_reason` and board Status | Current Issue state, board item, and Status options around native transition |
| Comment | Issue comments | Current Issue/comments before append and both reads after |

Capability identifiers in the mapping describe GitHub semantics, not universal harness tool names. The selected connection must expose equivalent capabilities at runtime. If it exposes ordinary Issue operations but not native sub-issues or dependencies, those operations remain unsupported for that connection. If GitHub is the selected tracker and board Status is not visible, do not complete `status.read` or `status.transition` from Issue open/closed alone.

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Capabilities | [capabilities.json](capabilities.json) | Discovering runtime support and maximum honest fidelity |
| Common protocol | [project-management contract](../project-management/contract.json) | Constructing an operation or result envelope |
