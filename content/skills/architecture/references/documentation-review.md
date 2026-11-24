# Documentation Review Checklist

## Contents
- Quick Documentation Check
- Architecture Presentation Review
- API Documentation Review
- Diagrams
- Quality Indicators

Quality checklist for reviewing project documentation: guides, READMEs, contributing docs, API docs, and architecture topics. Changelog entries follow the release skill's Changelog rules.

## Quick Documentation Check

For every documentation change:

```
[ ] Spelling and grammar correct
[ ] Code examples compile/run
[ ] Links work
[ ] No outdated information
```

## Architecture Presentation Review

Substantive model, rationale, applicability, drift, and migration review follow the architecture skill's Critical Rules. For presentation:

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

## Diagrams

Use Mermaid for diagrams in project documentation, and only where one clarifies a multi-component flow, data flow, schema, or state machine.

```
[ ] Node names match real components, services, or tables in the code
[ ] One concern per diagram
[ ] States its purpose and the files it represents
[ ] Updated in the same change as the code it describes
```

## Quality Indicators

### Good Documentation

- Up-to-date with code
- Tested code examples
- Consistent formatting and tone
- Clear, concise language, organized around what the reader needs to do

### Signs of Poor Documentation

- Broken links
- Untested code samples
- Outdated information
- Missing error scenarios
