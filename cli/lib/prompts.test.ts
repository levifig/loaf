/**
 * Prompt Helpers Tests
 *
 * Tests for the shared CLI prompt utilities. Focuses on non-TTY behavior
 * since interactive TTY prompts can't be reliably tested in CI.
 */

import { describe, it, expect } from "vitest";
import { isTTY, askYesNo, askChoice } from "./prompts.js";

describe("isTTY", () => {
  it("returns a boolean", () => {
    expect(typeof isTTY()).toBe("boolean");
  });
});

describe("askYesNo", () => {
  it("returns false when stdin is not a TTY (CI/piped)", async () => {
    // In test environments, stdin.isTTY is typically undefined/false
    const result = await askYesNo("Test? [y/N] ");
    expect(result).toBe(false);
  });
});

describe("askChoice", () => {
  it("returns defaultChoice when stdin is not a TTY", async () => {
    const options = ["alpha", "beta", "gamma"];
    const format = (opt: string, i: number) => `  ${i + 1}. ${opt}`;
    const result = await askChoice("Choose: ", options, format, "beta");
    expect(result).toBe("beta");
  });

  it("works with non-string types", async () => {
    const options = [1, 2, 3];
    const format = (opt: number, i: number) => `  ${i + 1}. Option ${opt}`;
    const result = await askChoice("Choose: ", options, format, 2);
    expect(result).toBe(2);
  });
});
