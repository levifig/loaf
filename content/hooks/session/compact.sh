#!/usr/bin/env bash

set -euo pipefail

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"

if ! command -v loaf >/dev/null 2>&1; then
  exit 0
fi

cd "$PROJECT_DIR"
loaf session log 'compact(session): context compaction triggered' >/dev/null 2>&1 || true
