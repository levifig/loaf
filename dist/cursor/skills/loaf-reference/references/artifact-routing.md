# Artifact Routing

Choose by the named artifact and its authority, not by a guessed `.agents/` directory. These are current retrieval paths, not a universal artifact API or a claim that planned retirement has shipped.

| Requested object | Authority and supported read/write | Maintenance boundary |
|------------------|------------------------------------|----------------------|
| Shared work, definition of done, hierarchy, status | Configured native tracker through Project Management and the selected provider; Shape defines, Implement consumes, Ship verifies | No ongoing local issue synchronization or second work record |
| Persisted Handoff | Project-scoped SQLite body: discover with `loaf handoff list` or `loaf search <query>`, then `loaf handoff show <id-or-alias>`; write with `loaf handoff new --title <title> --body-file <path>` | Handoff owns deliberate preparation and delivery; preserve existing IDs and legacy files |
| Journal-only checkpoint or decision | Project-scoped SQLite journal: `loaf journal search <query>`, then `loaf journal show <entry-id>`; capture with `loaf journal log` | Foundations owns ordinary capture and selective synthesis; no session entity or required checkpoint |
| Direct transfer | Context delivered through an authorized harness surface; may be explicitly unpersisted | Do not claim an ID or database body unless persistence was observed |
| Explicit file or portable snapshot | Read the supplied path as authored evidence; a snapshot may be stale and does not prove access to its source store | Preserve user content; re-read current authoritative sources before acting |
| Council | Body in `.agents/councils/`; `loaf council list` / `loaf council show <ref>` reads file-backed deliberation and currently requires initialized state; `loaf council new` writes the file and registers identity | No council archive command; retain the file and use separately authorized Housekeeping for cleanup |
| Temporary report or deepening proposal | Authored analysis in chat or `.agents/reports/`; read/edit the actual file, without requiring a database row, status, ID, or report command | Housekeeping reviews each report; exact approval precedes moving, archival, or deletion |
| Maintained knowledge or architecture | Repository-configured Git paths, commonly `docs/knowledge/` and `docs/architecture/`; read current topics and deliberately update them | Knowledge Base and Architecture own their conventions; a private report is not automatically promoted |
| Spark, Idea, Exploration | Private SQLite records through their owning skills: `loaf spark list/show`, `loaf idea list/show`, `loaf exploration list/context`; capture/checkpoint through those skills | Private captures do not select shared work; native promotion requires provider readback |
| Historical local work and report helpers | Existing `loaf issue/task/plan/intent/...` and report lifecycle commands remain compatibility surfaces; consult exact CLI help for an existing record | Retirement remains pending in native work; preserve historical bodies, receipts, and file-backed reads rather than starting new work there |

For a named saved Handoff, start with its source project and SQLite discovery, even when no ID was supplied. A packet recorded only as a journal entry instead uses journal search/show. An explicit file request is a legitimate file read, not evidence that every Handoff lives on disk.

When the CLI, source project, database, or file is unavailable, say which source cannot be read and what is missing. Do not initialize another database to retrieve an existing record, claim that a journal hit is a persisted Handoff body, or silently substitute a file or snapshot as authority. An authorized direct transfer may still be useful, but label its persistence and access limits honestly. Record IDs are project-scoped retrieval addresses, not cross-project access grants.

Compatibility report working files remain authored Markdown; finalized generated projections must be updated through the owning report body and `loaf report finalize`, not hand-edited as a store. This does not give ordinary temporary reports a universal report lifecycle. Planned removal of compatibility commands does not make their current implementations or historical contents disappear.
