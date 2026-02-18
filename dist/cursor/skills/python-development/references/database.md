# Python Database Operations

## Contents
- Stack Decisions
- Async Connection Convention
- Migration Workflow
- Critical Rules

Async database operations with SQLAlchemy 2.0 and Alembic.

## Stack Decisions

| Component | Choice | Why |
|-----------|--------|-----|
| ORM | SQLAlchemy 2.0 | Mapped types, async-native |
| Driver | asyncpg | Fastest PostgreSQL async driver |
| Migrations | Alembic | Autogenerate from models |
| Models | `DeclarativeBase` + `Mapped[]` | Type-safe column definitions |

## Async Connection Convention

Standard setup: `create_async_engine` with `pool_size=5`, `max_overflow=10`, `pool_pre_ping=True`. Session via `async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)`.

Key conventions:
- Always use `expire_on_commit=False` for async sessions
- Use `pool_pre_ping=True` for connection health checks
- Use Repository pattern for CRUD (class wrapping session)
- Use `selectinload()` for eager loading relationships

## Migration Workflow

```bash
alembic revision --autogenerate -m "add users table"
alembic upgrade head
alembic downgrade -1
```

Every schema change goes through Alembic. Never modify tables directly.

## Critical Rules

### Always
- Use async session and queries
- Close sessions properly (async context manager)
- Use `relationship()` for foreign keys
- Create indexes for query columns
- Use migrations for all schema changes

### Never
- Use sync SQLAlchemy in async apps
- Forget to await database calls
- Skip migrations for schema changes
- Use string queries without parameterization
