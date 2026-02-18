# CI/CD Troubleshooting

## Contents
- Triage Decision Tree
- Common Failures
- Environment Differences
- Cache Issues
- Resolution Checklist

Debugging CI failures, environment drift, and cache problems.

## Triage Decision Tree

```
CI Failed
+-- Same code passes locally?
|   +-- YES --> Environment difference (version, env vars, permissions)
|   +-- NO --> Fix the actual bug
+-- Flaky (sometimes passes)?
|   +-- Race conditions, shared state, timeouts
+-- Always fails in CI?
    +-- Runner resources, external service access, CI-specific config
```

## Common Failures

| Symptom | Likely Cause | First Check |
|---------|--------------|-------------|
| Tests pass locally, fail in CI | Environment difference | Python/Node version, env vars |
| Linter passes locally, fails in CI | Version mismatch | Lock file, tool version |
| Random failures | Flaky test or race condition | Run tests multiple times |
| Timeout | Resource limits or hanging test | CI runner specs |
| Permission denied | File permissions or docker | Container user |

## Environment Differences

| Variable | Local | CI | Impact |
|----------|-------|-----|--------|
| `TZ` | Local timezone | UTC | Time-dependent tests |
| `PYTHONPATH` | Project root | Not set | Import errors |
| `HOME` | Your home | `/home/runner` | Config file paths |
| File system | Case-insensitive (macOS) | Case-sensitive (Linux) | Import casing |

Project CI sets: `TZ=UTC`, `PYTHONPATH=${{ github.workspace }}`, `PYTHONUNBUFFERED=1`, `NODE_ENV=test`.

## Cache Issues

| Problem | Solution |
|---------|----------|
| Stale cache | Add version suffix to key: `-v2` |
| Cache miss when expected hit | Check if hash input files changed |
| Wrong deps despite cache hit | Include all dep files in hash key |
| Cache evicted | 7-day TTL; force invalidation via `gh api` |

Force invalidation: `gh api -X DELETE repos/{owner}/{repo}/actions/caches/{cache_id}`

## Resolution Checklist

- [ ] Python/Node version matches
- [ ] All dependencies at same version
- [ ] Environment variables set
- [ ] Working directory correct
- [ ] File permissions correct
- [ ] Services running (database, etc.)
- [ ] Cache not stale
- [ ] Tests run in same order
- [ ] No local uncommitted changes affecting test
