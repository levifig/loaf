---
id: SPEC-016
title: "Council Advisory Redesign — dynamic specialists, structured output"
source: direct
created: 2026-03-27T20:50:19Z
status: drafting
appetite: "Medium (3-5 sessions)"
---

# SPEC-016: Council Advisory Redesign — dynamic specialists, structured output

## Problem Statement

The current council-session skill is coupled to Loaf's fixed agent roster (backend-dev, frontend-dev, dba, etc.), requires a PM agent to orchestrate, pre-synthesizes recommendations that hide dissenting views, and bundles advisory + decision-recording + implementation into one monolithic workflow. The user never used it via `/council-session` — they expect councils to be triggered naturally from conversation and to function as an **advisory board**, not a decision-making pipeline.

Key gaps:
- Specialists are limited to existing Loaf agents, which may not match the question's domain
- The user doesn't see individual specialist reasoning — only a synthesis
- The workflow pushes toward a decision and implementation, but the user wants to review and decide independently

## Strategic Alignment

- **Vision:** Fits "agent-creates, human-curates" — agents provide perspectives, human makes the call
- **Personas:** Serves the product owner who needs structured input from multiple domains to make informed decisions
- **Architecture:** Pure skill rewrite — no CLI changes, no new agents. The calling session acts as coordinator (no PM agent). Uses Agent tool for specialist spawning. Designed to work within whatever session model SPEC-014 establishes

## Solution Direction

### No PM Agent — Session as Coordinator

The calling session (Claude itself) acts as coordinator: frames the question, determines composition, spawns specialists, collects responses, and presents results. No separate PM agent is spawned. This aligns with the direction in SPEC-014 where skills replace agent delegation.

### Dynamic Specialist Profiles

When a council is triggered, Claude analyzes the question and creates **specialist profiles** tailored to the topic. Each profile is:

1. A persona with defined expertise and perspective
2. Backed by a relevant Loaf skill if one exists (e.g., a "Python Backend Specialist" loads `python-development`), or prompt-only for domains without a matching skill
3. Given a perspective bias (e.g., "prioritize maintainability", "prioritize performance") to ensure genuine diversity of viewpoint

Council size is **adaptive** — Claude determines 3-7 specialists based on the question's breadth and complexity. Present the proposed composition to the user before spawning.

### Product/UX Default Seat

The coordinator includes a **Product/UX specialist** by default when the question touches user experience, product direction, or feature scope. This specialist reads VISION.md and STRATEGY.md and argues from the user's perspective, not the system's — asking "should we?" not just "can we?"

**Skip condition:** When the question is purely technical, exploratory, and has no direct impact on UX or product direction (e.g., "which serialization format for internal RPC?"), the coordinator may omit the product/UX seat. The coordinator states this judgment when presenting the proposed composition — user can override.

### Uniform Question

Claude generates a **single, well-framed question** from the conversation context. Every specialist receives the identical question along with the same background context. No specialist gets privileged information.

**Strategic context always provided:** Every specialist receives VISION.md + STRATEGY.md + ARCHITECTURE.md alongside the framed question, regardless of the question's domain. This prevents "technically correct but strategically wrong" advice. Specialists choose how much weight to give strategic context, but they can't claim they didn't have it.

### Two-Round Deliberation

- **Round 1:** All specialists answer independently, in parallel. No cross-talk.
- **Round 2:** Each specialist sees all Round 1 positions and can revise their position, rebut others, or strengthen their argument. Run in parallel.

### Structured Per-Specialist Output

Each specialist produces:

| Field | Description |
|-------|-------------|
| **Take** | Adapts to question type: ✅/❌ for yes/no decisions, ranked pick for multi-option comparisons (e.g., "1st: X, 2nd: Y, avoid: Z"), position statement for open-ended questions. Always a forced commitment, no hedging. |
| **Confidence** | High / Medium / Low with brief justification |
| **Position** | Their recommended approach (1-2 sentences) |
| **Pros/Cons** | Per option under consideration |
| **Nuances** | Edge cases, hidden trade-offs, things others might miss |
| **Suggestions** | Alternative approaches or modifications |

Take + confidence are complementary: "My take is ❌ but I'm only Medium confident because..." or "1st: Postgres, 2nd: CockroachDB — High confidence" is more useful than either alone.

Output density is **adaptive** — concise (~200 words) for straightforward topics, detailed (~500 words) for complex ones. Claude judges based on question complexity.

### Advisory Output

The council produces:
1. The framed question
2. Each specialist's full structured response (Round 1 + Round 2 revisions)
3. A **convergence summary**: where specialists agree, where they diverge, and the key trade-offs
4. **No recommendation** — the council presents, the user decides

After the user has reviewed, suggest relevant next steps (ADR, implementation, further research) without auto-proceeding.

### Council File

Persist every council to `.agents/councils/` for future reference. Archive via `loaf task archive` conventions after the decision is captured.

## Scope

### In Scope

- Rewrite `council-session` SKILL.md with the advisory model
- Dynamic specialist profile generation (skill-backed when available, prompt-only otherwise)
- Two-round deliberation flow
- Structured per-specialist output format
- Updated council file template
- Model-invoked triggering (already done — `user-invocable: false`)

### Out of Scope

- CLI commands for council management
- Persistent specialist profiles across sessions
- Voting or scoring mechanisms
- Changes to other skills or agents
- Council history search or analytics

### Rabbit Holes

- **Specialist prompt engineering perfection** — good enough profiles that produce diverse views are fine; don't over-optimize the persona prompts
- **Round 2 convergence loops** — two rounds max, no iterating until consensus
- **Skill matching heuristics** — simple keyword/domain matching is sufficient; don't build a recommendation engine

### No-Gos

- Council must never auto-decide or auto-implement
- No filtering or ranking specialist output before showing to user
- Don't require odd numbers of specialists (that's a voting artifact — we're not voting)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Round 2 adds too much latency | Medium | Medium | Round 2 agents run in parallel; total is ~2x single round |
| Specialists echo each other (low diversity) | Low | High | Perspective bias in profiles forces divergent viewpoints |
| Skill-backed specialists over-index on skill content | Low | Medium | Prompt instructs specialist to use skill as background, not as script |

## Open Questions

- [x] Should the convergence summary be generated by a separate agent or by the orchestrating Claude? → By the coordinator (main session). No separate agent needed.
- [x] How should the skill reference existing Loaf skills for specialist backing — by name, by domain keyword, or by a mapping table? → By domain keyword matching. Simple, not a recommendation engine (per rabbit holes).

## Test Conditions

- [ ] "Call a council about X" triggers the skill from natural language
- [ ] Specialist composition is proposed and shown before spawning
- [ ] Each specialist receives the identical framed question
- [ ] Round 1 responses are shown individually with full structured output
- [ ] Round 2 revisions are shown per-specialist
- [ ] Convergence summary identifies agreement and disagreement points
- [ ] No recommendation is made — user is asked to decide
- [ ] Council file is persisted to `.agents/councils/`
- [ ] Next steps are suggested but not auto-executed
- [ ] Product/UX specialist included when question touches UX or product direction
- [ ] Product/UX specialist omitted when question is purely technical — coordinator states this judgment
- [ ] User can override the product/UX seat decision
- [ ] Each specialist provides a take (✅/❌, ranked pick, or position) alongside confidence
- [ ] All specialists receive VISION.md + STRATEGY.md + ARCHITECTURE.md context

## Circuit Breaker

At 50% appetite: If specialist profile generation or round structure proves complex, simplify to single-round with prompt-only specialists. Add round 2 and skill-backing as follow-up.

At 75% appetite: Ship single-round advisory model. Round 2 deliberation becomes a follow-up spec.
