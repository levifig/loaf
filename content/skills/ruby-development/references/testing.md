# Rails Testing with Minitest

## Stack Decisions

- **Minitest over RSpec** — simpler, faster, built-in to Rails
- **Fixtures over factories** — loaded once per test run, predictable, fast

## Test Organization

```
test/
├── fixtures/        # YAML test data
├── models/          # Model unit tests
├── controllers/     # HTTP integration tests
├── system/          # Browser tests (JS, Turbo)
├── services/        # Service object tests
└── test_helper.rb   # Shared setup
```

## Conventions

- Access fixtures: `users(:alice)`, `posts(:published_post)`
- System tests: `driven_by :selenium, using: :headless_chrome`
- Mock external services and time, never mock your own models
- `travel_to` for time-dependent behavior
- All fixtures use same password for `sign_in_as` helper
- `assert_difference "Post.count", 1` for creation tests

## Running Tests

```bash
bin/rails test                           # All tests
bin/rails test:models                    # Model tests only
bin/rails test test/models/user_test.rb:10  # Single test (line number)
bin/rails test:system                    # Browser tests
```
