package state

import (
	"context"
	"fmt"
	"time"
)

// gatewayMonthLayout keys gateway counts by UTC month, matching the
// substr(created_at, 1, 7) fold of per-call rows in migration 0028.
const gatewayMonthLayout = "2006-01"

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
	_, err := s.db.ExecContext(ctx, `
INSERT INTO hook_gateway_counts (
  project_id, month, hook_id, tool_name, decision, reason_class,
  observed_harness, observed_harness_version, count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
ON CONFLICT (project_id, month, hook_id, tool_name, decision, reason_class, observed_harness, observed_harness_version)
DO UPDATE SET count = count + 1
`, projectID, time.Now().UTC().Format(gatewayMonthLayout), observation.HookID, observation.ToolName,
		observation.Decision, observation.ReasonClass, observation.ObservedHarness,
		observation.HarnessVersion)
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

// HookGatewayStatusForMonth totals the UTC month containing month. It is a
// local aggregate and never returns hook payloads.
func (s *Store) HookGatewayStatusForMonth(ctx context.Context, month time.Time) ([]HookGatewayStatus, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT project_id, observed_harness, decision, reason_class, SUM(count)
FROM hook_gateway_counts
WHERE month = ?
GROUP BY project_id, observed_harness, decision, reason_class
ORDER BY project_id, observed_harness, decision, reason_class
`, month.UTC().Format(gatewayMonthLayout))
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
