# Documentation Review Checklist

## Contents
- Quick Documentation Check
- Architecture Presentation Review
- API Documentation Review
- Changelog Review
- Quality Indicators

Quality checklist for reviewing project documentation.

## Quick Documentation Check

For every documentation change:

```
[ ] Spelling and grammar correct
[ ] Code examples compile/run
[ ] Links work
[ ] No outdated information
```

## Architecture Presentation Review

Use the architecture skill for substantive model, rationale, applicability, drift, and migration review. For presentation:

```
[ ] Descriptive topic title and clear navigation
[ ] Related subjects linked instead of duplicated
[ ] Code and technical-reference links resolve
[ ] Implemented behaviour and agreed pending direction are visibly distinguished
[ ] No forced ADR metadata or headings unless explicitly required
```

For deliberately selected ADRs, check the repository's actual template through Architecture's decision-record guidance; use its legacy guidance only for that historical convention. Do not invent alternatives to satisfy a presentation checklist.

## API Documentation Review

Each endpoint must include:

```
[ ] HTTP method and path
[ ] Description
[ ] Authentication requirements
[ ] Request body schema with types
[ ] Response: success and error schemas
[ ] Request/response examples
```

**Key rule:** Documentation reflects only implemented and released features. No future/planned endpoints.

## Changelog Review

```
[ ] Follows Loaf's Common Changelog profile
[ ] Grouped by: Changed, Added, Removed, Fixed
[ ] Most recent version first
[ ] Entries use imperative, self-describing release-facing prose
[ ] Entries link to the best public reference when available
[ ] Dates in ISO format (YYYY-MM-DD)
[ ] No internal spec/task IDs or verbatim commit dumps
```

## Quality Indicators

### Good Documentation

- Up-to-date with code
- Tested code examples
- Consistent formatting
- Clear, concise language

### Signs of Poor Documentation

- Broken links
- Untested code samples
- Outdated information
- Missing error scenarios
