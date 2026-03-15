# Agent Delegation

## Contents

- Core Principle
- What PM Does vs Delegates
- Agent Capability Matrix
- Delegation Decision Tree
- Spawn Patterns
- Anti-Patterns
- Agent Access Hierarchy

## Core Principle

**PM is the orchestrator, not the implementer.** All code changes, documentation edits, and implementation work MUST be delegated to specialized agents.

## What PM Does vs Delegates

### PM Does Directly

| Action | Tool |
|--------|------|
| Create/edit session files | Write |
| Create/edit council files | Write |
| Track tasks | TodoWrite/TodoRead |
| Manage external issues | Linear, GitHub |
| Read files for context | Read, Grep, Glob |
| Ask clarifying questions | AskUserQuestion (OpenCode: `question`) |
| Assign subagent work | TodoWrite (subagents read via TodoRead) |

### PM MUST Delegate

- Any code in `backend/`, `frontend/`, `src/`, `tests/`
- Documentation in `docs/`
- Configuration files (`.yaml`, `.json`, `.toml`)
- Infrastructure files (`Dockerfile`, `docker-compose.yml`)
- Database migrations
- Test files

**NO EXCEPTIONS** - even "trivial" 1-line fixes go through specialized agents.

## Agent Capability Matrix

### Implementation Agents

| Agent | Focus | Use For |
|-------|-------|---------|
| `Backend Dev` | Backend | Services, APIs, business logic, backend code review |
| `Frontend Dev` | Frontend | UI components, state management, frontend code review |
| `rails-dev` | Ruby on Rails | Controllers, models, views, Hotwire/Stimulus, ActiveRecord |
| `DBA` | Database | Schema design, migrations, indexes, query optimization |
| `DevOps` | Infrastructure | Docker, Kubernetes, CI/CD, monitoring |

### Quality Assurance Agents

| Agent | Focus | Use For |
|-------|-------|---------|
| `QA` | Tests | Unit tests, integration tests, fixtures, coverage |
| `security` | Security | Audits, vulnerabilities, OWASP, secrets, threat modeling |

### Documentation & Planning

| Agent | Focus | Use For |
|-------|-------|---------|
| `docs` | Documentation | API docs, ADRs, READMEs, guides |
| `product` | Planning | Requirements, roadmaps, feature specs, user stories |
| `Design` | UI/UX | Interface design, accessibility, design systems |

## Delegation Decision Tree

```
What type of work is needed?

|-- Code Implementation
|   |-- Python/FastAPI/Backend --> Backend Dev
|   |-- React/Next.js/Frontend --> Frontend Dev
|   |-- Ruby on Rails --> rails-dev
|   +-- Database Schema/Migrations --> DBA

|-- Infrastructure & Operations
|   |-- Docker/K8s/CI/CD --> DevOps
|   +-- Database Performance --> DBA

|-- Quality Assurance
|   |-- Test Implementation --> QA
|   +-- Security Audit --> security

|-- Code Review (Domain Dev)
|   |-- Backend Review --> Backend Dev
|   +-- Frontend Review --> Frontend Dev

|-- Documentation & Design
|   |-- Technical Documentation --> docs
|   |-- UI/UX Design --> Design
|   +-- Product Requirements --> product

+-- Complex Decision?
    +-- Council (5-7 agents, odd number)
```

## Spawn Patterns

**OpenCode requirement:** Interview the user with the `question` tool before drafting a plan or research strategy.

### Sequential (Dependencies)

Use when output of one agent is input to another:

```python
# Step 1: Schema first
Task(subagent_type="DBA", prompt="Create users table...")

# Wait for completion

# Step 2: Implementation uses schema
Task(subagent_type="Backend Dev", prompt="Implement user service...")

# Wait for completion

# Step 3: Tests use implementation
Task(subagent_type="QA", prompt="Write user tests...")
```

**Common sequences:**
- Schema -> Code -> Tests
- Design -> Implementation -> Code review -> Testing -> Security
- Implementation -> Tests -> Code review -> Security

### Parallel (Independent)

Use when work is truly independent:

```python
# Both can run simultaneously
Task(subagent_type="Backend Dev", prompt="Implement API...")
Task(subagent_type="Frontend Dev", prompt="Build UI...")
```

**Requirements for parallel:**
- No dependencies between tasks
- Defined API contract (for API + UI)
- Separate files/components

### Spawning Best Practices

1. **Be specific in prompts** - Include file paths, requirements, constraints
2. **One concern per agent** - Don't ask Backend Dev to also write tests
3. **Include context** - Session file, issue ID, previous outcomes
4. **Reference session** - `Session: .agents/sessions/YYYYMMDD-HHMMSS-name.md`
5. **Include skill hints** - Name the skills that should guide the agent's work

### Skill Hints

When delegating, explicitly name the skills that should guide the agent's work. This creates deterministic contracts instead of leaving skill selection to the model's discretion.

```python
# Explicit: agent knows which patterns to follow
prompt="... Follow python-development skill for FastAPI conventions.
        Follow database-design skill for schema decisions."

# Implicit: agent may or may not pick the right skill
prompt="... Build the API endpoint."
```

**When to include skill hints:**
- The task spans multiple skill domains (e.g., backend code + database schema)
- You want a specific skill's conventions followed (e.g., "follow ruby-development, not python-development")
- The agent has access to multiple language skills and the choice matters

**When to skip:**
- Single-domain tasks where the agent only has one relevant skill
- The task description already clearly implies the domain

### Example Task() Call

```python
Task(
    subagent_type="Backend Dev",
    prompt="""
    Implement POST /api/v1/users endpoint.

    Requirements:
    - Validate email format
    - Hash password with bcrypt
    - Return 201 with user object
    - Handle duplicate email (409)

    Follow python-development skill for FastAPI conventions.
    Follow foundations skill for commit and security patterns.

    Files:
    - src/api/users.py
    - src/models/user.py

    Session: .agents/sessions/20251210-143052-user-registration.md
    Linear: BACK-123
    """
)
```

## Anti-Patterns

| Anti-Pattern | Better Approach |
|--------------|-----------------|
| PM implementing code | PM orchestrates, always delegate |
| Asking Backend Dev for React | Spawn Frontend Dev |
| Single agent for database + backend + tests | Sequential: DBA, Backend Dev, QA |
| Parallel spawns with hidden dependencies | Make dependencies explicit, spawn sequentially |
| Spawning without session context | Create session first, reference in prompts |
| Council for simple decisions | Single agent or PM judgment |

## Agent Access Hierarchy

| Agent Type | External Issue Access | Reports To |
|------------|----------------------|------------|
| PM | Read/Write | User |
| Implementation agents | None | PM (via session) |
| Review agents (backend/frontend devs) | None | PM (via session) |
| Product agent | Read-only | PM |

**Key**: Only PM writes to external issue trackers. All other agents report through session files.
