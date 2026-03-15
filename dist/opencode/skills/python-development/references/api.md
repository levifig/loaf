# Python API Integration

## Contents
- Stack Decisions
- API Client Pattern
- Retry Convention
- Critical Rules

Building reliable API clients with httpx and tenacity.

## Stack Decisions

| Component | Choice | Why |
|-----------|--------|-----|
| HTTP Client | httpx (async) | Native async, connection pooling |
| Retry | tenacity | Composable retry strategies |
| Validation | Pydantic | Response parsing into typed models |

Never use `requests` (sync, blocking).

## API Client Pattern

Wrap httpx `AsyncClient` in a typed async context manager (`__aenter__`/`__aexit__`) that parses responses into Pydantic models.

Conventions:
- Always use `async with` for client lifecycle
- Parse responses into Pydantic models at the client boundary
- Set explicit timeouts (default 30s)
- Generic `get(endpoint, response_model: type[T]) -> T` method

## Retry Convention

Use tenacity with exponential backoff for transient failures:

```python
@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=4, max=10),
    retry=retry_if_exception_type((httpx.TimeoutException, httpx.NetworkError)),
)
```

Standard: 3 attempts, exponential backoff 4-10s, retry only on network/timeout errors.

## Critical Rules

### Always
- Use async HTTP clients (httpx)
- Implement retry logic with tenacity
- Validate responses with Pydantic
- Handle rate limiting
- Use context managers for client lifecycle

### Never
- Use requests library (sync, blocking)
- Ignore HTTP error status codes
- Skip timeout configuration
- Hardcode API credentials
