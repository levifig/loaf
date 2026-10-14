package state

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/levifig/loaf/vnext/migration/archive"
	"github.com/levifig/loaf/vnext/migration/retired"
)

func quoteArchiveIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// Check the actual backup schema as well as the maintained inventory. FTS
// virtual/shadow tables are rebuildable indexes, not independent source rows.
func retiredArchiveColumns(ctx context.Context, tx *sql.Tx) (map[string][]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM sqlite_schema WHERE type='table' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	columns := map[string][]string{}
	for _, name := range names {
		if name == "artifact_search" || name == "journal_search" || name == "docs_search" {
			continue
		}
		r, err := tx.QueryContext(ctx, `SELECT name FROM pragma_table_info(?) ORDER BY cid`, name)
		if err != nil {
			return nil, err
		}
		var cols []string
		projectScoped := false
		for r.Next() {
			var col string
			if err := r.Scan(&col); err != nil {
				r.Close()
				return nil, err
			}
			cols = append(cols, col)
			projectScoped = projectScoped || col == "project_id"
		}
		if err := r.Err(); err != nil {
			r.Close()
			return nil, err
		}
		if err := r.Close(); err != nil {
			return nil, err
		}
		if !projectScoped {
			continue
		}
		class, ok := ProjectScopedTableClass(name)
		if !ok || class == TableClassGone {
			return nil, fmt.Errorf("unclassified project-scoped source table %q", name)
		}
		columns[name] = cols
	}
	for _, name := range ProjectScopedLiveTables() {
		if len(columns[name]) == 0 {
			// Explicit journal-first migration drops both session tables. Accept
			// only that known shape, never a lone missing table or legacy linkage.
			if (name == "sessions" || name == "session_state_snapshots") && len(columns["sessions"]) == 0 && len(columns["session_state_snapshots"]) == 0 {
				legacyLink := false
				for _, col := range columns["journal_entries"] {
					legacyLink = legacyLink || col == "session_id"
				}
				if !legacyLink {
					continue
				}
			}
			return nil, fmt.Errorf("missing project-scoped source table %q", name)
		}
	}
	return columns, nil
}

func readRetiredArchive(ctx context.Context, tx *sql.Tx, projectID string) (retired.Content, error) {
	c := retired.Content{ProjectID: projectID, Tables: []retired.Table{}}
	columns, err := retiredArchiveColumns(ctx, tx)
	if err != nil {
		return c, err
	}
	totalRows, totalBytes := 0, 0
	for _, name := range ProjectScopedLiveTables() {
		t := retired.Table{Name: name, Columns: columns[name], Rows: []retired.Row{}}
		if len(t.Columns) == 0 {
			t.Absent = true
			c.Tables = append(c.Tables, t)
			continue
		}
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+quoteArchiveIdentifier(name)+` WHERE project_id=?`, projectID).Scan(&t.SourceRows); err != nil {
			return c, err
		}
		totalRows += t.SourceRows
		if totalRows > 200_000 {
			return c, fmt.Errorf("retired archive source exceeds row bound")
		}
		lengths := make([]string, 0, len(t.Columns))
		for _, col := range t.Columns {
			lengths = append(lengths, "COALESCE(length(CAST("+quoteArchiveIdentifier(col)+" AS BLOB)),0)")
		}
		var tableBytes int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(`+strings.Join(lengths, "+")+`),0) FROM `+quoteArchiveIdentifier(name)+` WHERE project_id=?`, projectID).Scan(&tableBytes); err != nil {
			return c, err
		}
		if tableBytes > int64(64<<20-totalBytes) {
			return c, fmt.Errorf("retired archive source exceeds payload byte bound")
		}
		selects := make([]string, 0, len(t.Columns)*2)
		for _, col := range t.Columns {
			q := quoteArchiveIdentifier(col)
			selects = append(selects, "typeof("+q+")", "CASE WHEN typeof("+q+") IN ('text','blob') THEN CAST("+q+" AS BLOB) ELSE "+q+" END")
		}
		rows, err := tx.QueryContext(ctx, `SELECT `+strings.Join(selects, ",")+` FROM `+quoteArchiveIdentifier(name)+` WHERE project_id=?`, projectID)
		if err != nil {
			return c, err
		}
		for rows.Next() {
			row := retired.Row{Values: make([]retired.Cell, len(t.Columns))}
			values := make([]any, len(t.Columns))
			pointers := make([]any, len(t.Columns)*2)
			for i := range values {
				pointers[2*i] = &row.Values[i].Type
				pointers[2*i+1] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				rows.Close()
				return c, err
			}
			for i, value := range values {
				cell := &row.Values[i]
				switch v := value.(type) {
				case nil:
					if cell.Type != "null" {
						rows.Close()
						return c, fmt.Errorf("unexpected NULL in %s", name)
					}
				case int64:
					cell.Data = []byte(strconv.FormatInt(v, 10))
				case float64:
					cell.Data = make([]byte, 8)
					binary.BigEndian.PutUint64(cell.Data, math.Float64bits(v))
				case []byte:
					cell.Data = append([]byte{}, v...)
				default:
					rows.Close()
					return c, fmt.Errorf("unexpected SQLite value in %s", name)
				}
				totalBytes += len(cell.Data)
				if totalBytes > 64<<20 {
					rows.Close()
					return c, fmt.Errorf("retired archive source exceeds payload byte bound")
				}
			}
			row.Disposition = retiredRowDisposition(t, row)
			t.Rows = append(t.Rows, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return c, err
		}
		if err := rows.Close(); err != nil {
			return c, err
		}
		if len(t.Rows) != t.SourceRows {
			return c, fmt.Errorf("source table %s does not reconcile", name)
		}
		c.Tables = append(c.Tables, t)
	}
	// Polymorphic bodies have no entity foreign key. Only bodies actually
	// consumed by a retained root can count as a migrated source row.
	roots := map[string]map[string]bool{"idea": {}, "handoff": {}}
	for _, t := range c.Tables {
		kind := ""
		if t.Name == "ideas" {
			kind = "idea"
		}
		if t.Name == "handoffs" {
			kind = "handoff"
		}
		if kind == "" {
			continue
		}
		for _, r := range t.Rows {
			id, _ := t.Value(r, "id")
			roots[kind][string(id.Data)] = true
		}
	}
	for i := range c.Tables {
		t := &c.Tables[i]
		if t.Name != "artifact_bodies" {
			continue
		}
		for j := range t.Rows {
			r := &t.Rows[j]
			kind, _ := t.Value(*r, "entity_kind")
			id, _ := t.Value(*r, "entity_id")
			bodyKind, _ := t.Value(*r, "body_kind")
			if roots[string(kind.Data)][string(id.Data)] && string(bodyKind.Data) == "markdown" {
				r.Disposition = "migrated"
			}
		}
	}
	return c, nil
}

// Only actual continuity inputs count as migrated. Everything else is retained
// as recovery evidence, including local work that still needs tracker disposition.
func retiredRowDisposition(t retired.Table, r retired.Row) string {
	switch t.Name {
	case "journal_entries", "journal_origins", "handoffs", "ideas", "sparks", "explorations", "exploration_checkpoints", "exploration_checkpoint_items", "issue_criterion_claims":
		return "migrated"
	case "artifact_bodies":
		// Body content may be mapped, but this supporting row (including source
		// hashes and metadata) is preserved only in the recovery archive.
	case "backend_mappings":
		kind, _ := t.Value(r, "entity_kind")
		locator, _ := t.Value(r, "external_url")
		if string(kind.Data) == "issue" {
			if _, ok := trackerLocatorV2(string(locator.Data)); ok {
				return "migrated"
			}
		}
	case "facts":
		kind, _ := t.Value(r, "kind")
		if string(kind.Data) == FactKindJournal || string(kind.Data) == FactKindHandoffRecorded {
			return "migrated"
		}
		for _, k := range append(sparkFactKinds(), ideaFactKinds()...) {
			if string(kind.Data) == k {
				return "migrated"
			}
		}
	}
	return "retired"
}

func sealRetiredArchive(c retired.Content, a archive.Archive) ([]byte, error) {
	c.BackupSHA256 = a.Content.Source.BackupSHA256
	c.SchemaVersion = a.Content.Source.LegacySchemaVersion
	c.ContinuitySHA256 = a.ContentSHA256
	// Primary retained tables have independent semantic source counts. Compare
	// them before sealing physical accounting so a dropped source cannot pass.
	counts := map[string]int{}
	for _, k := range a.Content.Source.Kinds {
		counts[k.Kind] = k.SourceRows
	}
	counts["journal_origins"] = a.Content.Source.JournalOriginRows
	for _, t := range c.Tables {
		if n, ok := counts[t.Name]; ok && n != t.SourceRows {
			return nil, fmt.Errorf("continuity/retired source count mismatch for %s", t.Name)
		}
	}
	return retired.Seal(c)
}
