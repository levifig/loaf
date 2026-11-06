-- Machine-local, content-free pre-tool gateway outcomes, counted per UTC month.
-- One row per tool call grew without bound and slowed every state open, so
-- existing per-call observations fold into monthly counts and leave hook_events.
-- Historic hook rows without gateway classifications stay untouched.
CREATE TABLE IF NOT EXISTS hook_gateway_counts (
  project_id TEXT NOT NULL,
  month TEXT NOT NULL,
  hook_id TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  decision TEXT NOT NULL CHECK (decision IN ('allow', 'warn', 'block')),
  reason_class TEXT NOT NULL,
  observed_harness TEXT NOT NULL,
  observed_harness_version TEXT NOT NULL,
  count INTEGER NOT NULL CHECK (count > 0),
  PRIMARY KEY (project_id, month, hook_id, tool_name, decision, reason_class, observed_harness, observed_harness_version),
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

INSERT INTO hook_gateway_counts (
  project_id, month, hook_id, tool_name, decision, reason_class,
  observed_harness, observed_harness_version, count
)
SELECT project_id, substr(created_at, 1, 7), hook_id, tool_name, decision, reason_class,
       observed_harness, observed_harness_version, COUNT(*)
FROM hook_events
WHERE hook_phase = 'pre-tool' AND tool_name IS NOT NULL AND decision IS NOT NULL
  AND reason_class IS NOT NULL AND observed_harness IS NOT NULL AND observed_harness_version IS NOT NULL
GROUP BY project_id, substr(created_at, 1, 7), hook_id, tool_name, decision, reason_class,
         observed_harness, observed_harness_version;

-- Drop the per-call index first so the delete does not maintain it.
DROP INDEX IF EXISTS idx_hook_events_observation;

DELETE FROM hook_events
WHERE hook_phase = 'pre-tool' AND tool_name IS NOT NULL AND decision IS NOT NULL
  AND reason_class IS NOT NULL AND observed_harness IS NOT NULL AND observed_harness_version IS NOT NULL;
