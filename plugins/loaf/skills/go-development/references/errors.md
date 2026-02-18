# Go Error Handling Reference

Error patterns and project conventions.

## Philosophy

Errors are values, not exceptions. Returned, checked, and handled explicitly.

## Project Conventions

| Pattern | Convention |
|---------|-----------|
| Wrapping | Always use `%w` to preserve error chain |
| Context | Add context when wrapping: `fmt.Errorf("open config: %w", err)` |
| Sentinels | Define `var ErrXxx = errors.New(...)` for known conditions |
| Checking | Use `errors.Is` for sentinels, `errors.As` for typed errors |
| Panic | Only for programmer errors and initialization failures |
| Recover | Only at boundaries (HTTP handlers, long-running services) |

## Always/Never

| Always | Never |
|--------|-------|
| Check errors immediately | Ignore without explicit reason |
| Add context when wrapping | Wrap without adding information |
| Use `%w` to preserve chain | Use `%v` when callers need `errors.Is` |
| Define sentinel errors for known conditions | Compare error strings |
| Panic only for programmer errors | Panic for invalid user input |
| Recover at boundaries | Recover and silently continue |
| Document which errors a function returns | Leave error conditions undocumented |
