# Plan: Build-Time Command Scoping for Claude Code

## Problem
In Claude Code, Loaf commands require the `loaf:` prefix (e.g., `/loaf:breakdown`), but source files use generic `/command` syntax. Currently, only three explicit placeholders are supported.

## Solution
Enhance the Claude Code build target to auto-detect and scope all `/command` references.

## Analysis

The build system already has `substituteCommands()` in `build/targets/claude-code.js` (lines 52-57):

```javascript
function substituteCommands(content) {
  return content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/loaf:implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/loaf:resume")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/loaf:orchestrate");
}
```

This is applied to:
- SKILL.md files (line 477)
- Reference files via `copyReferencesWithSubstitution()` (line 74)
- Command files (line 530)

## Changes

### 1. Enhance `substituteCommands()` in `build/targets/claude-code.js`

Replace the explicit placeholder approach with a regex that detects all Loaf commands:

```javascript
/**
 * Substitute command references with Claude Code scoped commands
 *
 * Handles:
 * - Explicit placeholders: {{IMPLEMENT_CMD}} -> /loaf:implement
 * - Generic slash commands: /breakdown -> /loaf:breakdown
 *   (only for known Loaf commands, not arbitrary text)
 */
function substituteCommands(content, knownCommands) {
  // First handle legacy placeholders for backward compatibility
  let result = content
    .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/loaf:implement")
    .replace(/\{\{RESUME_CMD\}\}/g, "/loaf:resume")
    .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/loaf:orchestrate");

  // Then auto-scope known commands: /command -> /loaf:command
  // Match /command at word boundary, not already scoped
  for (const cmd of knownCommands) {
    // Match /command but not /loaf:command or /other:command
    const pattern = new RegExp(`(?<!/\\w+:)\\/${cmd}(?=\\s|\\)|\\]|,|$|\\`)`, 'g');
    result = result.replace(pattern, `/loaf:${cmd}`);
  }

  return result;
}
```

### 2. Pass known commands to the function

Update call sites to pass the discovered commands list:

```javascript
// In copySkills():
const transformed = substituteCommands(matter.stringify(body, frontmatter), allCommands);

// In copyCommands():
const transformed = substituteCommands(matter.stringify(body, mergedFrontmatter), allCommands);

// In copyReferencesWithSubstitution():
writeFileSync(destPath, substituteCommands(content, allCommands));
```

### 3. Thread `allCommands` through the build

The `buildUnifiedPlugin()` function already discovers commands:
```javascript
const allCommands = discoverCommands(srcDir);
```

Pass this to functions that need it.

## Files Modified

1. `build/targets/claude-code.js` - Enhance substituteCommands(), update call sites

## Verification

1. Run `npm run build`
2. Check a command file in `plugins/loaf/commands/` for correct scoping
3. Check a skill file in `plugins/loaf/skills/` for correct scoping
4. Verify `/breakdown` becomes `/loaf:breakdown` in output
5. Verify `/loaf:implement` (already scoped) stays unchanged
6. Verify non-Loaf commands like `/help` are not modified
