# Security Patterns

## Contents
- Security Mindset
- STRIDE Threat Model
- Secret Management
- Security Headers
- Container Security
- Critical Rules

Security mindset, threat modeling, and project security conventions.

## Security Mindset

For every feature, ask:

1. How could this be exploited?
2. What happens if input is malicious?
3. What if authenticated but not authorized?
4. What if the system is partially compromised?

## STRIDE Threat Model

| Threat | Definition | Example |
|--------|------------|---------|
| **S**poofing | Impersonating something/someone | Stolen API key |
| **T**ampering | Modifying data or code | SQL injection |
| **R**epudiation | Denying actions taken | Missing audit logs |
| **I**nfo Disclosure | Exposing information | Error stack traces |
| **D**enial of Service | Disrupting service | Resource exhaustion |
| **E**levation of Privilege | Gaining unauthorized access | Broken access control |

## Secret Management

### Principles

1. **Never in code** -- secrets don't belong in source
2. **Never in logs** -- redact secrets from logging
3. **Least privilege** -- only access what you need
4. **Rotation** -- rotatable without downtime
5. **Encryption** -- at rest and in transit

### Log Redaction Pattern

```python
import structlog

def redact_secrets(_, __, event_dict):
    sensitive_keys = {"password", "secret", "token", "key"}
    for key in list(event_dict.keys()):
        if any(s in key.lower() for s in sensitive_keys):
            event_dict[key] = "[REDACTED]"
    return event_dict
```

### Never Log

- Passwords and credentials
- API keys and tokens
- Personal identifiable information (PII)
- Session tokens
- Encryption keys

## Security Headers

```python
@app.middleware("http")
async def add_security_headers(request, call_next):
    response = await call_next(request)
    response.headers["X-Content-Type-Options"] = "nosniff"
    response.headers["X-Frame-Options"] = "DENY"
    response.headers["X-XSS-Protection"] = "1; mode=block"
    response.headers["Strict-Transport-Security"] = "max-age=31536000"
    response.headers["Content-Security-Policy"] = "default-src 'self'"
    return response
```

## Container Security

| Check | Requirement |
|-------|-------------|
| Non-root user | UID 1000 |
| Read-only root filesystem | Yes |
| Capabilities dropped | Yes |
| `allowPrivilegeEscalation` | `false` |
| Image vulnerability scanning | Required |
| Minimal base images | Updated |

## Critical Rules

**Always:** Validate input at trust boundaries, log security events (without secrets), fail securely (deny by default), encrypt in transit and at rest, use parameterized queries, assume breach.

**Never:** Trust external input without validation, log secrets/credentials/PII, use default credentials, expose detailed errors to users, store secrets in code or version control.
