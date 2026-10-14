package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/levifig/loaf/vnext/migration/retired"
)

func writeMigrateRetiredHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: loaf migrate retired-show --archive <ABSOLUTE-FILE> --kind <spec|intent|task|brainstorm|plan|work-contract|issue> --id <LEGACY-ID> [--body [--body-id <BODY-OR-SNAPSHOT-ID>]]")
	fmt.Fprintln(out, "Verify an entire retired archive, then print the original typed row and bodies as JSON. --body writes one body's original bytes (refuses missing/ambiguous bodies). No database or login is required.")
}

func (r Runner) runMigrateRetired(args []string, out io.Writer) error {
	values := map[string]string{}
	bodyOnly := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--body" {
			if bodyOnly {
				return fmt.Errorf("duplicate --body")
			}
			bodyOnly = true
			continue
		}
		if arg != "--archive" && arg != "--kind" && arg != "--id" && arg != "--body-id" {
			return fmt.Errorf("unknown retired-show option %q", arg)
		}
		if values[arg] != "" || i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			return fmt.Errorf("%s requires exactly one value", arg)
		}
		i++
		values[arg] = args[i]
	}
	for _, key := range []string{"--archive", "--kind", "--id"} {
		if values[key] == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	if values["--body-id"] != "" && !bodyOnly {
		return fmt.Errorf("--body-id requires --body")
	}
	if err := requireAbsoluteCleanPath("archive", values["--archive"]); err != nil {
		return err
	}
	tables := map[string]string{"spec": "specs", "intent": "intents", "task": "tasks", "brainstorm": "brainstorms", "plan": "plans", "work-contract": "work_contracts", "issue": "issues"}
	table, ok := tables[values["--kind"]]
	if !ok {
		return fmt.Errorf("unsupported retired kind %q", values["--kind"])
	}
	info, err := os.Lstat(values["--archive"])
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > retired.MaxEncodedBytes {
		return fmt.Errorf("archive must be a bounded regular file, not a symlink")
	}
	file, err := os.Open(values["--archive"])
	if err != nil {
		return err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return errors.Join(fmt.Errorf("archive identity changed"), err, file.Close())
	}
	encoded, readErr := io.ReadAll(io.LimitReader(file, retired.MaxEncodedBytes+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return err
	}
	a, err := retired.Parse(encoded)
	if err != nil {
		return err
	}
	row, err := a.Record(table, values["--id"])
	if err != nil {
		return err
	}
	var selected retired.Table
	bodies := []retired.Row{}
	var body []byte
	bodyCount := 0
	for _, t := range a.Content.Tables {
		if t.Name == table {
			selected = t
		}
		if t.Name != "artifact_bodies" {
			continue
		}
		for _, candidate := range t.Rows {
			kind, _ := t.Value(candidate, "entity_kind")
			id, _ := t.Value(candidate, "entity_id")
			if (string(kind.Data) != values["--kind"] && string(kind.Data) != strings.ReplaceAll(values["--kind"], "-", "_")) || string(id.Data) != values["--id"] {
				continue
			}
			bodies = append(bodies, candidate)
			bodyID, _ := t.Value(candidate, "id")
			if values["--body-id"] != "" && string(bodyID.Data) != values["--body-id"] {
				continue
			}
			bodyCount++
			content, _ := t.Value(candidate, "content")
			body = content.Data
		}
	}
	related := retiredRelatedTables(a, values["--kind"], values["--id"])
	if values["--kind"] == "intent" {
		for _, t := range related {
			if t.Name == "intent_snapshots" {
				for _, candidate := range t.Rows {
					bodyID, _ := t.Value(candidate, "id")
					if values["--body-id"] != "" && string(bodyID.Data) != values["--body-id"] {
						continue
					}
					bodyCount++
					content, _ := t.Value(candidate, "body")
					body = content.Data
				}
			}
		}
	}
	if bodyOnly {
		if bodyCount != 1 {
			return fmt.Errorf("expected exactly one body or intent snapshot, found %d; use typed JSON and --body-id to select preserved bytes", bodyCount)
		}
		n, err := out.Write(body)
		if err == nil && n != len(body) {
			err = io.ErrShortWrite
		}
		return err
	}
	return writeJSON(out, struct {
		ProjectID   string          `json:"project_id"`
		Table       string          `json:"table"`
		Columns     []string        `json:"columns"`
		Record      retired.Row     `json:"record"`
		Bodies      []retired.Row   `json:"artifact_bodies"`
		BodyColumns []string        `json:"artifact_body_columns"`
		Related     []retired.Table `json:"related_tables"`
	}{a.Content.ProjectID, table, selected.Columns, row, bodies, artifactBodyColumns(a), related})
}

// Recover direct histories and metadata without traversing an unbounded graph
// or manufacturing a new current-state interpretation of retired work.
func retiredRelatedTables(a retired.Archive, kind, id string) []retired.Table {
	keys := map[string]string{}
	var contractProvider, contractRef string
	if kind == "work-contract" {
		root, err := a.Record("work_contracts", id)
		if err == nil {
			for _, t := range a.Content.Tables {
				if t.Name == "work_contracts" {
					provider, _ := t.Value(root, "provider")
					ref, _ := t.Value(root, "provider_ref")
					contractProvider = string(provider.Data)
					contractRef = string(ref.Data)
				}
			}
		}
	}
	switch kind {
	case "intent":
		for _, name := range []string{"intent_snapshots", "intent_deferrals", "intent_dispositions", "intent_operations"} {
			keys[name] = "intent_id"
		}
	case "issue":
		keys["issue_criteria"] = "issue_id"
	case "work-contract":
		for _, name := range []string{"work_contract_criteria", "work_contract_workspace"} {
			keys[name] = "contract_id"
		}
	case "spec":
		keys["tasks"] = "spec_id"
		keys["plans"] = "spec_id"
	}
	criteria := map[string]bool{}
	for _, t := range a.Content.Tables {
		if t.Name != "issue_criteria" && t.Name != "work_contract_criteria" {
			continue
		}
		key := keys[t.Name]
		if key == "" {
			continue
		}
		for _, r := range t.Rows {
			owner, _ := t.Value(r, key)
			if string(owner.Data) == id {
				criterion, _ := t.Value(r, "id")
				criteria[string(criterion.Data)] = true
			}
		}
	}
	result := []retired.Table{}
	for _, t := range a.Content.Tables {
		selected := retired.Table{Name: t.Name, Columns: t.Columns, Rows: []retired.Row{}}
		for _, r := range t.Rows {
			match := false
			if kind == "issue" && t.Name == "issue_identity" {
				match = true
			}
			if kind == "work-contract" && (t.Name == "work_contract_mappings" || t.Name == "work_contract_receipts") {
				provider, _ := t.Value(r, "provider")
				ref, _ := t.Value(r, "provider_ref")
				match = string(provider.Data) == contractProvider && string(ref.Data) == contractRef
			}
			if key := keys[t.Name]; key != "" {
				value, _ := t.Value(r, key)
				match = string(value.Data) == id
			}
			if (kind == "issue" && t.Name == "issue_criterion_claims") || (kind == "work-contract" && t.Name == "work_contract_criterion_claims") {
				child, _ := t.Value(r, "child_criterion_id")
				parent, _ := t.Value(r, "parent_criterion_id")
				match = criteria[string(child.Data)] || criteria[string(parent.Data)]
			}
			if t.Name == "aliases" || t.Name == "events" || t.Name == "entity_tags" || t.Name == "backend_mappings" {
				entityKind, _ := t.Value(r, "entity_kind")
				entityID, _ := t.Value(r, "entity_id")
				match = string(entityID.Data) == id && (string(entityKind.Data) == kind || string(entityKind.Data) == strings.ReplaceAll(kind, "-", "_"))
			}
			if match {
				selected.Rows = append(selected.Rows, r)
			}
		}
		if len(selected.Rows) > 0 {
			selected.SourceRows = len(selected.Rows)
			result = append(result, selected)
		}
	}
	return result
}

func artifactBodyColumns(a retired.Archive) []string {
	for _, t := range a.Content.Tables {
		if t.Name == "artifact_bodies" {
			return t.Columns
		}
	}
	return nil
}
