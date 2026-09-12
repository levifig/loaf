---
name: shape
description: Shapes existing pitch, conversation, or native intent into a bounded, independently reviewable tracker work contract. Use when work needs testable completion criteria, explicit exclusions, hierarchy, or dependencies before implementation. Produces a verified native record ready for implement and ship.
---

# Shape

Turn an existing pitch, conversation, or native issue into a bounded, independently reviewable [work contract](templates/work-contract.md), then route it once into canonical native tracker fields through [`project-management/v1`](../project-management/SKILL.md). The provider owns native identity and state; Loaf owns the questions and packet shape that make the definition actionable.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules

- As the first action, run `loaf journal log "skill(shape): <concise intent>"` against the current private local journal. If the write fails, report the failure and continue only when the work can safely proceed; never put invocation bookkeeping in the tracker.
- Confirm the destination, runtime capabilities, and whether a matching native record already exists before creation.
- If creation returns an ambiguous result, search and re-read native state; never repeat the create blindly.
- Synthesize already-known context from the pitch, conversation, and live native record before asking. Investigate available facts; ask only material unanswered choices whose answers would change outcome, criteria, safety, dependencies, or the basis of selected or resourced work.
- Stop when an implementer can deliver and a reviewer can evaluate without inventing intent or making unauthorized consequential decisions. Missing native registration or selection is distinct from missing substance; do not keep interviewing after the definition is sufficient.
- Preserve the problem narrative's intent while removing solution assumptions not required by constraints.
- Identify architectural implications and unresolved choices using the architecture skill; shaping does not automatically create an architecture document or decision record.
- Make the rideable increment concrete in the existing native definition and criteria: Rider, complete Journey, Entry point, observable Outcome, real Dogfood, Safety/integrity proof, Learning sought, and explicit Deferrals. These are required decisions, not required native headings or a new schema. An existing journey restored by maintenance is valid; one exercise may supply several answers.
- Set the title through the native work field, then fill every definition-packet field. Definition of done uses independently reviewable criteria; out of scope names what this work deliberately will not solve. Read [Reviewable Criteria](references/reviewable-criteria.md) when writing or changing criteria, exclusions, or material scope.
- Keep hierarchy and dependencies in their native relationship fields, not in comments or body prose alone. Use existing native owners, priorities, estimates, or cycles only when they are relevant, configured, supported, and authorized; never invent them.
- Decompose only when a child earns an independently useful, independently verifiable definition of done. Read [Decomposition](references/decomposition.md) before changing hierarchy.
- Read back the native record, definition, hierarchy, dependencies, and current status before declaring it ready. Writing a sufficient definition does not select the work. Move to Todo in the same turn when the human selects the work, including an existing explicit request to implement that concrete change; do not ask for the same selection again. Otherwise leave Backlog intent in place.
- The main agent executes common operations through the selected provider skill; delegation never changes semantics or authority.

## Verification

- The tracker-issued native reference is retained and the final read matches the intended native title and every routed definition field.
- Known context was reused; remaining questions, if any, were material rather than ceremonial.
- Each completion criterion names observable evidence and can be evaluated without reconstructing the pitch conversation.
- The named rider can complete the journey through its real entry point; dogfood, safety/integrity proof, learning sought, and deferrals are concrete without adding another work record or gate.
- Any foundation work is exercised by this journey or remains explicitly deferred; no child is only a future wheel.
- Out-of-scope boundaries prevent likely expansion rather than restating the goal.
- Parent/child and dependency relationships are confirmed through their native fields.
- Any unsupported provider feature is reported with honest fidelity; no prose substitute is presented as exact.

## Quick Reference

| Need | Common operation |
|------|------------------|
| Reuse known context | Read the pitch, conversation, and live native record before asking |
| Ask | Only a material unanswered choice |
| Find existing work | `work.read` |
| Mint native identity | `work.create` |
| Change native title | `work.update` |
| Write canonical body | `definition.write` |
| Set parent/child | `hierarchy.change` |
| Set blocking edge | `dependency.change` |
| Prove readiness | Read back all relevant native fields |
| Select for execution | `status.transition` to Todo only when the human selects the work; sufficiency is not selection |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Reviewable criteria | [reviewable-criteria.md](references/reviewable-criteria.md) | Writing independently evaluable criteria, exclusions, or material-change rules |
| Decomposition | [decomposition.md](references/decomposition.md) | Deciding whether one criterion deserves its own native child record |
