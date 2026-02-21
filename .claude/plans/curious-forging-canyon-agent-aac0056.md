# Post-Migration Cleanup Plan

5 targeted fixes across 12 files, followed by a build verification.

---

## Issue 1: Add `user-invocable: false` to orchestration sidecar

**File:** `src/skills/orchestration/SKILL.claude-code.yaml`

Current content (lines 1-5):
```yaml
# Claude Code extensions for Orchestration skill
# Base name/description in SKILL.md

agent: "{{AGENT:pm}}"
allowed-tools: Read, Write, Edit, Glob, Grep, TodoWrite, TodoRead
```

**Change:** Insert `user-invocable: false` on a new line 4 (before the `agent:` line).

Result:
```yaml
# Claude Code extensions for Orchestration skill
# Base name/description in SKILL.md

user-invocable: false
agent: "{{AGENT:pm}}"
allowed-tools: Read, Write, Edit, Glob, Grep, TodoWrite, TodoRead
```

---

## Issue 2: Add `## Contents` TOC to 8 skills

For each file, I will insert a `## Contents` section listing the major `##` headings. Placement: after the title `# ...` heading and any brief intro paragraph, before the first `##` section.

### 2a. `src/skills/foundations/SKILL.md`

Insert after line 15 ("Engineering foundations for consistent, secure, and well-documented code."), before line 17 ("## Philosophy"):

```markdown

## Contents
- Philosophy
- Quick Reference
- Topics
- Available Scripts
- Critical Rules
- Naming Conventions
- Commit Types
- Test Patterns
- Security Mindset
- Related Skills
```

### 2b. `src/skills/go-development/SKILL.md`

Insert after line 12 ("Patterns and best practices for Go development, grounded in Effective Go principles and community conventions."), before line 14 ("## Philosophy"):

```markdown

## Contents
- Philosophy
- Reference Files
- Quick Reference
- Core Principles Summary
- Integration with Foundations
```

### 2c. `src/skills/infrastructure-management/SKILL.md`

Insert after line 13 ("Infrastructure patterns for containerization, orchestration, CI/CD pipelines, and deployment automation."), before line 15 ("## Stack Overview"):

```markdown

## Contents
- Stack Overview
- Philosophy
- Quick Reference
- Topics
- Available Scripts
- Critical Rules
- CI Failure Triage
- Quick Diagnostics
```

### 2d. `src/skills/orchestration/SKILL.md`

Insert after line 14 ("Comprehensive patterns for PM-style orchestration: coordinating multi-agent work, managing sessions, running councils, delegating to specialized agents, and integrating with Linear."), before line 16 ("## Philosophy"):

```markdown

## Contents
- Philosophy
- Quick Reference
- Topics
- Configuration
- Artifact Locations
- Available Scripts
- Three-Phase Workflow
- When to Use PM Orchestration
- Critical Rules
```

### 2e. `src/skills/power-systems-modeling/SKILL.md`

Insert after line 15 ("Domain knowledge for overhead transmission line physics, thermal ratings, and mechanical analysis."), before line 17 ("## When to Use This Skill"):

```markdown

## Contents
- When to Use This Skill
- Key Reference Files
- Available Scripts
- Quick Reference
- Code Patterns
- Related Skills
```

### 2f. `src/skills/python-development/SKILL.md`

Insert after line 14 ("Comprehensive guide for modern Python 3.12+ development with FastAPI ecosystem."), before line 16 ("## When to Use This Skill"):

```markdown

## Contents
- When to Use This Skill
- Stack Overview
- Core Philosophy
- Quick Reference
- Topics
- Critical Rules
```

### 2g. `src/skills/ruby-development/SKILL.md`

Insert after line 12 ("The DHH/37signals way: convention over configuration, programmer happiness, and elegant simplicity."), before line 14 ("## Philosophy"):

```markdown

## Contents
- Philosophy
- Ruby Idioms
- Stack Overview
- Quick Reference
- Topics
- Critical Rules
- File Organization
- Decision Guide
```

### 2h. `src/skills/typescript-development/SKILL.md`

Insert after line 14 ("Comprehensive guide for modern TypeScript development with React ecosystem."), before line 16 ("## When to Use This Skill"):

```markdown

## Contents
- When to Use This Skill
- Stack Overview
- Core Philosophy
- Quick Reference
- Topics
- Critical Rules
```

---

## Issue 3: Fix stale comment in targets.yaml

**File:** `src/config/targets.yaml`

**Line 16:** `# Full Cursor support: skills, commands, agents, hooks`

**Change to:** `# Full Cursor support: agents, skills, hooks`

---

## Issue 4: Clarify wording in breakdown skill

**File:** `src/skills/breakdown/SKILL.md`

**Line 401:**
```markdown
- **implement** -- Run one task or orchestrate multiple tasks
```

**Change to:**
```markdown
- **implement** -- Run one task or coordinate multiple tasks
```

---

## Issue 5: Fix wrong recovery command in reference-session

**File:** `src/skills/reference-session/SKILL.md`

**Lines 189-190:**
```markdown
**Note**: No active session to update. Start a session first with
`/resume` to track this reference.
```

**Change to:**
```markdown
**Note**: No active session to update. Start a session first with
`/implement` to track this reference.
```

Rationale: `/resume` is for resuming EXISTING sessions. When no active session exists, `/implement` is the correct command to start new work.

---

## Final Step: Build verification

Run `npm run build` and confirm all 5 targets succeed (claude-code, opencode, cursor, codex, gemini).
