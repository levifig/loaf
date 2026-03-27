#!/usr/bin/env bash
# Post-merge housekeeping checklist
# Injects advisory checklist after a successful `gh pr merge`
# Post-tool hook for Bash (non-blocking, informational only)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Resolve paths for both source layout (post-tool/../) and built layout (hooks/)
if [[ -f "${SCRIPT_DIR}/../lib/json-parser.sh" ]]; then
  source "${SCRIPT_DIR}/../lib/json-parser.sh"
  INSTRUCTIONS="${SCRIPT_DIR}/../instructions"
else
  source "${SCRIPT_DIR}/lib/json-parser.sh"
  INSTRUCTIONS="${SCRIPT_DIR}/instructions"
fi

# Read hook input from stdin
HOOK_INPUT=$(cat)
COMMAND=$(parse_command "${HOOK_INPUT}")

# Only match gh pr merge commands
case "$COMMAND" in
  *"gh pr merge"*) ;;
  *) exit 0 ;;
esac

# Only inject on confirmed success -- if exit code is missing or non-zero, skip
EXIT_CODE=$(parse_exit_code "${HOOK_INPUT}")
[[ -z "$EXIT_CODE" || "$EXIT_CODE" != "0" ]] && exit 0

# Output the housekeeping checklist
cat "${INSTRUCTIONS}/post-merge.md"

exit 0
