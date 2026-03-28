/**
 * Shared CLI Prompt Helpers
 *
 * Reusable prompt utilities for interactive CLI commands.
 * All helpers are TTY-safe: they return defaults when stdin is not a terminal.
 */

import { createInterface } from "readline";

/**
 * Check if stdout is a TTY (interactive terminal).
 * Returns false when output is piped or in CI environments.
 */
export function isTTY(): boolean {
  return !!process.stdout.isTTY;
}

/**
 * Ask a yes/no question. Returns false if non-TTY.
 */
export function askYesNo(question: string): Promise<boolean> {
  if (!process.stdin.isTTY) {
    return Promise.resolve(false);
  }

  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    let resolved = false;
    rl.on("close", () => {
      if (!resolved) {
        resolved = true;
        resolve(false);
      }
    });
    rl.question(question, (answer) => {
      resolved = true;
      rl.close();
      resolve(answer.trim().toLowerCase().startsWith("y"));
    });
  });
}

/**
 * Ask the user to choose from a numbered list of options.
 * Returns defaultChoice if non-TTY or invalid input.
 *
 * @param question - Prompt text (e.g., "  Choose [1/2/3]: ")
 * @param options - Array of options to choose from
 * @param format - Function to format each option for display (called before prompting)
 * @param defaultChoice - Fallback for non-TTY or invalid input
 */
export function askChoice<T>(
  question: string,
  options: T[],
  format: (option: T, index: number) => string,
  defaultChoice: T,
): Promise<T> {
  if (!process.stdin.isTTY) {
    return Promise.resolve(defaultChoice);
  }

  // Print formatted options before prompting
  for (let i = 0; i < options.length; i++) {
    console.log(format(options[i], i));
  }

  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    let resolved = false;
    rl.on("close", () => {
      if (!resolved) {
        resolved = true;
        resolve(defaultChoice);
      }
    });
    rl.question(question, (answer) => {
      resolved = true;
      rl.close();
      const num = parseInt(answer.trim(), 10);
      if (num >= 1 && num <= options.length) {
        resolve(options[num - 1]);
      } else {
        resolve(defaultChoice);
      }
    });
  });
}
