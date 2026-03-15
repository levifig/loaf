# Test Debugging

## Contents
- Flaky Test Diagnosis
- Test Isolation Patterns
- State Pollution Detection
- Environment Differences
- Test Debugging Workflow

Strategies for diagnosing flaky tests, isolation failures, and environment-related test issues.

## Flaky Test Diagnosis

A flaky test passes sometimes and fails sometimes with the same code.

| Category | Symptoms | Fix |
|----------|----------|-----|
| **Timing** | Passes locally, fails in CI; adding `sleep()` helps | Wait for conditions with timeout, not fixed sleeps |
| **Order** | Passes alone, fails in suite; reordering changes results | Fix shared state; run with `--random-order` |
| **Non-deterministic data** | Different failures each time; works with specific seed | Control randomness with explicit seeds |

### Diagnosis Commands

```bash
# Run in random order to catch order dependencies
pytest --random-order

# Find the guilty test pair
pytest tests/test_a.py tests/test_b.py -v  # passes?
pytest tests/test_b.py tests/test_a.py -v  # fails?

# Run suspect test in isolation
pytest tests/test_orders.py::test_specific -v
```

## Test Isolation Patterns

### Database Isolation

Use transactional fixtures -- each test runs in a transaction that rolls back:

```python
@pytest.fixture
def db_session(engine):
    connection = engine.connect()
    transaction = connection.begin()
    session = Session(bind=connection)
    yield session
    session.close()
    transaction.rollback()
    connection.close()
```

### Time Isolation

Freeze time for deterministic tests:

```python
from freezegun import freeze_time

@freeze_time("2024-01-15 12:00:00")
def test_order_expiry():
    order = create_order()
    assert not order.expired
```

### External Service Isolation

Mock external services at the boundary:

```python
@pytest.fixture
def mock_payment_api(mocker):
    return mocker.patch(
        'app.services.payment.PaymentAPI.charge',
        return_value={'status': 'success', 'id': 'ch_123'}
    )
```

## State Pollution Detection

### Common Sources

| Source | Detection | Fix |
|--------|-----------|-----|
| Module globals | Check imports for mutations | Use fixtures, reset in teardown |
| Class variables | Look for `cls.` modifications | Use instance variables |
| Singletons | Check shared instances | Reset or mock singletons |
| Environment variables | Print env in failing test | Restore in teardown |
| File system | Check for temp files | Use tmp directories, clean up |
| Database | Check for leftover records | Transaction rollback |
| Caches | Check memoized values | Clear caches in setup |

### Teardown Pattern

```python
@pytest.fixture(autouse=True)
def reset_global_state():
    original_config = app.config.copy()
    yield
    app.config = original_config
    cache.clear()
```

## Environment Differences

| Local vs CI Difference | Solution |
|------------------------|----------|
| CI slower, races more likely | Use proper waits, not sleeps |
| CI has less memory/CPU | Check resource usage |
| CI runs tests in parallel | Ensure isolation |
| Different temp directories | Use portable paths |
| CI uses UTC | Freeze time or use UTC explicitly |

### Reproducing CI Locally

```bash
# Docker: Use same image as CI
docker run -it --rm -v $(pwd):/app -w /app python:3.11 bash

# Act: Run GitHub Actions locally
act -j test

# Environment parity
export CI=true TZ=UTC && pytest
```

## Test Debugging Workflow

1. **Isolate:** Run failing test alone (`pytest path::test_name -v`)
2. **Diagnose:** Add `caplog` or print statements for state inspection
3. **Minimize:** Remove setup pieces until bug disappears to find root cause
4. **Verify assumptions:** Print intermediate state to find where expectation diverges from reality

### Flaky Test Prevention

| Practice | Why |
|----------|-----|
| Factories over shared fixtures | Fresh data each test |
| Avoid shared state | Tests can't pollute each other |
| Mock time and randomness | Deterministic behavior |
| Transaction rollback | Auto-cleanup |
| Run in random order | Catch order dependencies |
| Set explicit timeouts | Fail fast on hangs |
