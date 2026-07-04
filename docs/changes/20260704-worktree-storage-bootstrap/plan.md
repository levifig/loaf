# Worktree Storage Bootstrap

## Summary

Fresh linked worktrees can still strand agents before useful work starts. The current SPEC-036 gate correctly refuses commands when a worktree has local `.agents/` state without a valid `.moved-to` back-pointer, but normal branch/worktree creation still materializes tracked `.agents/**` files. That means a new agent may hit the refusal before it has enough context to run the sanctioned migration.

This change should make fresh-worktree bootstrap boring: the first mutating Loaf action should either work directly when the worktree only contains tracked historical `.agents` material, or present a single safe command path that an agent can execute without being trapped by the gate.

## Direction

- Keep `.agents` centralized to the main worktree per ADR-013.
- Keep the refusal for real, un-migrated worktree-local agentic state.
- Add a narrow bootstrap path for fresh linked worktrees whose local `.agents` content is only checkout material that already exists in the main worktree.
- Preserve dry-run/explicit-apply behavior for migration when content movement is required.
- Treat this as the first branch-local change capsule: implementation notes, tests, and acceptance criteria live here while cross-change observations stay in the project journal.

## Acceptance Criteria

- In a fresh linked worktree, a non-destructive Loaf command such as `loaf --version`, `loaf build`, or `loaf journal context` does not dead-end on historical tracked `.agents/**` content.
- In a fresh linked worktree with only tracked `.agents/**` files identical to the main worktree, the bootstrap path creates or recognizes the `.agents/.moved-to` back-pointer without requiring manual file surgery.
- A linked worktree with genuinely divergent local `.agents/**` content still refuses non-migrate commands with the SPEC-036 nudge.
- `loaf migrate worktree-storage` remains dry-run by default and safe to run explicitly.
- Build/test workflows in a clean worktree can run without first knowing internal SPEC-036 history.
- Regression coverage creates a fresh linked worktree and proves the first intended Loaf command succeeds or self-heals safely.

## Non-Goals

- Do not re-centralize `.agents` somewhere outside the main worktree.
- Do not remove the SPEC-036 safety gate for real un-migrated local state.
- Do not revive session entities or session lifecycle vocabulary.
- Do not solve the broader change-capsule CLI in this change.
- Do not sync Linear tasks by default; this change is local/Git/SQLite scoped.

## Test Strategy

- Add or adjust native Go tests around `worktree_storage_migration.go` and runner dispatch.
- Cover at least these cases:
  - main checkout remains a no-op;
  - non-git directory remains rejected for migration;
  - fresh linked worktree with identical tracked `.agents` content can bootstrap;
  - linked worktree with divergent `.agents` content still refuses;
  - stale `.moved-to` still refuses;
  - explicit `migrate worktree-storage --apply` still preserves conflict policy behavior.
- Run:
  - `go test ./...`
  - `npm run test:smoke`
  - `npm run build`

## Review Notes

- This is intentionally narrower than the future CR/change-capsule system. It uses the `docs/changes/YYYYMMDD-slug/` layout as a pilot artifact while the implementation remains normal code/tests.
- The observed dogfood failure was `npm run build` in `/tmp/loaf-journal-terminology-pass`: `bin/loaf __generate-cli-ref` was refused until `loaf migrate worktree-storage --apply` ran.
- The desired user experience is that a new worktree created for an agent starts usable, not that every safety check becomes advisory.

## Linear Mapping

Recommended Linear shape:

- Issue: `CR: loaf/worktree-storage-bootstrap`
- Tasks only if useful:
  - Add bootstrap classification for identical tracked `.agents` content.
  - Add fresh-worktree regression coverage.
  - Update help/docs if the command output changes.

Do not sync these tasks by default until the team needs visible parallel execution.
