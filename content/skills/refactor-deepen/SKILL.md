---
name: refactor-deepen
description: >-
  Surfaces refactoring opportunities through a deepening lens — modules that
  hide complexity behind narrow interfaces. Not for renames, extractions, or
  generic restructuring (use implement). Use when looking for structural
  improvements, or when the user asks "is this module too shallow?" or "where
  should we deepen this code?" Produces analysis with candidates, dependency categories, and proposed
  deepened modules; actionable work belongs in the native tracker.
---

# Refactor-Deepen

Identify shallow modules and propose deepenings that hide complexity behind
narrow, stable interfaces. Vocabulary discipline is load-bearing: drift to
"boundary," "service," "component," or "layer" defeats the entire point.

## Contents
- Critical Rules
- Verification
- Quick Reference
- Process
- Journal Commands
- Topics
- Related Skills

**Input:** $ARGUMENTS (module path, area, or "scan the repo")

---

## Critical Rules

Before the first journal write, select the command in [Journal Commands](#journal-commands). Use that form for every entry in this skill and its references.

### First Action: Self-Log

Before anything else, log invocation to the project journal:

```bash
loaf journal log "skill(refactor-deepen): <module or area>"
```

### Vocabulary Discipline

The source taxonomy is **eight terms**. Use them verbatim. Do not substitute
"boundary," "service," "component," or "layer" — those words are imprecise
and erode the lens.

| Term | One-line cue |
|------|--------------|
| **Module** | A unit of code with a name, a public surface, and hidden internals |
| **Interface** | The narrow public surface a module exposes |
| **Implementation** | What lives behind the interface |
| **Depth** | Ratio of hidden complexity to interface surface (more hidden = deeper) |
| **Seam** | A point where the call graph can be cleanly cut and substituted |
| **Adapter** | The translation layer between an internal interface and an external concern |
| **Leverage** | How much downstream code is simplified per unit of interface |
| **Locality** | Whether related concerns live near each other or are scattered |

Full semantics live in [references/language.md](references/language.md). Read
that file on every invocation before naming anything.

### Glossary Integration

The skill *pressure-tests* glossary terms — it consumes existing entries and
adds new ones when a deepening clearly names a structural module.

- **Before naming a candidate module**, check the glossary:
  ```bash
  loaf kb glossary check <term>
  ```
  - Canonical hit → use the existing term verbatim.
  - Avoided-alias hit → use the canonical replacement, surface the swap to the user.
  - Unknown → proceed; if the term proves load-bearing, add it (next rule).
- **When a deepening clearly names a structural module**, add it canonically:
  ```bash
  loaf kb glossary upsert "<term>" \
    --definition "<one-line definition>" \
    --avoid "<alias-1>,<alias-2>"
  ```
  Use `upsert` (high commitment), not `propose`. A named deepening is not exploratory.
- If a glossary write fails, surface the error verbatim and continue without
  synthesizing partial state.

### Grilling Protocol

The interview imports the shared
[templates/grilling.md](templates/grilling.md) template — relentless
interview, walk the decision tree, recommend per question, prefer exploration
when the codebase can answer. Do not re-derive the protocol; follow it.

### Interface Design: 3 Unprimed Sub-Agents

When the grilling loop reaches interface design for a candidate, spawn
**exactly 3 sub-agents with identical briefs**. Do not prime them with
opposing constraints (no "Agent 1 minimal, Agent 2 flexible"). Variety must
emerge from sampling, not manufactured opposition. If the three designs
converge accidentally, the fallback is a 4th agent or a rerun with different
seeds — *not* introducing priming. See
[references/interface-design.md](references/interface-design.md).

### Analysis Output

Present the investigation in the current conversation by default. Use [templates/plan.md](templates/plan.md) as the proposal structure: scope and evidence, candidate module, dependency category, proposed deepened module, tests that should survive, and rejected alternatives. This is analysis, not a second work contract.

For a broad scan or advisory report, check candidate names with `loaf kb glossary check`, but do not invoke `loaf kb glossary upsert` or offer plugin review. A report does not commit the project to a structural module.

When the user requests a saved proposal or persistence is useful for an authorized review, write an authored temporary report under `.agents/reports/<YYYYMMDDHHMMSS>-<descriptive-slug>.md`. Read the current UTC time, create the directory only when writing, and avoid overwriting an existing report. Follow Housekeeping for later maintenance; saving analysis does not authorize deletion or archival.

Concrete actionable findings go through Project Management's capture rule into the configured native tracker. Use Shape to establish a work contract and Implement for selected work. Do not create new `.agents/plans/` files or `loaf plan` records for this workflow. Preserve existing proposal files and compatibility records; this guidance does not migrate or delete them. Durable architectural conclusions belong in the repository's maintained architecture topics when deliberately adopted.

### Whole-Repo Inventory

For broad scans, inventory source and project-owned docs first. Exclude
generated, vendored, cached, virtualenv, and build output trees unless the user
explicitly asks to inspect them.

Default excludes:

- `node_modules/`, `.next/`, `.venv/`, `venv/`, `__pycache__/`
- `dist/`, `dist-cli/`, `plugins/`, `build/`, `coverage/`, `.turbo/`
- generated lock/vendor output and other tool caches

Use `rg --files` with `-g` excludes or targeted `find` commands rather than a
blind recursive listing. If an excluded tree appears relevant, name it as an
assumption and ask before pulling it into context.

### Termination

Return the proposal, the location of any explicitly saved report, and the native references of any captured follow-ups. State which decisions remain open. Analysis completion does not select or authorize implementation; honor an existing explicit implementation request through the normal native work prerequisite.

### Codex Review (Opt-In, Plugin-Gated)

After a focused proposal is saved for review, check whether the `codex` plugin is
available in the current harness — i.e. whether the `codex:codex-rescue`
agent (or the `codex:rescue` skill) is exposed in this session's available
skills/agents list. Loaf's install-time tool detection discovers external CLIs
like the Codex binary; it does **not** report Claude-Code-plugin presence
inside the running session. The skill-level ambient-availability check is the
right contract here — what we care about is "can I route a review *right now*,"
not "is the Codex CLI installed somewhere on this machine." If a future Loaf
surface adds plugin-level detection, this rule can switch to it.

- **Always** make this offer plugin-gated and opt-in. Ask once, verbatim:
  > Want a Codex review of this deepening before commit?
  Wait for an affirmative reply before invoking anything.
- **Always** route an accepted review through the `codex:codex-rescue`
  agent (or the `codex:rescue` skill if that's what the harness exposes),
  passing the proposal report path and a pointer to
  [references/deepening.md](references/deepening.md) for vocabulary context.
- **Never** mention Codex if the plugin is absent — proceed straight to the
  proposal result.
- **Never** invoke Codex review by default, on a non-affirmative reply, or
  on a "skip" — return the proposal result.

---

## Verification

- The project journal contains a `skill(refactor-deepen):` entry as the first action
- The glossary, architecture overview, relevant owning topics, and applicable legacy decisions were read before the candidate list was presented, using the repository's actual paths
- Every candidate module name was checked via `loaf kb glossary check` before
  being proposed
- Output uses the eight source terms verbatim — zero occurrences of
  "boundary," "service," "component," or "layer" in their place
- Interface Design spawned exactly 3 sub-agents with identical briefs
- Whole-repo scans excluded generated/vendor/cache/build trees unless the user
  explicitly asked to inspect them
- Broad or advisory analysis did not mutate the glossary or create a new local work record
- Any saved proposal is an authored temporary report, with no invented lifecycle or work identity
- Concrete actionable follow-ups were captured through Project Management; existing files and compatibility records were preserved
- Codex review is offered only for a saved focused proposal when the plugin is available, and runs only on an affirmative reply
- The result identifies the proposal, evidence, native follow-ups, and remaining decisions

---

## Quick Reference

### Glossary CLI Cheat Sheet

| Verb | When |
|------|------|
| `loaf kb glossary check <term>` | Before naming any candidate module |
| `loaf kb glossary upsert <term> --definition <d> --avoid <list>` | When a deepening names a structural module |
| `loaf kb glossary list` | At interview start (see grilling protocol) |

`propose` and `stabilize` are not used by this skill — they belong to
architecture and exploratory flows.

### When to Deepen vs. Leave Alone

| Signal | Deepen? |
|--------|---------|
| Interface surface ≈ implementation surface | Yes — module is shallow |
| Callers reach across the interface to internals | Yes — interface is leaky |
| One module concern is scattered across many files | Yes — locality is poor |
| Interface is narrow and implementation is large | No — already deep |
| Module is a thin pass-through to a true-external dependency | No — adapter, by design |

---

## Process

1. **Read context.** Read the glossary (via `loaf kb glossary list`), architecture overview, relevant owning topics, and applicable legacy decisions using the repository's actual paths. Follow the architecture skill for current authority and documentation conventions. Read
   [references/language.md](references/language.md) and
   [references/deepening.md](references/deepening.md).
2. **Survey candidates.** Walk the target module/area. Produce a numbered list
   of shallow-module candidates with one-line rationales using the eight terms.
3. **Pick a candidate** (with the user) and enter the grilling loop following
   [templates/grilling.md](templates/grilling.md). Surface canonical terms
   inline; challenge drift on contact.
4. **Classify dependencies.** For each candidate, classify dependencies by
   category (in-process, local-substitutable, ports-and-adapters, true-external)
   per [references/deepening.md](references/deepening.md).
5. **Design the interface** by spawning 3 unprimed sub-agents per the rules in
   [references/interface-design.md](references/interface-design.md). Present
   all three designs to the user; do not pre-rank.
6. **Present the analysis.** Use [templates/plan.md](templates/plan.md). Broad scans remain advisory and skip glossary mutation and plugin review.
7. **Save only when useful or requested.** Write a temporary report following Analysis Output; preserve existing records.
8. **Mutate the glossary** (`loaf kb glossary upsert`) only when a focused deepening clearly names a structural module canonically.
9. **Offer plugin review** only for a saved focused proposal and under the opt-in rules above.
10. **Capture and return.** Route concrete actionable findings through Project Management, then return the proposal and remaining decisions. Shape and Implement consume native work, not the report as a substitute contract.

---

## Journal Commands

Use the ordinary command `loaf journal log "type(scope): description"` with your harness's permissions. Apply only the labeled section for the harness you are running.

### Codex

With Loaf's managed basic-command policy installed, use `loaf journal log --execpolicy-safe "type(scope): description"` for every entry, including invocation logs. Keep `--execpolicy-safe` immediately after `journal log` and invoke PATH `loaf` as a standalone command, without an absolute executable pin or shell/environment wrapper. The flag does not install the policy or grant permission by itself; without the policy, use the ordinary command and Codex's approval flow.

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Vocabulary | [references/language.md](references/language.md) | Naming any module, interface, or seam — read first, every invocation |
| Deepening Patterns | [references/deepening.md](references/deepening.md) | Classifying dependencies and applying seam discipline |
| Interface Design | [references/interface-design.md](references/interface-design.md) | Running the parallel 3-agent Interface Design |
| Grilling Protocol | [templates/grilling.md](templates/grilling.md) | Interview discipline during the candidate-selection loop |
| Proposal Template | [templates/plan.md](templates/plan.md) | Structuring analysis or a saved temporary report |

---

## Related Skills

- **architecture** — Explains, audits, and records durable architecture decisions; it no longer consumes refactor-deepen's relentless interview template
- **knowledge-base** — Owns the underlying KB and glossary CLI verbs
- **implement** — Where generic refactors (rename, extract, restructure) belong
