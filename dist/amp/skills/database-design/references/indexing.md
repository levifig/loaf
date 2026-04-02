# Indexing

## Contents
- Index Type Selection
- When to Create Indexes
- Composite Index Column Order
- Maintenance
- Always / Never

Index design decisions and project conventions.

## Index Type Selection

| Type | Use Case |
|------|----------|
| B-tree | Default. Equality, range, sorting |
| GIN | Arrays, JSONB, full-text search |
| GiST | Geometric, range types |
| BRIN | Large tables with natural ordering (time-series) |
| Hash | Equality only (rarely needed) |

**Default to B-tree** unless you have a specific need.

## When to Create Indexes

**Create for:**
- WHERE clauses with high selectivity
- JOIN conditions
- ORDER BY (especially with LIMIT)
- Foreign keys (prevents full table scans on delete)

**Skip for:**
- Low-cardinality columns alone (e.g., status with 3 values)
- Small tables (seq scan may be faster)
- Write-heavy tables without read justification

## Composite Index Column Order

**Rule:** Equality columns first (most selective), range columns last.

Index on `(status, created_at)` supports:
- `WHERE status = 'active' ORDER BY created_at DESC` -- yes
- `WHERE status = 'active'` -- yes
- `WHERE created_at > '...'` alone -- no (wrong column first)

Use partial indexes for common filters: `CREATE INDEX ... WHERE deleted_at IS NULL`.

Use covering indexes (`INCLUDE`) to avoid table lookups when the tradeoff (larger index) is justified.

## Maintenance

- Monitor unused indexes: `pg_stat_user_indexes WHERE idx_scan = 0`
- Use `REINDEX CONCURRENTLY` for bloated indexes (Postgres 12+)
- Schedule regular `ANALYZE` or rely on autovacuum

## Always / Never

| Always | Never |
|--------|-------|
| Index foreign keys | Create indexes without measuring need |
| Consider selectivity before indexing | Index every column "just in case" |
| Use CONCURRENTLY for production indexes | Create indexes during peak traffic |
| Monitor index usage and remove unused | Keep unused indexes indefinitely |
| Match index order to query patterns | Assume column order doesn't matter |
| Use partial indexes for common filters | Index boolean columns alone |
