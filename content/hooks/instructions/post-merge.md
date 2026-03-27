# Pre-Merge Checklist

Complete these steps on the feature branch before merging.

1. **Update CHANGELOG.md:**
   Add entries under `[Unreleased]` describing what the PR ships.

2. **Bump version** in `package.json`:
   - `feat` commit -> minor bump (or dev bump if pre-release)
   - `fix` commit -> patch bump
   - Breaking change -> major bump

3. **Move `[Unreleased]` to versioned section:**
   `## [X.Y.Z] - YYYY-MM-DD`

4. **Rebuild all targets:**
   ```
   npx loaf build
   ```

5. **Commit and push** the changelog + version bump to the PR branch.

---

# Post-Merge Housekeeping

Complete these steps on main after merging.

1. **Switch to main and pull:**
   ```
   git checkout main && git pull --rebase
   ```

2. **Mark tasks done:**
   ```
   loaf task update TASK-XXX --status done
   ```

3. **Mark spec done** (if all tasks for the spec are now complete).

4. **Archive session file** -- update frontmatter:
   - `status: complete`
   - `archived_at: "YYYY-MM-DDTHH:MM:SSZ"`
   - `archived_by: "agent-pm"`
   - Move file to `.agents/sessions/archive/`

5. **Archive completed tasks and specs:**
   ```
   loaf task archive --spec SPEC-XXX
   loaf spec archive SPEC-XXX
   ```

6. **Commit housekeeping:**
   ```
   git add -A && git commit -m "chore: close TASK-XXX session, archive artifacts"
   ```

7. **Delete merged feature branch:**
   ```
   git branch -d feat/xxx
   ```
