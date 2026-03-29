#!/usr/bin/env bash
# SessionStart hook: Validate SOUL.md is present
# If missing, inject from canonical template and warn

set -euo pipefail

SOUL_PATH="SOUL.md"
TEMPLATE_PATH="${CLAUDE_PLUGIN_ROOT}/templates/soul.md"

if [[ -f "$SOUL_PATH" ]]; then
  # SOUL.md exists — pass silently
  exit 0
fi

# SOUL.md missing — attempt to restore from template
if [[ -f "$TEMPLATE_PATH" ]]; then
  cp "$TEMPLATE_PATH" "$SOUL_PATH"
  echo "⚠ SOUL.md was missing — restored from canonical template."
  echo "The Warden identity (Arandil) is now active."
  exit 0
fi

# Neither exists — warn but don't block
echo "⚠ SOUL.md not found and no template available."
echo "The Warden identity will not be active for this session."
exit 0
