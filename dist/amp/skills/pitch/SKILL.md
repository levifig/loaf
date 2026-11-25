---
name: pitch
description: >-
  Conducts problem discovery and option exploration before work is shaped, and
  starts or resumes a checkpointed Exploration. Use when direction is undecided
  or solution-led, or the user asks to explore options or "help me think
  through" a choice. Produces a problem narrative with options weighed, not an
  implementation. Not for evidence gathering on a known question (use research),
  quick capture (use idea), context transfer to another agent (use handoff), or
  bounding a chosen direction (use shape).
version: 0.6.0-rc.3
---

# Pitch

Discover the problem with the user, widen the options before judging them, then render the [problem narrative](templates/problem-narrative.md). Pitch may read relevant native tracker context through [`project-management/v1`](../project-management/SKILL.md), but it does not mint a substitute work record.

## Contents

- Critical Rules
- Verification
- Quick Reference
- Topics

## Critical Rules

- Ask one consequential question at a time. Include a recommendation when choices have meaningful tradeoffs.
- Stay in the problem space until affected people, current reality, desired outcome, constraints, and important unknowns are explicit.
- Seek the first rideable journey: identify the smallest useful end-to-end change for a real operator without designing it. Sequence useful outcomes, never storage, backend, API, UI, or verification layers.
- Treat a proposed solution as one option and as evidence about the problem, not as an accepted implementation.
- Before judging, list the options with their tradeoffs, including doing nothing and a redesign of the surrounding approach. Do not commit to one until the alternatives are on the table.
- Record each rejected option and why it fell. The narrative's handoff carries them so shape does not reopen them blindly.
- Capture side ideas as sparks while they arise, using the spark capture command, and keep going; triage decides what happens to them.
- Weigh the options against VISION and STRATEGY when the project has them. If evidence shows a premise there is wrong, say so and hand that evidence to reflect; never edit a Core Doc from pitch.
- Read an existing native record when the user supplies a reference or when duplicate context materially affects the pitch.
- Do not create shared work during pitch unless the human explicitly says to file it. That publish is the triage action: a native intent record in Backlog, not Todo, written as the project-management skill's [Record Writing](../project-management/references/record-writing.md) reference describes rather than as the narrative's headings. Otherwise fill an existing native record the same way, following [Draft Tracker Writes](../project-management/SKILL.md#draft-tracker-writes), or hand the narrative and destination uncertainty to shape. Discovery may keep a solution as a hypothesis; speculative is not authorized.
- When an undecided inquiry will outlast this conversation, its context window, or the current harness, keep it as an Exploration and checkpoint it as the loaf-core skill's continuity reference describes.
- Render the narrative with the template's headings and completed prose. Omit template field-marker comments and authoring instructions from chat responses and saved narratives; the markers belong only in the template source.
- Read [Interview Guide](references/interview-guide.md) when the initial request is solution-heavy or crosses several problem domains.

## Verification

- Every required problem-narrative field is filled with observed or explicitly uncertain information.
- The narrative explains who is affected and what changes if the problem is solved.
- The first useful operator journey is nameable, and any sequencing is expressed as outcomes rather than component layers.
- Proposed implementation details are separated from constraints and evidence.
- The options considered include doing nothing and a redesign, and every rejected option names why it fell.
- Side ideas raised along the way exist as sparks, not only in prose.
- Options were weighed against VISION and STRATEGY where they exist; any premise found wrong was handed to reflect, and no Core Doc was edited.
- No shared work was filed unless the human asked for it.
- Existing tracker context is attributed to its native reference and has not been copied into a second work authority.
- The handoff states whether shape should create a native record, update an existing one, or leave an unshaped Backlog intent in place.

## Quick Reference

| Signal | Next question |
|--------|---------------|
| Solution first | What failure or unmet need makes this solution attractive? |
| Only one option named | What else would meet the need, including doing nothing or redesigning around it? |
| Broad audience | Who feels the problem most directly? |
| Vague success | What observable reality would be different? |
| Conflicting goals | Which outcome wins, and why? |
| Existing native reference | What does the current record already establish? |
| Component-layer sequence | What is the first complete journey a real operator could use? |

| Operation | Command |
|-----------|---------|
| Capture a side idea | `loaf spark capture --scope <scope> --text <text>` |
| Resume an Exploration | `loaf exploration context <ref> --json` |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Interviewing | [interview-guide.md](references/interview-guide.md) | Narrowing a solution-heavy, ambiguous, or multi-party direction |
