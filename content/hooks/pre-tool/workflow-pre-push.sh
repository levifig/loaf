#!/usr/bin/env bash
# Hook: Pre-Push Advisory (INFORMATIONAL)
# PreToolUse hook - reads JSON from stdin per Claude Code hooks API
#
# Triggers on git push commands
# Shows branch naming and safety reminders (non-blocking)

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

# Read hook input from stdin
INPUT=$(cat)
COMMAND=$(parse_command "$INPUT")

# Only match git push
case "$COMMAND" in
  *"git push"*) ;;
  *) exit 0 ;;
esac

# Detect force-push flags and target branch
FORCE_PUSH=false
[[ "$COMMAND" == *"--force"* || "$COMMAND" == *"-f "* || "$COMMAND" == *" -f" ]] && FORCE_PUSH=true

# Determine target branch: check command args first, fall back to current branch
TARGET_BRANCH=""
if [[ "$COMMAND" =~ git\ push\ .*\ (main|master)($|\ ) ]]; then
  TARGET_BRANCH="${BASH_REMATCH[1]}"
fi
if [[ -z "$TARGET_BRANCH" ]]; then
  TARGET_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
fi

# Output advisory reminders
cat "${INSTRUCTIONS}/pre-push.md"

# Extra warning for force-push to protected branches
if [[ "$TARGET_BRANCH" == "main" || "$TARGET_BRANCH" == "master" ]] && $FORCE_PUSH; then
  echo "" >&2
  echo "WARNING: Force-pushing to $TARGET_BRANCH. This rewrites shared history." >&2
fi

# Non-blocking - always exit 0
exit 0
