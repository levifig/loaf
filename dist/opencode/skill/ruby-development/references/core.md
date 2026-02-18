# Rails Core Principles

## Contents
- Rails 8 Defaults
- File Organization
- Credentials

## Rails 8 Defaults

| Component | Purpose |
|-----------|---------|
| Solid Queue | Database-backed job processing (no Redis) |
| Solid Cache | Database-backed caching |
| Solid Cable | Database-backed WebSockets |
| Propshaft | Simple asset pipeline (no compilation) |
| Import Maps | JavaScript without bundling (no Node.js) |
| Kamal | Zero-downtime Docker deployments |

## File Organization

```
app/
├── controllers/     # HTTP request handling
├── models/          # Business logic and data
├── views/           # Templates and partials
├── services/        # Complex multi-model operations
├── jobs/            # Background processing (Solid Queue)
├── mailers/         # Email sending
├── helpers/         # View helpers
└── javascript/
    └── controllers/ # Stimulus controllers
```

## Architectural Principles

- **Database-first**: Active Record at center, database constraints enforce integrity
- **RESTful by default**: Resources map to routes, 7 standard actions
- **HTML over the wire**: Hotwire/Turbo for SPA-like speed without JS complexity
- **No external dependencies**: Solid Queue/Cache/Cable replace Redis

## Credentials

```bash
EDITOR="code --wait" bin/rails credentials:edit
EDITOR="code --wait" bin/rails credentials:edit --environment production
```

Access: `Rails.application.credentials.dig(:stripe, :secret_key)`
