# Schema Design

## Contents
- Primary Key Strategy
- Project Conventions
- Soft Delete vs Hard Delete
- Always / Never

Schema design decisions and project conventions.

## Primary Key Strategy

| Type | Pros | Cons | Use For |
|------|------|------|---------|
| UUID v4 | Globally unique, safe to expose | 16 bytes, fragments indexes | Distributed systems, public-facing IDs |
| ULID | Sortable by time, globally unique | Less common library support | Event sourcing, time-series, logs |
| Serial/Identity | Compact (4-8 bytes), sequential | Exposes row count, conflicts in distributed writes | Internal tables, single-writer |
| Composite | Enforces relationship at DB level | Complicates joins | Junction tables, natural keys |

**Default:** UUID v4 for public-facing, serial for internal-only.

## Project Conventions

| Decision | Convention |
|----------|-----------|
| Normalization | Start at 3NF; denormalize deliberately with measured justification |
| Denormalization | Maintain normalized source; create materialized views for read optimization |
| Foreign keys | Default to `ON DELETE RESTRICT`; cascades hide data loss |
| Timestamps | Use `TIMESTAMPTZ` (timezone-aware) |
| Money | Use `DECIMAL`, never `FLOAT` |
| Audit columns | `created_at`, `updated_at` on every table; use triggers for `updated_at` |
| Nullability | `NOT NULL` unless truly optional |

## Soft Delete vs Hard Delete

| Approach | Pros | Cons |
|----------|------|------|
| Soft delete (`deleted_at`) | Audit trail, easy restore | Query complexity, storage growth |
| Hard delete | Simple queries, storage efficient | No recovery |

When using soft delete:
- Add partial index on `deleted_at IS NULL` for common queries
- Include `deleted_by` for audit trail
- All queries must filter `WHERE deleted_at IS NULL`

## Always / Never

| Always | Never |
|--------|-------|
| Define explicit primary keys | Use reserved words as column names |
| Add NOT NULL unless truly optional | Store multiple values in one column |
| Use appropriate data types | Use FLOAT for money (use DECIMAL) |
| Add foreign key constraints | Trust application code alone for integrity |
| Use TIMESTAMPTZ for timestamps | Use magic values (-1, 9999) for special meanings |
| Document non-obvious column purposes | Store derived data without invalidation strategy |
