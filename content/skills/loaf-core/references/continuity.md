# Ordinary Continuity

Continuity is part of ordinary work, not an invocation or conversation-closing ceremony. Record significant decisions and discoveries when they happen, with enough rationale and source context to be useful later. Use the journal command selected in the loaf-core skill's Journal Commands section; rendered Markdown is a projection, never the write surface.

Before completing substantial work, handing off, or compacting context, check whether useful reasoning is missing from existing records. Read recent entries when needed. If something useful is missing, write one concise `wrap(scope)` checkpoint: abandoned approaches and why, unresolved uncertainty, and the next useful step when applicable. Omit empty fields and do not repeat raw events. Routine interactions need no redundant checkpoint.

For example, a journal may already contain the chosen retry policy but omit why a fixed delay was abandoned and which rate-limit behavior remains unverified. A checkpoint should preserve that missing connection, rather than repeat every command or test result. If the decision entry already explains it, another checkpoint adds nothing.

The existing `wrap` entry type is optional synthesis. It creates no session entity, status, lifecycle, or completion gate; a conversation without a checkpoint is valid. Existing entries and `loaf journal recent --since-last-wrap` remain usable. The standalone Wrap skill is retired.

Upgrade removes unchanged Loaf-owned Wrap skill copies through the existing deprecation mechanism and reports customized or unowned copies for manual handling. Legacy OpenCode/Cursor `commands/wrap.md` files have no ownership receipt, so preview and apply preserve them and require manual review and relocation or removal before retrying. Their presence does not create a new continuity requirement.

Recover context with `loaf journal recent`, `loaf journal search <query>`, `loaf journal show <id>`, and `loaf journal context`. Confirm relevant sources, Git state, and native tracker state before acting: remembered context is evidence, not current work authority. If the journal is unavailable, report the failure and any unpersisted reasoning; do not silently create a second store or claim capture succeeded.

Compaction guidance reinforces this discipline only where the harness delivers it. It emits instructions, not an automatic transcript summary or journal write. Ordinary capture must work without a compaction or end event; abrupt endings can still lose reasoning that was never recorded.

Responsibilities remain separate:

- Loaf Core owns ordinary capture and selective synthesis.
- Project Management owns capture of concrete actionable follow-ups in the native tracker.
- Implement and Ship own delivery evidence, checks, and accurate Git-state reporting.
- Housekeeping owns artifact maintenance under its approval rules.
- Handoff remains the existing explicit transfer workflow; ordinary capture does not require invoking it.

Checkpointing grants no authority to implement Backlog work, create tasks or agents, send messages, commit, push, release, install, or clean up. Preserve useful reasoning within the authorization already given.

## Explorations

An undecided inquiry that will outlast one conversation, context window, or harness is kept as an Exploration: a private record of immutable checkpoints and provenance. It has no status, owner, or lifecycle. Resuming means reading its portable context and appending new facts, never changing state. You decide what an Exploration means and when to checkpoint; the CLI only validates and performs the operation you request.

Check `loaf exploration list` before creating a duplicate. Start one with `loaf exploration create --title <title> [--from <source>]...`, linking the journal entries, handoffs, reports, or findings that motivated it. Checkpoint at every meaningful plateau and before the context window gets hostile:

```bash
loaf exploration checkpoint <ref> --purpose <p> --conclusions <c> --unresolved <u> --next <n> [--item candidate:<text>]... [--item evidence:<text>]... [--operation-id <key>]
```

Every checkpoint carries all four portable fields, each readable without this conversation. Each is capped at 4096 UTF-8 bytes; oversize input is rejected, never truncated:

- **purpose:** what is being explored and why, restated for a stranger.
- **conclusions:** what is established so far, including rejected options and why they fell.
- **unresolved:** the open question or decision the inquiry turns on.
- **next:** the recommended next action, concrete enough for a fresh agent.

Larger detail goes in ordered `candidate:` and `evidence:` items or a related report. Never store transcripts, prompts, or tool output; curate the meaning instead. Name native tracker references in the checkpoint; spark and idea references stay there too, because tracker records point only to places everyone with tracker access can reach.

A conversation handle or log path is provenance, never context. Record one only when it helps later navigation (`loaf conversation create`, `loaf conversation handle add`, `loaf exploration conversation add`), and never infer identity from the current session. Handles do not make an Exploration resumable; only a portable checkpoint does.

Resume anywhere with `loaf exploration context <ref> --json`. The four-field core returns whole; optional layers report counts, truncation, and their expansion command. Treat unavailable handles as lost; the checkpoint is the context. If `portable_context_present` is false, rebuild understanding from linked sources and write the missing checkpoint first. Read the tracker records a checkpoint names before continuing: if one is done, cancelled, or a duplicate, say so instead of silently reopening it.

To park an inquiry, checkpoint it honestly. An Exploration is never paused or closed; it waits, resumable from its checkpoint. A checkpoint never files tracker work; filing a direction follows the pitch skill's filing rule.
