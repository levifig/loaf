/**
 * CLI Reference Generator
 *
 * Generates cli-reference/SKILL.md content by introspecting the actual CLI
 * structure from Commander.js definitions. This ensures the documentation
 * stays in sync with the code.
 */

import { Command } from "commander";

interface CommandInfo {
  name: string;
  description: string;
  subcommands?: SubcommandInfo[];
  options?: OptionInfo[];
}

interface SubcommandInfo {
  name: string;
  description: string;
  usage?: string;
  options?: OptionInfo[];
}

interface OptionInfo {
  flags: string;
  description: string;
  defaultValue?: string;
}

/**
 * Extract command structure from a Commander.js program
 */
export function extractCliStructure(program: Command): CommandInfo[] {
  const commands: CommandInfo[] = [];

  // Get all subcommands of the main program
  const subcommands = program.commands;

  for (const cmd of subcommands) {
    const cmdInfo: CommandInfo = {
      name: cmd.name(),
      description: cmd.description() || "",
    };

    // Extract subcommands if any
    if (cmd.commands && cmd.commands.length > 0) {
      cmdInfo.subcommands = cmd.commands.map((sub) => {
        const subInfo: SubcommandInfo = {
          name: sub.name(),
          description: sub.description() || "",
        };

        // Extract options
        if (sub.options && sub.options.length > 0) {
          subInfo.options = sub.options.map((opt) => ({
            flags: opt.flags,
            description: opt.description || "",
            defaultValue: opt.defaultValue,
          }));
        }

        return subInfo;
      });
    }

    // Extract options at command level
    if (cmd.options && cmd.options.length > 0) {
      cmdInfo.options = cmd.options.map((opt) => ({
        flags: opt.flags,
        description: opt.description || "",
        defaultValue: opt.defaultValue,
      }));
    }

    commands.push(cmdInfo);
  }

  return commands;
}

/**
 * Generate SKILL.md content from CLI structure
 */
export function generateCliReferenceSkill(commands: CommandInfo[]): string {
  const sections: string[] = [];

  // Header / Frontmatter
  sections.push(`---
name: cli-reference
description: >-
  Documents the Loaf CLI commands and when to use them. Reference for
  {{IMPLEMENT_CMD}}, {{ORCHESTRATE_CMD}}, and all loaf
  subcommands. Use when you need to know which CLI command to invoke.
  Not for skill documentation (use the skill's own SKILL.md) or for
  understanding build internals.
---

# Loaf CLI Reference

Quick reference for all Loaf CLI commands. Each command includes its purpose, common usage patterns, and when to use it.

**Note:** This file is auto-generated from the CLI source code. Do not edit manually.
`);

  // Global Commands section
  sections.push(`## Global Commands

### {{IMPLEMENT_CMD}}
Orchestrates implementation sessions through agent delegation and batch execution.

**Use when:**
- User asks "implement this" or "start working on TASK-XXX"
- Starting a new spec implementation
- Resuming work after context loss

**Usage:**
- {{IMPLEMENT_CMD}} TASK-XXX — Load task, auto-create session
- {{IMPLEMENT_CMD}} SPEC-XXX — Resolve all tasks, build dependency waves
- {{IMPLEMENT_CMD}} TASK-XXX..YYY — Expand range, build waves
- {{IMPLEMENT_CMD}} "description" — Ad-hoc session

### {{ORCHESTRATE_CMD}}
Coordinates multi-agent work: agent delegation, session management, Linear integration.

**Use when:**
- Managing sessions and delegating to agents
- Running council workflows
- Coordinating cross-cutting work

---
`);

  // Individual command sections
  for (const cmd of commands) {
    sections.push(generateCommandSection(cmd));
  }

  // Command substitution reference
  sections.push(`## Command Substitution Reference

The following placeholders are substituted at build time per target:

| Placeholder | Claude Code | OpenCode | Cursor |
|-------------|-------------|----------|--------|
| \`{{IMPLEMENT_CMD}}\` | \`/implement\` | \`/implement\` | \`@loaf/implement\` |
| \`{{ORCHESTRATE_CMD}}\` | \`/implement\` | \`/implement\` | \`@loaf/implement\` |

---

## Quick Decision Guide

**Need to start working?** → \`{{IMPLEMENT_CMD}} TASK-XXX\`

**Need to continue after restart?** → \`loaf session start\` then \`{{IMPLEMENT_CMD}}\`

**Need to coordinate agents?** → \`{{ORCHESTRATE_CMD}}\`

**Made changes to skills?** → \`loaf build && loaf install --to <target>\`

**Want to see what's in progress?** → \`loaf task list --active\`

**Ready to archive completed work?** → \`loaf task archive TASK-XXX\`

**Need to check knowledge freshness?** → \`loaf kb check\`
`);

  return sections.join("\n");
}

function generateCommandSection(cmd: CommandInfo): string {
  const parts: string[] = [];

  parts.push(`## ${capitalizeFirst(cmd.name)} Management`);
  parts.push("");
  parts.push(`### \`loaf ${cmd.name}\``);
  parts.push(cmd.description);
  parts.push("");

  if (cmd.subcommands && cmd.subcommands.length > 0) {
    parts.push("**Subcommands:**");
    parts.push("");
    parts.push("| Subcommand | Purpose |");
    parts.push("|------------|---------|");

    for (const sub of cmd.subcommands) {
      parts.push(`| \`loaf ${cmd.name} ${sub.name}\` | ${sub.description} |`);
    }

    parts.push("");

    // Add usage examples for subcommands with options
    const subcommandsWithOptions = cmd.subcommands.filter(
      (s) => s.options && s.options.length > 0
    );

    if (subcommandsWithOptions.length > 0) {
      parts.push("**Options:**");
      parts.push("");

      for (const sub of subcommandsWithOptions) {
        parts.push(`- \`loaf ${cmd.name} ${sub.name}\`:`);
        for (const opt of sub.options!) {
          parts.push(`  - \`${opt.flags}\` — ${opt.description}`);
        }
        parts.push("");
      }
    }
  }

  // Add example usage section
  parts.push("**Usage:**");
  parts.push("```bash");

  // Generate usage examples based on subcommands
  if (cmd.subcommands && cmd.subcommands.length > 0) {
    // Find subcommands that are commonly used
    const commonSubcommands = cmd.subcommands.slice(0, 3);
    for (const sub of commonSubcommands) {
      parts.push(`loaf ${cmd.name} ${sub.name}`);
    }
  } else {
    parts.push(`loaf ${cmd.name}`);
  }

  parts.push("```");
  parts.push("");
  parts.push("---");
  parts.push("");

  return parts.join("\n");
}

function capitalizeFirst(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Generate JSON output of CLI structure for agent consumption
 */
export function generateCliJson(program: Command): string {
  const commands = extractCliStructure(program);
  return JSON.stringify(
    {
      name: "loaf",
      description: "Loaf — An Opinionated Agentic Framework",
      commands,
    },
    null,
    2
  );
}
