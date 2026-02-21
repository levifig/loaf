---
name: review-sessions
description: >-
  Reviews agent artifacts in .agents/ and provides session hygiene recommendations. Use when
  the user asks "review my sessions" or "clean up artifacts."
---

# Review Sessions

Review ALL agent artifacts in `.agents/` and provide hygiene recommendations.

## Contents
- Sessions Review
- Session Archival Process
- Councils Review
- Reports Review
- Summary Table
- CHANGELOG Integration Check
- Where Knowledge Belongs

**CRITICAL: Review EVERY file, not samples or averages.**
**Never reference `.agents/` files in output docs outside `.agents/`.**
**After user confirmation, auto-move archived artifacts and update `.agents/` links.**
**Archived artifacts are retained indefinitely.**

## 1. Sessions Review

For EACH session file in `.agents/sessions/` and `.agents/sessions/archive/`:

### A. Read and Summarize
- Title, status, Linear issue (query Linear for current status), dates, current state, council/report summaries present

### B. Check for Issues
- [ ] References missing files? -> **Stale**
- [ ] Linear issue closed? -> **Ready for archive**
- [ ] Status "completed" but not archived? -> **Ready for archive**
- [ ] File in archive but status not `archived`? -> **Fix status**
- [ ] Archived missing `archived_at` or `archived_by`? -> **Add metadata**
- [ ] >7 days since last update with no activity? -> **Review for staleness**
- [ ] Council file exists but no session summary? -> **Update before archive**
- [ ] Reports linked but unprocessed/unsummarized? -> **Update before archive**

### C. Check for Extraction Needs
- [ ] Contains `lessons_learned`? -> Extract to relevant docs
- [ ] Contains unrecorded `decisions`? -> Suggest ADR creation
- [ ] Contains `remaining_work`/`next_steps`/`issues_discovered`/`technical_debt`? -> Check if tracked in Linear
- [ ] Council outcomes or report summaries missing? -> Capture before archive

### D. Present Per Session
Summary (2-3 lines), issues found, extraction recommendations, recommendation: **Extract & Archive** / **Archive** / **Keep**

## 2. Session Archival Process

**Never auto-archive.** For each flagged session:
1. Show extraction checklist with specific items
2. Confirm council outcomes and report summaries captured
3. Ask user: "Extract and archive" / "Archive" / "Keep"
4. If extracting, perform extractions first
5. Archive: set status `archived`, `archived_at`, `archived_by`
6. Auto-move and update `.agents/` references

## 3. Councils Review

For EACH council file in `.agents/councils/` and archive:
- Topic, date, linked session, decision outcome, session summary status
- Flag: orphaned (no session), missing session summary, >14 days old, decision should be ADR
- Archive only after session summary exists (status + `archived_at` + `archived_by` + move)

## 4. Reports Review

For EACH file in `.agents/reports/` and archive. Report frontmatter following [report template](templates/report.md).

Archive prerequisites: processed, linked session archived, session captures conclusions, `archived_at` + `archived_by` set.

## 5. Summary Table

```
SESSIONS (N total):
  Ready for archive: N (list titles)
  Need extraction: N (list items)
  Keep active: N

COUNCILS (N total):
  Clean: N
  Orphaned/stale: N
  Missing session summary: N

REPORTS (N total):
  Processed and archived: N
  Missing frontmatter: N
  Awaiting session summary: N
```

## 6. CHANGELOG Integration Check

For sessions with completed work:
- Check for `## CHANGELOG Draft` section and integration status
- Report pending drafts with session title and Linear issue
- Auto-suggest `/update-changelog` if drafts are pending

## 7. Where Knowledge Belongs

| Information Type | Destination |
|------------------|-------------|
| Work tracking | Linear issues |
| Implementation details | Git commits |
| Architectural decisions | `docs/decisions/ADR-XXX-*.md` |
| Migration patterns | `docs/DATABASE_ARCHITECTURE.md` |
| API contracts | `docs/api/` |
| Test patterns | `docs/TESTING.md` |
| Remaining work | Linear backlog |
| User-facing changes | CHANGELOG.md |
| Council outcomes | Session summary |
| Report conclusions | Session summary |
| Archived artifacts | `.agents/<type>/archive/` |
