---
name: ruby-development
description: >-
  Covers Ruby and Rails 8+ development: Hotwire, Solid Queue, Minitest, and
  Rails conventions.
version: 1.17.2
---

# Ruby Development

The DHH/37signals way: convention over configuration, programmer happiness, and elegant simplicity.

## Stack Overview

### Rails 8 (Default)

| Component | Default | Alternative |
|-----------|---------|-------------|
| Database | SQLite (dev/prod) | PostgreSQL |
| Background Jobs | Solid Queue | Sidekiq |
| Caching | Solid Cache | Redis |
| WebSockets | Solid Cable | Redis |
| Assets | Propshaft + Import Maps | esbuild |
| CSS | Tailwind (standalone) | Bootstrap |
| Deployment | Kamal | Capistrano |

### Beyond Rails

| Use Case | Tool |
|----------|------|
| Microservices | Sinatra, Roda |
| API-only | Grape |
| Full-stack alternative | Hanami |
| CLI tools | Thor, TTY |
| Gem development | Bundler |

## Topics

| Topic | Reference | Use When |
|-------|-----------|----------|
| Core | [core.md](references/core.md) | Writing idiomatic Ruby, organizing Rails files |
| Models | [models.md](references/models.md) | Creating models, validations, associations, migrations |
| Controllers | [controllers.md](references/controllers.md) | Building RESTful actions, strong params, filters |
| Views | [views.md](references/views.md) | Creating templates, partials, layouts, helpers |
| Hotwire | [hotwire.md](references/hotwire.md) | Adding Turbo Drive/Frames/Streams, Stimulus |
| API | [api.md](references/api.md) | Building JSON APIs, versioning, authentication |
| Jobs | [jobs.md](references/jobs.md) | Setting up Solid Queue, Active Job, recurring jobs |
| Security | [security.md](references/security.md) | Implementing authentication, authorization, CSRF |
| Testing | [testing.md](references/testing.md) | Writing Minitest, fixtures, system tests |
| Performance | [performance.md](references/performance.md) | Fixing N+1 queries, caching, indexing |
| Deployment | [deployment.md](references/deployment.md) | Deploying with Kamal, Docker, zero-downtime |
| Styling | [styling.md](references/styling.md) | Setting up Tailwind CSS, responsive design |
| Services | [services.md](references/services.md) | Creating service objects, result patterns |
| Mobile | [mobile.md](references/mobile.md) | Building Hotwire Native, Bridge Components |
| Debugging | [debugging.md](references/debugging.md) | Using binding.irb, byebug, Rails console |

## Critical Rules

### Always

- Follow Ruby/Rails naming conventions exactly
- Use generators (`bin/rails g`) for scaffolding
- Write tests first (TDD with Minitest)
- Keep controllers thin (under 10 lines per action)
- Return proper HTTP status codes
- Use strong parameters for all user input
- Index foreign keys in migrations

### Never

- Fight conventions without excellent reason
- Put business logic in controllers or views
- Skip model validations
- Use `render` after successful mutations (use `redirect_to`)
- Store secrets in code (use `credentials:edit`)
- Write custom CSS when Tailwind utilities exist
- Reach for microservices before the monolith fails
