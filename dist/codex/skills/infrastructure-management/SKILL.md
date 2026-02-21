---
name: infrastructure-management
description: 'Covers Docker, Kubernetes, GitOps, CI/CD pipelines, and container security.'
version: 1.17.2
---

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

## Topics

| Topic | Reference File | Use When |
|-------|----------------|----------|
| Docker | `references/docker.md` | Writing Dockerfiles, optimizing builds, adding health checks |
| Kubernetes | `references/kubernetes.md` | Creating deployments, services, probes, resource limits |
| GitOps | `references/gitops.md` | Setting up ArgoCD, Kustomize, sync policies |
| CI/CD | `references/ci-cd.md` | Building GitHub Actions workflows, caching, secrets |
| Troubleshooting | `references/troubleshooting.md` | Debugging CI failures, version conflicts, cache issues |

## Available Scripts

| Script | Usage | Description |
|--------|-------|-------------|
| `scripts/check-dockerfile.sh` | `check-dockerfile.sh <file>` | Validate Dockerfile best practices |
| `scripts/validate-k8s-manifest.py` | `validate-k8s-manifest.py <file>` | Check K8s manifest for required fields |

## Critical Rules

### Always

- Use multi-stage builds to minimize image size
- Run containers as non-root user (UID 1000)
- Include health checks in all services
- Pin specific image versions (no `:latest`)
- Set resource requests AND limits
- Use `npm ci` / `pip-sync` in CI (not install)
- Commit lockfiles to version control

### Never

- Commit secrets to version control
- Use `:latest` tags in production
- Skip security scanning in CI
- Deploy without rollback capability
- Store state in containers
- Run as root in production

## CI Failure Triage

```
CI Failed
+-- Same code passes locally?
|   +-- YES --> Check environment differences
|   |   +-- Python/Node version
|   |   +-- Environment variables
|   |   +-- File permissions
|   |   +-- Installed dependencies
|   +-- NO --> Fix the actual bug
+-- Flaky (sometimes passes)?
|   +-- Check for race conditions, shared state, timeouts
+-- Always fails in CI?
    +-- Check runner resources (memory, timeout)
    +-- Check external service access
    +-- Check CI-specific config
```
