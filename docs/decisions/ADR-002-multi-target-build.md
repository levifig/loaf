---
id: ADR-002
title: Multi-Target Build System
status: Accepted
date: 2025-12-01
---

# ADR-002: Multi-Target Build System

## Decision

Loaf compiles from a single source tree to 5+ target formats (Claude Code, Cursor, OpenCode, Codex, Gemini).

## Context

AI coding tools fragment across providers but skills should be portable. Writing separate skills for each tool is unsustainable.

## Rationale

- Single source of truth for all skill content
- Target-specific adaptations handled by sidecar files and build transformers
- New targets added by creating a new transformer, not rewriting skills
- Consistent quality across all platforms

## Consequences

- Build step required before distribution (`npm run build`)
- Sidecar files needed for tool-specific fields
- Build system is a critical path dependency
- Each target transformer must be maintained
