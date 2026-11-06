---
topics:
  - cloud
  - compatibility
  - synchronization
covers:
  - internal/state/path.go
  - internal/auth/store.go
  - internal/auth/provision.go
  - internal/cli/attach_gate.go
  - internal/cli/attach_run.go
  - internal/cli/cloud_environment_bootstrap.go
  - internal/cli/cloud_environment_test.go
  - .agents/setup
  - .agents/resume
  - .cursor/environment.json
  - .cursor/loaf-cloud-install.sh
  - .cursor/loaf-cloud-start.sh
last_reviewed: 2026-11-06
---

# Cloud and CI Attach Walkthrough

> **Compatibility guide:** This walkthrough exercises the shipped synchronization surface. The accepted private-continuity protocol has pre-cutover code and tests but no public destination attach, network sync, or serve journey. See [Private Synchronization](../architecture/private-synchronization.md) and the [Private Sync Threat Model](../security/private-sync-threat-model.md).

## Contents

- Overview
- Prerequisites
- Environment variables
- Cursor Cloud Agents
- Amp Orbs
- CI (GitHub Actions)
- Failure modes
- Verification

This walkthrough covers preparing an ephemeral host with no Loaf installation or attach state, then minting a project-scoped client credential and attaching it on Cursor Cloud, Amp Orbs, or CI. Existing Orb snapshots can retain prior installation and auth state; a fresh clone alone does not prove a fresh machine environment.

## Overview

Three layers stack on cloud hosts:

1. **Bootstrap** — this repository's install scripts build the Loaf CLI and run `loaf install --to <harness>` under `LOAF_PROJECT_ENV=1` so skills, commands, and hooks resolve from the project checkout (`.cursor/`, `.amp/`, `.agents/skills/`). External Amp consumers use the separate pinned release bootstrap linked below.
2. **Secrets** — per-project environment secrets carry the bundled client credential minted by `loaf auth link`. Never paste the operator master key or account admin secret into cloud or CI secrets.
3. **Attach** — the start script (or an explicit CI step) runs `loaf attach`, which reads `.agents/loaf.conf`, `LOAF_CLIENT_TOKEN`, and optionally `LOAF_SYNC_URL`, then completes the unattended ceremony: HTTPS health → auth → pull → decrypt probe → capability check → HLC skew check → identity match.

When attach enforcement is active and the environment is unattached, every substrate-touching `loaf` command exits nonzero with a machine-readable refusal. SessionStart journal context hooks fail closed the same way. Exempt commands: `--version`, help, and the auth/attach/install surface needed to become attached.

Enforcement becomes active when the machine-local auth directory contains an admin wire, client wire, or attach-state file, or when `LOAF_CLIENT_TOKEN` is nonempty. A wholly unconfigured environment is not gated. Successful attach persists the client wire and attached state, so unsetting the environment token afterward does not detach that environment.

## Prerequisites

Complete these steps once on a trusted operator machine before configuring cloud or CI environments.

### 1. Run a sync relay

Self-host the sync server (`loaf serve`) or use your deployed relay. The endpoint must be HTTPS in production (TLS-terminated reverse proxy is fine).

```bash
loaf serve --listen :8080 --db /path/to/sync.sqlite
```

### 2. Create the substrate account

```bash
loaf auth setup --endpoint https://sync.example.com --server-db /path/to/sync.sqlite
```

Store the Emergency Kit offline. The admin wire stays on the operator machine only.

### 3. Mint a project-scoped client credential

From the project checkout (`.agents/loaf.conf` must exist with `project_id`):

```bash
loaf auth link cursor:myproject --project "$(jq -r .project_id .agents/loaf.conf)"
```

Copy the one-line bundled client wire printed after `auth link ok`. This string is the value for `LOAF_CLIENT_TOKEN` in cloud and CI secrets.

Connection names are arbitrary but should encode harness and project for audit (`cursor:loaf`, `amp:loaf`, `ci:loaf`).

## Environment variables

| Variable | Required | Surface | Purpose |
|----------|----------|---------|---------|
| `LOAF_CLIENT_TOKEN` | Yes (cloud/CI attach) | All | Bundled client credential from `loaf auth link` |
| `LOAF_PROJECT_ENV` | Yes (cloud harness install) | Cursor Cloud, Amp | Routes `loaf install` to project-local `.cursor/`, `.amp/`, `.agents/skills/` |
| `LOAF_SYNC_URL` | No | All | Override sync endpoint embedded in the bundled credential |
| `LOAF_CONNECTION_NAME` | No | All | Connection name recorded in attach state (defaults to token id) |

Bootstrap scripts in this repository export `LOAF_PROJECT_ENV=1`. Cursor Cloud does not carry install-time shell exports into agent sessions, so the start script re-exports it. Prefer also setting `LOAF_PROJECT_ENV=1` as a durable project environment variable in the harness UI when available.

## Cursor Cloud Agents

Keeping the four bootstrap scripts is an explicit exception for this repository's current Cursor Cloud / Amp Orb contract. Local tests that require those paths do not prove the hosts require Bash.

This repository ships Cursor Cloud bootstrap artifacts:

| File | Role |
|------|------|
| `.cursor/environment.json` | Points install/start scripts |
| `.cursor/loaf-cloud-install.sh` | Builds CLI, runs `loaf install --to cursor` |
| `.cursor/loaf-cloud-start.sh` | Re-exports `LOAF_PROJECT_ENV`, pre-warms attach when token present |

### Operator setup

1. Open the Cursor Cloud project settings for this repository.
2. Add **project environment variables** (secrets):
   - `LOAF_CLIENT_TOKEN` — the bundled wire from `loaf auth link`
   - `LOAF_PROJECT_ENV` — `1` (recommended as a durable var, not only via start script)
   - `LOAF_SYNC_URL` — optional override if the relay URL differs from the credential
3. Confirm `.cursor/environment.json` references the install and start scripts (committed in-repo).

### What happens on each agent session

**Install/build phase** (`.cursor/loaf-cloud-install.sh`):

```bash
make build-cli   # when native binary missing for host
export PATH="$ROOT/bin:$PATH"
export LOAF_PROJECT_ENV=1
loaf install --to cursor --yes
```

**Start phase** (`.cursor/loaf-cloud-start.sh`):

```bash
export PATH="$ROOT/bin:$PATH"
export LOAF_PROJECT_ENV=1
if [[ -n "${LOAF_CLIENT_TOKEN:-}" ]]; then
  loaf attach
fi
```

After attach succeeds, private-continuity commands such as `loaf journal recent` can proceed, subject to their normal state checks. If attach enforcement is active and attach has not succeeded, substrate commands and SessionStart context emission refuse. A missing token on a wholly unconfigured host does not activate that gate; an explicit `loaf attach` still reports the missing credential.

### Manual verification on a cloud agent

```bash
loaf attach --json          # should exit 0 when secrets are correct
loaf journal recent         # should succeed after attach
loaf journal context --from-hook  # SessionStart path; refuses when enforcement is active and unattached
```

## Amp Orbs

The commands below describe this Loaf source repository's development bootstrap. For DojoHQ or another external Loaf-enabled consumer with no Loaf checkout or Go toolchain, first install and prove the pinned release with [Amp Orb Consumer Bootstrap](../cloud/amp-orb-consumer-bootstrap.md). Private continuity attachment remains a separate step after that readiness proof.

Amp uses `.agents/setup` (fresh orb) and `.agents/resume` (wake) instead of Cursor's environment.json.

| File | Role |
|------|------|
| `.agents/setup` | Builds CLI, runs `loaf install --to amp` |
| `.agents/resume` | Re-runs project-environment install on wake, then attaches when `LOAF_CLIENT_TOKEN` is configured |

### Operator setup

1. In Amp project configuration, add **project secrets / environment variables**:
   - `LOAF_CLIENT_TOKEN` — bundled wire from `loaf auth link`
   - `LOAF_PROJECT_ENV` — `1`
   - `LOAF_SYNC_URL` — optional
2. The committed `.agents/resume` already runs `loaf attach` after installation when `LOAF_CLIENT_TOKEN` is configured. After fresh setup, run attach explicitly before commands that require the attached environment.

### What happens on orb prepare

**Setup** (`.agents/setup`):

```bash
go run ./cmd/loafdev build-go   # when bin/loaf or the host native binary is missing
export PATH="$ROOT/bin:$PATH"
export LOAF_PROJECT_ENV=1
loaf install --to amp --yes
```

**Resume** (`.agents/resume`):

```bash
export PATH="$ROOT/bin:$PATH"
export LOAF_PROJECT_ENV=1
loaf install --to amp --yes
if [[ -n "${LOAF_CLIENT_TOKEN:-}" ]]; then
  loaf attach
fi
```

After fresh setup, run attach when the secret is present; resume already performs this step:

```bash
loaf attach
```

## CI (GitHub Actions)

CI runners are ephemeral: no attach state persists between jobs. Each job that touches the substrate must install Loaf, export secrets, and attach before substrate commands.

### Repository secrets

Add these in **Settings → Secrets and variables → Actions**:

| Secret | Value |
|--------|-------|
| `LOAF_CLIENT_TOKEN` | Bundled wire from `loaf auth link` (use a `ci:<repo>` connection name) |
| `LOAF_SYNC_URL` | Optional relay override |

Do not commit credentials. Do not use the operator master key in CI.

### Example workflow fragment

```yaml
jobs:
  substrate-check:
    runs-on: ubuntu-latest
    env:
      LOAF_PROJECT_ENV: "1"
      LOAF_CLIENT_TOKEN: ${{ secrets.LOAF_CLIENT_TOKEN }}
      LOAF_SYNC_URL: ${{ secrets.LOAF_SYNC_URL }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build Loaf CLI
        run: |
          make build-cli
          echo "$PWD/bin" >> "$GITHUB_PATH"

      - name: Attach to substrate
        run: loaf attach --json

      - name: Run substrate command
        run: loaf journal recent --limit 5
```

Notes:

- `LOAF_PROJECT_ENV=1` keeps any in-job `loaf install` on project-local paths if needed.
- Attach runs every job; there is no persistent attach state on the runner.
- Use `loaf attach --json` in CI so failures emit structured refusals in logs.
- Jobs that only run `go test` against isolated temp state (`LOAF_DB` in tests) do not need attach unless they invoke substrate-touching CLI paths.

### CI attach failures

Common causes:

- **Missing secret** — the explicit attach step reports a missing credential. Later substrate commands refuse only when auth enforcement is active and the environment is still unattached; do not use an unconfigured host's successful local command as proof of attach.
- **Wrong project scope** — credential `project_id` does not match `.agents/loaf.conf`; attach exits with `attach-identity-mismatch`.
- **Reach/auth** — relay unreachable or token revoked; attach exits with `attach-reach-failed` or `attach-auth-failed`.

## Failure modes

Attach refusals are machine-readable. Use `--json` for automation.

| Code | Cause | Remedy |
|------|-------|--------|
| `attach-conf-missing` | No `.agents/loaf.conf` | Ensure conf ships with the clone |
| `attach-credential-missing` | `LOAF_CLIENT_TOKEN` empty | Add project secret from `loaf auth link` |
| `attach-credential-invalid` | Malformed bundled wire | Re-mint with `loaf auth link` |
| `attach-identity-mismatch` | Conf `project_id` ≠ credential scope | Mint a project-scoped token for this checkout |
| `attach-endpoint-insecure` | Non-HTTPS endpoint | Use TLS or set `LOAF_SYNC_URL` to HTTPS relay |
| `attach-reach-failed` | Relay health check failed | Verify relay URL and network egress |
| `attach-auth-failed` | Token rejected or revoked | Re-mint or un-revoke the connection |
| `attach-hlc-skew` | Clock grossly skewed from stream | Fix host NTP; retry attach |
| `environment-unattached` | Enforcement active, substrate command before successful attach | Run `loaf attach` or fix secrets |

## Verification

**Operator machine (local):**

```bash
loaf auth list                              # connection visible with last-seen after cloud attach
go test ./internal/cli/... -run CloudEnvironment -count=1
```

**Cloud / CI (after secrets configured):**

```bash
loaf attach --json                          # exit 0
loaf journal recent                         # exit 0 (substrate reachable)
```

**Unattached refusal (isolated negative check):**

Use a disposable auth home with an explicit unattached state. The subshell isolates both auth and SQLite paths, contains no real credentials, and makes no network request. It leaves the fixture under the printed temporary directory for inspection; it does not remove or modify live auth state.

```bash
(
  attach_probe=$(mktemp -d) || exit 1
  export XDG_DATA_HOME="$attach_probe/data"
  export LOAF_DB="$attach_probe/loaf.sqlite"
  unset LOAF_CLIENT_TOKEN
  mkdir -p "$XDG_DATA_HOME/loaf/auth" || exit 1
  printf '%s\n' '{"attached":false}' > "$XDG_DATA_HOME/loaf/auth/attach.json" || exit 1
  cd "$attach_probe" || exit 1
  if loaf journal recent --json 2>"$attach_probe/refusal.json"; then
    printf '%s\n' 'Expected an unattached refusal.' >&2
    exit 1
  else
    probe_status=$?
  fi
  test "$probe_status" -eq 1 || exit 1
  jq -e '.code == "environment-unattached"' "$attach_probe/refusal.json" || exit 1
  printf 'Fixture: %s\n' "$attach_probe"
)
```

This proves the configured-unattached refusal path only. It does not prove relay reachability, successful attachment, or the destination protocol's availability.

Harness bootstrap script markers are tested by `TestCloudEnvironmentBootstrapArtifactsInstallCLI` in `internal/cli/cloud_environment_test.go`.
