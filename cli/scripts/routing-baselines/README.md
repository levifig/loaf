# Skill routing baselines

Measured results of `cli/scripts/eval-skill-routing.mjs`, kept so that changes to skill descriptions, such as merging skills, can be compared against how routing behaved before them. Each JSON file is the runner's own report, kept exactly as written. Its name gives the date, the model and the commit it measured.

## 2026-11-24: 38-skill inventory before the skill merges

Measured at commit `e5523b3e`. The run covered the `all` suite: 190 cases, made up of 131 core cases (each a request with one expected skill) and 59 conflict probes. Each model ran once, with a live call per case.

| Model | Overall | Core | Conflict probes | Merge probes |
|---|---|---|---|---|
| `claude-opus-5-5` | 174/190 (91.6%) | 121/131 | 53/59 | 25/27 |
| `claude-sonnet-5-5` | 171/190 (90.0%) | 120/131 | 51/59 | 23/27 |

These scores reflect a label correction made after the run.

- **The correction:** two merge probes were labelled `brainstorm` and `explore`, but those skills' own descriptions say requests like them belong to `pitch`. Their `expected` value is now `pitch`. The prompts are unchanged, so each recorded answer still applies, and the totals above were rescored from those answers.
- **The uncorrected scores:** the `passed` fields and totals inside the JSON files use the original labels. They give Opus 176/190 with 27/27 merge probes, and Sonnet 173/190 with 25/27.

Merge probes are the 27 conflict probes added for the merges. Each carries `expected_after`: the skill that should own the request once the overlapping skills are merged. `expected` is the right skill today. Two older discovery probes also carry `expected_after`. Today, 12 of the 27 merge probes already land on their `expected_after` owner on Opus, and 10 on Sonnet.

Per-case answers are in each file under `results`. Per-skill and per-group summaries are under `by_skill` and `by_group`.

### What the failures show

These patterns hold for both models unless a model is named.

- **Implementation requests go to project-management.** This covers "implement this", "start working on TASK-042" and "let's build this out". Its description sends concrete implementation requests through it first, so the failures reflect test expectations that predate that rule. Sonnet also sends a confirmed, selected "write the code" request there instead of to implement.
- **Loaf CLI operation has no clear owner.** Requests to upgrade Loaf, list command flags, or repair stale installed targets go to bootstrap, foundations or debugging instead of loaf-reference. loaf-reference's shipped description is about Flow semantics, not operating the CLI.
- **The discovery skills already overlap.**
  - explore, brainstorm and pitch are confused with one another and with research. On Sonnet, "Explore several approaches before we choose one" goes to project-management.
  - brainstorm's and explore's descriptions send user requests to pitch, but that route falls past the 250-character cut. The model sees only that they are not entry points, and the two relabelled probes still go to brainstorm or explore.
  - On Opus, the two older discovery probes answered pitch. Their `expected` values of explore and brainstorm predate that routing rule, like the implement cases.
  - An open-ended "poke around offline mode" request goes to research on Sonnet.
- **Linear requests leak.** "Triage the high-priority bugs in our Linear workspace" goes to triage. "Audit this Linear project" goes to research.
- **Some requests that use evidence go to research.** A request to sort what a dogfood run showed from what was only inferred goes to research instead of reflect.
- **Older vocabulary misroutes.** The probe about "deferred intents", a retired concept, goes to project-management instead of triage.
- **Ambiguous wording pulls toward ship.** On Sonnet, "Run the general quality checklist before we ship" goes to ship instead of foundations.

### Comparing after a merge

The runner scores each case only against `expected`. It also refuses to run while any case names a skill that no longer exists. So a merge change updates the cases in the same change that retires the skills:

1. Keep every prompt byte-identical, so the answers stay comparable with this baseline.
2. For each probe that has `expected_after`, set `expected` to that value.
3. For older probes without `expected_after` whose `expected` is a retired skill, set `expected` to the new owner or drop the probe. Seven probes are in this position: three for triage, and one each for loaf-reference, orchestration, strategy and documentation-standards.
4. Remove retired skills from `choices`.
5. Move the retired skills' core cases to their new owners, or drop the ones whose intent no longer applies.
6. Rerun on the same models and compare per-case answers with this baseline. Focus on the merge probes and the affected skills' core cases.

Some `expected_after` values are assumptions that the merge work must confirm:

- **foundations:** after the rename planned with the merges, read it as `loaf-core`. The rename change must update every `foundations` reference in the cases: the four merge probes, the older probes that name it, and its core-case key.
- **Proofreading a CONTRIBUTING guide:** `expected_after: architecture` assumes architecture takes over documentation-standards' review guidance. The merge plan doesn't name that owner yet.

### Limits

- The runner shows each skill's name and only the first 250 characters of its description. Harnesses show more, so these results measure the start of each description, not the full listing an agent sees.
- The system prompt frames the model as Claude Code and asks for a skill name only. Other harnesses, and real conversations with history, may route differently.
- There was one run per model and no repeats. A single run doesn't show how much the results vary.
- Only Anthropic models were measured.
- Replies may include thinking before the answer. The runner allows 1024 output tokens and reads the first text block.
- The `cost` figure in each file is unreliable. The runner's pricing table predates these models and falls back to older prices. Use the token counts instead: 637,865 input tokens per model, with 8,095 output tokens for Opus and 11,669 for Sonnet.
- The runs authenticated with an OAuth access token from `ant auth print-credentials --access-token`. The maintainer approved that route, which goes beyond the case-only scope first planned for this work.
