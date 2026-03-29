#!/bin/bash
# Hook: Comprehensive session end checklist (INFORMATIONAL)
# SessionEnd hook - agent-aware checklists before session closes
#
# Triggers when Claude Code session ends
# Displays agent-specific completion checklists

# Use CLAUDE_PROJECT_DIR if available, otherwise current directory
SESSIONS_DIR="${CLAUDE_PROJECT_DIR:-.}/.agents/sessions"

# Detect agent type from environment
AGENT_TYPE="${AGENT_TYPE:-unknown}"

# Count session files (excluding archive subdirectory)
SESSION_COUNT=0
if [ -d "$SESSIONS_DIR" ]; then
  SESSION_COUNT=$(find "$SESSIONS_DIR" -maxdepth 1 -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')
fi

if [ "$SESSION_COUNT" -eq 0 ]; then
  # No sessions, no checklist needed
  exit 0
fi

echo "# Session End Checklist"
echo ""
echo "**$SESSION_COUNT** active session file(s) found."
echo ""

# Agent-specific checklists
case "$AGENT_TYPE" in
  main)
    # Coordinator (main session): Full completion checklist
    echo "## Coordinator Checklist"
    echo ""
    echo "- [ ] All spawned implementers/reviewers completed their tasks"
    echo "- [ ] Session outcomes logged in session file"
    echo "- [ ] Linear issues synced with actual progress"
    echo "- [ ] Decisions captured (ADRs if architectural)"
    echo "- [ ] Session file \`## Current State\` is handoff-ready"
    echo "- [ ] **Code review completed** (spawn a reviewer if significant changes)"
    echo "- [ ] Completed sessions deleted (knowledge extracted)"
    echo "- [ ] Active sessions have clear next steps"
    echo ""
    echo "### Code Review Reminder"
    echo ""
    echo "If this session involved significant code changes, consider spawning a reviewer."
    ;;

  implementer)
    # Implementer profile: Outcome reminder
    echo "## Implementer Outcome Checklist"
    echo ""
    echo "- [ ] Tests pass for implemented changes"
    echo "- [ ] Code follows project style conventions"
    echo "- [ ] Session file updated with outcomes (if applicable)"
    echo "- [ ] Changes aligned with session requirements"
    echo "- [ ] Documentation updated (if significant changes)"
    ;;

  reviewer)
    # Reviewer profile: Review completion
    echo "## Reviewer Checklist"
    echo ""
    echo "- [ ] Review findings documented"
    echo "- [ ] Critical issues flagged appropriately"
    echo "- [ ] Session file updated with review outcomes"
    echo "- [ ] Recommendations captured for coordinator"
    ;;

  researcher)
    # Researcher profile: Research deliverables
    echo "## Researcher Checklist"
    echo ""
    echo "- [ ] Research findings documented"
    echo "- [ ] Options and trade-offs clearly presented"
    echo "- [ ] Session file updated with research outcomes"
    echo "- [ ] Recommendations captured for coordinator"
    ;;

  *)
    # Generic checklist
    echo "## General Checklist"
    echo ""
    echo "- [ ] Work outcomes captured appropriately"
    echo "- [ ] Session files current and handoff-ready"
    echo "- [ ] No loose ends or undocumented decisions"
    ;;
esac

echo ""
echo "---"
echo ""
echo "**Active Sessions**: $SESSION_COUNT"
echo "**Tip**: Session files persist across conversations. Keep them current or delete when complete!"

exit 0
