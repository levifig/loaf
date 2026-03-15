#!/usr/bin/env bash
# Remind about changelog updates after git commit
# Post-tool hook for Bash (non-blocking, informational only)

set -euo pipefail

# Check if this was a git commit command
COMMAND=""
if [[ -n "${TOOL_INPUT:-}" ]]; then
    COMMAND=$(echo "$TOOL_INPUT" | grep -oE '"command"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//' | sed 's/"$//' || true)
fi

# Only trigger on git commit (not amend, rebase, etc.)
if [[ ! "$COMMAND" =~ ^git[[:space:]]+commit[[:space:]] ]]; then
    exit 0
fi

# Only remind if CHANGELOG.md exists (project opted in to changelogs)
if [[ ! -f "CHANGELOG.md" ]]; then
    exit 0
fi

# Check if changelog has an [Unreleased] section
if ! grep -q '## \[Unreleased\]' CHANGELOG.md 2>/dev/null; then
    exit 0
fi

echo "CHANGELOG: Consider updating CHANGELOG.md with this commit."
echo "  Categories: Added | Changed | Fixed | Removed | Deprecated | Security"

exit 0
