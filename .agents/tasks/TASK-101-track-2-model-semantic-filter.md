---
id: TASK-101
title: 'Track 2: Model semantic filter'
status: todo
priority: P2
created: '2026-04-10T17:27:26.602Z'
updated: '2026-04-10T17:27:26.602Z'
spec: SPEC-029
---

# TASK-101: Track 2: Model semantic filter

## Description

Create a single Stop prompt hook that replaces both `journal-nudge` and `session-state-update`. The prompt presents the CLI sync's extraction summary (new entries since last sync) and asks the model to: (a) add decisions, discoveries, and context that the CLI cannot infer, (b) update `## Current State`. One Stop prompt, not two.

**File hints:**
- MODIFY: `config/hooks.yaml` — add Stop prompt hook, remove journal-nudge if still present
- MODIFY: `content/hooks/instructions/` or inline prompt — the model filter prompt text
- MODIFY: `cli/commands/session.ts` — `loaf session sync` output formatted for model consumption

## Acceptance Criteria

- [ ] Single Stop prompt hook presents bounded diff summary to model
- [ ] Model adds semantic entries (decisions, discoveries) that CLI cannot detect
- [ ] `## Current State` updates correctly via the merged prompt
- [ ] Empty/trivial turns produce no unnecessary entries
- [ ] Prompt references the sync summary, not the full conversation
- [ ] Previous `session-state-update` hook removed or merged

## Verification

```bash
npm run typecheck && npm run test && loaf build
```
