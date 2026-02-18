---
name: go-development
description: >-
  Covers idiomatic Go development with patterns from Effective Go. Includes
  project structure, goroutines and channels for concurrency, explicit error
  handling, table-driven tests, and common idioms (guard clauses, defer,
  embedding). Use when writing Go services, CLIs, or libraries, or when the user
  asks "how do I handle errors in Go?" or "what's the Go way to do X?"
---

# Go Skill

Patterns and best practices for Go development, grounded in Effective Go principles and community conventions.

## Reference Files

| File | Use When |
|------|----------|
| `references/core.md` | Setting up projects, naming variables, formatting code |
| `references/concurrency.md` | Working with goroutines, channels, or sync primitives |
| `references/errors.md` | Handling errors, creating custom errors, using panic/recover |
| `references/testing.md` | Writing table-driven tests, benchmarks, or examples |
| `references/idioms.md` | Using guard clauses, defer, embedding, or method receivers |

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Packages | lowercase, short, no underscores | `http`, `bytes`, `strconv` |
| Exported | PascalCase | `ReadFile`, `HTTPClient` |
| Unexported | camelCase | `parseConfig`, `internalState` |
| Interfaces (single method) | Method + "er" | `Reader`, `Stringer`, `Handler` |
| Getters | No "Get" prefix | `user.Name()` not `user.GetName()` |
| Acronyms | All caps when exported | `HTTPServer`, `xmlParser` |

## Key Decisions

| Area | Decision |
|------|----------|
| Error handling | Return errors, check immediately, no exceptions |
| Formatting | `gofmt` is non-negotiable |
| Concurrency | Channels for communication, goroutines are cheap |
| Interfaces | Small (1-3 methods), defined by consumer |
| Dependencies | Standard library first, minimize external deps |
| Generics | Use when type safety matters; prefer interfaces when behavior matters |
