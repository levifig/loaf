# Python Testing with pytest

## Contents
- Test Organization
- Fixture Conventions
- Async Testing
- Coverage Config
- pytest-specific Rules

## Test Organization

Directory layout: `tests/{unit,integration,e2e}/` with `conftest.py` at root for shared fixtures. Test naming: `test_<unit>_<scenario>` or `test_<unit>_<scenario>_<result>`.

## Fixture Conventions

- Use `@pytest.fixture` for all setup/teardown (never inline setup)
- Shared fixtures go in `conftest.py`
- Scope appropriately: default (function), `scope="session"` for expensive resources
- Async DB fixture: in-memory SQLite + rollback per test

```python
@pytest.fixture
async def db_session() -> AsyncSession:
    engine = create_async_engine("sqlite+aiosqlite:///:memory:")
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    async with async_session_maker() as session:
        yield session
        await session.rollback()
```

- Use `AsyncMock` for async service mocks
- Use `@patch("my_app.services.module.function")` for external dependencies

## Async Testing

- All async tests require `@pytest.mark.asyncio`
- FastAPI endpoint tests use httpx `AsyncClient` as a fixture: `AsyncClient(app=app, base_url="http://test")`

## Coverage Config

Minimum 80% coverage (`--cov-fail-under=80`). Standard markers: `unit`, `integration`, `slow`. See [core.md](core.md) for pytest config in pyproject.toml.

## pytest-specific Rules

### Always
- Use `@pytest.mark.parametrize` for input variations
- Use `@pytest.mark.asyncio` for all async tests

### Never
- Use `time.sleep()` -- use `pytest-asyncio` or mock time
- Skip `await` in async fixtures (causes silent failures)
