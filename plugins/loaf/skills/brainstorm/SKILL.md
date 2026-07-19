---
name: brainstorm
description: >-
  Preserves the structured divergent-thinking stance consumed by the explore
  workflow: option generation before judgment, trade-off analysis, and spark
  capture. Explore owns the user-facing inquiry lifecycle; reference this
  technique from it. Not a ...
user-invocable: false
argument-hint: '[idea or problem]'
version: 2.0.0-alpha.9
---

# Brainstorm

Generative thinking — expanding possibilities before narrowing. This stance is an internal technique consumed by `/loaf:explore`, which owns inquiry continuity through Explorations and portable checkpoints; invoke the technique from there rather than as a standalone workflow.

## Critical Rules

**Always**
- Diverge before converging — generate options before judging
- Connect exploration to VISION.md and STRATEGY.md context
- Document discarded options — they hold valuable reasoning
- Log invocation first: `loaf journal log "skill(brainstorm): <topic>"`
- Capture sparks (speculative byproducts) with `loaf spark capture --scope <scope> --text <text>`; brainstorm summaries may render or summarize them
- Set boundaries on exploration time
- Log outcome to the project journal: `loaf journal log "decision(scope): direction chosen and rationale"`

**Never**
- Prematurely commit to an option before full exploration
- Delete brainstorm documents — archive them for context preservation
- Process sparks during the main brainstorm — capture only, expand later
- Turn brainstorm into an interview — keep it exploratory

## Verification

After work completes, verify:
- Brainstorm captured in SQLite or summarized in an explicitly durable report
- Sparks captured with `loaf spark capture` and optionally summarized in `## Sparks`
- Spark lifecycle documented: unprocessed → promoted/discarded
- Brainstorm references strategic context from VISION/STRATEGY

## Quick Reference

### Mode Detection

| Input Pattern | Mode | Output |
|---------------|------|--------|
| Idea file reference | Idea Processing | Deep dive on captured idea |
| Problem/question | Problem Exploration | Exploratory options |
| Empty | Open Brainstorm | "What should we think about?" |

### Spark Format

```markdown
## Sparks

- **Title** -- one-line description
- **Title** -- one-line description
```

Sparks are: lightweight, byproducts, worth remembering. Mark as `*(promoted)*` or `*(abandoned)*` after processing.

Brainstorm documents are archived after sparks are processed — never deleted, since the exploration context has lasting value. SQLite spark state is the lifecycle source; draft markdown is a projection or narrative summary.

## Suggests Next

After a divergent pass, checkpoint the surrounding Exploration (`loaf exploration checkpoint`), then suggest `/loaf:shape` if a clear direction emerged or `/loaf:triage` to disposition captured sparks and ideas.

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Brainstorm Template | [templates/brainstorm.md](templates/brainstorm.md) | Creating structured brainstorm documents |
| Strategic Context | `strategy/references/` | Grounding exploration in project direction |
