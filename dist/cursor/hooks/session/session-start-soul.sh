#!/usr/bin/env bash
# SessionStart hook: Validate SOUL.md is present
# If missing, inject from canonical template and warn

set -eo pipefail

SOUL_PATH="SOUL.md"

if [[ -f "$SOUL_PATH" ]]; then
  # SOUL.md exists — pass silently
  exit 0
fi

# Attempt restore from template (path varies by target runtime)
TEMPLATE_PATH="${CLAUDE_PLUGIN_ROOT:-}/templates/soul.md"

if [[ -n "${CLAUDE_PLUGIN_ROOT:-}" ]] && [[ -f "$TEMPLATE_PATH" ]]; then
  cp "$TEMPLATE_PATH" "$SOUL_PATH"
  echo "⚠ SOUL.md was missing — restored from canonical template."
  echo "The Warden identity (Arandil) is now active."
  exit 0
fi

# No template available (non-Claude target or template missing) — warn but don't block
echo "⚠ SOUL.md not found. Copy content/templates/soul.md to project root to enable the Warden identity."
exit 0
