---
description: >-
  Serves as the sole quality gate for implemented native tracker work. Use when
  a candidate diff needs independent review, criterion verification, and a
  landing decision. Produces an evidence-backed quality verdict and verified
  native transition when authorized.
version: 0.6.0-rc.3
---

# Ship

Evaluate the candidate against the live [work contract](templates/work-contract.md), repository state, and observed verification evidence. Passing tests are necessary but do not replace criterion-by-criterion review.

## Contents

- Critical Rules
- Post-Merge Reconciliation
- Verification
- Quick Reference
- Journal Commands
- Topics

## Critical Rules

Before the first journal write, select the command in [Journal Commands](#journal-commands). Use that form for every entry in this skill and its references.

- As the first action, run `loaf journal log "skill(ship): <concise intent>"` against the current private local journal. If the write fails, report the failure and continue only when the work can safely proceed; never put invocation bookkeeping in the tracker.
- Re-read the canonical record, relationships, workflow state, and relevant collaboration through [`project-management/v1`](../project-management/SKILL.md).
- Inspect the actual candidate diff and repository instructions. Do not accept an implementer's summary as evidence.
- Verify the explicitly declared implemented work set against the canonical tracker records and actual diff. Require changed Unreleased coverage for every implemented record, regardless of change size or category, and review each note for semantic accuracy. Run the repository's shared coverage validator against the actual review base and candidate; passing structural checks do not establish accurate or complete scope. Missing coverage blocks Approve and Done. A release-only exception must prove movement of already-landed notes without new implementation; a release title, tag, or skip label is insufficient. Report whether the hosted check is actually required by native rules; local hooks and model guidance alone are not a universal mechanical barrier.
- Review correctness, maintainability, security boundaries, scope, and every completion criterion with fresh eyes; use independent read-only reviewers when available and proportionate.
- Map each criterion and relevant scope boundary to the contract's specified proof. Reuse existing valid evidence; do not create a duplicate log. A named human verdict is usable only where the contract explicitly defines judgmental acceptance, and it must match the designated authority, method and considerations, and the reviewed candidate. Human signoff cannot replace specified objective checks or safety proof. Required unavailable proof or judgment remains unproven. Scope creep is not a bonus that can be silently shipped.
- Record a durable disposition discoverable from the issue, directly or through a stable existing linked review or PR: candidate identity, criterion-to-evidence outcomes, necessary judgment and acceptance authority, relevant scope changes, and limitations. Write it as [Record Writing](../project-management/references/record-writing.md) describes: lead with one to three plain lines giving the verdict, what it rests on, and anything a person must do; add plain bullets only for scope changes or limitations people should know; put candidate identity, criterion-to-evidence detail, and commands in collapsed agent notes. Solo and team use the same contract; reuse native comments or review surfaces rather than a second ledger.
- Review the rideable increment inside this same quality gate: can the named rider use the result today, does the journey exercise the machinery added, what real dogfood and integrity evidence support the claims, and could less preserve the same safe outcome?
- Do not accept material contract drift by recording it in the verdict. If the candidate changes promised outcome, criteria, scope or safety, dependencies, or the basis of selected or resourced work, pause acceptance and return it to shape. Material changes need applicable authorization (existing authorization counts; assignment, write access, or reviewer status alone is never product authority), an updated canonical definition and native fields with authoritative readback, and a retained reason, authority, and impact; then review the candidate against that resulting contract. Keep same-turn responsibilities; do not add extra ceremony or repeated approval when authorization already exists.
- Return a clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence to implement and rerun affected gates after fixes. Unclear or changed contract substance, or a material scope or commitment shift, returns to shape; do not bounce ordinary fixes to shaping or absorb unauthorized scope in implement. Inaccessible required proof or absent authority is blocked; do not waive the criterion. Material means uncertainty or change in what was promised or authorized, not the severity of an in-scope coding bug.
- Build review or change-request text from the live native contract and observed diff, not from a synchronized local projection.
- Preserve external-action authority. Acceptance is separate from landing authority. A quality verdict never authorizes commit, push, publication, or destructive action; commits and feature-branch pushes keep their own rules outside ship.
- Before marking an issue Done or merging a PR that closes it, re-read the issue and confirm every Verification box is ticked, as [Record Writing](../project-management/references/record-writing.md#verification-boxes) describes. An open box blocks closing and merging unless its reason is written on the box; stop and name each open box without one. A tick counts only with a recorded result for the reviewed candidate: treat a box ticked without one, ticked on an earlier tree, or a person-only box ticked by an agent as open, then rerun its check or return it to implement. Tick a box yourself only after running its check and recording the result in the review; ticking is the only definition write ship makes. A check only a person can do, such as a customer sign-off, stays for that person. Never write a reason yourself to clear the gate, and remember that a reason clears only this gate, never a criterion's proof. A merge can close linked issues automatically, as Linear's GitHub integration does, so confirm the boxes before the merge, not after.
- Merge only through ship. A bare `gh pr merge`, or a direct merge or push into the default branch, is never an authorized landing path.
- Merge authority depends on how ship started. When the user invoked ship explicitly, by name or slash command, or directly asked to merge this specific PR, an Approve verdict authorizes the merge; do not ask again. When ship was inferred from context or started on your own initiative, ask with your harness's structured question tool, if it has one, before merging. Request changes or Blocked never merges.
- Immediately before merging, re-read the PR and the Verification boxes on each issue it closes. Its head must be the reviewed candidate or have the same tree (`git rev-parse <sha>^{tree}`), as after an autosquash that only reshaped commits; a changed tree needs re-review first. Every required check must have passed; a failing required check blocks the merge. Under the merge or rebase strategy, `fixup!`, `squash!`, or `amend!` commits still on the branch block the merge until autosquashed; under squash they disappear in the landing. Merge with the project's merge strategy as the git-workflow skill resolves it against repository settings and branch rules; when that strategy, or every method, is ruled out, do not run the merge and report the conflicting setting and rule. Pin the head you verified (`gh pr merge --match-head-commit <sha>` on GitHub), and author the landing message that strategy calls for from the reviewed outcome.
- Transition native workflow state only after the corresponding repository event is proven and authorized.
- Re-read the changed native record and any evidence comment; never mark work complete from a mutation request alone.
- After an authorized merge, finish the Post-Merge Reconciliation below before reporting local checkout cleanup as complete. Remote landing remains true even if local cleanup is blocked; report both states separately instead of silently skipping the local work.
- Release remains a separate retroactive ceremony over already-landed work.

## Post-Merge Reconciliation

Only after authoritative PR readback confirms `MERGED`, the merge commit, the exact head, the base branch, and the base repository identity:

1. Inspect the current checkout with `git status --porcelain` and `git worktree list`. If it is dirty, inaccessible, or the base branch belongs to another worktree, preserve it and report **local reconciliation pending** with the specific reason. Do not stash, reset, or switch through someone else's work.
2. Resolve `<baseRemote>` from `git remote -v` against the authoritative PR base repository's host and owner/name, accounting for equivalent SSH and HTTPS URLs. `origin` may be a fork; its name or the branch's configured upstream is not proof of that identity. If no matching remote can be verified, preserve the checkout and report local reconciliation pending. In a clean available checkout, switch to the confirmed base and fast-forward from that verified remote; never merge the old feature branch back into the base:
   ```bash
   git switch <baseRefName>
   git pull --ff-only <baseRemote> <baseRefName>
   git rev-parse <baseRefName> FETCH_HEAD
   git merge-base --is-ancestor <mergeCommit> <baseRefName>
   git status --porcelain
   ```
   Require every command to succeed, both refs to name the same commit, the observed PR merge commit to be an ancestor of the local base, and a clean checkout. `FETCH_HEAD` is the base just fetched from the verified remote, including when its fetch mapping does not create a conventional remote-tracking ref. If any check, switch, or fast-forward refuses, leave local reconciliation pending and report the observed state; do not rebase, force, or reset to make it pass.
3. Independently transition the canonical native work to its authorized completed state only after the remote merge is proven and its Verification boxes still read ticked or carrying their reasons; if one is open without a reason, name it and report the issue as not complete instead of transitioning. Re-read the resulting state. A local checkout blocker does not undo a verified remote merge or justify skipping the tracker readback.
4. Report the disposition of the feature branch, remote branch, and any linked feature worktree. Inspect cleanliness, activity, and dependent work before any removal. Squash and rebase merges rewrite the feature commits, so the feature tip is usually not an ancestor of the base: if an authorized `git branch -d <headRefName>` refuses, retain the branch and report it; never silently escalate to `-D`. Worktree removal and remote branch deletion require their own applicable authorization and safety checks. Do not remove a worktree while running inside it.

Report **Landed remotely; local reconciliation pending** when the PR and tracker are complete but the checkout cannot be safely reconciled. Report local reconciliation complete only after the verified base repository, fetched-base equality, merge-commit ancestry, and clean-checkout readback. Do not treat installing or activating a local runtime as part of this Git reconciliation.

## Verification

- The reviewed commit or diff identity is exact and the working tree contains no unexplained changes.
- Every completion criterion and relevant scope boundary has the contract's specified proof from the candidate, tests, or native state. A named human verdict counts only where the contract explicitly defines judgmental acceptance and matches the designated authority, method and considerations, and the reviewed candidate; implementer summary or a general human signoff is not acceptance and cannot replace specified objective checks or safety proof.
- The durable disposition is discoverable from the issue and names candidate, criterion-to-evidence outcomes, required judgment or authority, relevant scope changes, and limitations; it leads with plain lines a person can read, with agent detail collapsed.
- The complete journey works from its named entry point to observable outcome; no future-only machinery is presented as delivered value.
- Dogfood and safety/integrity evidence match the contract, and learning plus remaining deferrals are reported honestly.
- Required focused, affected, full-suite, format, lint, static-analysis, and build gates were run in proportion to risk.
- All findings have an evidence-backed disposition and fixes were re-reviewed.
- Load the project-management skill and apply its Capture Deferred Work rule to deferred actionable findings; a follow-up issue never excuses an unmet completion criterion for this candidate.
- Affected architectural claims and implementation gaps were checked against the diff and evidence using the architecture skill's maintenance rules; a documentation update alone is not proof of delivery.
- Before any close or merge, every Verification box on each issue the merge closes or ship marks Done was ticked with a recorded result for the reviewed candidate, or carried a person's reason; ship ticked only boxes whose checks it ran and recorded, and left person-only checks for their person.
- Any merge followed the authority rule for how ship started, used the merge strategy the git-workflow skill resolved, pinned a head whose tree matches the reviewed candidate, left no autosquash commits under merge or rebase, and had every required check passing.
- Any native transition matches the observed landed state and was confirmed by readback.
- After an authorized merge, the PR merge identity, local base/remote base equality and clean checkout, native state, and branch/worktree dispositions were read back; any incomplete local reconciliation is named rather than presented as finished.

## Quick Reference

| Verdict | Meaning |
|---------|---------|
| Approve | Criteria, boundaries, and quality gates are proven, and every Verification box is ticked or carries its reason on the box. Merge now when ship was explicitly invoked or the merge was directly requested; otherwise ask first |
| Request changes | A clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence returns to implement; unclear or changed contract substance, or a material scope or commitment shift, returns to shape. Do not absorb unauthorized scope by recording it. |
| Blocked | Required proof is inaccessible; a Verification box is open with no reason on it and ship cannot run its check, because only a person can or it can only run after the merge; or required authority, connection, or repository event is unavailable. Do not waive the criterion |
| Landed | Repository event and final native state were both observed; report local reconciliation separately, including any pending checkout or branch cleanup |

## Journal Commands

Use the ordinary command `loaf journal log "type(scope): description"` with your harness's permissions. Apply only the labeled section for the harness you are running.

### Codex

With Loaf's managed basic-command policy installed, use `loaf journal log --execpolicy-safe "type(scope): description"` for every entry, including invocation logs. Keep `--execpolicy-safe` immediately after `journal log` and invoke PATH `loaf` as a standalone command, without an absolute executable pin or shell/environment wrapper. The flag does not install the policy or grant permission by itself; without the policy, use the ordinary command and Codex's approval flow.

## Topics

No supporting references. Use the [tracker update](templates/tracker-update.md) for a concise evidence-bearing verdict when collaborators need it.
