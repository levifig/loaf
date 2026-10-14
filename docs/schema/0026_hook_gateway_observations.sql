-- Machine-local, content-free observations for pre-tool gateway evaluations.
-- Historic hook rows retain their nullable legacy columns and remain readable.
ALTER TABLE hook_events ADD COLUMN tool_name TEXT;
ALTER TABLE hook_events ADD COLUMN decision TEXT CHECK (decision IN ('allow', 'warn', 'block'));
ALTER TABLE hook_events ADD COLUMN reason_class TEXT;
ALTER TABLE hook_events ADD COLUMN observed_harness TEXT;
ALTER TABLE hook_events ADD COLUMN observed_harness_version TEXT;

CREATE INDEX IF NOT EXISTS idx_hook_events_observation
  ON hook_events (project_id, created_at, observed_harness, decision);
