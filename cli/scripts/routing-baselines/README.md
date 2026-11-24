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

### Case changes since this baseline

- **brainstorm and explore folded into pitch, strategy into reflect:** explore's three core cases now expect `pitch` and strategy's three expect `reflect`. brainstorm's three core cases were dropped because each asks for "the brainstorm technique" by name, which no longer exists. The four discovery and three strategy merge probes now expect their `expected_after` owner. So do the older "generate a few possible directions" and "help me explore a few" probes, which also carry `expected_after: pitch`. The older "Define the product direction and personas" probe now expects `reflect`. The retired skills are gone from every probe's `choices`. Group names are unchanged so per-group results stay comparable. The suite is now 187 cases.

## 2026-11-24: after folding brainstorm, explore and strategy into pitch and reflect

Measured at commit `606c0766`, with the same `all` suite of 187 cases, once per model. These runs measure only this merge: they predate the other planned merges and the `foundations` rename.

The comparison joins each case to its baseline answer by prompt. It scores both runs against the skill that should own the request after the merges (`expected_after` where present). Explore's moved core cases count as right in the baseline when they reached explore, which owned them then. "Affected" means the 28 cases whose owner is now pitch or reflect, plus any case either run answered with one of those skills or a retired one.

| Model | Run's own score | All shared cases, against the post-merge owner | Affected cases |
|---|---|---|---|
| `claude-opus-5-5` | 174/187 | 159 → 163 | 21 → 25 of 28 |
| `claude-sonnet-5-5` | 171/187 | 154 → 160 | 18 → 25 of 28 |

**What improved:**
- Every discovery and strategy merge probe now lands on pitch or reflect on both models.
- So do the older "generate a few possible directions" and "help me explore a few" probes, and "Define the product direction and personas".

**What regressed:**
- **Resuming an Exploration has no owner.** "Resume the workflow exploration where we left off" goes to handoff on both models. On Sonnet, "Start an exploration of the caching options and checkpoint it" goes to research. Explore's description claimed these requests. Now the checkpoint guidance lives in Foundations' continuity reference, and no skill's description names Explorations.
- **One Opus core case moved to idea.** "I have a concept to pitch" goes to idea. Sonnet still routes it to pitch. With one run per model, this may be run-to-run variation.
- **Two unrelated Sonnet core cases moved.** "Which loaf command shows the recent project journal?" goes to foundations, and "Ready to merge this PR" goes to git-workflow. Neither skill's description changed, so these are likely variation too.

**Still wrong in both runs:** the dogfood "observed versus inferred" request still goes to research on both models.

Token use was 582,723 input tokens per model, with 5,496 output tokens for Opus and 8,662 for Sonnet. The `cost` field is unreliable for the reason given under Limits.

### Limits

- The runner shows each skill's name and only the first 250 characters of its description. Harnesses show more, so these results measure the start of each description, not the full listing an agent sees.
- The system prompt frames the model as Claude Code and asks for a skill name only. Other harnesses, and real conversations with history, may route differently.
- There was one run per model and no repeats. A single run doesn't show how much the results vary.
- Only Anthropic models were measured.
- Replies may include thinking before the answer. The runner allows 1024 output tokens and reads the first text block.
- The `cost` figure in each file is unreliable. The runner's pricing table predates these models and falls back to older prices. Use the token counts instead: 637,865 input tokens per model, with 8,095 output tokens for Opus and 11,669 for Sonnet.
- The runs authenticated with an OAuth access token from `ant auth print-credentials --access-token`. The maintainer approved that route, which goes beyond the case-only scope first planned for this work.

## 2026-11-24: after folding four skills into loaf-core, architecture and release

Measured at commit `4562252f`, with 34 skills: `foundations` renamed to `loaf-core`, and `knowledge-base`, `documentation-standards`, `debugging` and `security-compliance` folded into `loaf-core`, architecture and release ([#470](https://github.com/levifig/loaf/issues/470)). The run used the `all` suite once per model, with the cases updated as "Comparing after a merge" describes. Prompts are unchanged.

| Model | Overall | Core | Conflict probes | Affected cases |
|---|---|---|---|---|
| `claude-opus-5-5` | 171/190 | 118/131 | 53/59 | 31/33 |
| `claude-sonnet-5-5` | 171/190 | 119/131 | 52/59 | 30/33 |

The totals are not directly comparable with the earlier baseline, because some cases now expect different skills. Compare per case instead. "Affected cases" are the 33 cases whose baseline expectation, `expected_after` or choices named a retired skill. On Opus, all 33 passed in the baseline.

### What changed per case

These hold for both models unless a model is named.

- **Moved cleanly:** all debugging, security, knowledge-file and documentation-formatting core cases now reach their new owners. So do the merge probes for stale knowledge files, the changelog entry, root-cause debugging and the SSO threat review.
- **Regressed on the affected cases:**
  - "Proofread the CONTRIBUTING guide draft…" answers `none` on both models. It went to documentation-standards before; architecture's description does not yet claim proofreading guides.
  - On Opus, "Run the general quality checklist before we ship" now goes to ship. Sonnet already did that in the baseline.
  - On Sonnet, "Prepare a routine checkpoint so I can resume this work next session" now goes to handoff. The start of `loaf-core`'s description mentions handoff but no longer mentions finishing work or compacting context.
- **Pulled into `loaf-core`:** "Which loaf command shows the recent project journal?" went to loaf-reference before and now goes to `loaf-core`. Two other Loaf CLI cases that already failed (journal flags and repairing stale targets) now fail into `loaf-core` instead of `foundations` or `debugging`.
- **Unrelated movement:** discovery probes owned by the brainstorm and explore merge moved between pitch, explore and architecture on both models. One run per model cannot separate that from noise.

### Rerun after tuning two descriptions

The first post-merge run above came in below the baseline, so two descriptions were tuned at commit `84b4b28e`, and the full suite ran again once per model. Prompts and cases are unchanged.

- The start of `loaf-core`'s description names quality checks and saving reasoning before finishing, handoff or compaction, and says "continuity checkpoints" instead of "journaling".
- The start of architecture's description names proofreading guides such as README or CONTRIBUTING.

| Model | Overall | Core | Conflict probes | Affected cases |
|---|---|---|---|---|
| `claude-opus-5-5` | 174/190 | 121/131 | 53/59 | 33/33 |
| `claude-sonnet-5-5` | 175/190 | 122/131 | 53/59 | 33/33 |

- **Every affected case passes on both models,** matching Opus's 33/33 in the baseline. The CONTRIBUTING proofreading, quality checklist and routine checkpoint cases now reach their owners. Sonnet's quality checklist case, which went to ship in the baseline, now passes too.
- **No pull into `loaf-core`:** "Which loaf command shows the recent project journal?" is back with loaf-reference. "Diagnose why the installed Loaf targets are stale and repair them" still fails, now into `loaf-core`; in the baseline it failed into debugging.
- **Changes outside the affected cases,** from one run each:
  - On Opus, "Explain which component owns shared work and why; do not edit" went to loaf-reference instead of architecture. It passed in the baseline and the first post-merge run, so the shorter architecture description in `84b4b28e` may have caused it; one run cannot tell that apart from variation.
  - On Sonnet, "Ready to merge this PR" went to git-workflow instead of ship. Neither skill changed in this merge.
