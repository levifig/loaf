---
id: ADR-001
title: Adopt Agent Skills Open Standard
status: Accepted
date: 2025-12-01
---

# ADR-001: Adopt Agent Skills Open Standard

## Decision

Loaf skills follow the [Agent Skills open standard](https://agentskills.io/specification) published by Anthropic and adopted by the AAIF (Linux Foundation).

## Context

Multiple skill/instruction formats competed in 2025: .cursorrules, AGENTS.md, custom formats. Anthropic published Agent Skills as an open standard in December 2025. Microsoft, OpenAI, Atlassian, Figma, Cursor, GitHub adopted it.

## Rationale

- Portability: one skill format works across Claude Code, Copilot, Codex, Cursor, Gemini, Windsurf, Zed
- Industry backing: AAIF unifies MCP, AGENTS.md, and Agent Skills under one standards body
- Ecosystem: skills.sh (57K+), SkillsMP (351K+) — growing marketplaces
- Future-proof: the standard will outlive any single tool

## Consequences

- SKILL.md is the required entrypoint for every skill
- Standard frontmatter fields only in SKILL.md (tool-specific fields go in sidecars)
- Description quality is critical (drives auto-invocation accuracy)
