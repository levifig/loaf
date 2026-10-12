# Ordinary Continuity

Continuity is part of ordinary work, not an invocation or conversation-closing ceremony. Record significant decisions and discoveries when they happen, with enough rationale and source context to be useful later. Use the journal command selected in Foundations; rendered Markdown is a projection, never the write surface.

Before completing substantial work, handing off, or compacting context, check whether useful reasoning is missing from existing records. Read recent entries when needed. If something useful is missing, write one concise `wrap(scope)` checkpoint: abandoned approaches and why, unresolved uncertainty, and the next useful step when applicable. Omit empty fields and do not repeat raw events. Routine interactions need no redundant checkpoint.

For example, a journal may already contain the chosen retry policy but omit why a fixed delay was abandoned and which rate-limit behavior remains unverified. A checkpoint should preserve that missing connection, rather than repeat every command or test result. If the decision entry already explains it, another checkpoint adds nothing.

The existing `wrap` entry type is optional synthesis. It creates no session entity, status, lifecycle, or completion gate; a conversation without a checkpoint is valid. Existing entries and `loaf journal recent --since-last-wrap` remain usable. The standalone Wrap skill is retired.

Upgrade removes unchanged Loaf-owned Wrap skill copies through the existing deprecation mechanism and reports customized or unowned copies for manual handling. Legacy OpenCode/Cursor `commands/wrap.md` files have no ownership receipt, so preview and apply preserve them and require manual review and relocation or removal before retrying. Their presence does not create a new continuity requirement.

Recover context with `loaf journal recent`, `loaf journal search <query>`, `loaf journal show <id>`, and `loaf journal context`. Confirm relevant sources, Git state, and native tracker state before acting: remembered context is evidence, not current work authority. If the journal is unavailable, report the failure and any unpersisted reasoning; do not silently create a second store or claim capture succeeded.

Compaction guidance reinforces this discipline only where the harness delivers it. It emits instructions, not an automatic transcript summary or journal write. Ordinary capture must work without a compaction or end event; abrupt endings can still lose reasoning that was never recorded.

Responsibilities remain separate:

- Foundations owns ordinary capture and selective synthesis.
- Project Management owns capture of concrete actionable follow-ups in the native tracker.
- Implement and Ship own delivery evidence, checks, and accurate Git-state reporting.
- Housekeeping owns artifact maintenance under its approval rules.
- Handoff remains the existing explicit transfer workflow; ordinary capture does not require invoking it.

Checkpointing grants no authority to implement Backlog work, create tasks or agents, send messages, commit, push, release, install, or clean up. Preserve useful reasoning within the authorization already given.
