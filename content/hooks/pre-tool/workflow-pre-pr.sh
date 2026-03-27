#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Resolve paths for both source layout (pre-tool/../) and built layout (hooks/)
if [[ -f "${SCRIPT_DIR}/../lib/json-parser.sh" ]]; then
  source "${SCRIPT_DIR}/../lib/json-parser.sh"
  INSTRUCTIONS="${SCRIPT_DIR}/../instructions"
else
  source "${SCRIPT_DIR}/lib/json-parser.sh"
  INSTRUCTIONS="${SCRIPT_DIR}/instructions"
fi

# Read hook input — Claude Code: stdin JSON; OpenCode: TOOL_NAME + TOOL_INPUT env vars
if [[ -n "${TOOL_INPUT:-}" ]]; then
  INPUT="{\"tool_name\":\"${TOOL_NAME:-}\",\"tool_input\":${TOOL_INPUT}}"
else
  INPUT=$(cat)
fi
COMMAND=$(parse_command "$INPUT")

# Only match gh pr create
case "$COMMAND" in
  *"gh pr create"*) ;;
  *) exit 0 ;;
esac

# Check if CHANGELOG.md has actual entries (list items) under [Unreleased]
if [[ -f "CHANGELOG.md" ]] && \
   sed -n '/^## \[Unreleased\]/,/^## \[/p' CHANGELOG.md | grep -q "^- "; then
  # Entries exist — pass through with format reminder
  cat "$INSTRUCTIONS/pre-pr-format.md"
  exit 0
else
  # Missing or empty — block until CHANGELOG is updated
  cat "$INSTRUCTIONS/pre-pr-checklist.md" >&2
  exit 2
fi
