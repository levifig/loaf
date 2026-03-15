# Rails Controllers

## Conventions

- Controllers handle HTTP only — no business logic
- Keep actions under 10 lines
- One controller per resource
- Seven RESTful actions maximum

## Redirect vs Render

| Situation | Use | Status |
|-----------|-----|--------|
| Successful mutation | `redirect_to` | 302/303 |
| Validation failure | `render` | 422 |
| Not found | `render` | 404 |

- Use `flash.now` with `render`, regular `flash` with `redirect_to`
- Return `status: :unprocessable_entity` on validation failures

## Error Handling

`rescue_from ActiveRecord::RecordNotFound` in `ApplicationController` for centralized 404 handling.

## Response Formats

Use `respond_to` with `format.html`, `format.turbo_stream`, and `format.json` blocks for multi-format controllers.
