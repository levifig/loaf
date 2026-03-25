#!/bin/bash
# Hook: KB consolidation prompt (INFORMATIONAL)
# SessionEnd hook - reminds about stale knowledge at session end
#
# Reads the nudge tracking file created by kb-staleness-nudge post-tool hook
# and reports how many stale knowledge files were flagged during the session

# Match the nudge file key from the post-tool hook
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"
nudge_file="/tmp/loaf-kb-nudged-${PPID}-$(echo "$PROJECT_DIR" | md5sum 2>/dev/null | cut -c1-8 || echo "$PROJECT_DIR" | md5 2>/dev/null | cut -c1-8 || echo "default")"

if [ ! -f "$nudge_file" ]; then
  exit 0
fi

# Read the list of nudged files before cleanup
nudged_files=()
while IFS= read -r line; do
  [ -n "$line" ] && nudged_files+=("$line")
done < "$nudge_file"

# Clean up the tracking file
rm -f "$nudge_file"

nudged_count=${#nudged_files[@]}

if [ "$nudged_count" -gt 0 ]; then
  echo ""
  echo "## Knowledge Base"
  echo ""
  echo "**${nudged_count}** stale knowledge file(s) were flagged during this session:"
  echo ""
  for f in "${nudged_files[@]}"; do
    echo "- \`$f\`"
  done
  echo ""
  echo "Run \`loaf kb check\` to review or \`loaf kb review <file>\` to mark as reviewed."
fi

exit 0
