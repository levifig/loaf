/**
 * Commands Module
 *
 * Shared module for command substitution across all build targets.
 * Provides universal unscoped substitution that all targets use.
 * Claude Code adds an optional post-substitution scoping pass separately.
 */

// ─────────────────────────────────────────────────────────────────────────────
// Implementation
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Create a command substitution function for the given target.
 *
 * Returns a function that performs universal unscoped substitution:
 * - {{IMPLEMENT_CMD}} -> /implement
 * - {{ORCHESTRATE_CMD}} -> /implement
 *
 * Note: {{RESUME_CMD}} has been removed - use {{IMPLEMENT_CMD}} for resuming sessions.
 *
 * Claude Code retains an optional post-substitution pass that adds
 * /loaf: scoping (e.g., /implement -> /loaf:implement), but this is
 * applied separately in the target transformer.
 *
 * @param targetName - The target name (reserved for future use)
 * @returns A function that substitutes command placeholders in content
 */
export function createCommandSubstituter(targetName: string): (content: string) => string {
  // targetName is reserved for future target-specific substitutions
  void targetName;

  return (content: string): string => {
    return content
      .replace(/\{\{IMPLEMENT_CMD\}\}/g, "/implement")
      .replace(/\{\{RESUME_CMD\}\}/g, "/implement")
      .replace(/\{\{ORCHESTRATE_CMD\}\}/g, "/implement");
  };
}
