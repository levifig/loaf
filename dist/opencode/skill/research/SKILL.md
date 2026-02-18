---
name: research
description: >-
  Conducts project assessment, topic investigation, brainstorming, and vision
  evolution. Covers assessing project state, exploring problem spaces,
  generating ideas, and evolving strategic direction. Use when stepping back to
  understand the big picture, or when the user asks "what's the current state?"
  or "help me think through this problem." Produces state assessments, research
  findings with ranked options, or vision change proposals. Not for multi-agent
  coordination, session management, or task delegation (use orchestration).
---

# Research

Patterns for zooming out, investigating topics, and evolving project direction.

## Contents
- Philosophy
- Quick Reference
- Input Parsing and Mode Detection
- Research Workflow
- Research Modes
- Confidence Hierarchy
- Interview Patterns
- Output Formats
- Integration with Workflow
- When to Research
- When NOT to Research
- Critical Rules
- Scripts
- Related Skills

**Input:** $ARGUMENTS

## Philosophy

**Research is about understanding before doing.**

Research activities:
1. Assess project state and lessons learned
2. Explore problem spaces with structured inquiry
3. Generate and refine ideas through brainstorming
4. Evolve strategic direction when evidence supports it

Research is NOT about:
- Implementing solutions
- Making tactical decisions
- Writing code or specifications

## Quick Reference

| Need | Action |
|------|--------|
| Understand project state | Review sessions, docs, recent changes |
| Explore a topic | Structured inquiry with confidence hierarchy |
| Generate ideas | Brainstorm with interview and synthesis |
| Evolve VISION | Propose changes with evidence |

## Input Parsing and Mode Detection

Parse `$ARGUMENTS` to determine mode:

| Input Pattern | Mode | Action |
|---------------|------|--------|
| "project state" / "catch me up" / empty | State Assessment | Review docs, sessions, recent work |
| Topic or question | Topic Investigation | Structured inquiry |
| "let's brainstorm" / "ideas for X" | Brainstorming | Interview and synthesis |
| "should we change direction?" / "update VISION" | Vision Evolution | Evidence-based proposal |

## Research Workflow

### State Assessment

**Trigger:** Empty input, "project state", "catch me up", "where are we"

#### Steps

1. **Read project documents:**
   - `docs/VISION.md` -- Current direction
   - `docs/STRATEGY.md` -- Personas, market, problem space
   - `docs/ARCHITECTURE.md` -- Technical constraints

2. **Check ideas and specs:**
   - `.agents/ideas/*.md` -- Pending ideas
   - `docs/specs/SPEC-*.md` -- Active specifications

3. **Review recent sessions:**
   - `.agents/sessions/*.md` -- Implementation history
   - Note lessons learned, open questions

4. **Check recent commits:**
   ```bash
   git log --oneline -20
   ```

5. **Synthesize findings** (see Output Formats below)

### Topic Investigation

**Trigger:** Specific topic or question

#### Steps

1. **Clarify the question:**

   Use `AskUserQuestion` to understand:
   - What specifically are you trying to understand?
   - What context do you already have?
   - What decision will this inform?

2. **Check project context first:**
   - Existing decisions in `docs/decisions/ADR-*.md`
   - Relevant sections in ARCHITECTURE.md
   - Previous session discussions

3. **Apply confidence hierarchy** (see section below)

4. **For library/framework questions:**
   - Use Context7 MCP if available
   - Cross-reference with official documentation
   - Check for version-specific information

5. **Synthesize findings** (see Output Formats below)

## Research Modes

### Mode 1: Project State Assessment

**Trigger:** "What's the current state?" / "Catch me up"

**Process:**
1. Read existing docs (VISION, ARCHITECTURE, REQUIREMENTS)
2. Check recent sessions for lessons learned
3. Review recent commits for context
4. Identify gaps, contradictions, or stale information
5. Summarize current state with recommendations

**Output:** State assessment with actionable insights

### Mode 2: Topic Exploration

**Trigger:** "Research X" / "How should we approach Y?"

**Process:**
1. Interview user to understand the question deeply
2. Apply confidence hierarchy for sources
3. Synthesize findings with tradeoffs
4. Present options with recommendations

**Output:** Research findings with options and recommendation

### Mode 3: Brainstorming

**Trigger:** "Let's brainstorm" / "Ideas for X"

**Process:**
1. Interview to understand constraints and goals
2. Generate diverse options (quantity over quality initially)
3. Filter through constraints
4. Refine promising directions
5. Present shaped options for decision

**Output:** Curated options with pros/cons

### Mode 4: Vision Evolution

**Trigger:** "Should we change direction?" / "Update VISION"

**Process:**
1. Gather evidence (sessions, feedback, market changes)
2. Identify what's changed since last VISION update
3. Propose specific changes with rationale
4. Get user approval before any edits

**Output:** VISION.md change proposal (not direct edit)

## Confidence Hierarchy

When researching, prioritize sources in this order:

```
1. Project context (highest confidence)
   - VISION.md, ARCHITECTURE.md, REQUIREMENTS.md
   - Session files and lessons learned
   - Existing codebase patterns

2. Authoritative documentation
   - Context7 for library docs
   - Official documentation sites
   - RFCs and specifications

3. Community knowledge
   - Stack Overflow (verified answers)
   - GitHub issues and discussions
   - Blog posts from recognized experts

4. General web (lowest confidence)
   - Search results
   - Forum posts
   - Unverified sources
```

**Rule:** Always check project context first. External research should supplement, not replace, understanding of existing decisions.

### Confidence Levels

Rate findings by confidence:

| Level | Meaning |
|-------|---------|
| **High** | Official docs, verified in project context, or tested |
| **Medium** | Authoritative source, consistent with multiple references |
| **Low** | Community source, single reference, or inference |

Always cite sources with confidence levels.

## Interview Patterns

### Before Any Research

Ask clarifying questions:

| Area | Questions |
|------|-----------|
| Scope | What specifically are you trying to understand? |
| Context | What do you already know? What have you tried? |
| Constraints | What constraints should guide the research? |
| Output | What decision will this research inform? |

### For Topic Exploration

```
1. What problem are we trying to solve?
2. What constraints exist (technical, time, team)?
3. What would success look like?
4. What have we already considered and rejected?
5. Are there non-obvious angles to explore?
```

### For Brainstorming

```
1. What's the ideal outcome if constraints didn't exist?
2. What's the minimum viable approach?
3. What approaches have we seen work elsewhere?
4. What would we do if we had 10x the resources? 1/10?
5. What's the contrarian view?
```

## Output Formats

### State Assessment

```markdown
## Project State Assessment

**Date:** YYYY-MM-DD

### Current Position
- [Summary of where the project stands]

### Strategic Context
- **Vision:** [Brief summary]
- **Key personas:** [Who we're building for]
- **Current focus:** [Active specs/work]

### Recent Progress
- [Key accomplishments from recent sessions]

### In Flight
| Spec/Task | Status | Notes |
|-----------|--------|-------|
| SPEC-001 | implementing | [progress] |
| SPEC-002 | approved | [next up] |

### Ideas Pipeline
- [Idea 1] -- raw
- [Idea 2] -- raw

### Lessons Learned (Recent)
- [Insights from implementation feedback]

### Open Questions
- [Unresolved decisions or gaps]

### Recommendations
1. [Actionable next step]
2. [Actionable next step]
```

### Research Findings

```markdown
## Research: [Topic]

**Question:** [What we're trying to understand]

### Summary
[One-paragraph answer with confidence level]

### Key Findings
1. [Finding 1]
2. [Finding 2]

### Project Context
[How this relates to our existing decisions/architecture]

### Options Considered

#### Option A: [Name]
- **Pros:** ...
- **Cons:** ...
- **Best for:** ...

#### Option B: [Name]
- **Pros:** ...
- **Cons:** ...
- **Best for:** ...

### Recommendation
[Which option and why, with caveats]

### Sources
- [Source 1 with confidence level]
- [Source 2 with confidence level]

### Implications
- [What this means for our work]

### Open Questions
- [What's still unclear]
```

### Vision Change Proposal

```markdown
## VISION.md Change Proposal

**Proposed by:** [Session/Date]
**Evidence:** [What prompted this]

### Current State
[Quote relevant VISION.md section]

### Proposed Change
[New text]

### Rationale
[Why this change is warranted]

### Impact
- [What this changes downstream]
- [ADRs that may need updating]
- [Requirements that may be affected]

### Approval Required
This proposal requires user approval before implementation.
```

## Research vs Other Skills

| Need | Skill |
|------|-------|
| Understand current state | `/research` |
| Investigate a factual question | `/research` |
| Generate ideas or options | `/brainstorm` |
| Explore a problem space | `/brainstorm` |
| Update strategy from learnings | `/reflect` |
| Make technical decisions | `/architecture` |

## Context7 Usage

For library/framework questions, use Context7 if available:

```
1. mcp__plugin_context7_context7__resolve-library-id
   - Find the library ID

2. mcp__plugin_context7_context7__query-docs
   - Query specific documentation
```

This provides authoritative, version-specific documentation.

## Integration with Workflow

Research fits at the top of the product development hierarchy:

```
RESEARCH -> VISION -> ARCHITECTURE -> REQUIREMENTS -> SPECS -> TASKS
    |         |
    +---------+
    evolves
```

**Research informs Vision changes.** Vision changes cascade down through Architecture, Requirements, and Specs.

## When to Research

| Situation | Research Mode |
|-----------|---------------|
| Starting new project | Project state + Topic exploration |
| Stuck on approach | Topic exploration + Brainstorming |
| Finishing major work | Project state (capture learnings) |
| External change | Vision evolution |
| Quarterly review | Project state + Vision evolution |

## When NOT to Research

- **When action is clear** - Don't research to avoid doing
- **When already decided** - Check ADRs first
- **When time-critical** - Act, then document learnings
- **For trivial questions** - Just answer them

## Critical Rules

### Always
- Interview before researching
- Check project context first
- Cite sources with confidence levels
- Present options, let user decide
- Get approval before editing VISION

### Never
- Edit VISION without explicit approval
- Research indefinitely (set time bounds)
- Ignore existing project decisions
- Present research as implementation plan
- Skip the interview step

## Output Expectations

Research produces **findings**, not decisions.

**Good output:**
- "Based on X, Y, and Z, the options are A and B"
- "The documentation says X with high confidence"
- "Our ADR-003 already decided Y"

**Bad output:**
- "Let's do X" (that's a decision, not research)
- "I think we should..." (presents options, not conclusions)

## Scripts

| Script | Usage | Description |
|--------|-------|-------------|
| `check-sessions.sh` | Review recent sessions | Find lessons learned |
| `git-context-summary.sh` | Summarize git history | Understand recent changes |

## Related Skills

- **orchestration** - For acting on research findings
- **foundations** - For documentation standards
- **brainstorm** - For generative exploration
- **reflect** - For updating strategy post-shipping
- **architecture** - For making technical decisions
- **strategy** - For discovering strategic context
