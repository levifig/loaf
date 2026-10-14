package retired

import (
	"bytes"
	"testing"
)

func TestArchivePreservesTypedBytesAndRejectsAnyChangedByte(t *testing.T) {
	c := Content{ProjectID: "project", BackupSHA256: string(bytes.Repeat([]byte("a"), 64)), ContinuitySHA256: string(bytes.Repeat([]byte("b"), 64)), SchemaVersion: 30,
		Tables: []Table{{Name: "specs", Columns: []string{"id", "project_id", "body", "empty", "blob", "nullable", "integer", "real"}, SourceRows: 1,
			Rows: []Row{{Disposition: "retired", Values: []Cell{{Type: "text", Data: []byte("spec")}, {Type: "text", Data: []byte("project")}, {Type: "text", Data: []byte{0xff, 0, 13, 10}}, {Type: "text", Data: []byte{}}, {Type: "blob", Data: []byte{}}, {Type: "null"}, {Type: "integer", Data: []byte("9223372036854775807")}, {Type: "real", Data: make([]byte, 8)}}}}}},
	}
	encoded, err := Seal(c)
	if err != nil {
		t.Fatal(err)
	}
	a, err := Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	r, err := a.Record("specs", "spec")
	if err != nil || !bytes.Equal(r.Values[2].Data, c.Tables[0].Rows[0].Values[2].Data) {
		t.Fatalf("record bytes: %v", err)
	}
	for i := range encoded {
		changed := bytes.Clone(encoded)
		changed[i] ^= 1
		if _, err := Parse(changed); err == nil {
			t.Fatalf("accepted changed byte %d", i)
		}
	}
	c.Tables[0].SourceRows++
	if _, err := Seal(c); err == nil {
		t.Fatal("accepted unreconciled source count")
	}
}
