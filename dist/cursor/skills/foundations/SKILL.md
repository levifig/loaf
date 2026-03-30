---
name: foundations
description: >-
  Establishes code style, naming conventions, TDD discipline, verification
  checklists, and code review standards. Use when writing or reviewing code
  quality, running verification checks, or setting up review processes. Not for
  git workflow (use git-workflow), debugging (use debugging), security (use
  security-compliance), or documentation (use documentation-standards).
version: 2.0.0-dev.5
---

# Code Standards

Engineering foundations for consistent, high-quality code.

## Contents
- Topics
- Available Scripts
- Critical Rules
- Naming Conventions
- Test Patterns

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Code Style | [references/code-style.md](references/code-style.md) | Writing Python/TypeScript code, naming variables |
| TDD | [references/tdd.md](references/tdd.md) | Writing tests first, red/green/refactor cycle |
| Verification | [references/verification.md](references/verification.md) | Verifying work before claiming done |
| Code Review | [references/code-review.md](references/code-review.md) | Requesting or receiving code reviews |
| Review | [references/review.md](references/review.md) | Conducting structured code reviews |
| Permissions | [references/permissions.md](references/permissions.md) | Configuring tool allowlists, sandbox, agent permissions |
| Observability | [references/observability.md](references/observability.md) | Instrumenting services, logging, metrics, tracing |
| Production Readiness | [references/production-readiness.md](references/production-readiness.md) | Validating services are ready for production |

## Available Scripts

| Script | Usage | Description |
|--------|-------|-------------|
| `scripts/check-python-style.py` | `check-python-style.py <dir>` | Check Python style (type hints, docstrings) |
| `scripts/check-test-naming.sh` | `check-test-naming.sh <dir>` | Check test file/function naming |

## Critical Rules

### Always

- Use type hints on all public functions
- Validate inputs at trust boundaries

### Never

- Use bare `except:` clauses
- Log sensitive data or stack traces to users

## Naming Conventions

| Context | Python | TypeScript |
|---------|--------|------------|
| Files | `snake_case.py` | `PascalCase.tsx` (components) |
| Functions | `snake_case` | `camelCase` |
| Classes | `PascalCase` | `PascalCase` |
| Constants | `UPPER_SNAKE` | `UPPER_SNAKE` |
| Tests | `test_<unit>_<scenario>_<result>` | `describe/it` blocks |

## Test Patterns

Scenario-based fixture naming:

- `*_perfect` - Complete, valid data (happy path)
- `*_degraded` - Partial data, quality issues
- `*_chaos` - Edge cases, malformed data

Coverage target: 70% minimum across all components.
