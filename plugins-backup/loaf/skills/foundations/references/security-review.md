# Security Review Checklist

## Contents
- Quick Security Check
- Authentication and Access Control
- Injection Prevention
- Configuration and Deployment
- API and Frontend Security
- Secrets Management

Security-focused review checklist for PRs and code changes.

## Quick Security Check

For every PR:

```
[ ] No hardcoded secrets, keys, or passwords
[ ] No sensitive data in logs
[ ] Input validation present at boundaries
[ ] SQL/NoSQL queries are parameterized
```

## Authentication and Access Control

```
[ ] Authorization checked on every endpoint
[ ] Default deny -- explicit allow
[ ] CORS configured restrictively
[ ] Directory traversal prevented (no user input in file paths)
[ ] Rate limiting on sensitive operations
[ ] JWT/session tokens validated server-side
[ ] Strong password requirements enforced
[ ] Account lockout after failed attempts
```

## Injection Prevention

```
[ ] SQL queries parameterized (ORM or bound parameters)
[ ] OS commands use subprocess with list args (never shell=True with user input)
[ ] XML parsing disables external entities
[ ] NoSQL queries use safe APIs
```

## Configuration and Deployment

```
[ ] Debug mode disabled in production
[ ] Default credentials changed
[ ] Error messages don't leak internals
[ ] Security headers configured (X-Content-Type-Options, X-Frame-Options, HSTS, CSP)
[ ] Dependencies pinned to specific versions
[ ] Dependency scanning in CI
[ ] Unnecessary features disabled
```

## API and Frontend Security

**API:**
```
[ ] Authentication on all non-public endpoints
[ ] Rate limiting configured
[ ] Request size limits
[ ] Content-Type validation
```

**Frontend:**
```
[ ] XSS prevention (output encoding)
[ ] CSP headers configured
[ ] Cookies: HttpOnly, Secure, SameSite
[ ] No sensitive data in localStorage
```

## Secrets Management

```
[ ] No secrets in code or version control
[ ] No secrets in logs
[ ] Secrets rotated regularly
[ ] Access to secrets audited
[ ] Secrets in environment variables or vault
```
