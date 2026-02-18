---
name: foundations
description: >-
  Establishes code quality and engineering standards. Covers code style
  conventions, conventional commit messages, documentation standards (ADRs, API
  docs, changelogs), and security patterns (input validation, secrets
  management, OWASP). Use when reviewing code, preparing commits, writing
  documentation, or when the user asks "how should I format this commit?" or
  "what's the security best practice for X?" Produces validated commits,
  formatted documentation, and security-reviewed code. Not for language-specific
  patterns, framework APIs, or deployment (use language skills).
user-invocable: false
allowed-tools: 'Read, Write, Edit, Bash, Glob, Grep'
---

# Code Standards

Engineering foundations for consistent, secure, and well-documented code.

## Contents
- Topics
- Available Scripts
- Critical Rules
- Naming Conventions
- Commit Types
- Test Patterns

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Code Style | `references/code-style.md` | Writing Python/TypeScript code, naming variables |
| Commits | `references/commits.md` | Writing commit messages, creating PRs, branching |
| Diagrams | `references/diagrams.md` | Creating Mermaid diagrams, visualizing architecture |
| Documentation | `references/documentation.md` | Writing ADRs, API docs, changelogs |
| Security | `references/security.md` | Threat modeling, managing secrets, compliance checks |
| Debugging | `references/debugging.md` | Systematically debugging issues with hypotheses |
| Hypothesis Tracking | `references/hypothesis-tracking.md` | Managing multiple hypotheses during investigation |
| Test Debugging | `references/test-debugging.md` | Fixing flaky tests, isolation issues, state pollution |
| TDD | `references/tdd.md` | Writing tests first, red/green/refactor cycle |
| Verification | `references/verification.md` | Verifying work before claiming done |
| Code Review | `references/code-review.md` | Requesting or receiving code reviews |
| Permissions | `references/permissions.md` | Configuring tool allowlists, sandbox, agent permissions |

## Available Scripts

| Script | Usage | Description |
|--------|-------|-------------|
| `scripts/check-commit-msg.sh` | `check-commit-msg.sh <file>` | Validate commit message format |
| `scripts/check-python-style.py` | `check-python-style.py <dir>` | Check Python style (type hints, docstrings) |
| `scripts/check-test-naming.sh` | `check-test-naming.sh <dir>` | Check test file/function naming |
| `scripts/validate-adr.py` | `validate-adr.py <file>` | Validate ADR structure |
| `scripts/check-changelog-format.sh` | `check-changelog-format.sh <file>` | Check micro-changelog format |
| `scripts/check-secrets.sh` | `check-secrets.sh <dir>` | Scan for hardcoded secrets |
| `scripts/validate-compliance.py` | `validate-compliance.py <file>` | Validate security checklist completion |

## Critical Rules

### Always

- Use type hints on all public functions
- Write atomic commits (one logical change)
- Use imperative mood in commit messages
- Validate inputs at trust boundaries
- Log security events (without secrets)
- Include micro-changelog at document bottom

### Never

- Commit secrets, passwords, or API keys
- Document APIs before they ship
- Use bare `except:` clauses
- Force push to main/master
- Log sensitive data or stack traces to users
- Skip commit signing without explicit permission

## Naming Conventions

| Context | Python | TypeScript |
|---------|--------|------------|
| Files | `snake_case.py` | `PascalCase.tsx` (components) |
| Functions | `snake_case` | `camelCase` |
| Classes | `PascalCase` | `PascalCase` |
| Constants | `UPPER_SNAKE` | `UPPER_SNAKE` |
| Tests | `test_<unit>_<scenario>_<result>` | `describe/it` blocks |

## Commit Types

| Type | Use For | Version Impact |
|------|---------|----------------|
| `feat` | New features | Minor bump |
| `fix` | Bug fixes | Patch bump |
| `refactor` | Code restructuring | None |
| `docs` | Documentation only | None |
| `test` | Test additions/updates | None |
| `chore` | Maintenance, deps | None |
| `perf` | Performance improvements | Patch bump |

## Test Patterns

Scenario-based fixture naming:

- `*_perfect` - Complete, valid data (happy path)
- `*_degraded` - Partial data, quality issues
- `*_chaos` - Edge cases, malformed data

Coverage target: 70% minimum across all components.
