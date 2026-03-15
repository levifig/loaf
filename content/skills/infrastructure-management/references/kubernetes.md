# Kubernetes Patterns

## Contents
- Project Conventions
- Resource Sizing
- Health Probes
- Labeling
- Checklist

Project K8s deployment conventions and sizing guidance.

## Project Conventions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Security context | `runAsNonRoot: true`, UID 1000 | Matches Docker image user |
| Privilege escalation | `allowPrivilegeEscalation: false` | Least privilege |
| Root filesystem | `readOnlyRootFilesystem: true` | Prevent runtime modifications |
| Capabilities | `drop: [ALL]` | Remove all Linux capabilities |
| Image tags | Commit SHA or semver, never `latest` | Deterministic deploys |
| Service type | `ClusterIP` default; `LoadBalancer` for external | Minimize exposure |
| Secrets | External-secrets or sealed-secrets, never in ConfigMaps | Separate concerns |

## Resource Sizing

| Workload Type | Memory Request | Memory Limit | CPU Request | CPU Limit |
|---------------|----------------|--------------|-------------|-----------|
| API Service | 256Mi | 512Mi | 100m | 500m |
| Background Worker | 512Mi | 1Gi | 200m | 1000m |
| Batch Job | 1Gi | 2Gi | 500m | 2000m |

QoS precedence: Guaranteed (requests==limits) > Burstable (requests<limits) > BestEffort (none).

HPA defaults: `minReplicas: 2`, `maxReplicas: 10`, target CPU utilization 70%.

## Health Probes

| Probe | Purpose | Endpoint | Timing |
|-------|---------|----------|--------|
| Liveness | Restart if fails | `/health` | initial: 10s, period: 30s, failures: 3 |
| Readiness | Remove from service if fails | `/ready` | initial: 5s, period: 10s |
| Startup | For slow-starting apps | `/health` | failures: 30, period: 5s (150s max) |

## Labeling

Standard labels for all resources:

| Label | Purpose |
|-------|---------|
| `app.kubernetes.io/name` | Application name |
| `app.kubernetes.io/component` | Component within app (api, worker) |
| `app.kubernetes.io/version` | Semver or SHA |

Standard annotations: `prometheus.io/scrape: "true"`, `prometheus.io/port`.

## Checklist

- [ ] Security context configured
- [ ] runAsNonRoot: true
- [ ] Resource requests AND limits set
- [ ] Liveness and readiness probes defined
- [ ] Secrets not in ConfigMaps
- [ ] No `latest` image tags
- [ ] Labels follow Kubernetes conventions
