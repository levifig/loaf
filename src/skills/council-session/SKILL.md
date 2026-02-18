---
name: council-session
description: >-
  Convenes multi-agent council deliberations for complex decisions requiring diverse
  perspectives. Covers agent composition selection, parallel perspective gathering,
  synthesis, and decision recording. Use when a decision needs input from 5-7 specialized
  agents, or when the user asks "let's get a council opinion" or "I need multiple
  perspectives on this." Produces council files with agent perspectives, synthesis, and
  recorded decisions. Not for single-agent work or implementation (use implement).
---

# Council Deliberation Session

You are the {{AGENT:pm}} agent convening a council of specialized agents for multi-perspective deliberation.

## Contents
- Step 1: Parse Decision Topic
- Step 2: Determine Council Composition
- Step 3: Create Council File
- Step 4: Spawn Council Agents
- Step 5: Collect and Synthesize
- Step 6: Present to User
- Step 7: Record Decision
- Step 8: Consider ADR
- Step 9: Implement Decision
- Quality Checklist
- Remember

**Input**: `$ARGUMENTS` (the decision topic or question)

---

## Step 1: Parse Decision Topic

Extract: **what** needs deciding, **why** a council (vs single agent), **context/constraints**, known **options**.

If unclear, **ask user for clarification** before proceeding.

---

## Step 2: Determine Council Composition

Select **5-7 agents** (MUST be odd number). Reference `council-workflow/reference/council-composition.md`.

Present composition to user with rationale per agent. **Wait for explicit approval** before proceeding.

---

## Step 3: Create Council File

Generate timestamps, then create council file following [council template](templates/council.md).

---

## Step 4: Spawn Council Agents

**CRITICAL**: Spawn ALL agents in a **single response** (parallel).

Each agent gets:
- The decision question and options
- Relevant context
- Domain-specific analysis factors
- Instruction to focus on THEIR domain expertise only

---

## Step 5: Collect and Synthesize

After ALL agents respond:

1. Document each perspective in council file (recommendation, rationale, concerns)
2. Create synthesis: consensus points, key disagreements, trade-off analysis per option, overall recommendation with confidence level

Reference `council-workflow/reference/synthesis-patterns.md` for techniques.

---

## Step 6: Present to User

Present synthesis with:
- Summary of deliberation
- Each option with pros/cons and recommendation status
- Core trade-offs between top options
- Council recommendation (or explicit deferral if genuinely ambiguous)
- Ask user to choose

**CRITICAL**: **Wait for explicit user response.** Don't proceed until user decides.

---

## Step 7: Record Decision

After user decides, update council file:
- Chosen option, decided by, timestamp, rationale
- Action items with agent assignments
- Update frontmatter status to `completed`

If session file exists, update with decision reference and council outcome summary.

After session includes outcome summary: archive council (status + `archived_at` + `archived_by` + move to `.agents/councils/archive/`, update `.agents/` references).

---

## Step 8: Consider ADR

If decision is architectural, ask user if an ADR should be created. If approved, create following format from `council-workflow/reference/decision-recording.md`.

---

## Step 9: Implement Decision

If decision requires implementation:
1. Propose implementation plan with agent assignments
2. **Wait for user approval**
3. Spawn implementation agents with council decision context

---

## Quality Checklist

Before presenting:
- [ ] All agents spawned in parallel
- [ ] All perspectives collected
- [ ] Synthesis identifies consensus and disagreements
- [ ] Recommendation (or deferral) provided
- [ ] Council file created and updated

After decision:
- [ ] Decision recorded in council file
- [ ] Session file updated with outcome summary
- [ ] Council archived after session summary exists
- [ ] ADR consideration addressed
- [ ] Implementation plan proposed (if applicable)

---

## Remember

- **Always odd number** of agents (5 or 7)
- **Parallel spawning** (all at once in single response)
- **User approval required** at composition and decision steps
- **PM coordinates, doesn't vote**
- **Councils advise, users decide**
- **Document everything** (council file, session file, ADR if architectural)

Reference `council-workflow` skill for detailed guidance.
