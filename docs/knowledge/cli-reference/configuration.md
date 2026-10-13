# Configuration Maintenance

The guided answer to "is this project's Loaf config current, and what fixes it?"
Work top to bottom; stop when the checks pass.

## Diagnose first

Run `loaf config check --json` and read the structured findings — never scrape the human-readable output. It validates `.agents/loaf.json` and installed Loaf-managed hook config, and flags missing safe defaults, stale installed-target config, and malformed fields. For an explicit `integrations.github.tracker: true` election, the `github_destination` object reports the configured `repository` and opaque `project_id`, local errors when present, and `live_verification: "not-performed"`. `loaf version` displays the same project destination separately from running distribution identity and remains informational for incomplete config or outside a project. Neither command contacts GitHub or proves that the configured node is a usable board.

When GitHub is selected, finish the existing diagnostic ceremony through the [GitHub destination diagnosis](../../../content/skills/github/SKILL.md#destination-diagnosis), using an already exposed, identity-verified harness connection. Return one report combining local validity with exact repository and board ID, current live title, observed access, and Status options: Backlog, Todo, In Progress, and Done are required, with In Review reported when present. Confirmed missing routing or lanes is a configuration defect; unavailable access or capability leaves live evidence unverified. A `404` does not prove deletion, a successful read does not prove write permission, and Issue open/closed never substitutes for board lanes.

## Safe repairs

`loaf config check --fix` creates missing safe defaults and refreshes stale installed-target hook config. Review what it changed, then re-run `loaf config check --json` to confirm it passes. `--fix` only touches mechanical, reversible repairs: it never elects a tracker, invents or replaces a repository or board ID, fetches live board metadata, or changes opaque provider-specific fields.

## Project-owned choices (ask the user)

These are decisions, not defaults. The CLI cannot guess them — ask, then record:

- **GitHub account** — `integrations.github.account`, the login the project's `gh`
  commands must run as. The `github-account` enforcement hook checks it against
  `gh auth status`; a mismatch tells the user to run `gh auth switch`.
- **GitHub tracker and destination** — when the user selects GitHub as the canonical tracker, record `integrations.github.tracker: true`, `integrations.github.repository` (`owner/name`), and `integrations.github.project_id` (the exact opaque Projects v2 global node ID). Account-only settings, `enabled`, a legacy board title, a Git remote, or `issue.authority` do not elect GitHub. IDs are opaque: local syntax validation cannot establish object type, existence, or access. Read the current title live for display; never store title, lane, or access snapshots as routing identity. There is no CLI setter; edit narrowly after the project-owned choice is confirmed, rerun `loaf config check --json`, and complete the connected diagnosis above.
- **Legacy issue identity (migration compatibility only)** — historical `issue.authority` (`local`, `linear`, or `github`) and `issue.prefix` belong to frozen local issue commands; they do not elect the current native tracker. The prefix is the local alias token (`VCAM`) or the historical Linear team key (`ENG`). `loaf issue identity --prefix` / `--authority` remains a compatibility setter, and `--fix` never invents these fields on an existing file. Preserve leftover local rows until an explicitly authorized migration; do not use these commands as ongoing work authority or synchronization.
- **Integration routing** — preserve provider-specific `integrations.*` toggles and opaque fields; they do not elect GitHub. When Linear is active, record the exact server exposed by the harness as `integrations.linear.mcp_server_name`. The Linear or bootstrap skill records this project-owned value; Loaf does not install or authenticate the connection.
- **Which harnesses to install** — the targets passed to `loaf install --to <target>`
  (or `--to all`).
- **Codex basic command policy** — `loaf install --to codex --codex-basic-commands` explicitly installs PATH `loaf` managed command rules for the centrally classified basic Loaf leaves, including hardened journal logging and approved readers, outside the workspace sandbox. Each rule keeps the subcommand prefix; the policy does not grant a bare `loaf` namespace or a writable Loaf data-directory root. Global `AGENTS.md` remains user-owned: no guidance block is installed or required. Upgrades remove only unchanged recorded legacy blocks in regular files; edited blocks, absent blocks, and symlinks are preserved and never block independent rule updates. Ownership records for preserved legacy guidance are retained, not treated as a requirement to recreate it. Unclassified and operator commands remain gated. Do not enable it implicitly.
- **Codex lifecycle hooks** — Codex `0.144.1` rejects Loaf's legacy flat hook schema. Loaf installs a current-schema `SessionStart` matcher group whose command is PATH `loaf journal context --from-hook --codex-hook`; POSIX installs omit the Windows variant, while Windows installs keep both command fields as that PATH form. Startup is model-visible proven only for the isolated `darwin-arm64` smoke; other lifecycle modes and automatic completion remain gated, and skill/project instructions plus explicit CLI retrieval remain the fallback.

## Hand-edit only where no setter exists

Some fields (for example `integrations.github.account`) have no CLI setter. Edit
`.agents/loaf.json` narrowly for those, leave everything else to the CLI, and
immediately re-validate with `loaf config check --json`. Never hand-edit
Loaf-managed hook files — regenerate them through `loaf build` and `loaf install`.

## Refresh installed targets

`loaf upgrade` updates already-installed targets and applies deprecation-manifest cleanup (removing retired skills and targets). Run it after config or skill changes so installed harnesses match the source. It is safe to run from anywhere: project surfaces are refreshed only inside a detected Loaf repo, and `--to <target>` narrows the sync to one already-installed target. Cursor, OpenCode, Amp, Claude Code, and Codex generated hook commands stay `loaf …` on PATH; scoped `--select` fails closed if that PATH binary is missing or lacks required `loaf check --hook` or standalone `loaf check` commands, and does not publish `{XDG_DATA_HOME}/loaf/pinned/loaf`. Codex basic-command policy upgrades rewrite unmodified managed rules from an absolute pin to PATH `loaf` plus the same classified prefixes, and preserve modified or foreign policy.

`loaf install` is for onboarding — deploying Loaf into a folder that does not have it, or adding a harness that is not installed yet. Inside a Loaf repo it no-ops the project part and points at `loaf upgrade`.

If the Codex Auto-journal rule was previously enabled, upgrade refreshes it only while the installed body still matches Loaf's recorded digest. Unowned or locally modified rule files are preserved and reported as conflicts.

## Confirm project alignment

`loaf doctor` checks symlinks, stale files, and version drift across the checkout without changing anything. `loaf doctor --fix` offers each safe repair separately and defaults each y/N prompt to no; declined repairs remain reported failures. Use `loaf doctor --fix --force` only when every offered repair should be accepted without prompting, including in non-interactive automation. This catches drift that config-level checks do not.

## Record the decisions

Log the project-owned choices you settled with the user:
`loaf journal log "decision(config): ..."`.
