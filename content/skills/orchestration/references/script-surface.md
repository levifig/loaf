# Script Surface

## Contents
- Retirement Rule
- Retired Helpers
- Existing Commands

Orchestration no longer ships helper scripts. Shared operations use existing
`loaf` commands. Session and tracker helpers were retired rather than recreated.

## Retirement Rule

A helper gets a recorded native replacement or an evidence-backed retirement.
Do not restore local tracker authority, session lifecycles, or `loaf linear *`
commands that wrap a provider. Do not fold a git status dump into
`loaf journal context` — the continuity digest is not that dump.

Keep a script only when it is still invoked and can stay a tiny local CLI with
no provider API. None of the former orchestration helpers met that bar.

## Retired Helpers

| Script | Disposition | Why |
|--------|-------------|-----|
| `new-session.sh` | Retired | Already a stub. There is no session to create. Log with `loaf journal log`; read continuity with `loaf journal context`. |
| `git-context-summary.sh` | Retired | Unused git dump (uncommitted files, recent commits, ahead/behind). `loaf journal context` already names the branch and journal branch recency. The digest is not this dump. |
| `extract-magic-words.sh` | Retired | Overlaps `loaf journal log --detect-linear`. No remaining caller needed a second read-only listing or a Linear client. |
| `new-council.sh` | Existing command | Use `loaf council new --title <title> --body-file <path>` (or `--message`). Do not recreate `YYYYMMDD-HHMMSS-topic.md` or a `council:` frontmatter block. |
| `validate-council.py` | Retired | Validated the obsolete filename/frontmatter ceremony (PyYAML). `loaf council show` / `loaf council list` read the current artifact contract. |
| `format-progress.sh` | Retired | Unused Linear paste formatter. No remaining caller. Do not invent `loaf linear format-progress`. |
| `check-linear-format.py` | Retired | Unused Linear comment linter. No remaining caller and no provider API to wrap. |
| `suggest-team.py` | Retired | Linear team routing against local config, with a stub workspace-teams fetch. The Linear skill and harness MCP own team selection. No new tracker client. |
| `get-config.py` | Retired | Linear `.agents/config.json` lookup (comments even named `loaf.json`). `loaf config check` owns Loaf config; there is no `loaf config get`. |
| `validate-roadmap.py` | Retired (explicit exception) | Roadmaps are not first-class (bootstrap: series-prep is not roadmap planning). Not kept as a silent skill-local script. If roadmap artifacts become first-class, add a narrow local `loaf` validator then — not a tracker client. |

## Existing Commands

| Need | Command |
|------|---------|
| Log work | `loaf journal log "type(scope): description"` |
| Continuity digest | `loaf journal context` |
| Detect commit magic words | `loaf journal log --detect-linear` |
| Create a council artifact | `loaf council new --title <title> --body-file <path>` |
| Read councils | `loaf council show` / `loaf council list` |
| Loaf project config | `loaf config check` |
