# GitOps Patterns

## Contents
- Project GitOps Stack
- Repository Structure
- Deployment Strategies
- Sync Policies
- Checklist

GitOps conventions with ArgoCD and Kustomize.

## Project GitOps Stack

| Component | Tool | Notes |
|-----------|------|-------|
| GitOps controller | ArgoCD | Declarative, self-healing |
| Templating | Kustomize (primary), Helm (third-party charts) | Kustomize for app overlays, Helm for vendor charts |
| Progressive delivery | Argo Rollouts | Blue/green and canary strategies |
| Secrets | sealed-secrets or external-secrets | Never plain-text in Git |
| Multi-cluster | ApplicationSets | Template-driven per-cluster |

## Repository Structure

```
gitops-repo/
+-- apps/
|   +-- base/                # Base configurations
|   +-- overlays/            # Environment-specific
|       +-- dev/
|       +-- prod/
+-- clusters/                # Cluster configs
+-- infrastructure/          # Shared components
```

Environments are Kustomize overlays, not separate branches.

## Deployment Strategies

| Strategy | Use Case | Config |
|----------|----------|--------|
| Blue/Green | Full cutover with preview validation | `autoPromotionEnabled: false` |
| Canary | Gradual rollout with metrics gates | Step weights: 10% -> 50% -> 100% |
| Rolling Update | Default K8s, no Argo Rollouts needed | `maxUnavailable: 25%` |

Key conventions:
- Use sync waves (`argocd.argoproj.io/sync-wave`) for dependency ordering
- Use resource hooks (`argocd.argoproj.io/hook: PreSync`) for migrations
- Canary analysis templates gate promotion on success-rate metrics

## Sync Policies

| Policy | Use Case |
|--------|----------|
| `prune: true` | Remove resources not in Git |
| `selfHeal: true` | Revert manual cluster changes |
| `ApplyOutOfSyncOnly=true` | Sync only changed resources |
| `CreateNamespace=true` | Auto-create target namespace |

Default: automated sync with prune + selfHeal enabled.

## Checklist

- [ ] Git repo as single source of truth
- [ ] Environment-specific overlays
- [ ] Automated sync enabled
- [ ] Secrets encrypted (sealed-secrets)
- [ ] Sync waves for dependencies
- [ ] Rollback strategy defined
