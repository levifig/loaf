---
name: git-workflow
description: >-
  Covers branching strategies, commit conventions, PR creation, and the
  project's merge strategy (squash, merge commit, or rebase). Use when creating
  branches, writing commits, creating PRs, choosing how a PR lands, or managing
  git history. Provides patterns for collaborative git workflows. Not for code
  style (use foundations), CI/CD pipelines (use infrastructure-management), or
  reviewing and merging a PR (use ship).
---

# Git Workflow

Git conventions for branching, commits, PRs, and merge strategy.

## Contents
- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules

- Use Conventional Commits format for all commit messages
- Working-branch commits are complete implementation checkpoints. Keep each one atomic and buildable enough to support review, diagnosis, and safe continuation; do not commit partial or knowingly broken work.
- A pull request is one shippable unit. Merge it only through the ship skill, which owns merge authority; never run a bare `gh pr merge` or merge directly into the default branch.
- Land each PR with the project's merge strategy. Read `git.merge_strategy` from `.agents/loaf.json` (`squash`, `merge`, or `rebase`; `loaf config check` validates it). When it is unset, read the repository's enabled merge methods (on GitHub, `gh repo view --json squashMergeAllowed,mergeCommitAllowed,rebaseMergeAllowed`): use squash when it is enabled, otherwise the only enabled method. If merge and rebase are both enabled without squash, or the methods cannot be read, ask and recommend recording the answer in `git.merge_strategy`. A configured strategy the repository does not allow blocks the merge; never switch methods silently.
- Author the landing for that strategy (see Merge Strategy below). Under squash, write a deliberate Conventional Commit title and extended description; never accept the automatic dump of branch commit messages. Under merge or rebase, every branch commit lands on the default branch as written, so each must already meet the checkpoint rule above.
- Related child or stacked branches join their shippable root without synthetic topology. Verify ancestry with `git merge-base --is-ancestor <root> <child>`, then use `git merge --ff-only <child>` from the root branch. If ancestry diverged, stop and reassess instead of creating a merge commit by default.
- Never create a merge commit merely to assemble related feature work. Under the squash or rebase strategy, a merge commit on the default branch is an exception: preserve one only when topology, provenance, or a long-lived integration is itself durable project information, and record the explicit rationale.
- Independent shippable roots must not disappear inside one landing solely because they share an implementation branch. Split them into separate reviewed PRs, or obtain an explicit human decision that one atomic landing is safer.
- One branch per shippable root. Read the canonical native tracker record through the selected provider, then use the repository's branch convention (normally `feat/<slug>`, `fix/<slug>`, or `chore/<slug>`). Related child work shares the parent's branch and PR when the tracker hierarchy says it is one shipping unit.
- Keep branch history clean under every merge strategy. Before a commit is pushed, rewrite it freely: `git commit --amend` for the tip, `git commit --fixup=<sha>` and then `git rebase --autosquash <base>` for an earlier commit. Once a PR is under review, push fixes as `fixup!` commits so reviewers see only what changed, then autosquash once before merging. Commits already on the default branch are product history: correct them with a new commit or `git revert`, never a rewrite.
- Push a rewritten branch with `git push --force-with-lease`, never plain `--force`. Loaf does not decide which branches may be force-pushed; the host's branch protection or rulesets stop force-pushes to the default, release, and shared branches.
- Never push without explicit user confirmation

## Verification

- Commit messages follow unscoped Conventional Commits format (`type: description`)
- Every working-branch commit is a complete checkpoint, and the candidate diff represents exactly one shippable root
- Stacked work was verified as ancestral and assembled with `--ff-only`; no incidental merge commit exists
- Before a merge or rebase landing, no `fixup!`, `squash!`, or `amend!` commits remain on the branch
- Independent shippable roots have separate PRs unless an explicit human decision documents why one atomic landing is safer
- Branch is up to date with its base branch before creating the PR
- PR title is under 70 characters with PR# suffix convention
- The merge strategy came from `git.merge_strategy` or the unset fallback above, and the landing matches it: a squash title and body, or a merge-commit body, describes the shipped outcome rather than replaying implementation commits; under rebase, every branch commit is a clean checkpoint

## Quick Reference

| Action | Command/Pattern |
|--------|----------------|
| Branch naming | Repository convention, normally `feat/{slug}`, `fix/{slug}`, or `chore/{slug}`; include the native tracker key only when the repository convention calls for it |
| Commit format | `type: description` |
| Fix the tip commit | `git commit --amend` (before push) |
| Fix an earlier commit | `git commit --fixup=<sha>`, then `git rebase --autosquash <base>` (non-interactive) |
| Push a rewritten branch | `git push --force-with-lease` |
| Assemble stacked child | Verify `git merge-base --is-ancestor <root> <child>`, then run `git merge --ff-only <child>` from the root |
| Select merge strategy | `.agents/loaf.json` → `"git": {"merge_strategy": "squash"}` (or `"merge"`, `"rebase"`); unset falls back as described in Critical Rules |
| Merge a PR | Only through ship |
| PR creation | `gh pr create --title "..." --body "..."` |

### Merge Strategy

| Strategy | Ship runs | What lands on the default branch | What to author |
|----------|-----------|----------------------------------|----------------|
| `squash` | `gh pr merge <N> --squash --body ...` | One commit per PR | Keep GitHub's `PR title (#N)` subject; write a 2-4 line outcome body |
| `merge` | `gh pr merge <N> --merge --body ...` | Every branch commit plus a merge commit | Keep GitHub's `Merge pull request #N from owner/branch` subject, which release attribution reads; write an outcome body |
| `rebase` | `gh pr merge <N> --rebase` | Every branch commit, replayed with new hashes and no merge commit | Nothing at merge time; each branch commit must already be clean and name the work reference when the project uses one |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Commits | `references/commits.md` | Writing commit messages, fixing commits with amend or fixup and autosquash, creating PRs, branching, authoring the landing for each merge strategy, curating CHANGELOG entries, pre-PR/pre-push/post-merge hooks |
| PR shipping | ship skill | Reviewing, verifying, and merging one PR; ship is the only merge path and owns merge authority |
| Release ritual | release skill | Publishing a version from already-landed work |
