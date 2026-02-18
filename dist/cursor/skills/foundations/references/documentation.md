# Documentation Standards

## Contents
- Core Principle
- Document Hierarchy
- Architecture Decision Records
- API Documentation
- Micro-Changelog
- Critical Rules

Project documentation conventions, ADR format, and API doc rules.

## Core Principle

**Document after shipping, not before.** API documentation reflects ONLY what is implemented and released. Future features belong in Linear issues, not docs.

## Document Hierarchy

```
docs/
├── PRD.md                # What the product should be (vision)
├── ARCHITECTURE.md       # How the product is built (design)
├── IMPLEMENTATION.md     # What the product currently is (status)
├── QUICK_REFERENCE.md    # One-page command reference
├── api/                  # API docs (implemented features only)
│   ├── openapi.yaml
│   └── endpoints/
└── decisions/            # Architecture Decision Records
    ├── ADR000-template.md
    └── ADR001-*.md
```

| Document | Purpose | Updates When |
|----------|---------|--------------|
| **PRD.md** | Product vision (timeless) | Vision changes |
| **ARCHITECTURE.md** | Technical design | Architecture changes |
| **IMPLEMENTATION.md** | Current status | Features ship |
| **API docs** | Implemented endpoints | Features ship |

## Architecture Decision Records

**When to write:** Technology choices, architectural patterns, integration approaches, security decisions. **Skip for:** library version updates, bug fixes, performance tweaks, style changes.

**File naming:** `docs/decisions/ADRXXX-short-descriptive-title.md`

### ADR Template

```markdown
# ADR-XXX: Title

**Decision Date**: YYYY-MM-DD

**Status**: Proposed | Accepted | Deprecated | Superseded

## Context
[Situation, problem, constraints]

## Decision
[Use "We will..." not "It was decided..."]

## Consequences
### Positive
- [Benefit 1]

### Negative
- [Drawback 1]

## Alternatives Considered
### Alternative 1: [Name]
[Brief description and why rejected]
```

**Status lifecycle:** `Proposed -> Accepted -> (Deprecated | Superseded)`

## API Documentation

**The implemented-only rule:** `Feature Request -> Implementation -> Tests Pass -> Release -> Update API Docs`

Deprecation markers in OpenAPI:
```yaml
paths:
  /api/v1/legacy-endpoint:
    get:
      deprecated: true
      description: |
        **Deprecated since**: v1.5.0
        **Removal planned**: v2.0.0
```

## Micro-Changelog

Track changes at the **bottom** of individual documents:

```markdown
---

## Changelog

- 2025-11-14 - Added section on agent instructions
- 2025-11-12 - Updated architecture overview
```

**Format:** `- YYYY-MM-DD - Short description`, reverse chronological. Log section additions, significant updates, restructuring, corrections. Skip typos and formatting.

## Critical Rules

**Always:** Last Updated timestamp (YYYY-MM-DD), micro-changelog at bottom, reference files not inline code, keep docs minimal.

**Never:** Document APIs before they ship, include `.agents/` links outside `.agents/` artifacts, use lengthy code samples, add planning details to docs.
