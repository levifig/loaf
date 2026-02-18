# Observability

## Contents
- Logging Conventions
- Metrics Conventions
- Health Endpoints
- Alerting Principles

Project observability conventions. For language-specific tooling, see language skills.

## Logging Conventions

See [code-style.md](./code-style.md) for structured logging principles.

| Level | Use For |
|-------|---------|
| ERROR | Actionable failures requiring attention |
| WARN | Degraded state, potential issues |
| INFO | Business events, state transitions |
| DEBUG | Development troubleshooting (off in prod) |

| Always | Never |
|--------|-------|
| Structured JSON in production | Log sensitive data (PII, credentials, tokens) |
| Correlation/request IDs | String interpolation for log messages |
| Log at service boundaries | Log inside tight loops |
| Relevant context (user_id, resource_id) | Entire request/response bodies |
| Rate-limit repetitive messages | DEBUG level in production |

## Metrics Conventions

**RED Method** (request-driven services): Rate, Errors, Duration (p50, p95, p99)

**USE Method** (resources): Utilization, Saturation, Errors

| Always | Never |
|--------|-------|
| Consistent naming conventions | High-cardinality labels |
| Units in metric names (_seconds, _bytes) | Unbounded label values (user IDs, URLs) |
| Expose p50, p95, p99 for latencies | Only averages for latency |

## Health Endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| /health | Liveness probe | 200 if process running |
| /ready | Readiness probe | 200 if can serve traffic |
| /metrics | Prometheus scrape | Metrics in exposition format |

- **Liveness:** Minimal check -- process is alive, not deadlocked
- **Readiness:** Dependencies available, can handle requests; include dependency health
- Return structured JSON with component status

## Alerting Principles

- Alert on symptoms, not causes
- Every alert must be actionable
- Include runbook links in alert messages

| Severity | Response | Example |
|----------|----------|---------|
| Critical | Immediate page | Service down, data loss risk |
| High | Within 1 hour | Significant degradation |
| Medium | Within 1 day | Elevated errors, approaching limits |
| Low | Next sprint | Minor anomalies |

Language-specific tooling: see `python-development`, `typescript-development`, `ruby-development` skills.
