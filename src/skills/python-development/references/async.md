# Python Async Patterns

## Contents
- Concurrency Conventions
- Sync/Async Bridge
- Critical Rules

Async conventions for asyncio-based Python projects.

## Concurrency Conventions

- Prefer `asyncio.TaskGroup` (Python 3.11+) over `asyncio.gather` for structured concurrency
- Use `asyncio.timeout()` for explicit timeout bounds
- Use `@asynccontextmanager` for async resource lifecycle

When calling multiple independent async operations, use TaskGroup:

```python
async with asyncio.TaskGroup() as tg:
    tasks = [tg.create_task(fetch(url)) for url in urls]
results = [task.result() for task in tasks]
```

## Sync/Async Bridge

Run blocking sync code from async context via executor:

```python
async def run_sync_in_executor(func, *args):
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, partial(func, *args))
```

Run async from sync context: `asyncio.run(async_function())`

## Critical Rules

### Always
- Use `async with` for async context managers
- Track created tasks or use TaskGroup
- Handle exceptions in concurrent tasks
- Type hint as `Awaitable[T]` or `Coroutine`

### Never
- Block the event loop with sync I/O (`time.sleep`, file I/O)
- Forget to await coroutines
- Use async for CPU-bound work (use `run_in_executor`)
- Create tasks without tracking references
