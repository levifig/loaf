# CI/CD Patterns

## Contents
- Pipeline Stages
- Project CI Configuration
- Caching Strategies
- Secrets Management
- Checklist

Pipeline stage ordering, caching, secrets, and project CI conventions.

## Pipeline Stages

```
Lint --> Test --> Security --> Build --> Deploy
```

| Stage | Purpose | Fail Fast |
|-------|---------|-----------|
| Lint | Code style, formatting | Yes |
| Test | Unit, integration tests | Yes |
| Security | Vulnerability scanning | Yes |
| Build | Container image creation | No (main only) |
| Deploy | Environment deployment | No (manual gate) |

## Project CI Configuration

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CI Platform | GitHub Actions | Native GitHub integration |
| Concurrency | Cancel in-progress on same ref | Avoid wasted compute |
| Build trigger | Push to main/develop, PRs to main | Standard branch flow |
| Container registry | ghcr.io | Bundled with GitHub |
| Docker cache | GitHub Actions cache (`type=gha`) | No external cache service |
| Deploy mechanism | GitOps (commit image tag to gitops repo) | Auditable, declarative |

Key conventions:
- Lint and type-check run before tests (fail fast)
- Tests run with real database services (Postgres 17)
- Build step only runs on `main` branch
- Use `npm ci` (not `npm install`) for lockfile fidelity
- Pin tool versions exactly in lockfiles

## Caching Strategies

| What | Strategy |
|------|----------|
| Python deps | `actions/setup-python` with `cache: "pip"` |
| Node deps | `npm ci` with built-in cache |
| Docker layers | BuildKit GHA cache (`cache-from: type=gha`) |
| Pre-commit | `actions/cache` keyed on `.pre-commit-config.yaml` hash |

## Secrets Management

| Type | Use Case |
|------|----------|
| Repository secrets | Shared across workflows |
| Environment secrets | Per-environment (staging/prod) |
| Organization secrets | Shared across repos |

Secrets never appear in workflow files. Use `environment:` blocks for production gates.

## Checklist

- [ ] Lint and type check before tests
- [ ] Security scanning in pipeline
- [ ] Tests run with real database
- [ ] Coverage reports uploaded
- [ ] Docker caching enabled
- [ ] Secrets not in workflow files
- [ ] Concurrency control set
- [ ] Environment gates for production
