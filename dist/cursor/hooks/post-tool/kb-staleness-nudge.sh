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

# Parse the edited file path from TOOL_INPUT JSON
# TOOL_INPUT contains the tool's input as a JSON string
if [ -z "${TOOL_INPUT:-}" ]; then
  exit 0
fi

# Extract file_path (primary field for Edit/Write tools)
file_path=$(echo "$TOOL_INPUT" | grep -oE '"file_path"[[:space:]]*:[[:space:]]*"[^"]+"' | sed 's/.*: *"//' | sed 's/"$//' || true)

if [ -z "$file_path" ]; then
  exit 0
fi

# Check which knowledge files cover this path
result=$(loaf kb check --file "$file_path" --json 2>/dev/null || true)
if [ -z "$result" ] || [ "$result" = "[]" ]; then
  exit 0
fi

# Track already-nudged files per session
# PPID is the parent process (Claude Code), stable across hook invocations in one session
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"
nudge_file="/tmp/loaf-kb-nudged-${PPID}-$(echo "$PROJECT_DIR" | md5sum 2>/dev/null | cut -c1-8 || echo "$PROJECT_DIR" | md5 2>/dev/null | cut -c1-8 || echo "default")"

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

  # On closing brace, check if we have a stale file to report
  if echo "$line" | grep -q '}'; then
    if [ -n "$current_stale" ] && [ -n "$current_file" ]; then
      # Check if already nudged this session
      if [ -f "$nudge_file" ] && grep -qF "$current_file" "$nudge_file"; then
        # Already nudged, skip
        :
      else
        # Record as nudged
        echo "$current_file" >> "$nudge_file"
        echo "Knowledge file \`$current_file\` covers this code and may be stale. Consider reviewing with \`loaf kb review $current_file\`."
      fi
    fi
    current_file=""
    current_stale=""
  fi
done <<< "$result"

exit 0
