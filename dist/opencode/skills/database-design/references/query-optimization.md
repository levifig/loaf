# Query Optimization

## Contents
- EXPLAIN ANALYZE Red Flags
- Common Anti-Patterns
- Pagination Strategy
- Connection Pooling
- Always / Never

Query performance conventions and decision guidance.

## EXPLAIN ANALYZE Red Flags

Always run `EXPLAIN ANALYZE` before optimizing. Watch for:

| Red Flag | Meaning |
|----------|---------|
| Seq Scan on large table with selective WHERE | Missing or unused index |
| Large estimated vs actual row difference | Stale statistics (run `ANALYZE`) |
| High loops count in nested operations | N+1 at the database level |
| Sort without index support | Add index matching ORDER BY |

## Common Anti-Patterns

| Anti-Pattern | Fix |
|--------------|-----|
| Function on indexed column (`WHERE LOWER(email) = ...`) | Functional index or store normalized |
| `SELECT *` | Explicit column list |
| `OR` preventing index use | Rewrite as `UNION` for separate index scans |
| `NOT IN` with subquery (NULL issues) | Use `NOT EXISTS` instead |
| N+1 queries | Eager loading, batch `WHERE IN (...)`, or JOIN |

## Pagination Strategy

| Method | Use When |
|--------|----------|
| Offset (`LIMIT/OFFSET`) | Small datasets, need page jumping |
| Cursor (`WHERE id > ? LIMIT N`) | Large datasets, infinite scroll, real-time data |

**Default to cursor pagination** for APIs. Offset degrades at scale.

## Connection Pooling

| Setting | Guidance |
|---------|----------|
| Pool size | Start at 2x CPU cores, measure and adjust |
| Idle timeout | 10-30 seconds |
| Max lifetime | 30 min - 1 hour (rotate connections) |
| Formula | `max_connections > (pool_size * app_instances) + admin_connections` |

## Always / Never

| Always | Never |
|--------|-------|
| Use EXPLAIN ANALYZE before optimizing | Optimize based on intuition alone |
| Fetch only needed columns | SELECT * in production code |
| Use parameterized queries | Concatenate user input into SQL |
| Set statement timeouts | Allow unbounded query execution |
| Monitor slow query logs | Ignore database performance metrics |
| Test with production-scale data | Benchmark on empty tables |
