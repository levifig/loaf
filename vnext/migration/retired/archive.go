// Package retired preserves original SQLite values outside the destination fact
// store. Checksums detect corruption, not malicious replacement of an archive.
package retired

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
)

const Format = "loaf-retired-records-archive"
const Version = 1
const MaxEncodedBytes = 128 << 20

// Cell preserves SQLite's storage class. Text/blob data is base64 in JSON,
// integer data is decimal int64, and real data is eight IEEE-754 bytes (big endian).
type Cell struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

type Row struct {
	Disposition string `json:"disposition"`
	Values      []Cell `json:"values"`
}

type Table struct {
	Absent     bool     `json:"absent,omitempty"`
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	SourceRows int      `json:"source_rows"`
	Rows       []Row    `json:"rows"`
}

type Content struct {
	ProjectID        string  `json:"project_id"`
	BackupSHA256     string  `json:"backup_sha256"`
	ContinuitySHA256 string  `json:"continuity_sha256"`
	SchemaVersion    int     `json:"schema_version"`
	Tables           []Table `json:"tables"`
}

// TableCount accounts for physical source rows, not destination fact counts.
// Migrated means represented by the continuity export, possibly with declared
// effective-state mappings. Every original row is preserved here as evidence.
type TableCount struct {
	Absent       bool   `json:"absent,omitempty"`
	Table        string `json:"table"`
	SourceRows   int    `json:"source_rows"`
	MigratedRows int    `json:"migrated_rows"`
	RetiredRows  int    `json:"retired_rows"`
	SHA256       string `json:"sha256"`
}

// Manifest is reusable by rehearsal and later activation receipts. It contains
// no row content and makes no claim that open local work has been dispositioned.
type Manifest struct {
	ProjectID        string       `json:"project_id"`
	BackupSHA256     string       `json:"backup_sha256"`
	ContinuitySHA256 string       `json:"continuity_sha256"`
	ContentSHA256    string       `json:"content_sha256"`
	Tables           []TableCount `json:"tables"`
}

type Archive struct {
	Format   string   `json:"format"`
	Version  int      `json:"version"`
	Content  Content  `json:"content"`
	Manifest Manifest `json:"manifest"`
}

func digest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func checksum(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == sha256.Size && hex.EncodeToString(b) == s
}

func manifest(c Content) (Manifest, error) {
	m := Manifest{ProjectID: c.ProjectID, BackupSHA256: c.BackupSHA256, ContinuitySHA256: c.ContinuitySHA256, Tables: []TableCount{}}
	if c.ProjectID == "" || !checksum(c.BackupSHA256) || !checksum(c.ContinuitySHA256) || c.SchemaVersion < 1 || len(c.Tables) == 0 || len(c.Tables) > 100 {
		return m, fmt.Errorf("invalid retired archive source identity or inventory")
	}
	previous := ""
	total := 0
	for _, t := range c.Tables {
		if t.Absent {
			if t.Name <= previous || (t.Name != "sessions" && t.Name != "session_state_snapshots") || len(t.Columns) != 0 || t.SourceRows != 0 || len(t.Rows) != 0 {
				return m, fmt.Errorf("invalid absent legacy table")
			}
			previous = t.Name
			h, err := digest(t)
			if err != nil {
				return m, err
			}
			m.Tables = append(m.Tables, TableCount{Table: t.Name, Absent: true, SHA256: h})
			continue
		}
		if t.Name <= previous || len(t.Columns) == 0 || len(t.Columns) > 100 || t.SourceRows != len(t.Rows) {
			return m, fmt.Errorf("table inventory or source count does not reconcile: %s", t.Name)
		}
		previous = t.Name
		columns := map[string]bool{}
		projectColumn := -1
		for i, col := range t.Columns {
			if col == "" || columns[col] {
				return m, fmt.Errorf("invalid columns in %s", t.Name)
			}
			columns[col] = true
			if col == "project_id" {
				projectColumn = i
			}
		}
		if projectColumn < 0 {
			return m, fmt.Errorf("table %s has no project identity", t.Name)
		}
		count := TableCount{Table: t.Name, SourceRows: t.SourceRows}
		last := ""
		for _, r := range t.Rows {
			total++
			if total > 200_000 || len(r.Values) != len(t.Columns) {
				return m, fmt.Errorf("retired archive row bounds exceeded")
			}
			switch r.Disposition {
			case "migrated":
				count.MigratedRows++
			case "retired":
				count.RetiredRows++
			default:
				return m, fmt.Errorf("unknown row disposition")
			}
			p := r.Values[projectColumn]
			if p.Type != "text" || string(p.Data) != c.ProjectID {
				return m, fmt.Errorf("cross-project archive row")
			}
			for _, cell := range r.Values {
				switch cell.Type {
				case "null":
					if cell.Data != nil {
						return m, fmt.Errorf("NULL cell has data")
					}
				case "text", "blob":
					if cell.Data == nil {
						return m, fmt.Errorf("non-NULL byte cell has null data")
					}
				case "integer":
					n, err := strconv.ParseInt(string(cell.Data), 10, 64)
					if err != nil || strconv.FormatInt(n, 10) != string(cell.Data) {
						return m, fmt.Errorf("invalid integer cell")
					}
				case "real":
					if len(cell.Data) != 8 {
						return m, fmt.Errorf("invalid real cell")
					}
				default:
					return m, fmt.Errorf("unknown SQLite storage class")
				}
			}
			encoded, err := json.Marshal(r)
			if err != nil {
				return m, err
			}
			if string(encoded) <= last {
				return m, fmt.Errorf("duplicate or unordered rows in %s", t.Name)
			}
			last = string(encoded)
		}
		var err error
		count.SHA256, err = digest(t)
		if err != nil {
			return m, err
		}
		m.Tables = append(m.Tables, count)
	}
	var err error
	m.ContentSHA256, err = digest(c)
	return m, err
}

// Seal sorts rows and seals all original cells and counts in one canonical file.
func Seal(c Content) ([]byte, error) {
	sort.Slice(c.Tables, func(i, j int) bool { return c.Tables[i].Name < c.Tables[j].Name })
	for i := range c.Tables {
		type orderedRow struct {
			row     Row
			encoded []byte
		}
		ordered := make([]orderedRow, len(c.Tables[i].Rows))
		for j, r := range c.Tables[i].Rows {
			encoded, err := json.Marshal(r)
			if err != nil {
				return nil, err
			}
			ordered[j] = orderedRow{r, encoded}
		}
		sort.Slice(ordered, func(i, j int) bool { return bytes.Compare(ordered[i].encoded, ordered[j].encoded) < 0 })
		for j, r := range ordered {
			c.Tables[i].Rows[j] = r.row
		}
	}
	m, err := manifest(c)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(Archive{Format: Format, Version: Version, Content: c, Manifest: m})
	if err != nil {
		return nil, err
	}
	if len(b)+1 > MaxEncodedBytes {
		return nil, fmt.Errorf("retired archive exceeds encoded byte limit")
	}
	return append(b, '\n'), nil
}

// Parse verifies the whole manifest before exposing any content. Requiring the
// exact canonical encoding rejects changes to whitespace and duplicate keys too.
func Parse(encoded []byte) (Archive, error) {
	var a Archive
	if len(encoded) > MaxEncodedBytes {
		return a, fmt.Errorf("retired archive exceeds encoded byte limit")
	}
	d := json.NewDecoder(bytes.NewReader(encoded))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return Archive{}, fmt.Errorf("decode retired archive: %w", err)
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return Archive{}, fmt.Errorf("trailing retired archive data")
	}
	if a.Format != Format || a.Version != Version {
		return Archive{}, fmt.Errorf("unsupported retired archive format/version")
	}
	m, err := manifest(a.Content)
	if err != nil {
		return Archive{}, err
	}
	want, err := json.Marshal(Archive{Format: Format, Version: Version, Content: a.Content, Manifest: m})
	if err != nil {
		return Archive{}, err
	}
	if !bytes.Equal(encoded, append(want, '\n')) {
		return Archive{}, fmt.Errorf("retired archive checksum or canonical bytes mismatch")
	}
	return a, nil
}

func (a Archive) Record(table, id string) (Row, error) {
	var found *Row
	for _, t := range a.Content.Tables {
		if t.Name != table {
			continue
		}
		for _, r := range t.Rows {
			if v, ok := t.Value(r, "id"); ok && v.Type == "text" && string(v.Data) == id {
				if found != nil {
					return Row{}, fmt.Errorf("ambiguous legacy id")
				}
				copy := r
				found = &copy
			}
		}
	}
	if found == nil {
		return Row{}, fmt.Errorf("legacy record not found")
	}
	return *found, nil
}

func (t Table) Value(r Row, column string) (Cell, bool) {
	for i, name := range t.Columns {
		if name == column && i < len(r.Values) {
			return r.Values[i], true
		}
	}
	return Cell{}, false
}
