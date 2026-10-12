# Command Routing

Choose the authority before choosing a command. The generated command index records implemented CLI help, including compatibility commands; its presence does not authorize those commands as the current work flow.

| Intent | Route |
|--------|-------|
| Shape new bounded work | Shape writes the canonical native tracker record through the configured Project Management provider |
| Implement selected work | Implement reads the selected native contract and uses repository-native Git mechanics |
| See or change shared work state | Read or mutate the native tracker through the selected provider, then verify readback |
| Continue after a restart | `loaf journal context`; find a specific checkpoint with `loaf journal search <query>` and `loaf journal show <id>` |
| Load a named saved Handoff | In its source project, `loaf handoff list` or `loaf search <query>`, then `loaf handoff show <id-or-alias>` |
| Read a council | `loaf council list/show <ref>` reads its authored file and currently requires initialized SQLite state; an explicit file path can be read directly |
| Review a temporary report or proposal | Read the authored file; no database row, universal lifecycle, or compatibility report command is required |
| Read an explicit snapshot | Read the supplied path and revalidate current authoritative sources before acting |
| Skills or content changed | `loaf build`; installation or upgrade is a separate explicitly scoped action |
| Check knowledge freshness | `loaf kb check` |
| Inspect an existing local task, plan, issue, or report lifecycle record | Use exact command help for historical compatibility; do not create a second shared work authority |
| Import legacy `.agents` Markdown into SQLite | Explicit migration only: `loaf migrate markdown --dry-run`, then authorized `--apply` |

The shipped Loaf Reference's Artifact Routing topic supplies the family-specific storage and maintenance boundaries. Handoff owns deliberate transfers; Foundations owns ordinary capture. Missing state or a missing CLI is an access limitation, not a reason to invent another store or initialize a fallback for an existing record.

## JSON diagnosis surfaces

Prefer structured output when available:

- `loaf config check --json` — project config and installed hook validity
- `loaf state doctor --json` / `loaf state status --json` — SQLite health and readiness
- `loaf check --hook <id> --json` — one enforcement hook's result
- `loaf check commit-msg|secrets|changelog|compliance|test-naming|dockerfile|k8s-manifest|bounds|units|standard-refs --json` — standalone validators
- `loaf kb check --json` — knowledge staleness against Git history
- `loaf journal recent --json` — private timeline; current shared work comes from the provider
- `loaf migrate markdown --dry-run --json` — migration simulation/inventory, not ongoing synchronization

The troubleshooting reference covers diagnostic selection and isolated state. The markdown-migration reference owns import authority, origin reclaim, and insert-only behavior. Compatibility readiness checks describe legacy local records; they do not verify a native work contract.
