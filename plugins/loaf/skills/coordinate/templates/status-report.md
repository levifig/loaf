# Status Report

Use this shape for the coordinator's current report. Rebuild it from live thread, tracker, and code-host state each time; it is a message to the operator, never a stored record. Include every ongoing thread, not only those needing the operator. Leave out a section that would be empty.

Start each entry with 🔴 when the operator must decide, approve, or act, and 🔵 otherwise. Waiting on the coordinator or on another thread is 🔵. Link each thread by its name.

## Now

[Threads acting or holding a turn.]

🔴 [Linked thread name]

**Status:** [What exists and whether it shipped or lives only in the thread.]

**Next action:** [Who acts next and the exact action.]

## Next

[Threads ready and waiting for a turn, a dependency, or an answer, in queue order.]

## Later

[Threads parked, or work not yet started.]

## Housekeeping

**Archived:** [Comma-separated linked names of threads archived since the last report, without reasons.]

**Suggestions:**

- [Linked thread name]: [recommended housekeeping action and the decision needed.]
