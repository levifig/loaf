# Rails Views

## Conventions

- Views display data only — no business logic
- Pass locals explicitly to partials (not instance variables)
- Use `dom_id(record)` for element IDs
- Collection rendering: `<%= render @posts %>`

## Logic Extraction

| Bad | Good |
|-----|------|
| `<% if current_user && current_user.admin? %>` | `<% if admin? %>` (helper) |
| `<%= post.created_at.strftime("%B %d") %>` | `<%= format_date(post.created_at) %>` |
| `<%= Post.where(published: true).count %>` | `<%= @published_count %>` (controller) |

For complex display logic, use presenters (SimpleDelegator subclass).
