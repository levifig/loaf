---
name: ruby-development
description: >-
  Covers Ruby and Rails 8+ development following the DHH/37signals way. Includes
  Ruby idioms, Hotwire, Solid Queue, Minitest, gem development, and CLI tools.
  Use when building Rails apps, writing background jobs, or when the user asks
  "how do I add Hotwire?" or "what's the Rails convention for X?"
version: 1.16.0
---

# Ruby Development

The DHH/37signals way: convention over configuration, programmer happiness, and elegant simplicity.

## Philosophy

### The Rails Doctrine

- **Optimize for programmer happiness** - Ruby's beauty isn't accidental
- **Convention over configuration** - Sensible defaults, deviate only with reason
- **The menu is omakase** - Trust the chef's curated choices
- **No one paradigm** - Mix OOP, functional, metaprogramming as needed
- **Exalt beautiful code** - Code is read more than written
- **Provide sharp knives** - Power tools for capable developers
- **Value integrated systems** - The majestic monolith over distributed chaos
- **Progress over stability** - Embrace change, don't fear upgrades
- **Push up a big tent** - Welcome diverse approaches

### Calm Development

From "It Doesn't Have to Be Crazy at Work":
- Work sustainably, not heroically
- 40-hour weeks produce better code than crunch
- Meetings are a last resort
- Deep work requires long, uninterrupted stretches

## Ruby Idioms

### Expressive Code

```ruby
# Prefer readable over clever
users.select(&:active?).map(&:email)

# Guard clauses over nested conditionals
def process(user)
  return unless user.active?
  return if user.banned?

  # Happy path at normal indentation
  send_notification(user)
end

# Implicit returns
def full_name
  "#{first_name} #{last_name}"
end

# Trailing conditionals for simple cases
send_email if user.subscribed?
raise ArgumentError, "Invalid" unless valid?
```

### Duck Typing

```ruby
# Check behavior, not class
def export(target)
  target.write(data) if target.respond_to?(:write)
end

# Accept anything that quacks like a duck
class Report
  def initialize(formatter)
    @formatter = formatter  # Any object with #format method
  end
end
```

### Blocks and Iterators

```ruby
# Blocks for configuration
Rails.application.configure do |config|
  config.cache_store = :memory_store
end

# Blocks for resource management
File.open("data.txt") do |file|
  process(file.read)
end  # File automatically closed

# Custom iterators
def each_with_status
  items.each_with_index do |item, i|
    yield item, "#{i + 1}/#{items.size}"
  end
end
```

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

| Use Case | Tool | Notes |
|----------|------|-------|
| Microservices | Sinatra, Roda | When Rails is overkill |
| API-only | Grape | Lightweight REST APIs |
| Full-stack alternative | Hanami | Clean architecture |
| CLI tools | Thor, TTY | Command-line interfaces |
| Gem development | Bundler | Standard tooling |
| Background jobs | Sidekiq | When you need Redis anyway |

## Quick Reference

```bash
# Rails 8 app
rails new myapp --database=postgresql

# Sinatra microservice
bundle gem my_service --no-test

# CLI tool
bundle gem my_cli --no-test
# Add thor to gemspec

# Generators (Rails)
bin/rails g model User name:string email:string
bin/rails g controller Posts index show
bin/rails g authentication

# Development
bin/dev                    # Start with Procfile.dev
bin/rails test             # Run tests
bin/rails db:migrate       # Run migrations

# Deployment (Kamal)
kamal setup && kamal deploy
```

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
| Assets | [assets.md](references/assets.md) | Configuring Propshaft, Import Maps |
| Mobile | [mobile.md](references/mobile.md) | Building Hotwire Native, Bridge Components |
| Debugging | [debugging.md](references/debugging.md) | Using binding.irb, byebug, Rails console |

## Critical Rules

### Always

- Follow Ruby/Rails naming conventions exactly
- Write expressive, readable code over clever code
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

## File Organization

### Rails

```
app/
├── controllers/     # HTTP request handling
├── models/          # Business logic and data
├── views/           # Templates and partials
├── services/        # Complex operations
├── jobs/            # Background processing
├── javascript/      # Stimulus controllers
│   └── controllers/
└── assets/
    └── stylesheets/ # Tailwind CSS
```

### Gem

```
my_gem/
├── lib/
│   ├── my_gem.rb         # Entry point
│   └── my_gem/
│       ├── version.rb
│       └── ...
├── test/                  # Minitest
├── my_gem.gemspec
└── Gemfile
```

## Decision Guide

| Need | Solution |
|------|----------|
| Web application | Rails 8 |
| CRUD operations | RESTful controllers |
| Dynamic updates | Turbo Frames/Streams |
| Complex business logic | Service objects |
| Background processing | Solid Queue jobs |
| External API | API controller + token auth |
| Mobile app | Hotwire Native |
| Microservice | Sinatra or Roda |
| CLI tool | Thor + TTY |
| Shared library | Gem |
