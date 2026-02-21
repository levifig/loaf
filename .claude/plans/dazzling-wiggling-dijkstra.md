# Plan: Pre-Push Validation Hook

## Goal

Create a blocking pre-tool hook that validates `git push` commands before allowing them to execute. The hook ensures code quality gates are met before pushing to remote.

## Checks to Implement

| Check | Condition | Message |
|-------|-----------|---------|
| Version bump | `package.json` version unchanged since last git tag | "Version not bumped since last release" |
| CHANGELOG | No new entries in CHANGELOG.md since last tag | "CHANGELOG.md not updated" |
| Build | `npm run build` fails | "Build failed - fix before pushing" |

**Failure mode:** Blocking (exit 2 to prevent push)

## Implementation

### 1. Create Hook Script

**File:** `src/hooks/pre-tool/foundations-validate-push.sh`

```bash
#!/bin/bash
set -euo pipefail

# Read JSON input from stdin
HOOK_INPUT=$(cat)
TOOL_NAME=$(echo "$HOOK_INPUT" | jq -r '.tool_name // ""')
COMMAND=$(echo "$HOOK_INPUT" | jq -r '.tool_input.command // ""')

# Only process Bash tool
[[ "$TOOL_NAME" != "Bash" ]] && exit 0

# Only process git push commands
[[ ! "$COMMAND" =~ ^git[[:space:]]+push ]] && exit 0

ERRORS=()

# Check 1: Version bump since last tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [[ -n "$LAST_TAG" ]]; then
    CURRENT_VERSION=$(jq -r '.version' package.json 2>/dev/null || echo "")
    TAG_VERSION=$(git show "$LAST_TAG:package.json" 2>/dev/null | jq -r '.version' || echo "")
    if [[ "$CURRENT_VERSION" == "$TAG_VERSION" ]]; then
        ERRORS+=("Version not bumped since $LAST_TAG (still $CURRENT_VERSION)")
    fi
fi

# Check 2: CHANGELOG updated since last tag
if [[ -n "$LAST_TAG" ]]; then
    CHANGELOG_CHANGED=$(git diff "$LAST_TAG" --name-only -- CHANGELOG.md 2>/dev/null || echo "")
    if [[ -z "$CHANGELOG_CHANGED" ]]; then
        ERRORS+=("CHANGELOG.md not updated since $LAST_TAG")
    fi
fi

# Check 3: Build succeeds
if ! npm run build --silent 2>/dev/null; then
    ERRORS+=("Build failed (npm run build)")
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
```

### 2. Register Hook in hooks.yaml

**File:** `src/config/hooks.yaml`

Add to `hooks.pre-tool` section:

```yaml
- id: foundations-validate-push
  skill: foundations
  script: hooks/pre-tool/foundations-validate-push.sh
  matcher: "Bash"
  blocking: true
  timeout: 60000  # 60s for build
  description: "Validates version bump, CHANGELOG, and build before git push"
```

### 3. Build and Test

```bash
npm run build
```

## Files to Create/Modify

| File | Action |
|------|--------|
| `src/hooks/pre-tool/foundations-validate-push.sh` | Create |
| `src/config/hooks.yaml` | Add hook registration |

## Verification

1. **Test blocking on version not bumped:**
   - Reset version to match last tag
   - Attempt `git push` → should block

2. **Test blocking on CHANGELOG not updated:**
   - Revert CHANGELOG to last tag state
   - Attempt `git push` → should block

3. **Test blocking on build failure:**
   - Introduce syntax error in source
   - Attempt `git push` → should block

4. **Test success path:**
   - Bump version, update CHANGELOG, ensure build passes
   - Attempt `git push` → should succeed

5. **Build succeeds:**
   ```bash
   npm run build
   ```

## Edge Cases

- **No tags exist:** Skip version/CHANGELOG checks (first release)
- **No package.json:** Skip version check
- **No CHANGELOG.md:** Skip CHANGELOG check
- **Bypass:** User can use `git push --no-verify` if truly needed (Claude Code respects this)
