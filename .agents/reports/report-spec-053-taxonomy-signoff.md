---
id: report-spec-053-taxonomy-signoff
report_kind: gate
state_id: "report:d4e082ada4a9b3428efaa254"
status: final
title: Spec 053 Taxonomy Signoff
---

# SPEC-053 Taxonomy Sign-Off Packet

Date: 2026-06-24
Spec: SPEC-053
Task: TASK-394
Status: pending user sign-off

## Purpose

This packet records the evidence and proposed decisions for the breaking taxonomy entries that SPEC-053 gates. It is intentionally not an implementation of the production entries. No retired skill, retired agent, or opt-in pack removal should be added to config/deprecations.json until the user explicitly signs off.

## Current Evidence

- config/deprecations.json is still production-empty: retired_targets, retired_skills, retired_agents, relocations, and aliases are empty arrays.
- thermo-nuclear-code-quality-review is absent from content/skills and generated dist/plugins outputs, so the remaining action is old-install cleanup through a retired skill manifest entry.
- debugging is present in content/skills/debugging and generated outputs. Its Claude sidecar currently has user-invocable: true and no disable-model-invocation field.
- librarian is present as content/agents/librarian.md with only content/agents/librarian.claude-code.yaml. Generated Cursor and OpenCode agent Markdown exists without target-specific sidecar boundary files.
- The deep evaluation recommends retiring thermo, treating debugging as a visibility/routing-only decision, moving language/domain packs to opt-in install packaging, and either retiring or fully wiring librarian.

## Proposed Decisions For Sign-Off

1. Retire old installed thermo-nuclear-code-quality-review surfaces.
   - Recommended: yes.
   - Implementation after sign-off: add a retired_skills manifest entry for known shared skill homes so install --upgrade removes stale installed copies while reporting the tombstone reason and default one-release window.
   - Rationale: source skill is already absent from the package; users with older installs need explicit cleanup.

2. Debugging routing.
   - Recommended: tighten description only now, validated under SPEC-051 routing eval; do not add disable-model-invocation yet.
   - Alternative: add disable-model-invocation: true if the desired policy is reference-only/manual-only.
   - Not recommended: user-invocable: false as the fix, because it hides menu visibility but does not stop model routing.

3. Language/domain reference packs.
   - Recommended: record the opt-in packaging decision, but defer production removals until install profiles exist.
   - Implementation after sign-off: do not add retired_skills entries for go-development, python-development, typescript-development, ruby-development, or power-systems-modeling in this slice unless profile selection and de-selection semantics are implemented.
   - Rationale: the mechanism can clean removals, but the product still needs a way for users to select packs intentionally.

4. Librarian profile.
   - Recommended: retire librarian unless a concrete orchestration/wrap caller is added in the same branch.
   - Implementation after sign-off if retired: add retired_agents entries for known agent homes, remove source/generated librarian profile files, and report tombstone cleanup on upgrade.
   - Implementation after sign-off if kept: add Cursor/OpenCode sidecars and wire the profile from orchestration or wrap so the .agents-only boundary is preserved and the profile is reachable.

## Manifest Entries To Add Only After Sign-Off

The following are illustrative shapes, not active entries:

```json
{
  "retired_skills": [
    {
      "skill": "thermo-nuclear-code-quality-review",
      "since": "v2.0.0-pre.20260614235428",
      "reason": "retired by SPEC-053 taxonomy decision; not part of the Loaf core skill suite",
      "skill_homes": [
        "${HOME}/.agents/skills",
        "${HOME}/.config/agents/skills",
        "${HOME}/.config/opencode/skills"
      ]
    }
  ],
  "retired_agents": [
    {
      "agent": "librarian",
      "since": "v2.0.0-pre.20260614235428",
      "reason": "retired by SPEC-053 taxonomy decision; native session commands cover this lifecycle",
      "agent_homes": [
        "${HOME}/.cursor/agents",
        "${XDG_CONFIG_HOME}/opencode/agents"
      ]
    }
  ]
}
```

## Required User Sign-Off

Before TASK-394 can be completed, choose:

- thermo cleanup: approve or reject.
- debugging: description-only, disable-model-invocation, or fold into foundations.
- language/domain packs: defer removals until install profiles, or implement opt-in removals now.
- librarian: retire, or wire with Cursor/OpenCode sidecars plus an actual caller.

## Gate Status

SPEC-053 remains open. TASK-394 remains in progress until the signed-off production entries are committed and verified. TASK-395 remains blocked behind that sign-off and final gate review.

<!-- loaf:render kind=report contract=durable-doc-v1 -->
