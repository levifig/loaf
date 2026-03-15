#!/usr/bin/env bash
# Linear MCP server wrapper: loads LINEAR_API_KEY (or 1Password ref op://...) and runs mcp-remote.
# Used by the Loaf plugin so the manifest does not contain bash substring syntax that
# Claude Code's MCP validator misparses as an env var (e.g. KEY#op://).

set -e
if command -v direnv >/dev/null 2>&1; then
  eval "$(direnv export bash 2>/dev/null)" || true
fi

KEY="${LINEAR_API_KEY:-}"
if [[ "$KEY" == op://* ]] && command -v op >/dev/null 2>&1; then
  KEY="$(op read "$KEY" 2>/dev/null)" || KEY=""
fi

if [ -n "$KEY" ]; then
  exec npx -y mcp-remote https://mcp.linear.app/mcp --header "Authorization: Bearer $KEY"
else
  exec npx -y mcp-remote https://mcp.linear.app/mcp
fi
