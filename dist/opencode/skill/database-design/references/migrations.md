# Migrations

## Contents
- Project Conventions
- Zero-Downtime Rules
- Backward Compatibility
- Data vs Schema Migrations
- Always / Never

Safe, reversible migration patterns for zero-downtime deployments.

## Project Conventions

| Decision | Convention |
|----------|-----------|
| Naming | `{timestamp}_{action}_{target}.sql` (e.g., `20240115120000_create_users_table.sql`) |
| Actions | create, add, drop, rename, modify |
| Reversibility | Every migration has a rollback path; backup before irreversible ops |
| Index creation | Always `CONCURRENTLY` in production |
| Large data migrations | Separate scripts, not blocking deployment |

## Zero-Downtime Rules

| Operation | Safe Approach |
|-----------|---------------|
| Add column | Add as nullable; backfill; then add NOT NULL constraint |
| Remove column | Stop writing -> deploy ignoring column -> drop in later migration |
| Rename column | Add new -> backfill -> write to both -> read from new -> drop old |
| Add index | `CREATE INDEX CONCURRENTLY` |

## Backward Compatibility

| Safe | Unsafe |
|------|--------|
| Add nullable column | Add NOT NULL without default |
| Add table | Drop table |
| Add index (concurrently) | Drop index (if used) |
| Widen column (VARCHAR(50) -> VARCHAR(100)) | Narrow column |

**Rule:** Old code must continue working during migration.

## Data vs Schema Migrations

| Type | Purpose | When to Run |
|------|---------|-------------|
| Schema (DDL) | Structure changes | Deploy time |
| Data (DML) | Content changes | After schema, before/after deploy |

Separate large data migrations into independent scripts:
```
001_add_status_column.sql        # Schema
001_backfill_status.sql          # Data (run separately)
002_add_status_not_null.sql      # Schema (after backfill)
```

## Always / Never

| Always | Never |
|--------|-------|
| Test migrations on production-like data | Run untested migrations in production |
| Include rollback scripts | Deploy irreversible changes without backup |
| Use transactions where supported | Mix DDL and DML in same transaction (some DBs) |
| Run migrations in a specific order | Assume migration order across branches |
| Coordinate migrations with code deploys | Drop columns before removing code references |
| Document expected duration for large migrations | Block deployments with long-running migrations |
