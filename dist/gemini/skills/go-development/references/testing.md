# Go Testing Reference

Testing conventions. Follows `foundations` testing principles.

## Project Conventions

| Decision | Convention |
|----------|-----------|
| Test location | Next to source: `foo.go` -> `foo_test.go` |
| Pattern | Table-driven tests for multiple cases |
| Race detection | Always run with `-race` in CI |
| Helpers | Use `t.Helper()` in all test utilities |
| Parallelism | Use `t.Parallel()` where safe (no shared state) |
| Assertions | Standard library preferred; testify when readability improves |

Run commands:
- `go test ./...` -- all packages
- `go test -race ./...` -- detect data races
- `go test -cover ./...` -- coverage report
- `go test -bench=. -benchmem` -- benchmarks

## Always/Never

| Always | Never |
|--------|-------|
| Use table-driven tests for multiple cases | Duplicate test logic |
| Name test cases descriptively | Use `test1`, `test2` |
| Run tests with `-race` in CI | Ship without race detection |
| Use `t.Helper()` in test utilities | Obscure failure locations |
| Test edge cases (nil, empty, boundary) | Only test happy path |
| Use `t.Parallel()` where safe | Parallelize tests with shared state |
