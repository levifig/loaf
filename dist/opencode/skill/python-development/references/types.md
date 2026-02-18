# Python Type Safety

## Contents
- Conventions
- Mypy Configuration
- Critical Rules

Type hint conventions and mypy configuration.

## Conventions

- **Union syntax**: Use `X | None` (not `Optional[X]`)
- **Collections**: Use `collections.abc` (`Sequence`, `Mapping`) over `typing` equivalents
- **Structural typing**: Prefer `Protocol` over ABC for interfaces
- **TypedDict**: Use for structured dicts with `Required`/`NotRequired`
- **Annotated**: Use `Annotated[int, Field(gt=0)]` for constrained types
- **TypeGuard**: Use for runtime narrowing functions
- **ParamSpec**: Use for decorator typing that preserves signatures
- **Type aliases**: Use PEP 695 `type Point = tuple[float, float]`

## Mypy Configuration

```toml
[tool.mypy]
python_version = "3.12"
strict = true
warn_return_any = true
warn_unused_configs = true

[[tool.mypy.overrides]]
module = "tests.*"
disallow_untyped_defs = false
```

Always run mypy in strict mode. Relax only for test files.

## Critical Rules

### Always
- Annotate all public functions
- Use strict mypy configuration
- Prefer Protocol over ABC
- Use TypedDict for structured dicts
- Run mypy in CI/CD

### Never
- Use `Any` without justification
- Ignore mypy errors without `# type: ignore[reason]`
- Skip type hints on public APIs
