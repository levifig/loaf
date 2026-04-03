#!/usr/bin/env bash
# Regenerate TASKS.md when task files change
# Post-tool hook for Edit|Write on .agents/tasks/**/*.md

set -euo pipefail

# Only run if the edited file is in .agents/tasks/
if [[ -n "${TOOL_INPUT:-}" ]]; then
    # Parse the file path from TOOL_INPUT JSON
    FILE_PATH=$(echo "$TOOL_INPUT" | grep -oE '"file_path"[[:space:]]*:[[:space:]]*"[^"]+"' | sed 's/.*: *"//' | sed 's/"$//' || true)

    # Check if it's a task file
    if [[ "$FILE_PATH" == *".agents/tasks/"*".md" ]]; then
        # Find the project root (where .agents/ exists)
        PROJECT_ROOT=$(pwd)
        while [[ "$PROJECT_ROOT" != "/" && ! -d "$PROJECT_ROOT/.agents" ]]; do
            PROJECT_ROOT=$(dirname "$PROJECT_ROOT")
        done

        if [[ -d "$PROJECT_ROOT/.agents" && -x "$PROJECT_ROOT/scripts/generate-task-board.sh" ]]; then
            "$PROJECT_ROOT/scripts/generate-task-board.sh" > /dev/null 2>&1 || true
        fi
    fi
fi

exit 0
