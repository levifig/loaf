# Loaf's Historical Living ADR Convention

Read this only when maintaining or interpreting records under Loaf's historical revise-in-place convention. ADRs in general are not legacy: Architecture's decision-record reference covers new exceptional records. Existing files alone do not require more records, and these historical rules do not apply to architecture topics.

## Follow the repository's contract

Prefer its own template, lifecycle, and numbering. Do not renumber historical records, fabricate dates or alternatives, or force migration during an ADR-scoped task. A user explicitly requesting an ADR has chosen a format; explain any substantive scope mismatch without imposing a ritual approval gate.

For records using Loaf's historical convention, the five statuses mean:

| Status | Meaning |
|--------|---------|
| Proposed | A choice awaiting acceptance |
| Accepted | The agreed choice within its applicability; implementation may still be pending |
| Rejected | A proposal explicitly declined before acceptance |
| Deprecated | No longer binding in this home, whether retired or rehomed; explain which and identify the new home when one exists |
| Superseded | Responsibility transferred to named replacement ADRs with reciprocal links |

Revise a continuing topic in place under its existing ID. For this legacy convention, a material revision sets `revised:` and adds a matching `## Revisions` note; cosmetic edits do not. Preserve the original creation date, add transition dates only from evidence, and retain meaningful replaced framing among the actual alternatives. Supersession is for topic transfer or split, not ordinary evolution.

If preserving this convention without an existing repository template, use the legacy ADR template listed in the Architecture skill. Verify status-specific dates and sections, reciprocal replacement links, index accuracy, and the difference between acceptance and implementation. Do not silently reinterpret unknown metadata; report the gap.

Run `loaf kb validate --legacy-adr <file> [<file>...]` for the historical filename, frontmatter, date-format, and required-section checks. This opt-in command replaces Loaf's former `validate-adr.py` helper and needs no Python or Git. It preserves that helper's limited metadata syntax and date-shape checks; it does not verify reciprocal links, factual dates, acceptance, or architectural correctness. Ordinary `loaf kb validate` does not apply these conventions to architecture topics or exceptional decision records.
