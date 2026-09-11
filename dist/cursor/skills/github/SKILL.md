---
name: github
description: >-
  Maps project-management/v1 operations to GitHub repository Issue semantics
  through an exposed harness-native connection. Use when the selected canonical
  tracker is GitHub. Produces verified native outcomes without configuring
  credentials or a provider client.
version: 0.5.0
---

# GitHub

Apply [`project-management/v1`](../project-management/contract.json) through a GitHub-capable connection already exposed by the current harness. The [capability mapping](capabilities.json) describes semantic ceilings; the runtime connection decides which operations are actually available for the selected repository.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules

- Discover exposed GitHub connections and verify the exact destination before mutation: `owner/repository` plus the configured Projects v2 board when GitHub is the canonical tracker. Refer to that board by its title. If the title matches the repository or Loaf project name, call it the project board. Never identify it as `Project #N` or a `users/<login>/projects/<n>` path — those numbers are user- or org-scoped, not repo-scoped. Do not assume a connection, repository, or board from the provider name or local Git remote. A working `gh` authenticated as the project account that can read the destination repository and named board is a GitHub connection, including on Amp when the only MCP server is Linear. Another provider's MCP does not hide GitHub.
- Prefer an already configured, usable GitHub MCP that matches the intended account. If none is available, an installed authenticated `gh` in the main agent's permitted shell is a valid exposed GitHub capability, including `gh api`, before asking the user to configure MCP. Unrelated Linear MCP does not determine GitHub authority or hide GitHub. Select the connection matching the intended account before any resource access. Authenticated `gh` fallback does not bypass the account requirement or observed permissions.
- Anchor the expected GitHub account to `integrations.github.account`. Resolve the host from explicit project configuration or user context, never from the repository owner. Inspect only nonsecret account metadata.
- Verify actor identity through the selected connection's own transport before resource access. For MCP, use that connector's trustworthy principal identity or metadata capability; a `gh` identity is not evidence of the MCP actor. If the connector identity is missing, ambiguous, or mismatched, that MCP connection is unusable. The main agent may then try authenticated `gh` fallback, and must separately verify the `gh` actor in the same shell context with `gh api --hostname <host> user --jq .login`, accounting for environment overrides. Stop on a mismatched or unverifiable identity. Repeat actor verification through the actual chosen transport immediately before mutations, after known context changes, and on same-connection readback, naming the explicit repository and host.
- Do not authenticate, refresh, or globally switch accounts, extract credentials, or add routing wrappers. The installed `github-account` hook may still switch the global active account; this guidance does not change, bypass, or isolate that hook. Preflight and readback cannot eliminate a concurrent switching race and cannot prove the mutation actor. If concurrent switching makes identity uncertain, stop.
- Inspect runtime capabilities and permissions for the selected repository. `connection.describe` is only the first signal because GitHub has no reliable non-mutating write-capability probe; use successful safe native reads as additional operation-level evidence. Downgrade an unavailable mapping to `advisory`, `manual`, or `unsupported` without inventing a fallback representation.
- Accept the connection's observed permission evidence without requiring one header family: classic OAuth, fine-grained credentials, and GitHub Apps can expose different permission metadata. Never inspect, print, or infer the credential itself.
- Use repository Issues as canonical work records and the Issue body as the canonical definition. Search with GitHub's `is:issue` qualifier, read candidate matches, and reject every issue-shaped object carrying a `pull_request` property before reading or mutating it as an Issue. Apply the same check to relationship results.
- Use GitHub's native sub-issue relationships for hierarchy and native issue dependencies for directed blocked-by/blocking edges. Arbitrary related edges are unsupported. Never emulate either relationship with labels, task-list prose, links, or comments.
- Orient dependency writes from the blocked Issue: when the target blocks another Issue, write the target as a `blocked_by` edge on that other Issue, then read back both the target's `blocking` collection and the other's `blocked_by` collection.
- GitHub as the canonical tracker requires an active Projects v2 board whose Status field includes the team's lanes (at least Backlog, Todo, In Progress, and Done). If the connection cannot see that named board or Status field, stop and prompt. Do not treat Issue open/closed as Todo or Backlog, and never encode lanes in labels or comments.
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
- Return the common result envelope instead of GitHub-shaped success claims.

## Verification

- The selected connection was observed in the current harness and the exact GitHub repository plus the named board was read.
- The expected account was taken from `integrations.github.account` and the host from explicit configuration or context, never the repository owner. The selected connection matched that account before resource access. Actor identity was verified through that connection's own transport; `gh` identity was not treated as MCP evidence. Missing, ambiguous, mismatched, or unverifiable identity stopped work before resource operations.
- Every requested operation maps to all visible runtime capabilities named by the `before`, `execute`, and `after` phases in `capabilities.json`.
- Every target record and related record was confirmed to be an Issue rather than a pull request before mutation, and create duplicate detection used an `is:issue` search followed by candidate reads.
- A mutation is `confirmed` only when the final native read shows the intended Issue field, body, relationship, state and state reason, board Status, or comment. A duplicate transition additionally requires native `duplicate_of` readback matching the requested canonical Issue identity. A lane transition is confirmed only when the board Status matches the requested option.
- Missing sub-issue, dependency, state-reason, board, or comment capabilities are reported at their observed fidelity rather than represented through labels, prose, or another field. A missing board when GitHub is the tracker is a configuration gap: prompt, do not invent workflow.
- No connection configuration, credential material, provider transport, local work copy, synchronization path, or persistent mapping was created.

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
