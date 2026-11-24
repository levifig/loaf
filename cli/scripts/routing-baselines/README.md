# Skill routing baselines

Measured results of `cli/scripts/eval-skill-routing.mjs`, kept so that changes to skill descriptions, such as merging skills, can be compared against how routing behaved before them. Each JSON file is the runner's own report. Its name gives the date, the model and the commit it measured.

## 2026-11-24: 38-skill inventory before the skill merges

Measured at commit `e5523b3e`. The run covered the `all` suite: 190 cases, made up of 131 core cases (each a request with one expected skill) and 59 conflict probes. Each model ran once, with a live call per case.

| Model | Overall | Core | Conflict probes | Merge probes |
|---|---|---|---|---|
| `claude-opus-5-5` | 176/190 (92.6%) | 121/131 | 55/59 | 27/27 |
| `claude-sonnet-5-5` | 173/190 (91.1%) | 120/131 | 53/59 | 25/27 |

Merge probes are the 27 conflict probes that carry `expected_after`. They cover each skill pair the planned merges affect. `expected` is the right skill today, and `expected_after` is the skill that should own the request once the overlapping skills are merged. To check a merge, rerun the same probes and score them against `expected_after`.

Per-case and per-skill results are in each file, under `results`, `by_skill` and `by_group`.

### What the failures show

- **Implementation requests go to project-management.** All three core cases phrased as "implement this", "start working on TASK-042" and "let's build this out" go there on both models. Its description sends concrete implementation requests through it first, so these failures reflect test expectations that predate that rule, not misrouting.
- **Loaf CLI operation has no clear owner.** Requests to upgrade Loaf, list command flags, or repair stale installed targets go to bootstrap, foundations or debugging instead of loaf-reference. loaf-reference's shipped description is about Flow semantics, not operating the CLI.
- **The discovery skills already overlap.** explore, brainstorm and pitch are confused with one another and with research on both models.
- **Sonnet misses two merge probes.** A confirmed, selected request to write code went to project-management instead of implement. An open-ended "poke around offline mode" request went to research instead of pitch.

### Limits

- The runner shows each skill's name and only the first 250 characters of its description. Harnesses show more, so these results measure the start of each description, not the full listing an agent sees.
- The system prompt frames the model as Claude Code and asks for a skill name only. Other harnesses, and real conversations with history, may route differently.
- There was one run per model and no repeats. A single run doesn't show how much the results vary.
- Only Anthropic models were measured.
- Replies may include thinking before the answer. The runner allows 1024 output tokens and reads the first text block.
- The `cost` figure in each file is unreliable. The runner's pricing table predates these models and falls back to older prices. Use the token counts instead: 637,865 input tokens per model, with 8,095 output tokens for Opus and 11,669 for Sonnet.
- The runs authenticated with an OAuth access token from `ant auth print-credentials --access-token`.
