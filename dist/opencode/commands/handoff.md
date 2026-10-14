---
description: >-
  Creates deliberate context transfers for a named recipient or future
  conversation. Use when asked to prepare another agent to continue bounded
  work, deliver transfer context, or load a saved handoff. Produces a concise
  packet through an authorized channel, with SQLite persistence when needed. For
  ordinary capture, completion summaries, or parked-work checkpoints, use
  foundations.
subtask: false
version: 0.6.0-rc.3
---

# Handoff

Give a recipient enough context to take the next useful action without reconstructing the originating conversation.

**Input:** $ARGUMENTS

## Contents

- Critical Rules
- Verification
- Quick Reference
- Process
- Loading a Saved Handoff
- Journal Commands
- Related Skills

## Critical Rules

Before the first journal write, select the command in [Journal Commands](#journal-commands). Use that form for every entry in this skill.

- As the first action, run `loaf journal log "skill(handoff): <recipient and focus>"` against the current private journal. If unavailable, report the failure and continue only when the transfer can safely proceed.
- Use Handoff for a deliberate recipient-focused transfer. Interpret "prepare the next session" by intent: routine resumption and parked-work synthesis use Foundations and the managed continuity instructions, without automatically creating a packet.
- Infer the recipient, destination, bounded scope, and next action from available context. Ask only when ambiguity materially changes the transfer.
- Preparing or persisting a packet does not authorize spawning agents, creating tasks, sending messages, starting implementation, or expanding the recipient's existing authority. Use a delivery tool only when that action is already authorized.
- Re-read relevant native tracker and Git state when preparing the packet, and instruct the recipient to re-read them before acting. A snapshot is contextual evidence, never a work contract or current workflow state; Backlog does not authorize execution.
- Include only useful purpose, scope and authorization, current evidence, and next action. Add constraints, uncertainties, decisions and rationale, or suggested skills when they help; omit empty sections and large copied excerpts. Use the [packet template](templates/handoff.md) as prompts, not mandatory paperwork.
- Redact secrets, credentials, personal or customer data, and private external identifiers. Verify that the chosen channel and referenced resources are usable by the intended recipient.
- Apply Project Management's Capture Deferred Work rule to actionable follow-ups; a packet or journal entry does not replace the required native issue.
- Preserve existing handoff IDs, records, statuses, imported content, and legacy files. This workflow introduces no deprecation, deletion, migration, or cleanup authority.

## Verification

- The recipient can identify the purpose, permitted scope, relevant evidence, and next useful action without the original conversation.
- Delivery matches the actual outcome: prepared, delivered through an authorized channel, persisted and read back, or explicitly unpersisted because storage was unavailable.
- A persisted record's body and source project identity match on readback. Retrieval instructions work in the recipient's context; a source-local ID alone is insufficient when that store is inaccessible.
- The recipient re-reads current sources before acting and stays within existing authorization.
- Ordinary resumption creates no unnecessary transfer artifact. No existing records, files, statuses, or runtime selection changed as a side effect.

## Quick Reference

| Need | Transfer surface |
|------|------------------|
| Recipient can consume the current context; no independent retention needed | Authorized direct harness delivery |
| Transfer must survive independently, recipient cannot rely on current harness context, or operator requests persistence | Existing SQLite-backed `loaf handoff new/show/list` |
| Operator explicitly requests a portable file | Transfer snapshot at the requested destination |
| No authorized delivery tool | Return the usable packet to the operator; report it as prepared |
| Required persistence unavailable | Report the limitation and return an explicitly unpersisted packet when useful |
| Routine checkpoint or parked work | Foundations ordinary capture and selective synthesis |

## Process

1. **Frame the transfer.** Identify who will continue, where they can receive context, what they may do, and the next useful action. A future conversation can be the recipient; a named person or running agent is not required.
2. **Gather current evidence.** Read relevant native issues, Git status/branch/commits, journal entries, and existing authored material. Reference their authoritative locations and summarize only what continuation needs. State important limits or unavailable sources honestly.
3. **Prepare a compact packet.** Use the template's useful prompts. Include the native work reference where relevant and the instruction to re-read current state. Record a concise checkpoint through Foundations only if useful reasoning is missing from the journal; do not duplicate the packet there.
4. **Choose and verify delivery.** Use the applicable path below, then report the actual outcome, usable retrieval or destination, and the intended next action. Tool success is evidence of that operation, not proof that a recipient has read or accepted the packet.

### Direct Harness Delivery

Prefer direct harness delivery when the recipient can consume the context and it need not survive independently. Use the already-authorized channel and verify its result; no handoff record or file is necessary. If delivery is not authorized or no suitable tool exists, return the complete packet to the operator as prepared, without claiming it was sent.

### Private Persistence

Use the existing current-runtime SQLite path when independent retention is needed or explicitly requested:

```bash
loaf handoff new --title "<transfer title>" --body-file <packet-file> --json
loaf handoff show <returned-reference> --json
```

The body file is staging input, not another continuity store; the CLI also accepts `--body -` for stdin. Read back the returned native reference in the same source project and verify the exact body and project identity. Keep the CLI's observed status unchanged; no frontmatter or manual status rewrite is required.

Return the native reference, identify the source project context, and give the recipient `loaf handoff show <returned-reference> --json` retrieval instructions. Confirm access: another project or machine may not have that store. In that case provide the complete packet through an authorized accessible channel, or an explicitly requested portable snapshot; distinguish the persisted source copy from recipient delivery.

If the runtime or store cannot provide required persistence, report the limitation and return an explicitly unpersisted packet where useful. Do not claim durable capture, silently create another store, run migrations, or activate pending continuity capabilities.

### Explicit Portable Snapshot

When the operator explicitly requests a file, write the packet at the requested destination without overwriting existing content. Include useful source references, observation time from the environment, and current-state/authorization limits. No prescribed filename, frontmatter, status, or cleanup lifecycle is needed. This is a portable snapshot, not a competing private continuity store; verify that the recipient can access it before reporting delivery.

## Loading a Saved Handoff

From the source project context, use `loaf handoff list --json` or `loaf search <query> --json` to discover a native handoff, then `loaf handoff show <reference> --json` to read its body. Do not infer a filesystem location from the name. An explicit portable-file path is read as that snapshot; a packet recorded as a journal entry uses `loaf journal search` and `loaf journal show` instead. SQLite storage does not make every record a journal entry.

After loading, re-read the relevant native tracker and Git state, identify anything changed since the packet, and continue only within existing authorization. If the source store or referenced evidence is inaccessible, report what is missing instead of claiming successful recovery.

## Journal Commands

Use the ordinary command `loaf journal log "type(scope): description"` with your harness's permissions. Apply only the labeled section for the harness you are running.

### Codex

With Loaf's managed basic-command policy installed, use `loaf journal log --execpolicy-safe "type(scope): description"` for every entry, including invocation logs. Keep `--execpolicy-safe` immediately after `journal log` and invoke PATH `loaf` as a standalone command, without an absolute executable pin or shell/environment wrapper. The flag does not install the policy or grant permission by itself; without the policy, use the ordinary command and Codex's approval flow.

## Related Skills

- **foundations** — Ordinary capture and selective journal checkpoints
- **project-management** — Native work authority and actionable follow-up capture
- **orchestration** — Bounded coordination when authorized
