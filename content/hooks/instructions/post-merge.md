**Reminder:** Ship owns post-merge reconciliation, but invoking Ship does not prove it happened. After an authorized merge, verify the remote event, local base, tracker state, and cleanup disposition before reporting local completion; otherwise say **local reconciliation pending**. This advisory checklist also covers manual merges.

# Post-Merge Housekeeping

Complete these steps only after authoritative PR readback confirms the merge succeeded. Hook command matching alone does not prove success.

1. **Confirm the merge and capture identity.** Read the PR's state, merge commit, base branch, head branch, number, URL, and linked native tracker reference. If the PR is not observed as merged, stop without changing tracker or Git state.

2. **Reconcile an available clean checkout.** Inspect `git status --porcelain` and `git worktree list` first. If the checkout is dirty or the base is in another worktree, leave it untouched and report local reconciliation pending. Otherwise switch to the PR base and fast-forward:
   ```
   git switch <baseRefName>
   git pull --ff-only origin <baseRefName>
   git rev-parse <baseRefName> origin/<baseRefName>
   git status --porcelain
   ```
   Confirm the refs match and the checkout is clean. Do not stash, reset, rebase, or merge the feature branch into the base to hide a refusal.

3. **Transition the canonical native tracker record.** Through the selected `project-management/v1` provider skill and harness-native connection, read the provider's valid statuses, perform the authorized completion transition, then read the same native record back. Report unsupported, failed, or indeterminate provider outcomes without creating a local fallback record.

4. **Report the feature worktree disposition.** From outside the feature worktree, inspect registered worktrees and confirm the candidate is clean and no agent is using it. Remove it only with applicable authorization:
   ```
   git worktree list
   git -C <worktree-path> status --short
   git worktree remove <worktree-path>
   ```
   If no linked worktree exists, continue. Never use `--force` without explicit user confirmation, and never remove the worktree while running inside it.

5. **Report the local and remote feature branch disposition.** Delete only when authorized and safe:
   ```
   git branch -d <headRefName>
   ```
   Squash and rebase merges rewrite the feature commits, so `-d` normally refuses after them; if it does, retain the branch and report it. Never silently force-delete it with `-D` or delete the remote branch.

6. **Log the landing:**
   ```
   loaf journal log "decision(ship): PR #N landed via <strategy> merge; <native-ref> completed"
   loaf journal log "commit(<hash>): <landed subject>"
   ```

7. **Suggest reflection** if the work produced key decisions or learnings.

8. **Suggest release only when appropriate** — if this PR completes a coherent rideable batch, publish later with `loaf release suggest` / `loaf release cut`. The PR is landed, not released, until that cut.
