# Post-Merge Housekeeping

Complete these steps on the current branch (main) after merging.

1. **Switch to main and pull:**
   ```
   git checkout main && git pull --rebase
   ```

2. **Mark tasks done:**
   Update each completed task:
   ```
   npx loaf task update TASK-XXX --status done
   ```

3. **Mark spec done** (if all tasks for the spec are now complete).

4. **Finalize CHANGELOG:**
   - If bumping version: move `[Unreleased]` entries to a new versioned section `## [X.Y.Z] - YYYY-MM-DD`.
   - If not bumping: leave entries under `[Unreleased]`.

5. **Bump version** in `package.json`:
   - `feat` commit -> minor bump (or dev bump if pre-release)
   - `fix` commit -> patch bump
   - Breaking change -> major bump

6. **Rebuild all targets:**
   ```
   npx loaf build
   ```

7. **Archive session file** -- update frontmatter:
   - `status: complete`
   - `archived_at: "YYYY-MM-DDTHH:MM:SSZ"`
   - `archived_by: "agent-pm"`
   - Move file to `.agents/sessions/archive/`

8. **Commit housekeeping:**
   ```
   git add -A && git commit -m "chore: bump to X.Y.Z, close TASK-XXX session"
   ```

9. **Archive completed tasks and specs:**
   ```
   loaf task archive --spec SPEC-XXX
   loaf spec archive SPEC-XXX
   ```

10. **Delete merged feature branch:**
    ```
    git branch -d feat/xxx
    ```
