# Python Debugging

## Contents
- Structured Logging
- pytest Debugging
- Breakpoints

Debug patterns and tools for Python.

## Structured Logging

Use structlog with bound context for request tracing:

```python
import structlog
logger = structlog.get_logger()

# Bind context that persists across log calls
logger = logger.bind(request_id=request.id, user_id=user.id)

# Structured key-value pairs, not f-strings
logger.info("order_created", order_id=order.id, total=order.total)
logger.error("payment_failed", order_id=order.id, error_code=e.code)
```

Convention: Use snake_case event names, structured key-value data. Never use f-string messages in structlog.

## pytest Debugging

```bash
pytest --pdb              # Drop into debugger on failure
pytest -x --pdb           # Stop on first failure + debugger
pytest -l                 # Show local variables in tracebacks
pytest -v -s              # Verbose + print statements visible
pytest tests/test_orders.py::test_create_order -v  # Single test
```

## Breakpoints

```python
breakpoint()              # Built-in (Python 3.7+)
if suspicious_condition:  # Conditional breakpoint
    breakpoint()
```
