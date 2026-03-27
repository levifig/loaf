---
name: cleanup
description: >-
  Reviews and cleans up agent artifacts in .agents/ — sessions, specs, plans,
  drafts, councils, and reports. Provides hygiene recommendations, archives
  completed work, and ensures extracted knowledge is preserved. Use when the
  user asks "clean up", "review sessions", "review artifacts", or "tidy up
  .agents/."
---

# Cleanup

Review ALL agent artifacts in `.agents/` and provide hygiene recommendations.

## Contents
- Sessions
- Tasks
- Specs
- Plans
- Drafts
- Councils
- Reports
- Summary Table
- Archival Process
- Where Knowledge Belongs

**CRITICAL: Review EVERY file, not samples or averages.**
**Never reference `.agents/` files in output docs outside `.agents/`.**
**After user confirmation, auto-move archived artifacts and update `.agents/` links.**
**Archived artifacts are retained indefinitely.**

---

## 1. Sessions

For EACH session file in `.agents/sessions/` and `.agents/sessions/archive/`:

### A. Read and Summarize
- Title, status, Linear issue (query Linear for current status), dates, current state

### B. Check for Issues
- [ ] References missing files? → **Stale**
- [ ] Linear issue closed? → **Ready for archive**
- [ ] Status "completed" but not archived? → **Ready for archive**
- [ ] File in archive but status not `archived`? → **Fix status**
- [ ] Archived missing `archived_at` or `archived_by`? → **Add metadata**
- [ ] >7 days since last update with no activity? → **Review for staleness**

### C. Check for Extraction Needs
- [ ] Contains `lessons_learned`? → Extract to relevant docs
- [ ] Contains unrecorded `decisions`? → Suggest ADR creation
- [ ] Contains `remaining_work`/`next_steps`/`technical_debt`? → Check if tracked
- [ ] CHANGELOG draft present but not integrated? → Flag for integration

### D. Present Per Session
Summary (2-3 lines), issues found, extraction recommendations.
Recommendation: **Extract & Archive** / **Archive** / **Keep**

---

## 2. Tasks

For EACH task in `.agents/tasks/` (not archive):

### Check for Issues
- [ ] Status `done` but not in archive/? → **Ready for archive**
- [ ] Status `done` but TASKS.json `file` field missing `archive/` prefix? → **Index drift**
- [ ] File exists on disk but not in TASKS.json? → **Orphan — run `loaf task sync --import`**
- [ ] TASKS.json entry exists but file not found on disk? → **Stale reference**
- [ ] Frontmatter disagrees with TASKS.json? → **Drift — run `loaf task sync`**

### Archival

Use the CLI to archive tasks — do NOT use raw `mv`:
```bash
loaf task archive TASK-040 TASK-041 TASK-042    # specific tasks
loaf task archive --spec SPEC-013               # all done tasks for a spec
```

### Present Per Task
ID, title, status, spec, age since completion.
Recommendation: **Archive** / **Keep**

---

## 3. Specs

For EACH spec file in `.agents/specs/` and `.agents/specs/archive/`:

### Check for Issues
- [ ] Status `complete` but not in archive/? → **Ready for archive**
- [ ] Status `implementing` but no active session or tasks? → **Stale**
- [ ] Status `drafting` with no activity >14 days? → **Review for staleness**
- [ ] Status `approved` but never started? → **Flag for prioritization**
- [ ] References tasks or sessions that don't exist? → **Stale references**
- [ ] Frontmatter missing required fields (id, title, status)? → **Fix metadata**

### Archival

Use the CLI to archive specs — do NOT use raw `mv`:
```bash
loaf spec archive SPEC-009 SPEC-013
```

### Present Per Spec
ID, title, status, appetite, task count (from TASKS.json if available).
Recommendation: **Archive** / **Keep** / **Flag for review**

---

## 4. Plans

For EACH plan file in `.agents/plans/`:

### Check for Issues
- [ ] Linked session is archived or doesn't exist? → **Orphaned — delete**
- [ ] Linked session is complete? → **Ready for cleanup**
- [ ] >14 days old with no linked active session? → **Stale — delete**
- [ ] No session link at all? → **Orphaned — delete**

Plans are ephemeral — they exist to support a session. When the session is archived, the plan should be deleted (not archived). The decisions and outcomes live in the session file and git history.

### Present Per Plan
Filename, linked session, session status, age.
Recommendation: **Delete** / **Keep**

---

## 5. Drafts

For EACH file in `.agents/drafts/`:

### Check for Issues
- [ ] Content was used to create a spec? → **Served its purpose — archive or delete**
- [ ] >30 days old with no related spec or active work? → **Stale — review for deletion**
- [ ] Contains unique research or analysis not captured elsewhere? → **Keep or extract**
- [ ] Duplicates information in an existing spec or ADR? → **Delete**

Drafts include brainstorms, research documents, and exploratory notes. They feed into specs and decisions but are not themselves durable artifacts.

### Present Per Draft
Filename, age, related specs (if any), size.
Recommendation: **Delete** / **Keep** / **Extract & Delete**

---

## 6. Councils

For EACH council file in `.agents/councils/` and archive:
- Topic, date, linked session, decision outcome
- Flag: orphaned (no session), >14 days old, decision should be ADR
- Archive only after session summary captures outcome

---

## 7. Reports

For EACH file in `.agents/reports/` and archive.
Report frontmatter follows [report template](templates/report.md).

Archive prerequisites: processed, linked session archived, `archived_at` + `archived_by` set.

---

## 8. Summary Table

```
SESSIONS (N total):
  Ready for archive: N (list)
  Need extraction:   N (list)
  Keep active:       N

TASKS (N total):
  Done (archive):     N (list)
  Orphans:            N
  Index drift:        N
  Active:             N

SPECS (N total):
  Complete (archive): N (list)
  Active:             N
  Stale:              N

PLANS (N total):
  Orphaned (delete):  N (list)
  Active:             N

DRAFTS (N total):
  Stale (review):     N (list)
  Active:             N

COUNCILS (N total): ...
REPORTS (N total): ...
```

---

## 9. Archival Process

**Never auto-archive or auto-delete.** For each flagged artifact:
1. Show checklist with specific items to extract or verify
2. Ask user: "Archive" / "Delete" / "Keep"
3. If extracting, perform extractions first
4. For sessions: set status `archived`, `archived_at`, `archived_by`, move to archive/
5. For tasks: use `loaf task archive TASK-XXX` (moves file + updates TASKS.json)
6. For specs: use `loaf spec archive SPEC-XXX` (moves file + updates TASKS.json)
7. For plans: delete (not archive — ephemeral)
8. For drafts: delete or archive based on user preference
9. Auto-update `.agents/` references after moves

**Final reconciliation:** After all archival is complete, run `loaf task sync` to ensure TASKS.json matches the filesystem. This catches any drift from manual file moves.

---

## 10. Where Knowledge Belongs

| Information Type | Destination |
|------------------|-------------|
| Work tracking | Linear issues / TASKS.json |
| Implementation details | Git commits |
| Architectural decisions | `docs/decisions/ADR-XXX-*.md` |
| API contracts | `docs/api/` |
| Remaining work | Linear backlog / TASKS.json |
| User-facing changes | CHANGELOG.md |
| Council outcomes | Session summary → archive |
| Report conclusions | Session summary → archive |
| Spec decisions | Spec file → `.agents/specs/` |
| Archived artifacts | `.agents/<type>/archive/` |
