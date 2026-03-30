#!/usr/bin/env bash
# SessionStart hook: Validate SOUL.md is present
# If missing, inject from canonical template and warn
# Portable across Claude Code, Cursor, and OpenCode.

set -eo pipefail

SOUL_PATH="SOUL.md"

if [[ -f "$SOUL_PATH" ]]; then
  exit 0
fi

# Locate template — works across all target layouts:
#   Claude Code: hooks/            → ../templates/soul.md  (1 up)
#   Cursor:      hooks/session/    → ../../templates/soul.md (2 up)
#   OpenCode:    plugins/hooks/    → ../../templates/soul.md (2 up)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE=""

# Try CLAUDE_PLUGIN_ROOT first (set by Claude Code runtime)
if [[ -n "${CLAUDE_PLUGIN_ROOT:-}" ]] && [[ -f "${CLAUDE_PLUGIN_ROOT}/templates/soul.md" ]]; then
  TEMPLATE="${CLAUDE_PLUGIN_ROOT}/templates/soul.md"
else
  # Walk up from script directory to find templates/soul.md
  dir="$SCRIPT_DIR"
  for _ in 1 2 3; do
    dir="$(dirname "$dir")"
    if [[ -f "$dir/templates/soul.md" ]]; then
      TEMPLATE="$dir/templates/soul.md"
      break
    fi
  done
fi

if [[ -n "$TEMPLATE" ]]; then
  cp "$TEMPLATE" "$SOUL_PATH"
  echo "⚠ SOUL.md was missing — restored from canonical template."
  echo "The Warden identity (Arandil) is now active."
  exit 0
fi

echo "⚠ SOUL.md not found and no template available."
exit 0
