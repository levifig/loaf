# Commit Conventions

## Contents
- Commit Message Format
- Commit Body
- Linear Integration
- Branch Naming
- History Model
- Pull Request Format
- Changelog Discipline
- Critical Rules
- Workflow Enforcement Hooks
- Semantic Versioning

Git commit, branch, and pull request standards.

## Commit Message Format

```
<type>: <description>

[optional body]

[optional footer]
```

### Types

| Type | Use For | Version Impact |
|------|---------|----------------|
| `feat` | New features | Minor bump |
| `fix` | Bug fixes | Patch bump |
| `refactor` | Code restructuring | None |
| `perf` | Performance improvements | Patch bump |
| `test` | Test additions/updates | None |
| `docs` | Documentation only | None |
| `chore` | Maintenance, deps, config | None |
| `ci` | CI/CD changes | None |
| `build` | Build system changes | None |

### Description Rules

- **Imperative mood**: "add feature" not "added feature"
- **Lowercase**: Start with lowercase after type
- **No period**: Don't end with a period
- **Short**: Under 72 characters
- **Focus on why**: The diff shows what

### Examples

```bash
# Good
feat: add thermal rating calculation
fix: prevent divide by zero in sag calculation
refactor: extract common validation logic

# Bad
feat: Added thermal rating calculation.  # Past tense, period
fix: Fixed the bug  # Vague, past tense
refactor: refactored code  # Redundant, past tense
```

## Commit Body

Add a body when:
- The "why" isn't obvious from title
- Trade-offs need documenting
- Implementation needs context

```
feat: add CIGRE TB 601 thermal model

Implement steady-state heat balance calculation per CIGRE TB 601.
Uses Newton-Raphson iteration for temperature convergence.

Key implementation notes:
- Natural convection below 0.5 m/s wind speed
- Film temperature for air property evaluation
- Tolerance: 0.1C for convergence
```

### What to Avoid in Body

- File lists (the diff shows this)
- Detailed code explanation
- Agent attribution
- Verbose descriptions

## Linear Integration

Use magic words in footer to link/close issues:

```
feat: add thermal rating API endpoint

Implement GET /api/towers/{id}/thermal-rating endpoint.

Closes BACK-123
```

### Keywords

| Keyword | Effect | Use For |
|---------|--------|---------|
| `Closes BACK-XXX` | Auto-closes on merge | Features, tasks |
| `Fixes BACK-XXX` | Auto-closes on merge | Bug fixes |
| `Resolves BACK-XXX` | Auto-closes on merge | Alternative |
| `Refs BACK-XXX` | Reference only | Related work |
| `Part of BACK-XXX` | Reference only | Partial work |

## Branch Naming

```
<type>/<description>
```

### Types

- `feat/` - New features (e.g., `feat/thermal-rating-cli`)
- `fix/` - Bug fixes
- `hotfix/` - Critical production fixes
- `release/` - Release preparation
- `chore/` - Maintenance, refactoring

### Rules

- Lowercase with hyphens (kebab-case)
- Short but descriptive (max 50 chars)
- Follow the repository's established branch convention; include the native tracker key only when that convention requires it

## History Model

Treat the three levels of Git history differently:

1. **Working-branch commits are implementation checkpoints.** Keep them atomic, coherent, and useful for review or diagnosis. They may record how a shippable outcome was built, but they are not automatically permanent product-history units.
2. **A pull request is one shippable unit.** It carries one reviewed root journey and lands through ship with the project's merge strategy (see [Merge Strategy](#merge-strategy)).
3. **The default branch is product history.** Each PR's landing should describe an observable, deployable outcome that can be reverted as a unit: the squash commit, the merge commit (`git revert -m 1 <merge>`), or, under rebase, the PR's replayed commits, each of which must stay buildable.

### Stacked or Child Branches

Related child work that belongs to one shippable root should not create an artificial merge bubble. Before assembling it, prove that the root is an ancestor of the child:

```bash
git merge-base --is-ancestor <root> <child>
git switch <root>
git merge --ff-only <child>
```

The second command must refuse if the histories diverged. When that happens, stop and inspect the competing changes; do not silently fall back to a merge commit or rewrite a shared branch.

### Fixing Commits

These rules hold under every merge strategy. Under squash, the branch commits are still what reviewers read and what `git bisect` walks before the merge; under merge or rebase, they also land on the default branch.

| Commit state | How to fix it |
|--------------|---------------|
| Local, not pushed | Rewrite freely: `git commit --amend` for the tip; `git commit --fixup=<sha>` and `git rebase --autosquash <base>` for an earlier commit |
| Pushed, not yet reviewed | Rewrite the same way, then `git push --force-with-lease` |
| Under review | Push `fixup!` commits instead of amending, so reviewers see only the change; autosquash once before the merge |
| On the default branch | Never rewrite; add a new commit or `git revert <sha>` |

Useful forms:

```bash
git commit --fixup=<sha>          # fold content into <sha>, keep its message
git commit --fixup=amend:<sha>    # fold content and replace <sha>'s message
git commit --fixup=reword:<sha>   # replace <sha>'s message only
git rebase --autosquash <base>    # fold every fixup!/squash!/amend! commit; no editor needed
git push --force-with-lease       # refuse if the remote moved since your last fetch
```

An autosquash changes commit hashes but not the final tree when nothing else changed. Compare `git rev-parse <old>^{tree}` with `git rev-parse <new>^{tree}` to show a reviewer, or ship, that the reviewed code is unchanged. Rebasing onto a moved base changes the tree and needs fresh review.

Never use plain `--force`. Loaf does not decide which branches may be force-pushed; protect the default, release, and shared branches with the host's branch protection or rulesets.

### Independent Roots and Merge Exceptions

Independent shippable roots need separate reviewed PRs even when implementation happened on one branch. Do not conceal them inside one giant landing merely for convenience. If splitting would make the landing less safe, obtain an explicit human decision that names why one atomic change is preferable.

Ordinary feature assembly never justifies a merge commit. Under the merge strategy, the PR's own merge commit is the normal landing. Under squash or rebase, a merge commit on the default branch is reserved for cases where the topology is itself durable evidence, such as a long-lived integration branch, provenance-sensitive upstream import, or deliberately preserved parallel history. Record that rationale before merging.

## Pull Request Format

### Title

Same format as commit messages (under squash, GitHub appends `(#N)` to the landed subject):

```
feat: add thermal rating calculation
```

### Description

Build the PR body from the canonical native tracker work contract and current verification evidence. Read the record through the selected `project-management/v1` provider skill, preserve its definition-of-done exactly, and use the ship template. Do not create a second local work record or include landing commit text in the PR body.

```
gh pr create --title "type: summary" --body-file <prepared-pr-body>
```

### Merge Strategy

The git-workflow skill selects the strategy (`git.merge_strategy`, or its fallback when unset). Author the landing for that strategy:

- **squash** — GitHub defaults the subject to `PR title (#N)`; keep it. Write a clean extended description: a one-line summary followed by bullet points grouped by feature area. Never use the automatic description that dumps every branch commit message; it is noisy and unhelpful in git history.
- **merge** — Every branch commit lands as written, so make each a clean checkpoint before review. Keep GitHub's `Merge pull request #N from owner/branch` subject; release attribution reads the branch name from it. Write the merge commit body as the outcome summary a squash would carry.
- **rebase** — Every branch commit is replayed onto the default branch with a new hash and no merge commit. Nothing added at merge time carries PR context, so each commit must be clean and should name the work reference when the project uses one.

Under every strategy:

- Assemble related stacked branches into their root with verified ancestry and `git merge --ff-only`; never create a merge commit merely to combine them
- Split independent shippable roots before the PR, or record an explicit human decision that one atomic landing is safer
- Don't push without explicit request. Merge only through the ship skill, which owns merge authority; release publishes a version later from already-landed work

## Changelog Discipline

`CHANGELOG.md` entries follow Loaf's Common Changelog profile: write for
humans, communicate the release impact, and curate git history instead of
copying it. Curate the `[Unreleased]` section as PRs land so the later
published release notes read as user-facing prose, not an internal worklog.

### Drop

Internal terms that have no meaning outside the team's working context:

- Internal work-unit numbering that is not the issue ID (issue IDs like `LOAF-42` belong in commits — release attribution reads them)
- Session, sprint, or branch references
- Internal terminology from skills/docs that isn't part of the user's mental model — e.g. `Q1`/`Q2`/`Q3` question numbers from a Triage Gate, internal gate-logic notation like `(Q1 OR Q2) AND Q3`, hook IDs that aren't user-facing
- "How the work got done" framing — interview steps, decomposition steps, review gates

### Keep

- **What changed** — commands, behaviors, file paths, config keys, hook names the user works with
- **Why it matters** — compatibility implications, migration notes, breaking-change call-outs
- **Public references** — `ADR-NNN` IDs, documented CLI flags, public file paths, named features

### Style

- Use backticks for all code references — file names, commands, config keys, hook names
- Write imperative, user-perspective prose: what upgrading changes, not how the work got built
- Keep each entry self-describing and one line whenever possible
- Link to the best public reference when available: PR, issue, ADR, release, or commit
- Group entries by Common Changelog category in order: `Changed`, `Added`, `Removed`, `Fixed`
- Prefix breaking entries with `**Breaking:**` and list them before non-breaking entries in the same category

### Auto-generated Entries

`loaf release suggest` drafts notes from landed issues; `loaf release cut` prepends them into `CHANGELOG.md`. Treat drafted notes as a draft: rewrite internal terms before cutting. Curate `[Unreleased]` as PRs land so the later cut reads as user-facing prose.

Before approving a release bump, compare `[Unreleased]` against the actual release range and remove scaffolding language introduced by specs, reviews, tasks, or session triage. If an entry only explains why the work was discovered or how the work was organized, it does not belong in the changelog.

## Critical Rules

### Always

- Write atomic working-branch checkpoints, each representing one coherent logical change
- Keep every checkpoint complete and buildable enough for review, diagnosis, and safe continuation
- Use imperative mood in messages
- Reference issue numbers when applicable
- Run the checks proportionate to that checkpoint before committing

### Never

- Commit a knowingly broken or internally incomplete checkpoint
- Skip commit signing (wait for user if it fails)
- Push without explicit user confirmation
- Use scoped commit subjects (for example, `feat(auth):`; write `feat:` instead)
- Include file lists in message
- Add agent attribution
- Mix unrelated changes
- Commit secrets or sensitive data
- Put work-unit IDs in the commit subject (use human-readable names). Issue aliases belong in the body so `loaf release suggest` can attribute the commit.

### ID References

- **IDs belong in footer, not subject line**
  - Bad: `feat: implement LOAF-42 invisible sessions`
  - Good: `feat: implement invisible sessions`
- Use descriptive names that are understandable without looking up IDs
- Issue aliases (`LOAF-42`) go in the body so release attribution can find them
- Linear issue IDs go in footer only (e.g., `Closes BACK-123`)

## Semantic Versioning

```
MAJOR.MINOR.PATCH[-PRERELEASE]

1.0.0 -> 1.0.1 (patch: bug fixes)
1.0.1 -> 1.1.0 (minor: new features)
1.1.0 -> 2.0.0 (major: breaking changes)
```

## Workflow Enforcement Hooks

Four hooks automatically enforce the conventions documented in this file:

| Hook | Timing | Behavior |
|------|-------|----------|
| `github-account` | Pre-tool (Bash) | Force-switch: switches the active `gh` account to the configured one before `gh` commands run (passes with a warning), exempting `gh auth` administration, and blocks only when the switch fails. It writes the shared global account pointer on every mismatched `gh` call -- read-only ones included -- so concurrent sessions on different identities collide on that pointer more often. Tracker GitHub preflight does not change, bypass, or isolate this hook. |
| `workflow-pre-pr` | Pre-tool (Bash) | Advisory: reminds about CHANGELOG [Unreleased] entries and PR format. Non-blocking. |
| `workflow-pre-push` | Pre-tool (Bash) | Advisory: reminders on `git push` — branch naming, uncommitted files, `--force-with-lease` for rewritten branches. Non-blocking. |
| `workflow-post-merge` | Post-tool (Bash) | Advisory: injects housekeeping checklist after a command match for `gh pr merge`; command matching does not prove a successful merge, so verify the result first. Non-blocking. |

These hooks read instruction templates from `hooks/instructions/` and run automatically when the corresponding git/gh commands are invoked.

Breaking changes use `feat!:` or `fix!:` and include:

```
BREAKING CHANGE: Description of breaking change.
```

### Pre-Release Versions

| Suffix | Meaning | When to use |
|--------|---------|-------------|
| `-alpha.N` | Alpha pre-release | Feature-complete but untested broadly |
| `-beta.N` | Beta pre-release | Testing with wider audience |
| `-rc.N` | Release candidate | Final validation before stable |

**Convention:**
- Use standard SemVer pre-release identifiers (`alpha`, `beta`, or `rc`) when publishing pre-release versions.
- `loaf release cut --bump` handles all bump types: `prerelease`, `release`, `major`, `minor`, `patch`

**Not required** — projects using simple `MAJOR.MINOR.PATCH` versioning can ignore pre-release suffixes entirely. This convention is for projects publishing staged pre-releases before stable releases.
