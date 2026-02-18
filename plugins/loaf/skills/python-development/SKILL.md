---
name: python-development
description: >-
  Covers Python 3.12+ development with uv project setup, FastAPI web services,
  Pydantic validation, async/await patterns, type hints with mypy, pytest
  testing, SQLAlchemy database operations, Polars data processing, httpx API
  clients, and Docker deployment. Use when building Python APIs, writing async
  code, or when the user asks "how do I validate data?" or "what's the best way
  to structure a Python project?" Not for schema design decisions or migration
  strategies (use database-design).
user-invocable: false
agent: 'backend-dev'
allowed-tools: 'Read, Write, Edit, Bash, Glob, Grep'
---

# Python Development

Modern Python 3.12+ development with FastAPI ecosystem.

## Stack Overview

| Layer | Default | Alternatives |
|-------|---------|--------------|
| Runtime | Python 3.12+ | - |
| Package Manager | uv | rye, poetry |
| Linter/Formatter | ruff | black + flake8 |
| Type Checker | mypy (strict) | pyright |
| Web Framework | FastAPI | Flask, Django |
| Validation | Pydantic v2 | - |
| ORM | SQLAlchemy 2.0 | - |
| Data Processing | Polars | Pandas |
| HTTP Client | httpx | aiohttp |
| Testing | pytest | - |
| Containerization | Docker | - |

## Topics

| Topic | Use For |
|-------|---------|
| [Core](references/core.md) | Project setup, pyproject.toml, modern Python features |
| [FastAPI](references/fastapi.md) | REST APIs, routing, dependency injection, middleware |
| [Pydantic](references/pydantic.md) | Data models, validation, settings management |
| [Async](references/async.md) | async/await, TaskGroup, context managers |
| [Types](references/types.md) | Type hints, mypy, Protocol, generics |
| [Testing](references/testing.md) | pytest, fixtures, mocking, async tests |
| [Database](references/database.md) | SQLAlchemy 2.0, Alembic migrations, transactions |
| [Data](references/data.md) | Polars, ETL pipelines, schema validation |
| [API Clients](references/api.md) | httpx, retries, rate limiting, error handling |
| [Deployment](references/deployment.md) | Docker, logging, OpenTelemetry, health checks |
| [Debugging](references/debugging.md) | pdb, structlog, pytest debugging, remote debugging |

## Critical Rules

### Always

- Use `async def` for I/O-bound operations
- Use Pydantic models for external input validation
- Use `pathlib.Path` for file operations
- Run mypy in CI/CD pipeline

### Never

- Block the event loop with sync I/O in async code
- Use mutable default arguments (`def foo(items=[])`)
- Skip validation on external input
- Hardcode configuration values (use pydantic-settings)
