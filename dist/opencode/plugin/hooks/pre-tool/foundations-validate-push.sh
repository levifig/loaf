#!/bin/bash
set -euo pipefail

# Pre-push validation hook
# Checks: version bump, CHANGELOG updated, build succeeds
# Language-aware: detects project type and runs appropriate checks

# Read JSON input from stdin
HOOK_INPUT=$(cat)
TOOL_NAME=$(echo "$HOOK_INPUT" | jq -r '.tool_name // ""')
COMMAND=$(echo "$HOOK_INPUT" | jq -r '.tool_input.command // ""')

# Only process Bash tool
[[ "$TOOL_NAME" != "Bash" ]] && exit 0

# Only process git push commands
[[ ! "$COMMAND" =~ ^git[[:space:]]+push ]] && exit 0

ERRORS=()

# Detect project type for version and build checks
HAS_PACKAGE_JSON=false
HAS_BUILD_SCRIPT=false
if [[ -f "package.json" ]]; then
    HAS_PACKAGE_JSON=true
    if jq -e '.scripts.build' package.json >/dev/null 2>&1; then
        HAS_BUILD_SCRIPT=true
    fi
fi

# Check 1: Version bump since last tag (only for projects with package.json version)
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [[ -n "$LAST_TAG" ]] && [[ "$HAS_PACKAGE_JSON" == "true" ]]; then
    CURRENT_VERSION=$(jq -r '.version // empty' package.json 2>/dev/null || echo "")
    if [[ -n "$CURRENT_VERSION" ]]; then
        TAG_VERSION=$(git show "$LAST_TAG:package.json" 2>/dev/null | jq -r '.version // empty' || echo "")
        if [[ -n "$TAG_VERSION" ]] && [[ "$CURRENT_VERSION" == "$TAG_VERSION" ]]; then
            ERRORS+=("Version not bumped since $LAST_TAG (still $CURRENT_VERSION)")
        fi
    fi
fi

# Check 2: CHANGELOG updated since last tag
if [[ -n "$LAST_TAG" ]]; then
    if [[ -f "CHANGELOG.md" ]]; then
        CHANGELOG_CHANGED=$(git diff "$LAST_TAG" --name-only -- CHANGELOG.md 2>/dev/null || echo "")
        if [[ -z "$CHANGELOG_CHANGED" ]]; then
            ERRORS+=("CHANGELOG.md not updated since $LAST_TAG")
        fi
    fi
fi

# Check 3: Build succeeds (only when a build system is detected)
if [[ "$HAS_BUILD_SCRIPT" == "true" ]]; then
    if ! npm run build --silent 2>/dev/null; then
        ERRORS+=("Build failed (npm run build)")
    fi
elif [[ -f "Makefile" ]] && grep -q '^build:' Makefile 2>/dev/null; then
    if ! make build 2>/dev/null; then
        ERRORS+=("Build failed (make build)")
    fi
elif [[ -f "Cargo.toml" ]]; then
    if ! cargo build 2>/dev/null; then
        ERRORS+=("Build failed (cargo build)")
    fi
elif [[ -f "go.mod" ]]; then
    if ! go build ./... 2>/dev/null; then
        ERRORS+=("Build failed (go build ./...)")
    fi
fi

# Report errors and block if any
if [[ ${#ERRORS[@]} -gt 0 ]]; then
    echo "❌ Pre-push validation failed:" >&2
    for err in "${ERRORS[@]}"; do
        echo "   • $err" >&2
    done
    echo "" >&2
    echo "Fix these issues before pushing, or use --no-verify to bypass." >&2
    exit 2
fi

echo "✓ Pre-push validation passed" >&2
exit 0
