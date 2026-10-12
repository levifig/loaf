package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/levifig/loaf/internal/project"
	"github.com/levifig/loaf/internal/state"
)

type telemetryStatusOutput struct {
	ContractVersion int                       `json:"contract_version"`
	Since           string                    `json:"since"`
	Rows            []state.HookGatewayStatus `json:"rows"`
}

func (r Runner) runTelemetry(args []string, out io.Writer, runtimeRoot string) error {
	if isHelpArg(args) {
		fmt.Fprintln(out, "Usage: loaf telemetry status [--json]")
		fmt.Fprintln(out, "Show local pre-tool gateway outcomes since the start of the current UTC month.")
		return nil
	}
	if len(args) < 1 || args[0] != "status" || len(args) > 2 || (len(args) == 2 && args[1] != "--json") {
		return fmt.Errorf("usage: loaf telemetry status [--json]")
	}
	root, err := project.ResolveRoot(runtimeRoot)
	if err != nil {
		return err
	}
	path, err := (state.PathResolver{StateHome: r.StateHome}).DatabasePath(root)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("local state database unavailable: %w", err)
	}
	store, err := state.OpenStoreReadOnly(path)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.RequireCurrentSchema(context.Background()); err != nil {
		return err
	}
	now := time.Now().UTC()
	since := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	rows, err := store.HookGatewayStatusSince(context.Background(), since)
	if err != nil {
		return err
	}
	if rows == nil {
		rows = []state.HookGatewayStatus{}
	}
	status := telemetryStatusOutput{ContractVersion: 1, Since: since.Format(time.RFC3339), Rows: rows}
	if len(args) == 2 {
		return writeJSON(out, status)
	}
	fmt.Fprintf(out, "Pre-tool gateway outcomes since %s (UTC):\n", since.Format("2006-01-02"))
	if len(rows) == 0 {
		fmt.Fprintln(out, "  No observations")
		return nil
	}
	for _, row := range rows {
		fmt.Fprintf(out, "  %s  %s  %s  %s  %d\n", row.ProjectID, row.ObservedHarness, row.Decision, row.ReasonClass, row.Count)
	}
	return nil
}
