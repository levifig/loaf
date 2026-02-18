# Python Production Deployment

## Contents
- Dockerfile Convention
- Settings Pattern
- Observability
- Lifecycle Management
- Critical Rules

Production-ready deployment patterns for Python applications.

## Dockerfile Convention

Multi-stage build with uv, non-root user:

```dockerfile
FROM python:3.12-slim as builder
WORKDIR /build
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev --no-install-project

FROM python:3.12-slim
RUN useradd -m -u 1000 appuser
WORKDIR /app
COPY --from=builder /build/.venv /app/.venv
COPY --chown=appuser:appuser src/ /app/src/
ENV PATH="/app/.venv/bin:$PATH" PYTHONUNBUFFERED=1
USER appuser
HEALTHCHECK --interval=30s --timeout=3s \
  CMD python -c "import httpx; httpx.get('http://localhost:8000/health')"
EXPOSE 8000
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

## Settings Pattern

Use `pydantic-settings` with `BaseSettings` + `SettingsConfigDict(env_file=".env")`. Cache with `@lru_cache` on getter. See [pydantic.md](pydantic.md) for full pattern.

## Observability

| Component | Choice |
|-----------|--------|
| Logging | structlog (JSON) |
| Tracing | OpenTelemetry + OTLP exporter |
| Health | `/health` (liveness) + `/health/ready` (readiness) |

Logging convention: JSON output via structlog with `contextvars` for request-scoped context.

## Lifecycle Management

Use FastAPI `lifespan` context manager for startup/shutdown (connect/disconnect resources). Gunicorn workers: `cpu_count() * 2 + 1`, worker class `uvicorn.workers.UvicornWorker`.

## Critical Rules

### Always
- Use multi-stage Docker builds with uv
- Run as non-root user
- Implement health checks (liveness + readiness)
- Use structured JSON logging
- Configure graceful shutdown via lifespan

### Never
- Run as root in containers
- Hardcode secrets in images
- Skip health checks
- Use development server in production
