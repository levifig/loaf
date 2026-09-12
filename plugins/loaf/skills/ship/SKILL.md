---
name: ship
description: >-
  Serves as the sole quality gate for implemented native tracker work. Use when
  a candidate diff needs independent review, criterion verification, and a
  landing decision. Produces an evidence-backed quality verdict and verified
  native transition when authorized.
user-invocable: true
argument-hint: '[PR number or URL]'
version: 0.5.0
---

# Ship

Evaluate the candidate against the live [work contract](templates/work-contract.md), repository state, and observed verification evidence. Passing tests are necessary but do not replace criterion-by-criterion review.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules

- As the first action, run `loaf journal log "skill(ship): <concise intent>"` against the current private local journal. If the write fails, report the failure and continue only when the work can safely proceed; never put invocation bookkeeping in the tracker.
- Re-read the canonical record, relationships, workflow state, and relevant collaboration through [`project-management/v1`](../project-management/SKILL.md).
- Inspect the actual candidate diff and repository instructions. Do not accept an implementer's summary as evidence.
- Review correctness, maintainability, security boundaries, scope, and every completion criterion with fresh eyes; use independent read-only reviewers when available and proportionate.
- Map each criterion and relevant scope boundary to the contract's specified proof. Reuse existing valid evidence; do not create a duplicate log. A named human verdict is usable only where the contract explicitly defines judgmental acceptance, and it must match the designated authority, method and considerations, and the reviewed candidate. Human signoff cannot replace specified objective checks or safety proof. Required unavailable proof or judgment remains unproven. Scope creep is not a bonus that can be silently shipped.
- Record a durable disposition discoverable from the issue, directly or through a stable existing linked review or PR: candidate identity, criterion-to-evidence outcomes, necessary judgment and acceptance authority, relevant scope changes, and limitations. Solo and team use the same contract; reuse native comments or review surfaces rather than a second ledger.
- Review the rideable increment inside this same quality gate: can the named rider use the result today, does the journey exercise the machinery added, what real dogfood and integrity evidence support the claims, and could less preserve the same safe outcome?
- Do not accept material contract drift by recording it in the verdict. If the candidate changes promised outcome, criteria, scope or safety, dependencies, or the basis of selected or resourced work, pause acceptance and return it to shape. Material changes need applicable authorization (existing authorization counts; assignment, write access, or reviewer status alone is never product authority), an updated canonical definition and native fields with authoritative readback, and a retained reason, authority, and impact; then review the candidate against that resulting contract. Keep same-turn responsibilities; do not add extra ceremony or repeated approval when authorization already exists.
- Return a clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence to implement and rerun affected gates after fixes. Unclear or changed contract substance, or a material scope or commitment shift, returns to shape; do not bounce ordinary fixes to shaping or absorb unauthorized scope in implement. Inaccessible required proof or absent authority is blocked; do not waive the criterion. Material means uncertainty or change in what was promised or authorized, not the severity of an in-scope coding bug.
- Build review or change-request text from the live native contract and observed diff, not from a synchronized local projection.
- Preserve external-action authority. Acceptance is separate from landing authority. A quality verdict does not itself authorize commit, push, merge, publication, or destructive action.
- Transition native workflow state only after the corresponding repository event is proven and authorized.
- Re-read the changed native record and any evidence comment; never mark work complete from a mutation request alone.
- Release remains a separate retroactive ceremony over already-landed work.

## Verification

- The reviewed commit or diff identity is exact and the working tree contains no unexplained changes.
- Every completion criterion and relevant scope boundary has the contract's specified proof from the candidate, tests, or native state. A named human verdict counts only where the contract explicitly defines judgmental acceptance and matches the designated authority, method and considerations, and the reviewed candidate; implementer summary or a general human signoff is not acceptance and cannot replace specified objective checks or safety proof.
- The durable disposition is discoverable from the issue and names candidate, criterion-to-evidence outcomes, required judgment or authority, relevant scope changes, and limitations.
- The complete journey works from its named entry point to observable outcome; no future-only machinery is presented as delivered value.
- Dogfood and safety/integrity evidence match the contract, and learning plus remaining deferrals are reported honestly.
- Required focused, affected, full-suite, format, lint, static-analysis, and build gates were run in proportion to risk.
- All findings have an evidence-backed disposition and fixes were re-reviewed.
- Load the project-management skill and apply its Capture Deferred Work rule to deferred actionable findings; a follow-up issue never excuses an unmet completion criterion for this candidate.
- Affected architectural claims and implementation gaps were checked against the diff and evidence using the architecture skill's maintenance rules; a documentation update alone is not proof of delivery.
- Any native transition matches the observed landed state and was confirmed by readback.

## Quick Reference

| Verdict | Meaning |
|---------|---------|
| Approve | Criteria, boundaries, and quality gates are proven; authorized landing may proceed separately |
| Request changes | A clear unmet criterion, ordinary defect, or obtainable missing in-scope evidence returns to implement; unclear or changed contract substance, or a material scope or commitment shift, returns to shape. Do not absorb unauthorized scope by recording it. |
| Blocked | Required proof is inaccessible, or required authority, connection, or repository event is unavailable; do not waive the criterion |
| Landed | Repository event and final native state were both observed |

## Topics

No supporting references. Use the [tracker update](templates/tracker-update.md) for a concise evidence-bearing verdict when collaborators need it.
