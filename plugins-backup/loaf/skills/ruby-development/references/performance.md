# Rails Performance

## N+1 Prevention

| Method | Use Case |
|--------|----------|
| `includes` | General preloading (smart: JOIN or separate) |
| `preload` | Force separate queries |
| `eager_load` | Force single LEFT OUTER JOIN |

Use `bullet` gem in development to detect N+1 queries.

## Index Guidelines

- Always index foreign keys
- Index columns used in WHERE and ORDER BY
- Consider composite indexes for common query patterns
- Remove unused indexes (they slow writes)

## Caching

- **Fragment caching**: `<% cache @post do %>` with `touch: true` on associations
- **Collection caching**: `render partial: "post", collection: @posts, cached: true`
- **Low-level caching**: `Rails.cache.fetch("key", expires_in: 1.hour) { compute }`

## Query Optimization

- `select(:id, :name)` for needed columns only
- `pluck(:email)` for single column arrays
- `find_each(batch_size: 1000)` for large datasets
- `counter_cache: true` to avoid COUNT queries

## Performance Checklist

- [ ] Foreign keys indexed
- [ ] N+1 queries eliminated (use Bullet)
- [ ] Large datasets paginated
- [ ] Expensive views cached
- [ ] Counter caches for counts
- [ ] Batch processing for bulk operations
