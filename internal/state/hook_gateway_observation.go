package state

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// HookGatewayObservation contains only bounded classifications. Callers must
// never pass tool arguments, file contents, or a policy message to this API.
type HookGatewayObservation struct {
	HookID          string
	ToolName        string
	Decision        string
	ReasonClass     string
	ObservedHarness string
	HarnessVersion  string
}

type HookGatewayStatus struct {
	ProjectID       string `json:"project_id"`
	ObservedHarness string `json:"observed_harness"`
	Decision        string `json:"decision"`
	ReasonClass     string `json:"reason_class"`
	Count           int64  `json:"count"`
}

func (s *Store) RecordHookGatewayObservation(ctx context.Context, projectID string, observation HookGatewayObservation) error {
	if observation.HookID == "" || observation.ToolName == "" || observation.ReasonClass == "" || observation.ObservedHarness == "" || observation.HarnessVersion == "" {
		return fmt.Errorf("hook gateway observation requires all classifications")
	}
	if !validGatewayToken(observation.HookID) || !validGatewayToken(observation.ObservedHarness) || len(observation.HarnessVersion) > 64 {
		return fmt.Errorf("hook gateway observation has invalid classifier")
	}
	switch observation.ToolName {
	case "unknown", "Bash", "Shell", "Edit", "StrReplace", "Write", "Read", "Glob", "Grep", "WebFetch", "WebSearch", "Task":
	default:
		return fmt.Errorf("hook gateway observation has invalid tool name")
	}
	switch observation.ReasonClass {
	case "passed", "policy", "invalid-payload", "advisory":
	default:
		return fmt.Errorf("hook gateway observation has invalid reason class")
	}
	switch observation.Decision {
	case "allow", "warn", "block":
	default:
		return fmt.Errorf("hook gateway observation has invalid decision")
	}
	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return fmt.Errorf("mint hook gateway observation ID: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO hook_events (
  id, project_id, hook_id, hook_phase, tool_name, decision,
  reason_class, observed_harness, observed_harness_version, created_at, updated_at
) VALUES (?, ?, ?, 'pre-tool', ?, ?, ?, ?, ?, ?, ?)
`, hex.EncodeToString(idBytes[:]), projectID, observation.HookID, observation.ToolName,
		observation.Decision, observation.ReasonClass, observation.ObservedHarness,
		observation.HarnessVersion, now, now)
	if err != nil {
		return fmt.Errorf("record hook gateway observation: %w", err)
	}
	return nil
}

func validGatewayToken(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' {
			continue
		}
		return false
	}
	return true
}

// HookGatewayStatusSince is a local aggregate and never returns hook payloads.
func (s *Store) HookGatewayStatusSince(ctx context.Context, since time.Time) ([]HookGatewayStatus, error) {
	// The UTC second prefix includes both exact-second and fractional RFC3339
	// timestamps at the boundary; comparing to an exact "...Z" would miss them.
	rows, err := s.db.QueryContext(ctx, `
SELECT project_id, observed_harness, decision, reason_class, COUNT(*)
FROM hook_events
WHERE hook_phase = 'pre-tool' AND decision IS NOT NULL AND reason_class IS NOT NULL AND created_at >= ?
GROUP BY project_id, observed_harness, decision, reason_class
ORDER BY project_id, observed_harness, decision, reason_class
`, since.UTC().Format("2006-01-02T15:04:05"))
	if err != nil {
		return nil, fmt.Errorf("query hook gateway status: %w", err)
	}
	defer rows.Close()
	var status []HookGatewayStatus
	for rows.Next() {
		var item HookGatewayStatus
		if err := rows.Scan(&item.ProjectID, &item.ObservedHarness, &item.Decision, &item.ReasonClass, &item.Count); err != nil {
			return nil, fmt.Errorf("scan hook gateway status: %w", err)
		}
		status = append(status, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read hook gateway status: %w", err)
	}
	return status, nil
}
