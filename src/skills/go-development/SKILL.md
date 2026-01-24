---
name: go-development
description: >-
  Covers idiomatic Go development with patterns from Effective Go. Includes project structure,
  goroutines and channels for concurrency, explicit error handling, table-driven tests, and
  common idioms (guard clauses, defer, embedding). Use when writing Go services, CLIs, or
  libraries, or when the user asks "how do I handle errors in Go?" or "what's the Go way to do X?"
---

# Go Skill

Patterns and best practices for Go development, grounded in Effective Go principles and community conventions.

## Philosophy

Go values **simplicity over cleverness**. The language deliberately omits features that add complexity without proportional benefit. This isn't a limitation—it's a design decision that produces readable, maintainable code.

### Core Principles

1. **Simplicity** - If there's a simple way and a clever way, choose simple
2. **Explicitness** - Error handling is explicit, not exceptional
3. **Conventions** - `gofmt` settles style debates; follow it
4. **Composition** - Prefer embedding and interfaces over inheritance
5. **Concurrency** - "Do not communicate by sharing memory; share memory by communicating"

### When to Use This Skill

- Writing Go services, CLIs, or libraries
- Designing concurrent systems with goroutines and channels
- Implementing idiomatic error handling
- Structuring Go projects
- Writing table-driven tests

## Reference Files

| File | Use When |
|------|----------|
| `references/core.md` | Setting up projects, naming variables, formatting code |
| `references/concurrency.md` | Working with goroutines, channels, or sync primitives |
| `references/errors.md` | Handling errors, creating custom errors, using panic/recover |
| `references/testing.md` | Writing table-driven tests, benchmarks, or examples |
| `references/idioms.md` | Using guard clauses, defer, embedding, or method receivers |

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Packages | lowercase, short, no underscores | `http`, `bytes`, `strconv` |
| Exported | PascalCase | `ReadFile`, `HTTPClient` |
| Unexported | camelCase | `parseConfig`, `internalState` |
| Interfaces (single method) | Method + "er" | `Reader`, `Stringer`, `Handler` |
| Getters | No "Get" prefix | `user.Name()` not `user.GetName()` |
| Acronyms | All caps when exported | `HTTPServer`, `xmlParser` |

### Common Patterns

**Error check immediately after call:**
```go
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()
```

**Guard clause (return early):**
```go
func process(data []byte) error {
    if len(data) == 0 {
        return errors.New("empty data")
    }
    // main logic here
}
```

**Table-driven test:**
```go
tests := []struct {
    name  string
    input int
    want  int
}{
    {"zero", 0, 0},
    {"positive", 5, 25},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := Square(tt.input)
        if got != tt.want {
            t.Errorf("Square(%d) = %d, want %d", tt.input, got, tt.want)
        }
    })
}
```

## Core Principles Summary

| Principle | Go Approach |
|-----------|-------------|
| Error handling | Return errors, check immediately, no exceptions |
| Formatting | `gofmt` is non-negotiable |
| Concurrency | Channels for communication, goroutines are cheap |
| Interfaces | Small (1-3 methods), defined by consumer |
| Dependencies | Standard library first, minimize external deps |
| Generics | Use when type safety matters; prefer interfaces when behavior matters |

## Integration with Foundations

This skill builds on `foundations` for:
- Code review practices
- Testing principles
- Documentation standards
- Security considerations

Reference `foundations` for universal patterns; this skill adds Go-specific idioms.
