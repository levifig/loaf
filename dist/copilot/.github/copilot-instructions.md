# Copilot Instructions

This file provides context and guidelines for GitHub Copilot to generate better code suggestions.

## Project Overview

This project uses the following technologies and patterns. Please follow these guidelines when generating code.

## Roles and Responsibilities

When working on different parts of the codebase, consider these specialized perspectives:

### backend-dev
Backend services developer. Detects stack and loads appropriate skill (Python, Rails, or TypeScript backend).

### dba
Database administrator for schema design, migrations, SQL optimization, and database architecture. Use for table changes, indexes, and query optimization.

### design
UI/UX designer and accessibility auditor. Use for design systems, component design, accessibility compliance, and user experience.

### devops
DevOps engineer for Docker, Kubernetes, CI/CD, and infrastructure. Use for containerization, deployment pipelines, and infrastructure changes.

### frontend-dev
Frontend developer for React, Next.js, and TypeScript. Builds accessible, performant user interfaces.

### pm
Project manager for orchestrating multi-agent work. Use when coordinating complex tasks, managing sessions, running councils, or delegating to specialized agents.

### qa
Quality assurance engineer for testing, code review, and security audits. Use for unit tests, integration tests, code review, and security checks.

## Technology Guidelines

### Design

# Design Principles

**Philosophy**: Design is problem-solving through systematic thinking, with accessibility and user needs at the center. Every design decision must be intentional, accessible, and meaningful.

## Quick Reference

| Topic | Key Principle | Reference |
|-------|--------------|-----------|
| Core | User-first, accessible by default, consistent, minimal | [reference/core.md](reference/core.md) |
| Color | Function first, 4.5:1 contrast minimum, never color-only | [reference/color.md](reference/color.md) |
| Typography | Legible, accessible, reinforce hierarchy | [reference/typography.md](reference/typography.md) |
| Spacing | 8pt grid, consistent rhythm, logical properties | [reference/spacing.md](reference/spacing.md) |
| Responsive | Mobile-first, content-out, fluid layouts | [reference/responsive.md](reference/responsive.md) |
| Accessibility | WCAG 2.1 AA minimum, keyboard navigable, screen reader compatible | [reference/a11y.md](reference/a11y.md) |
| Motion | Purposeful, natural, respect reduced motion | [reference/motion.md](reference/motion.md) |
| Systems | Tokens, components, patterns, governance | [reference/systems.md](reference/systems.md) |

## Core Principles

### 1. User-First
Every design decision must serve user needs and goals.
- Understand user context: Who, what, why, when, where?
- Validate assumptions with real users
- Measure outcomes that matter to users

### 2. Accessible by Default
Accessibility is not a feature - it is a requirement.
- WCAG 2.1 AA minimum, AAA as aspiration

...(see full documentation)

### Foundations

# Code Standards

Engineering foundations for consistent, secure, and well-documented code.

## Philosophy

**Code speaks first.** Well-structured code with clear names needs fewer comments. When comments are necessary, they explain WHY, not WHAT. Documentation lives close to code but separate from implementation details.

**Commits tell a story.** Each commit represents one coherent change. Messages use imperative mood and focus on intent. The git log should read like a narrative of the project's evolution.

**Security by default.** Every input is untrusted. Every error is generic to users. Every secret is externalized. Defense in depth, not security theater.

## Quick Reference

| Standard | Key Rule | Example |
|----------|----------|---------|
| **Code Style** | Type hints required, structured logging | `async def fetch(id: UUID) -> Result` |
| **Commits** | `<type>: <description>`, imperative mood | `feat: add thermal rating API` |
| **Documentation** | Document after shipping, not before | API docs reflect implemented endpoints only |
| **Security** | Validate all inputs at trust boundaries | Pydantic models at API layer |

## Topics

| Topic | Reference | Use For |
|-------|-----------|---------|
| Code Style | `reference/code-style.md` | Python/TypeScript conventions, naming, patterns |
| Commits | `reference/commits.md` | Commit messages, branches, PRs, Linear integration |
| Documentation | `reference/documentation.md` | ADRs, API docs, changelogs |
| Security | `reference/security.md` | Threat modeling, secrets, compliance |

...(see full documentation)

### Infrastructure

# Infrastructure

Infrastructure patterns for containerization, orchestration, CI/CD pipelines, and deployment automation.

## Stack Overview

| Layer | Technologies |
|-------|--------------|
| Containers | Docker, BuildKit, multi-stage builds |
| Orchestration | Kubernetes, Helm, Kustomize |
| GitOps | ArgoCD, Flux, Argo Rollouts |
| CI/CD | GitHub Actions, GitLab CI |
| Registries | GHCR, ECR, GCR, DockerHub |

## Philosophy

1. **Infrastructure as Code** - All configuration in version control
2. **GitOps** - Git as the single source of truth for deployments
3. **Security by Default** - Non-root, minimal images, no secrets in code
4. **Observability** - Health checks, probes, structured logging
5. **Reproducibility** - Pinned versions, lockfiles, deterministic builds

## Quick Reference

### Docker Essentials

```dockerfile
# Multi-stage build with non-root user
FROM python:3.12-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

FROM python:3.12-slim
RUN useradd --uid 1000 --create-home appuser
COPY --from=builder /install /usr/local
COPY --chown=appuser:appuser . .
USER appuser
HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost:8000/health || exit 1
ENTRYPOINT ["python", "-m", "app"]
```

### Kubernetes Pod Spec

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
  containers:
    - name: app
      image: app:v1.0.0  # Never :latest
      resources:
        requests: {memory: "256Mi", cpu: "100m"}
        limits: {memory: "512Mi", cpu: "500m"}

...(see full documentation)

### Orchestration

# PM Orchestration

Comprehensive patterns for PM-style orchestration: coordinating multi-agent work, managing sessions, running councils, delegating to specialized agents, and integrating with Linear.

## Philosophy

**PM is the orchestrator, not the implementer.**

The PM agent:
1. Creates issues and session files for tracking
2. Breaks down work into delegable tasks
3. Spawns specialized agents for implementation
4. Coordinates outcomes and updates external systems
5. Never implements code, tests, or documentation directly

Every release should be complete, polished, and delightful - no MVPs or quick hacks.

## Quick Reference

| Task | Action |
|------|--------|
| Multi-step work | Create session file, spawn agents |
| Complex decision | Convene council (5-7 agents, odd) |
| Linear update | Checkboxes, no emoji, no local paths |
| Feature planning | Now/Next/Later buckets, no dates |
| Agent selection | Match domain expertise to task |

## Topics

| Topic | Reference | Key Content |
|-------|-----------|-------------|
| Agent Delegation | [reference/delegation.md](reference/delegation.md) | Agent capabilities, spawn patterns, decision tree |
| Council Workflow | [reference/councils.md](reference/councils.md) | Composition, deliberation, synthesis, user approval |
| Session Management | [reference/sessions.md](reference/sessions.md) | Lifecycle, file format, handoffs, validation |
| Linear Integration | [reference/linear.md](reference/linear.md) | Update format, magic words, status conventions |
| Product Planning | [reference/planning.md](reference/planning.md) | Roadmaps, feature specs, acceptance criteria |

...(see full documentation)

### Python

# Python Development

Comprehensive guide for modern Python 3.12+ development with FastAPI ecosystem.

## When to Use This Skill

- Starting or configuring Python projects
- Building REST APIs with FastAPI
- Defining data models with Pydantic
- Implementing async/await patterns
- Adding type hints and mypy configuration
- Writing tests with pytest
- Working with databases (SQLAlchemy 2.0)
- Building data pipelines with Polars
- Integrating external APIs
- Deploying to production with Docker

## Stack Overview

| Layer | Default | Alternatives |
|-------|---------|--------------|
| Runtime | Python 3.12+ | - |
| Package Manager | uv | rye, poetry |
| Linter/Formatter | ruff | black + flake8 |
| Type Checker | mypy | pyright |
| Web Framework | FastAPI | Flask, Django |
| Validation | Pydantic v2 | - |
| ORM | SQLAlchemy 2.0 | - |
| Data Processing | Polars | Pandas |
| HTTP Client | httpx | aiohttp |
| Testing | pytest | - |
| Containerization | Docker | - |

## Core Philosophy

- **Explicit is better than implicit** — clear, obvious code
- **Simple is better than complex** — avoid unnecessary abstractions
- **Readability counts** — code is read more than written
- **Async by default** — non-blocking I/O for web services
- **Strict typing** — catch errors at development time
- **12-factor methodology** — environment-based configuration

## Quick Reference

### Project Setup with uv

```bash
uv init my-project --python 3.12
uv add fastapi uvicorn pydantic-settings
uv add --dev pytest ruff mypy
uv run pytest
```

### pyproject.toml Essentials

...(see full documentation)

### Rails

# Rails 8+ Development

Complete guide for building modern Rails applications following The Rails Way.

## Rails 8 Stack

| Component | Default | Alternative |
|-----------|---------|-------------|
| Database | SQLite (dev/prod) | PostgreSQL |
| Background Jobs | Solid Queue | Sidekiq |
| Caching | Solid Cache | Redis |
| WebSockets | Solid Cable | Redis |
| Assets | Propshaft + Import Maps | esbuild |
| CSS | Tailwind (standalone) | Bootstrap |
| Deployment | Kamal | Capistrano |

## Core Philosophy

- **Convention over configuration** - Follow Rails defaults
- **Embrace the monolith** - Most apps don't need microservices
- **Server-side rendering** - HTML over the wire via Hotwire
- **No build step** - Import Maps + Propshaft, no Node.js
- **Database-backed everything** - Solid Queue/Cache/Cable use your database
- **Test with fixtures** - Minitest + fixtures, not RSpec + factories

## Quick Reference

```bash
# New Rails 8 app
rails new myapp --database=postgresql

# Generators
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

| Topic | Reference | Key Patterns |
|-------|-----------|--------------|
| Core | [core.md](reference/core.md) | Framework principles, file organization, generators |
| Models | [models.md](reference/models.md) | Active Record, validations, associations, migrations |

...(see full documentation)

### Typescript

# TypeScript Development

Comprehensive guide for modern TypeScript development with React ecosystem.

## When to Use This Skill

- Starting or configuring TypeScript projects
- Building React components and applications
- Working with Next.js 14+ and App Router
- Implementing state management patterns
- Building forms with validation
- Integrating APIs and handling data fetching
- Writing tests for TypeScript applications
- Styling with Tailwind and CVA
- Optimizing application performance
- Implementing accessibility (a11y)
- Building React Native mobile apps
- Working with modern JavaScript (ESM)

## Stack Overview

| Layer | Default | Alternatives |
|-------|---------|--------------|
| Language | TypeScript 5+ | JavaScript (ESM) |
| Runtime | Node.js 20+ | Bun, Deno |
| Framework | Next.js 14+ | Vite, Remix |
| UI Library | React 18+ | - |
| State (Client) | Zustand | Context + Reducer |
| State (Server) | React Query | SWR |
| Forms | React Hook Form + Zod | - |
| Styling | Tailwind CSS + CVA | CSS Modules |
| Testing | Vitest + RTL | Jest |
| E2E Testing | Playwright | Cypress |
| Package Manager | pnpm | npm, yarn |

## Core Philosophy

- **Strict mode always** - catch errors at compile time
- **Server Components by default** - use Client Components only when needed
- **Type inference** - let TypeScript infer when obvious
- **Server state is different** - use React Query for API data
- **Accessibility is mandatory** - not optional
- **Mobile-first responsive** - design for small screens first
- **Measure before optimizing** - profile, then fix

...(see full documentation)

## Quality Checks (Manual)

Since Copilot doesn't support automated hooks, ensure you run these checks:

### Before Committing

- Check for hardcoded secrets before writing
- Validate changelog format
- Check code formatting
- Run security audit on bash commands
- Validate commit messages follow conventions
- Detect Linear magic words for auto-status updates
- Run mypy type checking
- Validate pytest configuration
- Progressive type checking
- Run ruff linting
- Run pytest tests
- Run bandit security scan
- Run TypeScript compiler check
- Analyze bundle size impact
- Check migration safety

### After Making Changes

- Run ruff after changes
- Run ESLint after changes
- Run RuboCop after changes
- Validate accessibility after changes
- Check design token usage
- Full accessibility audit

### General Guidelines

- Always add type hints/annotations
- Write tests for new functionality
- Follow existing code style
- Validate all user input
- Handle errors appropriately
- Keep functions small and focused
