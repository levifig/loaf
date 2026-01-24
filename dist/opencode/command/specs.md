---
description: Create feature specifications from requirements
version: 1.11.1
---

# Specs Command

Break requirements into shaped, implementable specifications.

**Input:** $ARGUMENTS

---

## Purpose

Specs define **shaped solutions** - bounded work with clear scope.

A spec is:
- **Shaped** - Direction, not blueprint
- **Bounded** - Clear in/out of scope
- **Testable** - Observable test conditions
- **Appetite-driven** - Fixed time, variable scope

See `orchestration/specs` reference for full spec methodology.

---

## CRITICAL: Interview First

**Before writing a spec, clarify:**

1. Which requirement section is this for?
2. What's the appetite (time budget)?
3. What scope trade-offs are acceptable?
4. What are the "rabbit holes" to avoid?
5. What approaches are explicitly forbidden (no-gos)?

Use `AskUserQuestion` to establish boundaries.

---

## Process

### Step 1: Parse Input

`$ARGUMENTS` should reference a requirement section.

Examples:
- "2.1 User Authentication"
- "REQUIREMENTS.md section 2.1"
- "user auth"

**If unclear:** Ask which requirement to specify.

### Step 2: Gather Context

Read in order:
1. **VISION.md** - Strategic alignment
2. **ARCHITECTURE.md** - Technical constraints
3. **REQUIREMENTS.md** - The requirement being specified
4. **Existing specs** - Avoid duplication

### Step 3: Interview for Shaping

Shape Up methodology - define boundaries, not tasks.

| Question | Purpose |
|----------|---------|
| What's definitely in scope? | Core functionality |
| What's definitely out of scope? | Prevent creep |
| What seems related but should be avoided? | Rabbit holes |
| What approaches are forbidden? | No-gos |
| How do we know it works? | Test conditions |
| How much time is it worth? | Appetite |
| What if we run out of time? | Circuit breaker |

### Step 4: Determine Appetite

| Appetite | Size | Suitable For |
|----------|------|--------------|
| Small | 1-2 days | Bug fixes, minor enhancements |
| Medium | 3-5 days | Feature additions, refactors |
| Large | 1-2 weeks | Major features (rare) |

**If work can't fit the appetite:** Needs more shaping or splitting.

### Step 5: Draft the Spec

Use spec template:

```yaml
---
id: SPEC-001
title: "User Authentication with OAuth"
requirement: "2.1 User Authentication"
created: YYYY-MM-DDTHH:MM:SSZ
status: drafting
appetite: "1 week"
---

# SPEC-001: User Authentication with OAuth

## Problem Statement

[From REQUIREMENTS - why this matters to users and business]

## Proposed Solution

[Shaped solution - high-level approach, not implementation details]

## Scope

### In Scope
- [Item 1]
- [Item 2]

### Out of Scope (Rabbit Holes)
- [Avoid this complexity]
- [Don't go here]

### No-Gos
- [Forbidden approach 1]
- [Forbidden approach 2]

## Test Conditions

- [ ] [Observable outcome 1]
- [ ] [Observable outcome 2]
- [ ] [Observable outcome 3]

## Implementation Notes

[Architecture references, technical constraints, relevant ADRs]

## Circuit Breaker

At 50% appetite: [What to do if we're not on track]
```

### Step 6: Generate Spec ID

Find next available ID:

```bash
ls docs/specs/SPEC-*.md docs/specs/archive/SPEC-*.md 2>/dev/null | \
  grep -oE 'SPEC-[0-9]+' | \
  sort -t- -k2 -n | \
  tail -1 | \
  awk -F- '{print $2 + 1}'
```

If no specs exist, start with `SPEC-001`.

### Step 7: Present for Approval

```markdown
## Proposed Specification

[Full spec content]

---

**Before approval, confirm:**
1. Is the scope correct?
2. Are the test conditions sufficient?
3. Is the appetite realistic?
4. Any missing rabbit holes or no-gos?
```

### Step 8: Await Approval

**Do NOT create spec file without explicit approval.**

User may:
- Approve as-is
- Adjust scope
- Change appetite
- Add constraints
- Request splitting

### Step 9: Create Spec File

After approval:

1. **Generate timestamps:**
   ```bash
   date -u +"%Y-%m-%dT%H:%M:%SZ"  # For frontmatter
   ```

2. **Create file:**
   - Location: `docs/specs/SPEC-{id}-{slug}.md`
   - Slug: kebab-case of title

3. **Announce completion:**
   ```
   Created: docs/specs/SPEC-001-user-auth-oauth.md

   Next: Use `/tasks SPEC-001` to break into atomic work items.
   ```

---

## Spec Lifecycle

```
drafting → approved → implementing → complete
```

| Status | Meaning |
|--------|---------|
| `drafting` | Being refined, not ready |
| `approved` | Ready for task breakdown |
| `implementing` | Tasks created, work in progress |
| `complete` | All tasks done, archive |

---

## Splitting Large Specs

When a spec is too big for its appetite:

```
SPEC-001-user-auth.md (large, 2 weeks)
        ↓ split into
SPEC-001a-oauth-integration.md (medium, 3 days)
SPEC-001b-session-management.md (small, 2 days)
SPEC-001c-login-ui.md (medium, 3 days)
```

Each sub-spec:
- Has its own appetite
- Can be worked independently
- References original requirement

---

## Test Conditions vs Acceptance Criteria

| Test Conditions (Spec) | Acceptance Criteria (Requirement) |
|------------------------|-----------------------------------|
| High-level outcomes | Detailed behaviors |
| For this shaped solution | For the full feature |
| May be subset of AC | Complete coverage |
| Implementation-aware | Implementation-agnostic |

Test conditions prove the spec is done.
Acceptance criteria prove the requirement is met.

---

## Guardrails

1. **Interview to shape** - Boundaries before details
2. **Respect appetite** - Fixed time, variable scope
3. **Document rabbit holes** - Prevent scope creep
4. **Clear test conditions** - Observable outcomes
5. **Circuit breaker** - Plan for running out of time
6. **Get approval** - Don't create without confirmation

---

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Implementation details | Keep high-level, shape not blueprint |
| No appetite | Always set time budget |
| Missing no-gos | Explicitly forbid bad approaches |
| Vague test conditions | Make observable and verifiable |
| Too ambitious | Split or increase appetite |

---

## Related Skills

- **orchestration/specs** - Full spec methodology
- **orchestration/product-development** - Where specs fit in hierarchy
- **orchestration/planning** - Shape Up methodology details
