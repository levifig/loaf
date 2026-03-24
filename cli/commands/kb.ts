/**
 * loaf kb command
 *
 * Parent command for knowledge base management. Subcommands will be
 * added in subsequent tasks (TASK-034, TASK-035).
 */

import { Command } from "commander";

export function registerKbCommand(program: Command): void {
  program
    .command("kb")
    .description("Knowledge base management");
}
