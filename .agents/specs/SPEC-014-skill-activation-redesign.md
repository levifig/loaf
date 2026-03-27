---
id: SPEC-014
title: Skill Activation Redesign — foundations decomposition + agent elimination
source: brainstorm
created: '2026-03-24T23:30:00.000Z'
status: drafting
appetite: Large (4+ sessions)
---

# SPEC-014: Skill Activation Redesign

## Problem Statement

Loaf's knowledge activation has two structural problems:

1. **The foundations skill is too broad to trigger precisely.** Its description ("Establishes code quality, commit conventions, documentation standards, and security patterns") covers 17 reference files spanning git workflow, debugging, security, code review, and documentation. When the context is "merge this PR," the skill's description doesn't match — and the squash merge conventions in `commits.md` don't load. Concrete incident: 2026-03-24, squash merge conventions missed during PR merge.

2. **Role-based agents add indirection without value.** All 8 role agents (pm, backend-dev, frontend-dev, qa, dba, design, devops, power-systems) are thin routers with 2-sentence system prompts that say "Your skills tell you how." Skills contain all domain knowledge. The routing layer adds nothing that description-based skill activation doesn't already provide. Meanwhile, some targets (Codex, Gemini) don't support agents at all and work fine with skills alone.

These problems compound: if skills are the only knowledge layer that works across all targets, they need precise descriptions for context-based activation. Broad skills and role-agent routing both work against this.

## Strategic Alignment

- **Vision:** Loaf as a portable agentic harness — skills-first architecture works across all targets, not just Claude Code
- **Architecture:** Follows the CLI-driven harness direction (SPEC-010 absorbed PM's task management). Skills are the universal knowledge layer; agents are infrastructure, not identity.
- **Best practices alignment:** Claude Platform, agentskills.io, Cursor, and Codex all recommend focused, well-described skills as the primary activation mechanism.

## Solution Direction

### Track A: Foundations Decomposition

Split the monolithic `foundations` skill into 4-5 focused skills, each with a precise description optimized for context-based triggering. Reference file content stays the same — this is reorganization, not rewriting.

**Proposed split:**

| New Skill | References (moved from foundations) | Description Focus |
|---|---|---|
| **git-workflow** | `commits.md` | Branching strategy, commit conventions, PR creation, squash merge workflow |
| **debugging** | `debugging.md`, `hypothesis-tracking.md`, `test-debugging.md` | Systematic debugging, hypothesis tracking, flaky test investigation |
| **security-compliance** | `security.md`, `security-review.md` | Threat modeling, secrets management, compliance checks |
| **documentation-standards** | `documentation.md`, `documentation-review.md`, `diagrams.md` | ADRs, API docs, changelogs, Mermaid diagrams |
| **foundations** (slimmed) | `code-style.md`, `tdd.md`, `verification.md`, `code-review.md`, `review.md`, `permissions.md`, `observability.md`, `production-readiness.md` | Code style, naming conventions, TDD, verification, review discipline |

Each new skill gets:
- `SKILL.md` with precise description (action verb opener, user-intent phrases, negative routing)
- `SKILL.claude-code.yaml` sidecar (`user-invocable: false`, no `agent:` field)
- `references/` directory with moved files
- Relevant hooks reassigned via `skill:` field in `config/hooks.yaml`

### Track B: Agent Elimination

Remove all 8 role-based agents. Keep 2 system agents (background-runner, context-archiver). Don't create replacement agents — built-in agents (Explore, Plan, general-purpose) cover isolation needs.

**What happens to each agent:**

| Agent | Action | Rationale |
|---|---|---|
| pm | **Remove** | Orchestration skill becomes always-on. Task management is CLI. |
| backend-dev | **Remove** | Language skills activate by context. |
| frontend-dev | **Remove** | Language skills activate by context. |
| qa | **Remove** | Testing discipline is in foundations. |
| dba | **Remove** | database-design skill activates by context. |
| design | **Remove** | interface-design skill activates by context. |
| devops | **Remove** | infrastructure-management skill activates by context. |
| power-systems | **Remove** | power-systems-modeling skill activates by context. |
| background-runner | **Keep** | Real procedural system prompt for async execution. |
| context-archiver | **Keep** | Real procedural system prompt for session preservation. |

**Agent reference updates in skills:**

Workflow skills that currently reference `{{AGENT:backend-dev}}` etc. will be updated to use natural-language subagent instructions. The current `{{AGENT:slug}}` substitution is purely cosmetic — it replaces with plain-text agent names at build time, not special syntax. Post-build, all targets see plain English.

After this change, skills use direct language: "Spawn a subagent for this implementation task with [relevant skills] preloaded." Claude Code interprets this via its Agent tool. Other targets either support subagents natively (Cursor) or don't (Codex, Gemini — the skill works inline in the main conversation). No build-time translation is needed because the instruction is the same everywhere — the harness decides how to execute it.

The `{{AGENT:...}}` substitution pattern and agent sidecar files (`.claude-code.yaml`, `.cursor.yaml`, `.opencode.yaml` per agent) are removed entirely.

**Council session redesign:**

The council-session skill currently spawns role agents for "diverse perspectives." After agent elimination, it will dynamically construct ephemeral pseudo-agents based on the discussion topic:

1. Analyze the topic and determine relevant perspectives (not roles — perspectives)
2. Construct per-perspective agent definitions with focused system prompts and relevant skills preloaded
3. Spawn each as a subagent, collect perspectives, synthesize

This produces better deliberation because perspectives are topic-specific ("security implications of this auth change") rather than generic role-play ("QA agent reviews").

### Track C: Description Audit

Audit and rewrite descriptions for all ~24 skills (reference and workflow) to optimize triggering. Descriptions are the *only* thing loaded at startup — Claude sees all skill descriptions in its system prompt and uses them to decide which skills to load. For `user-invocable: false` skills, the description is the entire triggering mechanism.

**Description formula (100-300 chars, max 1024):**

1. **Third-person action verb opener:** "Covers...", "Establishes...", "Coordinates..."
2. **Specific scope terms:** Include key terms Claude will match against ("commit messages", "PR creation", "squash merge")
3. **User-intent phrases:** "Use when creating PRs, writing commit messages, or merging branches"
4. **Negative routing:** "Not for code style (use foundations) or security checks (use security-compliance)" — critical for disambiguation between skills that could overlap
5. **Success criteria** (workflow skills only): "Produces a bounded spec with test conditions and circuit breaker"

**Example (git-workflow):**
```yaml
description: >-
  Covers branching strategy, commit message conventions, PR creation,
  and squash merge workflow. Use when creating branches, writing commit
  messages, creating or merging pull requests, or managing git history.
  Not for code style (use foundations) or CI/CD pipelines (use
  infrastructure-management).
```

### Track D: Build System + Config Cleanup

**Build system:**
- Remove `{{AGENT:...}}` substitution pattern from `cli/lib/build/lib/substitutions.ts` (`buildAgentMap()`, `substituteAgentNames()`)
- Remove agent discovery (`discoverAgents()`) from target transformers — or scope it to system agents only
- Remove agent sidecar files (`content/agents/*.{target}.yaml` for all 8 role agents)
- Keep system agent processing for background-runner and context-archiver (Claude Code only)
- Clean up agent-specific types in `cli/lib/build/types.ts`
- Remove agent output generation from claude-code.ts, cursor.ts, opencode.ts

**Config cleanup:**
- Remove `plugin-groups` from `config/hooks.yaml` entirely — they are legacy dead code (the build system ignores them, confirmed in code comments: "Legacy plugin groupings kept for documentation purposes"). Skill discovery is via filesystem scan; hook assignment is via each hook's `skill:` field.
- Update hook `skill:` fields to reference new skill names where appropriate (e.g., `check-secrets` → `skill: security-compliance`)
- Remove `agent:` field from all skill sidecar files (`.claude-code.yaml`)

## Scope

### In Scope

- Split foundations into 4-5 focused skills (new SKILL.md, sidecars, moved references)
- Slim down foundations to code quality core
- Delete 8 role-agent files from `content/agents/`
- Update all workflow skills that reference `{{AGENT:...}}` patterns
- Redesign council-session skill for dynamic perspective-based pseudo-agents
- Make orchestration skill universally active (not gated behind PM agent)
- Audit and rewrite descriptions for all ~24 skills
- Remove `plugin-groups` section from `config/hooks.yaml` (legacy dead code)
- Update hook `skill:` fields to reference new skill names
- Remove `agent:` field from all skill sidecars
- Remove agent-processing code from build target transformers
- Remove `{{AGENT:...}}` substitution pattern and `buildAgentMap()`/`substituteAgentNames()`
- Remove agent sidecar files for role agents
- Update `docs/ARCHITECTURE.md` to reflect new model
- Update `.claude/CLAUDE.md` / `.agents/AGENTS.md` if they reference agents

### Out of Scope

- Creating new persistent custom agents to replace role agents
- Rewriting reference file content (only reorganization)
- Changes to the CLI tool itself (commands, types, etc.)
- Skill evaluation framework or activation analytics (brainstorm spark)
- Changes to system agents (background-runner, context-archiver)
- Cross-harness testing beyond build output verification (manual testing on Cursor, Codex, etc.)

### Rabbit Holes

- Trying to build a "modes" or "presets" system — Claude Code doesn't have this primitive. Use skills + ephemeral agents instead.
- Over-engineering dynamic agent construction with project detection and smart skill selection — start simple (explicit skill lists in workflow skills), iterate later.
- Rewriting reference file content to match new skill boundaries — move files, don't rewrite them.
- Optimizing for targets that don't support agents (Codex, Gemini) beyond ensuring skills work — they already work.

### No-Gos

- Don't create persistent custom agents to replace role agents — the whole point is eliminating the routing layer
- Don't embed domain knowledge in agent system prompts — skills are the knowledge layer
- Don't break cross-harness build output — all targets must still build cleanly
- Don't change skill content beyond descriptions and organization — reference material stays intact

## Dependencies

| Dependency | Type | Status | Notes |
|---|---|---|---|
| None | — | — | This spec has no hard dependencies. It touches content, config, and build code but doesn't require new CLI features or external systems. |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Skill descriptions don't trigger as precisely as expected | Medium | Medium | Test with real prompts before and after. Iterate descriptions based on observed activation. |
| Council session redesign is more complex than expected | Medium | Low | Circuit breaker: ship with council flagged as "needs manual perspective specification" |
| Build system cleanup breaks a target | Low | High | Run `loaf build` for all targets after every change. Diff output against pre-change baseline. |
| Some workflow skills depend on agent behavior we haven't identified | Low | Medium | Grep for all `{{AGENT:` patterns and `agent:` sidecar fields. Map every reference before removing. |
| Orchestration skill is too noisy when always-active | Low | Medium | Review orchestration description — ensure it activates on task/session management contexts, not every conversation. |

## Resolved Questions

- **Plugin-groups:** Remove entirely. They're legacy dead code — the build system ignores them (confirmed in `hooks.yaml` comments and build target code). Skill discovery is via filesystem scan; hook assignment is via each hook's `skill:` field.
- **Hook reassignment:** Update hook `skill:` fields to reference the most relevant new skill (e.g., `check-secrets` → `security-compliance`, `validate-push` → `git-workflow`). This is organizational metadata only — hooks fire by `matcher:`, not by skill group. But correct attribution keeps config understandable.
- **Description length:** 100-300 chars is the sweet spot. Include scope terms, user-intent phrases, and negative routing. Max 1024 per spec. See Track C for the formula.

## Open Questions

- [ ] Should the foundations `scripts/` directory (check-commit-msg.sh, check-secrets.sh, etc.) be redistributed to new skills, or kept centrally? Scripts are referenced from hooks, not from SKILL.md.
- [ ] The `implement` skill currently delegates to agents for parallel execution. Without role agents, should it use built-in `general-purpose` for isolation, or construct ephemeral agents with skills preloaded? (Leaning toward natural-language spawning — let Claude decide based on context.)
- [ ] Should `orchestration` gain a `disable-model-invocation: false` + `user-invocable: false` combo to ensure it's always available to Claude but never manually invoked? Currently it's a workflow skill (`user-invocable: true`).

## Test Conditions

- [ ] `loaf build` succeeds for all 5 targets (claude-code, cursor, opencode, codex, gemini) with no agent-related output except system agents
- [ ] `npm run typecheck` passes
- [ ] `npm run test` passes
- [ ] No `{{AGENT:` patterns remain in any built output
- [ ] Each new skill from the foundations split has: SKILL.md with description, sidecar, references/ directory, registration in hooks.yaml
- [ ] Foundations skill's reference table no longer includes moved references
- [ ] All 8 role-agent files are deleted from `content/agents/`
- [ ] Workflow skills (implement, orchestration, council-session) work without role agents — spawning general-purpose or ephemeral agents for isolation
- [ ] Council-session skill can construct perspective-based pseudo-agents dynamically
- [ ] Orchestration skill loads universally (not gated behind PM agent)
- [ ] Every skill (reference and workflow) has an audited description with action verb opener and user-intent phrases
- [ ] `docs/ARCHITECTURE.md` reflects the skills-first model
- [ ] A prompt like "merge this PR" in a Loaf project activates the git-workflow skill (not just foundations)
- [ ] A prompt like "debug this flaky test" activates the debugging skill

## Circuit Breaker

**At 50% appetite:** Ship foundations decomposition (Track A) + description audit for new skills only. Agents still exist but are flagged for removal. Build system untouched.

**At 75% appetite:** Ship foundations decomposition + agent elimination (Tracks A + B) + build cleanup (Track D). Description audit limited to reference skills. Council redesign deferred.

**At 100% appetite:** Full spec: all four tracks including comprehensive description audit and council redesign.
