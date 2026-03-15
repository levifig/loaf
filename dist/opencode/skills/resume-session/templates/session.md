# Session File Template

**Location:** `.agents/sessions/YYYYMMDD-HHMMSS-<description>.md`

```yaml
---
session:
  title: "Clear description of work"
  status: in_progress
  created: "YYYY-MM-DDTHH:MM:SSZ"
  last_updated: "YYYY-MM-DDTHH:MM:SSZ"
  archived_at: "YYYY-MM-DDTHH:MM:SSZ"   # Required when archived
  archived_by: "agent-pm"                # Required when archived
  linear_issue: "PLT-XXX"               # If applicable
  linear_url: "https://linear.app/{{your-workspace}}/issue/PLT-XXX"
  branch: "username/plt-xxx-feature"
  task: "TASK-001"                       # If applicable
  spec: "SPEC-001"                       # If applicable

traceability:
  requirement: "2.1 User Authentication"
  architecture:
    - "Session Management"
  decisions:
    - "ADR-001"

plans: []
transcripts: []

orchestration:
  current_task: "Initial planning"
  spawned_agents: []
---

# Session: Brief Description

## Context
Why this session exists and what it aims to accomplish.

## Current State
Always handoff-ready summary of where things stand.

## Key Decisions
- Chose X over Y because Z

## Next Steps
- [ ] Immediate action items

## Resumption Prompt
<!-- Pre-write this section at session start for compaction resilience -->
Read this session file. Current state: [summary]. Next: [action].
```

## Filename Conventions

- **Task-coupled:** `YYYYMMDD-HHMMSS-task-XXX.md` (auto-generated)
- **Ad-hoc:** `YYYYMMDD-HHMMSS-<description>.md` (kebab-case)
