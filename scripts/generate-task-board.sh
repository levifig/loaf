#!/usr/bin/env bash
#
# generate-task-board.sh
#
# Generates .agents/TASKS.md from task files.
# Called by hook whenever task files change.
#
# Usage: ./scripts/generate-task-board.sh
#

set -euo pipefail

# Resolve paths relative to repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AGENTS_DIR="$REPO_ROOT/.agents"
TASKS_DIR="$AGENTS_DIR/tasks"
ARCHIVE_DIR="$TASKS_DIR/archive"
OUTPUT_FILE="$AGENTS_DIR/TASKS.md"

# Temporary files for sorting
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Extract YAML frontmatter field value
# Usage: get_field "field_name" "file_path"
get_field() {
    local field="$1"
    local file="$2"
    # Extract value between --- markers, handling quoted strings
    # Use grep || true to avoid failing on no match
    sed -n '/^---$/,/^---$/p' "$file" | \
        (grep -E "^${field}:" || true) | \
        head -1 | \
        sed -E "s/^${field}:[[:space:]]*//" | \
        sed -E 's/^"(.*)"/\1/' | \
        sed -E "s/^'(.*)'/\1/"
}

# Get spec filename from spec ID
# Usage: get_spec_filename "SPEC-001"
get_spec_filename() {
    local spec_id="$1"
    local spec_file
    spec_file=$(find "$AGENTS_DIR/specs" -name "${spec_id}-*.md" -type f 2>/dev/null | head -1)
    if [ -n "$spec_file" ]; then
        basename "$spec_file"
    else
        echo ""
    fi
}

# Format a task line for output
# Usage: format_task_line "task_file" "link_path"
format_task_line() {
    local task_file="$1"
    local link_path="$2"

    local id title spec spec_file
    id=$(get_field "id" "$task_file")
    title=$(get_field "title" "$task_file")
    spec=$(get_field "spec" "$task_file")
    spec_file=$(get_spec_filename "$spec")

    if [ -n "$spec_file" ]; then
        echo "- [${id}](${link_path}) - ${title} ([${spec}](specs/${spec_file}))"
    else
        echo "- [${id}](${link_path}) - ${title}"
    fi
}

# Format a completed task line (includes date)
# Usage: format_completed_line "task_file" "link_path"
format_completed_line() {
    local task_file="$1"
    local link_path="$2"

    local id title spec spec_file updated completed date_str
    id=$(get_field "id" "$task_file")
    title=$(get_field "title" "$task_file")
    spec=$(get_field "spec" "$task_file")
    spec_file=$(get_spec_filename "$spec")

    # Try completed field first, then updated, then file mtime
    completed=$(get_field "completed" "$task_file")
    if [ -z "$completed" ]; then
        completed=$(get_field "updated" "$task_file")
    fi

    # Parse ISO 8601 date to "YYYY-MM-DD HH:MM" format
    if [ -n "$completed" ]; then
        # Handle ISO 8601 format: 2026-01-24T14:35:00Z
        date_str=$(echo "$completed" | sed -E 's/T/ /' | sed -E 's/:[0-9]{2}Z$//' | cut -c1-16)
    else
        # Fallback to file modification time
        date_str=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$task_file" 2>/dev/null || \
                   stat -c "%y" "$task_file" 2>/dev/null | cut -c1-16)
    fi

    if [ -n "$spec_file" ]; then
        echo "- ${date_str} - [${id}](${link_path}) - ${title} ([${spec}](specs/${spec_file}))"
    else
        echo "- ${date_str} - [${id}](${link_path}) - ${title}"
    fi
}

# Collect tasks from a directory
# Usage: collect_tasks "directory" "is_archive"
collect_tasks() {
    local dir="$1"
    local is_archive="${2:-false}"

    if [ ! -d "$dir" ]; then
        return
    fi

    local task_file status priority link_path

    # Find task files
    # For non-archive, exclude the archive subdirectory
    local find_cmd
    if [ "$is_archive" = "true" ]; then
        find_cmd="find \"$dir\" -name \"TASK-*.md\" -type f"
    else
        find_cmd="find \"$dir\" -path \"$dir/archive\" -prune -o -name \"TASK-*.md\" -type f -print"
    fi

    eval "$find_cmd" | while read -r task_file; do
        status=$(get_field "status" "$task_file")
        priority=$(get_field "priority" "$task_file")

        # Default priority to P2 if missing
        if [ -z "$priority" ]; then
            priority="P2"
        fi

        # Calculate relative path from .agents/TASKS.md
        if [ "$is_archive" = "true" ]; then
            # Archive path: tasks/archive/YYYY-MM/TASK-XXX-name.md
            link_path=$(echo "$task_file" | sed "s|$AGENTS_DIR/||")
        else
            # Active path: relative from TASKS.md, preserving subdirectory structure
            link_path=$(echo "$task_file" | sed "s|$AGENTS_DIR/||")
        fi

        # Normalize status values
        case "$status" in
            in_progress|in-progress|inprogress)
                echo "$task_file|in_progress|$priority|$link_path" >> "$TMP_DIR/in_progress.txt"
                ;;
            done|completed|complete)
                echo "$task_file|done|$priority|$link_path" >> "$TMP_DIR/completed.txt"
                ;;
            todo|pending|"")
                echo "$task_file|todo|$priority|$link_path" >> "$TMP_DIR/todo.txt"
                ;;
            *)
                # Unknown status, treat as todo
                echo "$task_file|todo|$priority|$link_path" >> "$TMP_DIR/todo.txt"
                ;;
        esac
    done
}

# Main generation logic
main() {
    # Initialize temp files
    touch "$TMP_DIR/in_progress.txt"
    touch "$TMP_DIR/todo.txt"
    touch "$TMP_DIR/completed.txt"

    # Collect tasks from main directory (handles both root and active/ subdirectory)
    collect_tasks "$TASKS_DIR" false

    # Collect tasks from archive
    collect_tasks "$ARCHIVE_DIR" true

    # Start generating output
    {
        echo "# Tasks"
        echo ""

        # In Progress section
        echo "## In Progress"
        if [ -s "$TMP_DIR/in_progress.txt" ]; then
            while IFS='|' read -r task_file status priority link_path; do
                format_task_line "$task_file" "$link_path"
            done < "$TMP_DIR/in_progress.txt"
        else
            echo "_No tasks in progress_"
        fi
        echo ""

        # To Do section with priority subgroups
        echo "## To Do"
        echo ""

        local has_p1=false has_p2=false has_p3=false

        if [ -s "$TMP_DIR/todo.txt" ]; then
            # Check which priorities exist (|| true to avoid set -e failure)
            grep -q '|P1|' "$TMP_DIR/todo.txt" 2>/dev/null && has_p1=true || true
            grep -q '|P2|' "$TMP_DIR/todo.txt" 2>/dev/null && has_p2=true || true
            grep -q '|P3|' "$TMP_DIR/todo.txt" 2>/dev/null && has_p3=true || true

            # P1 - High
            if [ "$has_p1" = true ]; then
                echo "### P1 - High"
                grep '|P1|' "$TMP_DIR/todo.txt" | sort | while IFS='|' read -r task_file status priority link_path; do
                    format_task_line "$task_file" "$link_path"
                done
                echo ""
            fi

            # P2 - Normal
            if [ "$has_p2" = true ]; then
                echo "### P2 - Normal"
                grep '|P2|' "$TMP_DIR/todo.txt" | sort | while IFS='|' read -r task_file status priority link_path; do
                    format_task_line "$task_file" "$link_path"
                done
                echo ""
            fi

            # P3 - Low
            if [ "$has_p3" = true ]; then
                echo "### P3 - Low"
                grep '|P3|' "$TMP_DIR/todo.txt" | sort | while IFS='|' read -r task_file status priority link_path; do
                    format_task_line "$task_file" "$link_path"
                done
                echo ""
            fi
        else
            echo "_No pending tasks_"
            echo ""
        fi

        # Completed section (separator line before)
        echo "---"
        echo ""
        echo "## Completed"
        if [ -s "$TMP_DIR/completed.txt" ]; then
            # Sort by completed/updated date (reverse chronological)
            # First, create a sortable list with dates
            > "$TMP_DIR/completed_sorted.txt"
            while IFS='|' read -r task_file status priority link_path; do
                local completed sort_date
                completed=$(get_field "completed" "$task_file")
                if [ -z "$completed" ]; then
                    completed=$(get_field "updated" "$task_file")
                fi
                if [ -z "$completed" ]; then
                    # Fallback to file mtime in ISO format for sorting
                    completed=$(stat -f "%Sm" -t "%Y-%m-%dT%H:%M:%S" "$task_file" 2>/dev/null || \
                               date -r "$task_file" +"%Y-%m-%dT%H:%M:%S" 2>/dev/null || \
                               echo "1970-01-01T00:00:00")
                fi
                echo "$completed|$task_file|$link_path" >> "$TMP_DIR/completed_sorted.txt"
            done < "$TMP_DIR/completed.txt"

            # Now sort and output
            sort -r "$TMP_DIR/completed_sorted.txt" | while IFS='|' read -r sort_date task_file link_path; do
                format_completed_line "$task_file" "$link_path"
            done
        else
            echo "_No completed tasks_"
        fi
    } > "$OUTPUT_FILE"

    echo "Generated $OUTPUT_FILE"
}

main "$@"
