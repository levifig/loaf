#!/bin/bash
# Hook: Pre-compact context preservation with compact journal entry
# PreCompact hook - appends compact entry to session journals before compaction
#
# Triggers before conversation context is compacted
# Outputs instructions for agent to append compact entries

# Use CLAUDE_PROJECT_DIR if available, otherwise current directory
SESSIONS_DIR="${CLAUDE_PROJECT_DIR:-.}/.agents/sessions"

if [ ! -d "$SESSIONS_DIR" ]; then
  exit 0
fi

# Find recently modified sessions (active work)
RECENT_THRESHOLD=60
RECENT_SESSIONS=$(find "$SESSIONS_DIR" -maxdepth 1 -name "*.md" -type f -mmin -${RECENT_THRESHOLD} 2>/dev/null)

# Count active sessions
SESSION_COUNT=$(find "$SESSIONS_DIR" -maxdepth 1 -name "*.md" -type f ! -path "*/archive/*" 2>/dev/null | wc -l | tr -d ' ')

if [ -n "$RECENT_SESSIONS" ] && [ "$SESSION_COUNT" -gt 0 ]; then
  echo "# CONTEXT PRESERVATION REQUIRED"
  echo ""
  echo "**Action Required**: Before compaction, append compact entries to active session journals:"
  echo ""
  
  while read -r session; do
    [ -z "$session" ] && continue
    FILENAME=$(basename "$session")
    
    # Append compact entry via loaf session log
    echo "Run: loaf session log \"compact(session): Context compaction imminent - preserving work state\""
    echo "  (for session: $FILENAME)"
    echo ""
  done <<< "$RECENT_SESSIONS"
  
  echo "After appending compact entries, the transcript will be preserved in \`.agents/transcripts/\`."
  echo ""
  echo "Recent sessions: $SESSION_COUNT active session(s)"
fi

exit 0
