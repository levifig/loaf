# Push Reminders

- **Branch naming:** `<type>/<description>` (e.g., `feat/add-auth`, `fix/null-check`)
- **Check uncommitted files:** Run `git status` -- don't leave housekeeping files unstaged
- **Rewritten branch:** Push with `--force-with-lease`, never plain `--force`. The host's branch protection decides which branches refuse force-pushes.
