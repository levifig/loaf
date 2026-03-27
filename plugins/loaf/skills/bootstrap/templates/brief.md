# Brief Template

**Location:** `docs/BRIEF.md`

```yaml
---
source: file | text | folder | interview
original_path: ""       # Path to external source file, if copied into project
created: YYYY-MM-DDTHH:MM:SSZ
---

# Project Brief

## Problem Statement

[What problem does this project solve? Why does it exist? Be specific about
the pain -- vague problems produce vague solutions.]

## Target Users

[Who has this problem? Describe them concretely -- role, context, frequency
of the pain. Avoid "developers" or "users" without qualification.]

## Current Alternatives

[What do target users do today? Existing tools, manual workarounds, or
"nothing" are all valid answers. Understanding the status quo clarifies
what "better" means.]

## Vision

[What does success look like? Describe the end state, not the features.
A user who has this tool -- what changes for them?]

## Key Constraints

[Non-negotiable requirements: technical, legal, organizational, or
philosophical. Things that bound the solution space before design begins.]

- [Constraint 1]
- [Constraint 2]

## Open Questions

[Unresolved items that need answers before or during implementation.
Mark each with its urgency: blocking (must resolve before work starts)
or deferrable (can resolve during implementation).]

- [ ] [Question 1] -- blocking | deferrable
- [ ] [Question 2] -- blocking | deferrable
```

## Frontmatter Fields

| Field | Required | Values | Notes |
|-------|----------|--------|-------|
| `source` | Yes | `file`, `text`, `folder`, `interview` | How the brief entered the project |
| `original_path` | No | File path string | Only when copied from an external location |
| `created` | Yes | ISO 8601 timestamp | When the brief was persisted |

## Source Behaviors

- **`file`** -- Brief copied from an external file path. `original_path` records where it came from.
- **`text`** -- Brief provided as inline text in the `/loaf:bootstrap` invocation.
- **`folder`** -- Brief synthesized from multiple markdown files in a folder.
- **`interview`** -- Brief synthesized from a `/loaf:bootstrap` interview session (greenfield+empty mode).

When the brief already exists at `docs/BRIEF.md` within the project, use it in place -- do not copy or overwrite. Add frontmatter if missing.
