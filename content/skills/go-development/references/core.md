# Go Core Reference

## Contents
- Go Stack
- Project Structure
- Naming Conventions
- Always/Never

Go tooling, project layout, and naming decisions.

## Go Stack

| Component | Tool | Notes |
|-----------|------|-------|
| Runtime | Go 1.21+ | Garbage collected, statically linked binaries |
| Package Manager | Go Modules | `go.mod`, `go.sum` |
| Linter | `golangci-lint` | Aggregates multiple linters |
| Formatter | `gofmt` | Canonical formatting, no configuration |
| Testing | `go test` | Built-in, table-driven patterns |
| Documentation | `godoc` | Comments become documentation |

## Project Structure

```
project/
+-- go.mod              # Module definition
+-- go.sum              # Dependency checksums
+-- main.go             # Entry point (for commands)
+-- cmd/                # Multiple entry points
|   +-- myapp/
|       +-- main.go
+-- internal/           # Private packages (enforced by Go)
|   +-- config/
+-- pkg/                # Public library code (optional)
+-- *_test.go           # Tests alongside source
```

Key rules:
- `internal/` packages cannot be imported outside the module
- Tests live next to the code: `foo.go` -> `foo_test.go`
- One package per directory

## Naming Conventions

| Element | Rule | Examples |
|---------|------|----------|
| Packages | Short, lowercase, no underscores | `http`, `io`, `bufio` |
| Files | lowercase, underscores OK | `http_client.go` |
| Exported | PascalCase | `NewClient`, `ErrNotFound` |
| Unexported | camelCase | `parseHeader`, `defaultTimeout` |
| Interfaces | Behavior, often "-er" suffix | `Reader`, `Handler` |
| Getters | No "Get" prefix | `func (u *User) Name() string` |
| Acronyms | Consistent case | `HTTPServer`, not `HttpServer` |

Import grouping: standard library, then external, then internal (separated by blank lines).

## Always/Never

| Always | Never |
|--------|-------|
| Run `gofmt` before committing | Disable or configure gofmt |
| Use `go mod tidy` to clean deps | Vendor without good reason |
| Document exported symbols | Leave public API undocumented |
| Handle errors explicitly | Ignore errors with `_` without reason |
| Use `internal/` for private code | Expose implementation details |
| Prefer standard library | Add deps for trivial functionality |
