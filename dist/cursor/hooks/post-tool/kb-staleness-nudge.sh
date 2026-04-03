#!/usr/bin/env bash
# KB staleness nudge — alerts when editing files covered by stale knowledge
# Post-tool hook for Edit|Write
#
# Per-session single nudge: each knowledge file mentioned at most once.
# Tracks nudged files in a temp file keyed on PPID (stable within a session).

set -euo pipefail

# Check if loaf is available
if ! command -v loaf &>/dev/null; then
  exit 0
fi

# Parse the edited file path from TOOL_INPUT JSON.
# OpenCode/Amp pass the hook payload on stdin instead of exporting TOOL_INPUT.
tool_input="${TOOL_INPUT:-}"

if [ -z "$tool_input" ] && [ ! -t 0 ]; then
  tool_input=$(cat)
fi

if [ -z "$tool_input" ]; then
  exit 0
fi

# Extract file_path (primary field for Edit/Write tools)
file_path=$(echo "$tool_input" | grep -oE '"file_path"[[:space:]]*:[[:space:]]*"[^"]+"' | sed 's/.*: *"//' | sed 's/"$//' || true)

if [ -z "$file_path" ]; then
  exit 0
fi

# Check which knowledge files cover this path
result=$(loaf kb check --file "$file_path" --json 2>/dev/null || true)
if [ -z "$result" ] || [ "$result" = "[]" ]; then
  exit 0
fi

# Track already-nudged files per session
PROJECT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || printf '%s' "${CLAUDE_PROJECT_DIR:-$PWD}")"
BRANCH="$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || true)"
if [ -z "$BRANCH" ]; then
  DETACHED_SHA="$(git -C "$PROJECT_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)"
  BRANCH="detached-${DETACHED_SHA}"
fi
NUDGE_KEY="${PROJECT_DIR}:${BRANCH}"
nudge_hash="$(echo "$NUDGE_KEY" | md5sum 2>/dev/null | cut -c1-8 || md5 -q -s "$NUDGE_KEY" 2>/dev/null | cut -c1-8 || echo "default")"
nudge_file="/tmp/loaf-kb-nudged-${nudge_hash}"

# The JSON output is an array of objects with "file", "isStale", etc.
# Parse each entry: look for "file":"..." and "isStale":true pairs
# Process line-by-line, tracking the current object's fields
current_file=""
current_stale=""

while IFS= read -r line; do
  # Detect file field
  f=$(echo "$line" | grep -o '"file":[[:space:]]*"[^"]*"' | sed 's/"file":[[:space:]]*"//;s/"$//' || true)
  if [ -n "$f" ]; then
    current_file="$f"
  fi

  # Detect isStale field
  s=$(echo "$line" | grep -o '"isStale":[[:space:]]*true' || true)
  if [ -n "$s" ]; then
    current_stale="true"
  fi

  # Reset on isStale: false
  sf=$(echo "$line" | grep -o '"isStale":[[:space:]]*false' || true)
  if [ -n "$sf" ]; then
    current_stale=""
  fi

  # On closing brace, process the parsed entry
  if echo "$line" | grep -q '}'; then
    if [ -n "$current_file" ] && [ -n "$current_stale" ]; then
      if [ -f "$nudge_file" ] && grep -qF "$current_file" "$nudge_file"; then
        # Already tracked, skip
        :
      else
        echo "$current_file" >> "$nudge_file"
        echo "Knowledge file \`$current_file\` covers this code and may be stale. Consider reviewing with \`loaf kb review $current_file\`."
      fi
    fi
    current_file=""
    current_stale=""
  fi
done <<< "$result"

exit 0
