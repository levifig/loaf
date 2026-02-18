# Ruby Debugging

## Tools

| Tool | Use |
|------|-----|
| `binding.irb` | Built-in (Ruby 2.5+), opens interactive console |
| `binding.pry` | Enhanced REPL with `pry-rails` gem |
| `byebug` | Step debugger (`n`/`s`/`c`/`f`/`bt`) |
| `rails console --sandbox` | Test with auto-rollback on exit |

## Rails Logger

```ruby
Rails.logger.info "Order created", order_id: order.id
Rails.logger.tagged("OrderService", order.id) do
  Rails.logger.info "Starting processing"
end
```

## Useful Techniques

- `user.orders.to_sql` — see generated SQL in console
- `Rails.backtrace_cleaner.clean(e.backtrace)` — filtered backtraces
- `reload!` in console after code changes
