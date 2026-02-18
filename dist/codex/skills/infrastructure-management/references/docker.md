# Docker Patterns

## Contents
- Project Conventions
- Base Image Selection
- Checklist

Project container conventions and image selection guidance.

## Project Conventions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Build strategy | Multi-stage | Minimize image size and attack surface |
| Runtime user | `appuser` (UID 1000) | Matches K8s `runAsUser: 1000`, avoids volume permission issues |
| Health check | HTTP `/health` endpoint | Container orchestrator integration |
| Python env | `PYTHONUNBUFFERED=1` | Prevent stdout buffering in containers |
| Layer order | Dependencies before code | Maximize cache reuse |
| Secrets | Never baked in; BuildKit mounts or runtime env | Prevent leaks in image layers |
| Cleanup | Single `RUN` with `rm -rf /var/lib/apt/lists/*` | Reduce layer size |

## Base Image Selection

| Image | Size | Use Case |
|-------|------|----------|
| `scratch` | 0MB | Static Go binaries |
| `distroless` | ~2MB | Compiled languages |
| `alpine` | ~5MB | Need shell/packages |
| `*-slim` | ~50MB | Need glibc (Python, Ruby) |

Default: `python:3.12-slim` for Python services, `distroless` for Go binaries.

## Checklist

- [ ] Multi-stage build used
- [ ] Non-root user (UID 1000)
- [ ] Health check defined
- [ ] No secrets in image
- [ ] Specific version tags (no `latest`)
- [ ] .dockerignore present
- [ ] Layer order optimized
- [ ] apt lists cleaned up
- [ ] PYTHONUNBUFFERED=1 set
