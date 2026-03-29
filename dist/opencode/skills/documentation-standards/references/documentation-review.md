# Documentation Review Checklist

## Contents
- Quick Documentation Check
- ADR Review
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

## ADR Review

Every ADR must have:

```
[ ] Descriptive title
[ ] Status: Proposed/Accepted/Deprecated/Superseded
[ ] Context: Why this decision was needed
[ ] Decision: What was decided (uses "We will...")
[ ] Consequences: Trade-offs and impacts
[ ] Alternatives: At least one alternative considered
```

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
[ ] Grouped by: Added, Changed, Deprecated, Removed, Fixed, Security
[ ] Most recent version first
[ ] Links to version comparisons
[ ] Dates in ISO format (YYYY-MM-DD)
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
