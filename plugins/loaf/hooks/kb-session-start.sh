#!/bin/bash
# Hook: KB staleness advisory (INFORMATIONAL)
# SessionStart hook - surfaces stale knowledge count at session start
#
# Triggers on session start
# Displays count of stale knowledge files if any exist

# Check if loaf is available
if ! command -v loaf &>/dev/null; then
  exit 0
fi

# Get KB status as JSON
status=$(loaf kb status --json 2>/dev/null)
if [ $? -ne 0 ] || [ -z "$status" ]; then
  exit 0
fi

# Extract counts from JSON (no jq dependency)
# JSON fields: total_files, stale
stale=$(echo "$status" | grep -o '"stale":[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
total=$(echo "$status" | grep -o '"total_files":[[:space:]]*[0-9]*' | grep -o '[0-9]*$')

if [ -z "$stale" ] || [ "$stale" = "0" ]; then
  exit 0
fi

echo ""
echo "## Knowledge Base"
echo ""
echo "**${total}** knowledge files tracked. **${stale}** stale."
echo ""
echo "Run \`loaf kb check\` for details or \`loaf kb review <file>\` to mark reviewed."

exit 0
