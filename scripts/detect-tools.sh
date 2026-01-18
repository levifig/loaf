#!/usr/bin/env bash
#
# Detect installed AI coding tools
#
# Usage:
#   ./scripts/detect-tools.sh [--json]
#
# Output:
#   Space-separated list of detected tools, or JSON object with details
#
set -euo pipefail

# Output format
JSON_OUTPUT=false
if [[ "${1:-}" == "--json" ]]; then
    JSON_OUTPUT=true
fi

# Detection functions
detect_claude_code() {
    local result='{"installed": false}'

    if command -v claude &> /dev/null; then
        local version
        version=$(claude --version 2>/dev/null || echo "unknown")
        result='{"installed": true, "version": "'"${version}"'", "method": "cli"}'
    fi

    echo "$result"
}

detect_opencode() {
    local result='{"installed": false}'
    local config_dir="${HOME}/.config/opencode"

    if [[ -d "$config_dir" ]]; then
        result='{"installed": true, "config_dir": "'"${config_dir}"'", "method": "config"}'
    fi

    echo "$result"
}

detect_cursor() {
    local result='{"installed": false}'
    local config_dir="${HOME}/.cursor"
    local app_path=""

    # Check for app on macOS
    if [[ -d "/Applications/Cursor.app" ]]; then
        app_path="/Applications/Cursor.app"
    elif [[ -d "${HOME}/Applications/Cursor.app" ]]; then
        app_path="${HOME}/Applications/Cursor.app"
    fi

    # Check for Linux/Windows paths
    if [[ -z "$app_path" ]] && command -v cursor &> /dev/null; then
        app_path=$(which cursor)
    fi

    if [[ -d "$config_dir" ]] || [[ -n "$app_path" ]]; then
        result='{"installed": true'
        if [[ -d "$config_dir" ]]; then
            result+=', "config_dir": "'"${config_dir}"'"'
        fi
        if [[ -n "$app_path" ]]; then
            result+=', "app_path": "'"${app_path}"'"'
        fi
        result+=', "method": "app"}'
    fi

    echo "$result"
}

detect_copilot() {
    local result='{"installed": false}'

    # Check for GitHub CLI with copilot extension
    if command -v gh &> /dev/null; then
        if gh extension list 2>/dev/null | grep -q "copilot"; then
            result='{"installed": true, "method": "gh-extension"}'
        fi
    fi

    # Check for VS Code with Copilot
    local vscode_extensions=""
    if command -v code &> /dev/null; then
        vscode_extensions=$(code --list-extensions 2>/dev/null || echo "")
        if echo "$vscode_extensions" | grep -qi "copilot"; then
            result='{"installed": true, "method": "vscode"}'
        fi
    fi

    # Copilot is always "available" at project level
    if [[ "$result" == '{"installed": false}' ]]; then
        result='{"installed": true, "method": "project-level", "note": "Copy .github/ to project"}'
    fi

    echo "$result"
}

# Main
main() {
    local claude_code opencode cursor copilot
    claude_code=$(detect_claude_code)
    opencode=$(detect_opencode)
    cursor=$(detect_cursor)
    copilot=$(detect_copilot)

    if [[ "$JSON_OUTPUT" == true ]]; then
        echo "{"
        echo '  "claude-code": '"${claude_code}"','
        echo '  "opencode": '"${opencode}"','
        echo '  "cursor": '"${cursor}"','
        echo '  "copilot": '"${copilot}"
        echo "}"
    else
        local tools=()

        if echo "$claude_code" | grep -q '"installed": true'; then
            tools+=("claude-code")
        fi

        if echo "$opencode" | grep -q '"installed": true'; then
            tools+=("opencode")
        fi

        if echo "$cursor" | grep -q '"installed": true'; then
            tools+=("cursor")
        fi

        if echo "$copilot" | grep -q '"installed": true'; then
            tools+=("copilot")
        fi

        echo "${tools[*]}"
    fi
}

main
