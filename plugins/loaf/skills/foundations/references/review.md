# Code Review Checklist

## Contents
- Review Tiers
- Common Issues to Flag
- Review Response Templates

Tiered review checklist for code quality and correctness.

## Review Tiers

### Quick Review (< 50 lines)

```
[ ] Code compiles and tests pass
[ ] Naming is clear and descriptive
[ ] No obvious bugs or logic errors
[ ] No hardcoded secrets or credentials
```

### Standard Review

```
[ ] Functions have single responsibility
[ ] No deeply nested logic (max 3 levels)
[ ] No magic numbers (use named constants)
[ ] No commented-out code
[ ] All function parameters have types
[ ] Return types are explicit
[ ] Errors don't fail silently
[ ] Error messages include context
```

### Deep Review (critical paths, security-sensitive, architectural)

```
[ ] Follows existing codebase patterns
[ ] Abstractions at right level
[ ] Dependencies flow in correct direction
[ ] No N+1 queries
[ ] Input validated at boundaries
[ ] SQL uses parameterized queries
[ ] Authentication/authorization checked
[ ] Secrets not logged or exposed
[ ] New code has meaningful tests
[ ] Edge cases and error paths tested
[ ] Public APIs documented
```

## Common Issues to Flag

**Always flag:** Hardcoded secrets, SQL injection, missing input validation, unbounded loops/recursion, memory leaks, race conditions.

**Usually flag:** Missing error handling, no tests for new code, magic numbers, overly complex functions, copy-pasted code.

## Review Response Templates

### Approve

```
LGTM!

Minor suggestions (optional):
- [suggestion]
```

### Request Changes

```
Good progress. A few items to address:

**Must fix:**
- [blocking issue]

**Should fix:**
- [important but not blocking]

**Consider:**
- [optional improvement]
```
