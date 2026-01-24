---
description: 'Zoom out, brainstorm, assess project state, or evolve VISION'
version: 1.11.1
---

# Research Command

Step back from implementation to understand, explore, or brainstorm.

**Input:** $ARGUMENTS

---

## Mode Detection

Parse the input to determine research mode:

| Input Pattern | Mode | Action |
|---------------|------|--------|
| "project state" / "catch me up" / empty | State Assessment | Review docs, sessions, recent work |
| Topic description | Topic Exploration | Structured inquiry |
| "brainstorm" / "ideas for" | Brainstorming | Generate and refine options |
| "evolve vision" / "update vision" | Vision Evolution | Propose VISION changes |

---

## CRITICAL: Interview First

**Before any research, interview the user to understand:**

1. What specifically are you trying to understand?
2. What context do you already have?
3. What constraints should guide this research?
4. What decision will this inform?

Use `AskUserQuestion` to gather this context.

---

## Mode 1: Project State Assessment

**Trigger:** Empty input, "project state", "catch me up"

### Steps

1. **Read project documents:**
   - `docs/VISION.md` (if exists)
   - `docs/ARCHITECTURE.md` (if exists)
   - `docs/REQUIREMENTS.md` (if exists)

2. **Check recent sessions:**
   - List `.agents/sessions/*.md`
   - Read recent sessions for lessons learned
   - Note any unresolved issues or blockers

3. **Review recent commits:**
   ```bash
   git log --oneline -20
   ```

4. **Synthesize findings:**
   - Current position
   - Recent progress
   - Lessons learned
   - Open questions
   - Recommendations

5. **Present to user** with actionable next steps

---

## Mode 2: Topic Exploration

**Trigger:** Specific topic or question

### Steps

1. **Interview** to understand the question deeply

2. **Check project context first:**
   - Existing decisions in `docs/decisions/ADR-*.md`
   - Relevant sections in ARCHITECTURE.md
   - Previous session discussions

3. **Apply confidence hierarchy:**
   ```
   Project context > Context7 > Official docs > Community > Web
   ```

4. **For library/framework questions:**
   - Use Context7 MCP to get authoritative documentation
   - Cross-reference with official sources

5. **Synthesize findings:**
   - Summary with confidence level
   - Options with tradeoffs
   - Recommendation with rationale
   - Sources cited

6. **Present options** - Don't make the decision for the user

---

## Mode 3: Brainstorming

**Trigger:** "brainstorm", "ideas for", creative exploration

### Steps

1. **Interview to establish bounds:**
   - What's the ideal outcome?
   - What constraints exist?
   - What's been tried/rejected?

2. **Generate diverse options:**
   - Conventional approaches
   - Unconventional approaches
   - What if we had 10x resources?
   - What if we had 1/10 resources?
   - Contrarian view

3. **Filter through constraints:**
   - Technical feasibility
   - Time/resource budget
   - Alignment with VISION

4. **Refine promising directions:**
   - Shape the top 2-3 options
   - Identify risks and mitigations

5. **Present curated options** with pros/cons for decision

---

## Mode 4: Vision Evolution

**Trigger:** "evolve vision", "update vision", evidence of needed change

### Steps

1. **Gather evidence:**
   - What changed since last VISION update?
   - Session learnings that contradict VISION
   - External factors (market, technology, team)

2. **Read current VISION.md**

3. **Draft change proposal:**
   ```markdown
   ## VISION.md Change Proposal

   **Evidence:** [What prompted this]

   ### Current State
   [Quote relevant section]

   ### Proposed Change
   [New text]

   ### Rationale
   [Why this is warranted]

   ### Impact
   - [Downstream effects]
   ```

4. **Present proposal to user**

5. **WAIT FOR EXPLICIT APPROVAL**
   - Do NOT edit VISION.md without approval
   - User must explicitly confirm changes

6. **If approved:** Edit VISION.md with approved changes

---

## Output Format

### State Assessment

```markdown
## Project State Assessment

**Date:** [Generated timestamp]

### Current Position
[Summary]

### Recent Progress
- [Accomplishment 1]
- [Accomplishment 2]

### Lessons Learned
- [Insight 1]
- [Insight 2]

### Open Questions
- [Question 1]
- [Question 2]

### Recommendations
1. [Next step]
2. [Next step]
```

### Research Findings

```markdown
## Research: [Topic]

**Question:** [What we're investigating]

### Summary
[Answer with confidence level]

### Options

#### Option A: [Name]
- **Pros:** ...
- **Cons:** ...

#### Option B: [Name]
- **Pros:** ...
- **Cons:** ...

### Recommendation
[Which option and why]

### Sources
- [Source with confidence]
```

---

## Guardrails

1. **Always interview first** - Don't assume you understand the question
2. **Check project context** - Respect existing decisions
3. **Present options** - Don't make decisions for the user
4. **Cite sources** - With confidence levels
5. **Never edit VISION without approval** - Proposals only
6. **Time-bound research** - Don't go down rabbit holes indefinitely

---

## Context7 Usage

For library/framework questions, use Context7:

```
1. mcp__plugin_context7_context7__resolve-library-id
   - Find the library ID

2. mcp__plugin_context7_context7__query-docs
   - Query specific documentation
```

This provides authoritative, up-to-date documentation.

---

## Related Skills

- **research** - Full methodology for research modes
- **orchestration/product-development** - Where research fits in workflow
