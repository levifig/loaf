# Knowledge File Template

**Location:** `docs/knowledge/{slug}.md`

```yaml
---
topics: []                          # Required (min 1) -- semantic tags for routing
last_reviewed: YYYY-MM-DD           # Required -- ISO 8601 date of last human review
covers:                             # Recommended -- glob patterns from git root
  - "path/to/covered/**/*.ts"
consumers: []                       # Optional -- agent routing hints
depends_on: []                      # Optional -- other knowledge file names
implementation_status: in-progress  # Optional -- in-progress | stable | deprecated
---

# [Title]

Brief description of what this knowledge covers and why it exists.

## Key Rules

- [Rule 1]
- [Rule 2]

## [Domain Section]

[Domain-specific content organized by topic]

## Cross-References

- [Related knowledge file or ADR]
```
