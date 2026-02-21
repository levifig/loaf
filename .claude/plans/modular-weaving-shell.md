# Plan: Skill Routing & Resilience Improvements

## Context

OpenAI published a blog post on building production-grade agentic systems with skills, shell environments, and compaction. While targeted at their platform, the patterns are directly applicable to Loaf as a meta-framework that builds skills, agents, and commands for multiple AI coding tools.

Loaf already does several things well that the blog recommends (multi-target distribution, session-based state persistence, context management patterns, structured workflows). But there are concrete gaps in **skill routing precision**, **negative disambiguation**, **template placement**, **compaction-as-default philosophy**, and **security posture for networked agents**.

This plan applies the blog's ten tips and three patterns to Loaf, operating at two levels:
1. **Direct**: Improve the skills/agents/commands Loaf ships
2. **Meta**: Improve how `.agents/AGENTS.md` teaches others to write skills

**File hierarchy note:** The canonical guidelines file is `.agents/AGENTS.md`. Both `CLAUDE.md` and `.claude/CLAUDE.md` are symlinks to it. This plan also adds an `AGENTS.md` symlink at project root for agent compatibility.

---

## Changes

### 0. Create Root AGENTS.md Symlink

**Why:** Some agents look for `AGENTS.md` at project root. The canonical file is `.agents/AGENTS.md`; root `CLAUDE.md` and `.claude/CLAUDE.md` already symlink to it. Adding a root `AGENTS.md` symlink completes compatibility.

**Current state:**
- `.agents/AGENTS.md` - canonical file (real)
- `CLAUDE.md` -> `.agents/AGENTS.md` (symlink, exists)
- `.claude/CLAUDE.md` -> `../.agents/AGENTS.md` (symlink, exists)
- `AGENTS.md` - does not exist

**Action:** `ln -s .agents/AGENTS.md AGENTS.md`

### 1. Add Negative Routing to Skill Descriptions

**Why:** The blog's Tips 1 & 2 show that adding skills without negative examples reduces correct triggering by ~20%. Loaf's `description` frontmatter fields currently have "Use when..." but lack "Not for..." disambiguation. When 13 skills are loaded, the model needs clear boundaries.

**Confusable pairs identified:**
- `orchestration` vs `research` (both involve planning/assessment)
- `foundations` vs language skills (overlapping code standards)
- `database-design` vs `python-development` (both cover SQLAlchemy)
- `orchestration` vs `foundations` (both mention commits, PRs)

**What to change:** Add a trailing "Not for..." clause to the `description` field of each skill that has a confusable neighbor. Keep within the 1024-char limit.

**Files:**
- `src/skills/orchestration/SKILL.md` - Add: "Not for standalone research, brainstorming, or vision work (use research skill)."
- `src/skills/research/SKILL.md` - Add: "Not for multi-agent coordination, session management, or task delegation (use orchestration skill)."
- `src/skills/foundations/SKILL.md` - Add: "Not for language-specific patterns, framework APIs, or deployment (use language skills)."
- `src/skills/database-design/SKILL.md` - Add: "Not for ORM-level code patterns (use language skill) or infrastructure provisioning (use infrastructure-management)."
- `src/skills/python-development/SKILL.md` - Add: "Not for schema design decisions or migration strategies (use database-design)."
- `src/skills/typescript-development/SKILL.md` - Add: "Not for design system philosophy or accessibility auditing (use interface-design)."
- `src/skills/interface-design/SKILL.md` - Add: "Not for React/Next.js implementation details (use typescript-development)."
- `src/skills/infrastructure-management/SKILL.md` - Add: "Not for application code, database schemas, or CI pipeline logic in app repos."

**Pattern:** Append to existing description: `Not for [specific thing] ([alternative skill]).`

### 2. Add Success Criteria to Workflow Skills

**Why:** Blog Tip 1 says descriptions should answer "what does success look like?" Workflow skills (orchestration, research, foundations) are invoked procedurally and should define their outputs.

**What to change:** Add "Produces..." clause to workflow skill descriptions.

**Files:**
- `src/skills/orchestration/SKILL.md` - "Produces session files, council records, task breakdowns, and progress updates."
- `src/skills/research/SKILL.md` - "Produces state assessments, research findings with options, or vision change proposals."
- `src/skills/foundations/SKILL.md` - "Produces validated commits, formatted documentation, and security-reviewed code."

### 3. Embed Key Templates in SKILL.md (Not Just References)

**Why:** Blog Tip 3 says templates should live inside skills, not system prompts. Loaf already has good templates in reference files, but the most-used templates should be accessible without loading a reference.

**What to change:** Add condensed template summaries to the main SKILL.md files for the most frequently-used output formats. Keep full templates in references; add "Quick Template" sections to SKILL.md with the essential structure.

**Files:**
- `src/skills/orchestration/SKILL.md` - Add quick session file template (frontmatter + required sections)
- `src/skills/research/SKILL.md` - Already has templates (no change needed)
- `src/skills/foundations/SKILL.md` - Add quick commit message template and ADR skeleton

### 4. Reframe Compaction from Reactive to Proactive

**Why:** Blog Tip 4 says "treat compaction as default, not emergency fallback." Loaf's context-management.md frames compaction as something to manage when things go wrong ("context pollution", "warning signs"). The guidance should instead position compaction as a normal, expected part of long workflows.

**What to change:** Restructure the context-management.md reference to lead with proactive patterns and normalize compaction as routine.

**File:** `src/skills/orchestration/references/context-management.md`

**Changes:**
- Add new section "Design for Compaction" at the top, before troubleshooting
- Reframe "Warning Signs" as a secondary concern, not the primary framing
- Add explicit guidance: "For any workflow expected to exceed 15-20 exchanges, design the session file to serve as the compaction recovery point from the start"
- Add pattern: "Pre-write the resumption prompt section in the session file before starting, not as an emergency measure"

### 5. Add Explicit Skill References in Delegation Patterns

**Why:** Blog Tip 5 says explicit skill invocation creates deterministic contracts. When PM delegates to `backend-dev`, the prompt doesn't currently reference which skills to prioritize. This leaves skill selection to the model's discretion.

**What to change:** Update delegation reference to include skill hints in spawn prompts.

**File:** `src/skills/orchestration/references/delegation.md`

**Changes:**
- Update the "Example Task() Call" to include skill references:
  ```
  Follow patterns from the python-development skill for FastAPI conventions.
  Follow database-design skill for schema decisions.
  ```
- Add a "Skill Hints" subsection to "Spawning Best Practices" explaining when and how to reference specific skills in delegation prompts

### 6. Add Network + Skills Security Posture

**Why:** Blog Tip 6 warns that "combining skills with open network access creates a high-risk path for data exfiltration." Loaf's permissions.md covers tool allowlists but doesn't address the specific risk of skills + network access.

**What to change:** Add a "Network Access" section to the permissions reference.

**File:** `src/skills/foundations/references/permissions.md`

**Changes:**
- Add "Network Access Posture" section covering:
  - Risk: skills can instruct agents to fetch/send data over network
  - Default posture: no network access unless explicitly required
  - Pattern: scope `Bash(curl *)` / `WebFetch` to specific domains when needed
  - Never combine broad network access with broad file read permissions in the same agent session
- Add "Authenticated API Calls" subsection:
  - Use environment variables for credentials (never pass in prompts)
  - Agent prompts should reference `$ENV_VAR` patterns, not literal secrets
  - Prefer MCP tools (which handle auth) over raw network calls

### 7. Standardize Artifact Output Conventions

**Why:** Blog Tip 7 defines `/mnt/data` as the standard handoff boundary. Loaf has `.agents/sessions/` and `.agents/councils/` but no general convention for agent-produced artifacts (reports, analysis outputs, generated configs).

**What to change:** Add an "Artifact Output" section to the orchestration skill defining standard locations.

**File:** `src/skills/orchestration/SKILL.md`

**Changes:**
- Add to Quick Reference table: `Agent artifact` → `Write to .agents/{type}/` with frontmatter
- Add "Artifact Locations" section:

  | Artifact | Location | Archive |
  |----------|----------|---------|
  | Sessions | `.agents/sessions/` | `.agents/sessions/archive/` |
  | Councils | `.agents/councils/` | `.agents/councils/archive/` |
  | Transcripts | `.agents/transcripts/` | N/A |
  | Reports | `.agents/reports/` | N/A |
  | Tasks | `.agents/tasks/` | N/A |

### 8. Update Skill Development Guidelines in AGENTS.md

**Why:** Blog Pattern C treats skills as "enterprise workflow carriers" - living SOPs that evolve. The guidelines should encode the new patterns so future skill authors follow them.

**File:** `.agents/AGENTS.md` (canonical; `CLAUDE.md`, `.claude/CLAUDE.md`, and `AGENTS.md` all symlink here)

**Changes to "Description Best Practices" section:**

Add new items:

4. **Include negative routing** for disambiguation:
   ```yaml
   description: >-
     Covers Python development... Not for schema design
     decisions (use database-design) or deployment infrastructure
     (use infrastructure-management).
   ```

5. **Add success criteria** for workflow skills:
   ```yaml
   description: >-
     Conducts research... Produces state assessments,
     research findings with ranked options, or vision
     change proposals.
   ```

**Add new subsection "Template Embedding":**
- Frequently-used templates belong in SKILL.md (quick reference)
- Detailed templates go in references (loaded on demand)
- Templates reduce model improvisation and increase output consistency

**Add to Anti-Patterns table:**
| Skip negative routing for confusable skills | Add "Not for..." in description |
| Leave success criteria undefined for workflow skills | Add "Produces..." in description |

---

## Verification

1. **Build check**: `npm run build` succeeds after all changes
2. **Description length**: Verify no `description` field exceeds 1024 characters
3. **Frontmatter validation**: All modified SKILL.md files still parse correctly
4. **Grep for consistency**: All skills with confusable neighbors have "Not for..." clauses
5. **Manual review**: Read each modified description and verify the negative routing is accurate and doesn't create circular exclusions
6. **Diff review**: `git diff` to confirm only intended files changed
