# Reviewable Criteria

Shape writes a definition a later implementer and reviewer can use without reconstructing the conversation. Proportional detail beats complete paperwork.

## Sufficiency

Stop when the bounded outcome can be delivered and independently evaluated without inventing intent or making unauthorized consequential decisions. Missing native registration or selection is a tracker-establishment gap, not a reason to keep asking about already-known substance. Reuse established answers. Ask only when the unanswered choice would change outcome, criteria, safety, dependencies, or the basis on which work was selected or resourced.

## What a Criterion Names

Each completion criterion identifies relevant preconditions or scenario, an observable outcome, meaningful negative or preservation boundaries, and an obtainable evidence or acceptance basis. Conditions may be shared. Do not require Gherkin, one test per checkbox, or recipes that name files and functions. Prefer repeatable checks. Inherently judgmental outcomes name a review method, the considerations that matter, and who may accept them; do not invent metrics. Required proof or judgment that is unavailable remains unproven, not waived. Independent evidence means a reviewer can inspect or reproduce it; a second agent is not required.

## Examples

| Weak | Reviewable |
|------|------------|
| Keep interviewing after the pitch already names the rider, journey, and proof | Synthesize that context, write the five fields, read back, and stop |
| "Auth works" / "looks good" / "90% confidence" | Name the scenario, observable result, what must not break, and either a rerunnable check or a named reviewer plus acceptance considerations |
| A named human verdict used as proof for every criterion | Use the contract's specified proof; a human verdict only where judgmental acceptance is defined, matching authority, method, and the reviewed candidate |
| One new test per bullet, or "edit `foo.go` and add `Bar()`" | Shared conditions and implementer discretion; evidence a reviewer can inspect without an internal recipe |
| "Also migrate the other harness if time" | Put likely expansion in Out of Scope, or split only as an independently useful journey |
| Unresolved ownership on a multi-harness change left implicit | Ask that material choice; do not invent an owner, estimate, or team mode |
| Unchanged criteria while the work quietly doubled, a proposed weaker check to fit the diff, or accepting drift by recording it in the verdict | Pause acceptance and return to shaping with authorization and canonical readback; assignment, write access, or reviewer status is not product authority; do not silently lower the goal |
| Required subjective acceptance is unavailable | Leave the criterion unproven; do not waive it as a documentation detail |

## Exclusions and Change

Out of Scope prevents likely expansion. Routine in-scope implementation details remain implementer decisions. A material change to promised outcome, criteria, safety, dependencies, or the basis of selected or resourced work returns to shaping, not automatically to another pitch. Unchanged criteria with unexpectedly expanded work can still change the commitment.

Material changes need applicable authorization (existing authorization counts; assignment, write access, or reviewer role alone is not approval), an updated canonical definition with readback, and a retained reason: who or what authorized it and the material impact, in native history or a concise supported update. Do not snapshot a second contract or silently lower criteria to fit a result. Pause affected work when required approval is missing; unrelated authorized work may continue.

Solo and team use this same five-field contract. Use native owners, priorities, dependencies, estimates, or cycles only when they are relevant, configured, supported, and authorized. Capacity-impact recognition is qualitative unless the tracker already records it. Do not invent estimates, velocity, extra fields, states, team mode, or a shadow ledger.
