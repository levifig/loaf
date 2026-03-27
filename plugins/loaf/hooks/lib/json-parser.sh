#!/bin/bash
# JSON Parser Utilities
# Extract fields from Claude Code hook JSON input

parse_file_path() {
    # Extract file_path from hook JSON input
    # Args: $1 - JSON string
    # Returns: File path or empty string

    local json="$1"

    # Try to parse JSON with jq if available
    if command -v jq >/dev/null 2>&1; then
        echo "${json}" | jq -r '.file_path // empty' 2>/dev/null || echo ""
    else
        # Fallback to grep/sed
        echo "${json}" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)".*/\1/' || echo ""
    fi
}

parse_tool_name() {
    # Extract tool_name from hook JSON input
    # Args: $1 - JSON string
    # Returns: Tool name or empty string

    local json="$1"

    if command -v jq >/dev/null 2>&1; then
        echo "${json}" | jq -r '.tool_name // empty' 2>/dev/null || echo ""
    else
        echo "${json}" | grep -o '"tool_name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)".*/\1/' || echo ""
    fi
}

parse_agent_type() {
    # Extract agent_type from hook JSON input (if present)
    # Args: $1 - JSON string
    # Returns: Agent type or empty string

    local json="$1"

    if command -v jq >/dev/null 2>&1; then
        echo "${json}" | jq -r '.agent_type // empty' 2>/dev/null || echo ""
    else
        echo "${json}" | grep -o '"agent_type"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)".*/\1/' || echo ""
    fi
}

parse_parameters() {
    # Extract parameters object from hook JSON input
    # Args: $1 - JSON string
    # Returns: Parameters JSON object or empty string

    local json="$1"

    if command -v jq >/dev/null 2>&1; then
        echo "${json}" | jq -r '.parameters // empty' 2>/dev/null || echo ""
    else
        echo ""
    fi
}

parse_command() {
    # Extract tool_input.command from hook JSON input
    # Args: $1 - JSON string
    # Returns: Command string or empty string

    local json="$1"

    if command -v jq >/dev/null 2>&1; then
        echo "${json}" | jq -r '.tool_input.command // empty' 2>/dev/null || echo ""
    else
        # Fallback: extract "command" value from tool_input object
        echo "${json}" | grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)".*/\1/' || echo ""
    fi
}

parse_exit_code() {
    # Extract exit code from PostToolUse hook JSON input
    # Args: $1 - JSON string
    # Returns: Exit code (integer string) or empty string
    #
    # PostToolUse JSON includes tool_result with execution outcome.
    # Tries tool_result.exit_code first, then tool_result.returncode as fallback.

    local json="$1"

    if command -v jq >/dev/null 2>&1; then
        echo "${json}" | jq -r '.tool_result.exit_code // .tool_result.returncode // empty' 2>/dev/null || echo ""
    else
        # Fallback: try exit_code first, then returncode
        local code
        code=$(echo "${json}" | grep -o '"exit_code"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$' || true)
        if [[ -n "$code" ]]; then
            echo "$code"
        else
            code=$(echo "${json}" | grep -o '"returncode"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$' || true)
            echo "${code:-}"
        fi
    fi
}

# Export functions
export -f parse_file_path
export -f parse_tool_name
export -f parse_agent_type
export -f parse_parameters
export -f parse_command
export -f parse_exit_code
