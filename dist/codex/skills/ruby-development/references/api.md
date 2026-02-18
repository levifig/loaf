# Rails API

## Conventions

- Version from day one: `/api/v1` URL path versioning
- Base controller inherits `ActionController::API` (not `Base`)
- Token authentication with `ActionController::HttpAuthentication::Token`
- Centralized error handling: `rescue_from` in base controller

## Response Format

Consistent envelope: `{ data: ..., meta: { page, total_pages } }` for collections.

## HTTP Status Codes

| Action | Success | Failure |
|--------|---------|---------|
| GET | 200 OK | 404 Not Found |
| POST | 201 Created | 422 Unprocessable |
| PUT/PATCH | 200 OK | 422 Unprocessable |
| DELETE | 204 No Content | 404 Not Found |

## Rate Limiting

```ruby
# config/initializers/rack_attack.rb
Rack::Attack.throttle("api/ip", limit: 100, period: 1.minute) do |req|
  req.ip if req.path.start_with?("/api")
end
```

## Serialization

Use `as_json(only: [...], methods: [...])` for simple cases, jbuilder for complex responses.
