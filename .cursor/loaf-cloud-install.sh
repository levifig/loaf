#!/usr/bin/env bash
# Cursor Cloud Agent install script (LOAF-78).
# Builds the Loaf CLI for this host, then installs harness surfaces into the
# project environment so hooks/skills exist before the agent starts.
# Project attach token (LOAF-67): set LOAF_CLIENT_TOKEN from
# `loaf auth link --project` in the Cursor Cloud project environment; optional
# LOAF_SYNC_URL for sync endpoint.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64) arch=x64 ;;
  aarch64|arm64) arch=arm64 ;;
esac
NATIVE="$ROOT/bin/native/${os}-${arch}/loaf"

# bin/loaf is the host platform's binary; require both it and the native
# binary for this host before skipping the build (cloud agents are typically
# linux-*, not darwin). The build is Go only; there is no npm.
if [[ ! -x "$ROOT/bin/loaf" || ! -x "$NATIVE" ]]; then
  go run ./cmd/loafdev build-go
fi

export PATH="$ROOT/bin:$PATH"
export LOAF_PROJECT_ENV=1
loaf install --to cursor --yes

# Development toolchain provisioning for the Cloud Agent VM.
# The default Cursor image ships Go and Node but not the TypeScript compiler,
# which `make verify-generated` / `make ci-check` require to validate emitted
# plugin TypeScript. Provision tsc idempotently onto a PATH-visible location:
# prefer a user-writable npm prefix (npm's own Node dir, already on PATH), and
# fall back to a system install only if that fails.
if command -v npm >/dev/null 2>&1 && ! command -v tsc >/dev/null 2>&1; then
  node_prefix="$(dirname "$(dirname "$(command -v npm)")")"
  npm install -g --prefix "$node_prefix" typescript >/dev/null 2>&1 \
    || sudo env "PATH=$PATH" npm install -g typescript >/dev/null 2>&1 \
    || echo "loaf-cloud-install: could not provision tsc; 'make verify-generated' will be unavailable" >&2
fi

# Cap Go's parallel test/build actions on the constrained Cloud VM (typically
# 4 vCPU). The full `go test ./...` suite mixes CPU-bound pure-Go (WASM) SQLite
# work with Node-shelling capability tests; the default parallelism (one binary
# per core, each with GOMAXPROCS internal parallelism) oversubscribes the cores
# and intermittently fails a Node-pipe capability test. Persisting -p=2 in the
# user go env makes `make test` / `make verify-local` reliable here without
# editing the Makefile or mutating shell profiles.
if command -v go >/dev/null 2>&1; then
  current_goflags="$(go env GOFLAGS)"
  case " $current_goflags " in
    *" -p="*) : ;;                       # respect an existing operator override
    *)
      if [[ -n "$current_goflags" ]]; then
        go env -w GOFLAGS="$current_goflags -p=2"
      else
        go env -w GOFLAGS="-p=2"
      fi
      ;;
  esac
fi
