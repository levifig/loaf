---
description: Discover product requirements and update REQUIREMENTS.md
version: 1.11.1
---

# PRD Command

Interview to discover product requirements, update REQUIREMENTS.md.

**Input:** $ARGUMENTS

---

## Purpose

Requirements capture **what** the product must do, not **how**.

Good requirements are:
- **User-focused** - Written from user perspective
- **Testable** - Clear acceptance criteria
- **Organized** - Grouped by domain
- **Complete** - Cover edge cases and business rules

---

## CRITICAL: Interview First

**Before writing any requirements, understand:**

1. What feature area are we defining?
2. Who are the users affected?
3. What business rules apply?
4. What edge cases exist?
5. What's explicitly out of scope?

Use `AskUserQuestion` with non-obvious questions.

---

## Process

### Step 1: Parse Input

`$ARGUMENTS` should indicate the feature area or requirement section.

Examples:
- "user authentication"
- "section 2.1"
- "payment processing"

**If unclear:** Ask what feature area to define.

### Step 2: Gather Context

1. **Read VISION.md** - Strategic direction constrains requirements
2. **Read ARCHITECTURE.md** - Technical constraints
3. **Read existing REQUIREMENTS.md** - Don't duplicate

### Step 3: Interview with Non-Obvious Questions

Ask questions the user might not have considered:

| Category | Example Questions |
|----------|-------------------|
| Edge cases | "What happens if the user closes the browser mid-flow?" |
| Business rules | "Are there rate limits? Time-based restrictions?" |
| Persona conflicts | "How do admin users differ from regular users here?" |
| Failure modes | "What's the graceful degradation if this fails?" |
| Compliance | "Any audit logging requirements?" |
| Scale | "Expected volume? Concurrent users?" |

**Interview structure:**
```
1. Start with obvious questions to confirm understanding
2. Move to edge cases
3. Explore business rule conflicts
4. Check compliance/security needs
5. Confirm scope boundaries
```

### Step 4: Draft Requirements Section

Format:

```markdown
## [Number]. [Domain Name]

### [Number].[SubNumber] [Feature Name]

**Business Rules**
- [Rule 1]
- [Rule 2]

**User Stories**
As a [persona], I want to [action] so that [benefit].

**Acceptance Criteria**
- [ ] Given [context], when [action], then [outcome]
- [ ] Given [context], when [action], then [outcome]

**Edge Cases**
| Scenario | Expected Behavior |
|----------|-------------------|
| [Edge case] | [How system responds] |

**Out of Scope**
- [Explicitly excluded item]
```

### Step 5: Present for Approval

Show the drafted section to the user:

```markdown
## Proposed Addition to REQUIREMENTS.md

[Draft section]

---

**Questions before adding:**
1. Are these business rules accurate?
2. Any missing acceptance criteria?
3. Should anything be out of scope?
```

### Step 6: Await Approval

**Do NOT update REQUIREMENTS.md without explicit approval.**

User may:
- Approve as-is
- Request changes
- Add missing requirements
- Move items to out of scope

### Step 7: Update REQUIREMENTS.md

After approval:
1. Add new section to REQUIREMENTS.md
2. Maintain consistent numbering
3. Update table of contents if exists

---

## Requirements Format

### File Structure

```markdown
# Requirements

## 1. Core Platform

### 1.1 Feature A
...

### 1.2 Feature B
...

## 2. Identity & Access

### 2.1 User Authentication
...

### 2.2 Authorization
...
```

### Section Template

```markdown
### 2.1 User Authentication

**Business Rules**
- Users authenticate via OAuth (Google, GitHub)
- Sessions expire after 24 hours of inactivity
- Failed login attempts are logged for security audit
- Users can have at most 5 active sessions

**User Stories**

As a new user, I want to sign in with my existing Google account
so that I don't need to create another password.

As a returning user, I want my session to persist across browser tabs
so that I can work in multiple windows.

**Acceptance Criteria**
- [ ] Given a new user, when they click "Sign in with Google", then they are redirected to Google OAuth
- [ ] Given valid Google credentials, when OAuth completes, then user is logged in and redirected to dashboard
- [ ] Given an existing session, when user opens new tab, then they remain logged in
- [ ] Given 24 hours of inactivity, when user returns, then they must re-authenticate
- [ ] Given a logged-in user, when they click logout, then all session data is cleared

**Edge Cases**
| Scenario | Expected Behavior |
|----------|-------------------|
| OAuth provider unavailable | Show friendly error, suggest trying later |
| User revokes OAuth access | Invalidate session on next request |
| Multiple accounts same email | Prevent duplicate accounts, suggest login |
| Session cookie deleted | Redirect to login on next request |

**Out of Scope**
- Password-based authentication (security/maintenance burden)
- Apple/Microsoft OAuth (future consideration)
- MFA (separate feature area)
```

---

## Acceptance Criteria Format

Use Given-When-Then:

```markdown
Given [precondition/context],
when [action is performed],
then [expected outcome].
```

**Good criteria are:**
- Specific and testable
- Single behavior per criterion
- Observable outcome (not internal state)
- User-focused language

**Bad criteria:**
```markdown
- Login should be fast  # Not measurable
- Data should be correct  # Too vague
- System should handle errors  # No specific behavior
```

**Good criteria:**
```markdown
- [ ] Given valid credentials, when login submitted, then dashboard loads within 2 seconds
- [ ] Given invalid email format, when login attempted, then "Invalid email" error shown
- [ ] Given network failure during OAuth, when callback fails, then retry prompt displayed
```

---

## Numbering Convention

```
[Major].[Minor] [Feature Name]

1. Core Platform
   1.1 Dashboard
   1.2 Navigation

2. Identity & Access
   2.1 User Authentication
   2.2 Role-Based Access
```

When adding new sections:
- Check existing numbers
- Insert at appropriate position
- Update subsequent numbers if needed

---

## Guardrails

1. **Interview deeply** - Ask non-obvious questions
2. **Check constraints** - VISION and ARCHITECTURE bound requirements
3. **Write testable criteria** - Given-When-Then format
4. **Document edge cases** - What could go wrong?
5. **Explicit out of scope** - Prevent scope creep
6. **Get approval** - Don't update without user confirmation

---

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Requirements describe HOW | Focus on WHAT, leave HOW to specs |
| Missing edge cases | Interview for failure modes |
| Vague criteria | Use Given-When-Then |
| No out of scope | Explicitly list exclusions |
| Conflicting rules | Resolve before adding |

---

## Related Skills

- **orchestration/product-development** - Where requirements fit in hierarchy
- **orchestration/specs** - Breaking requirements into specifications
- **foundations** - Documentation standards
