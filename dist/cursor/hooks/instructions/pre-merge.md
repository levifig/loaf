STOP. `gh pr merge` belongs to the ship workflow. If ship is not running, do not merge; run ship instead.

Inside ship, before merging:

1. BOXES: Re-read each issue this merge will close. Every Verification box must be ticked or carry its reason on the box, as the project-management skill's record-writing reference defines them: a reason that misses its conditions, an agent's tick without a recorded result, or an agent's tick on a person-only box counts as open. If one is open, stop and name it instead of merging. A merge can close linked issues automatically, so this cannot wait until after.
2. STRATEGY: Use the project's merge strategy: `git.merge_strategy` in `.agents/loaf.json`, or the git-workflow skill's fallback when it is unset, checked against repository settings and the base branch's rules (rulesets and classic protection). If the strategy, or every method, is ruled out, stop and report the conflict instead of merging. Pass exactly one of `--squash`, `--merge`, or `--rebase`.
3. HEAD: Pin the reviewed head with `--match-head-commit <reviewed-sha>`.
4. MESSAGE: Author the landing for that strategy.
   - squash: keep the default "PR title (#N)" subject and pass a clean 2-4 sentence `--body`. NEVER use the automatic squash description that dumps every branch commit message.
   - merge: keep the default "Merge pull request #N from owner/branch" subject and pass a clean `--body` summarizing the outcome.
   - rebase: nothing is authored at merge time; the branch commits land as written.

Example (squash): write the reviewed outcome to a temporary message file, then pass it with `--body-file`:

```bash
gh pr merge N --squash --match-head-commit <reviewed-sha> --body-file <landing-message-file>
```

Describe the actual shipped outcome and verification. Do not imply that a planned compatibility retirement, cleanup command, or live installation has already shipped.
