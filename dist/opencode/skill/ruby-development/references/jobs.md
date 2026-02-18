# Background Jobs with Solid Queue

Database-backed background processing for Rails 8+. No Redis dependency.

## Queue Priorities

| Queue | Use Case | Polling |
|-------|----------|---------|
| `critical` | Payments, security | 0.1s |
| `default` | Standard operations | 1s |
| `mailers` | Email delivery | 1s |
| `low` | Reports, cleanup | 5s |

## Conventions

- **Idempotent jobs**: Check state before acting (`return if order.processed?`)
- **Pass IDs, not objects**: `perform(order_id)` not `perform(order)`
- **Retry per error type**: `retry_on Net::OpenTimeout, wait: :polynomially_longer, attempts: 5`
- **Discard on permanent failures**: `discard_on ActiveRecord::RecordNotFound`
- **Concurrency limits**: `limits_concurrency to: 1, key: ->(user_id) { user_id }`

## Recurring Jobs

```yaml
# config/recurring.yml
production:
  cleanup_sessions:
    class: CleanupSessionsJob
    schedule: every day at 3am
  sync_inventory:
    class: SyncInventoryJob
    schedule: every 15 minutes
```

## Testing

Test with `perform_now` (synchronous). Verify idempotency by calling twice.
