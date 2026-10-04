STOP. `gh pr merge` belongs to the ship workflow. If ship is not running, do not merge; run ship instead.

Inside ship, before merging:

1. STRATEGY: Use the project's merge strategy: `git.merge_strategy` in `.agents/loaf.json`, or the git-workflow skill's fallback when it is unset. Pass exactly one of `--squash`, `--merge`, or `--rebase`.
2. HEAD: Pin the reviewed head with `--match-head-commit <reviewed-sha>`.
3. MESSAGE: Author the landing for that strategy.
   - squash: keep the default "PR title (#N)" subject and pass a clean 2-4 sentence `--body`. NEVER use the automatic squash description that dumps every branch commit message.
   - merge: keep the default "Merge pull request #N from owner/branch" subject and pass a clean `--body` summarizing the outcome.
   - rebase: nothing is authored at merge time; the branch commits land as written.

Example (squash):
```
gh pr merge N --squash --match-head-commit <reviewed-sha> --body "$(cat <<'EOF'
Add loaf housekeeping CLI command that scans .agents/ directories and
recommends cleanup actions. Includes shared prompt helpers, scanner
engine, interactive/dry-run modes, and skill update.
EOF
)"
```
