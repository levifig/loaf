package state

import (
	"context"
	"fmt"
	"time"
)

// ContinuityHarnessStatus counts persisted writes, not sessions or tool calls.
// Supporting rows/items belong to their owning write and are not counted twice.
type ContinuityHarnessStatus struct {
	ProjectID       string `json:"project_id"`
	ObservedHarness string `json:"observed_harness"`
	Count           int64  `json:"count"`
}

func (s *Store) ContinuityHarnessStatusSince(ctx context.Context, since time.Time) ([]ContinuityHarnessStatus, error) {
	boundary := since.UTC().Format("2006-01-02T15:04:05")
	rows, err := s.db.QueryContext(ctx, `
WITH observations AS (
 SELECT project_id, observed_harness, created_at FROM journal_origins
 UNION ALL
 SELECT project_id, json_extract(payload, '$.observed_harness'),
        COALESCE(NULLIF(json_extract(payload, '$.updated_at'), ''), json_extract(payload, '$.created_at'))
 FROM facts WHERE kind != 'journal'
 UNION ALL
 SELECT project_id, observed_harness, created_at FROM explorations
 UNION ALL
 SELECT project_id, observed_harness, created_at FROM exploration_checkpoints
)
SELECT project_id, observed_harness, COUNT(*) FROM observations
WHERE observed_harness IS NOT NULL AND observed_harness != '' AND created_at >= ?
GROUP BY project_id, observed_harness ORDER BY project_id, observed_harness
`, boundary)
	if err != nil {
		return nil, fmt.Errorf("query continuity harness status: %w", err)
	}
	defer rows.Close()
	status := []ContinuityHarnessStatus{}
	for rows.Next() {
		var item ContinuityHarnessStatus
		if err := rows.Scan(&item.ProjectID, &item.ObservedHarness, &item.Count); err != nil {
			return nil, fmt.Errorf("scan continuity harness status: %w", err)
		}
		status = append(status, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read continuity harness status: %w", err)
	}
	return status, nil
}
