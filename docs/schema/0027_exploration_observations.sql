-- New exploration writes carry explicit harness provenance. Existing rows
-- remain untouched; their missing provenance must never be inferred.
ALTER TABLE explorations ADD COLUMN observed_harness TEXT;
ALTER TABLE explorations ADD COLUMN observed_harness_version TEXT;
ALTER TABLE exploration_checkpoints ADD COLUMN observed_harness TEXT;
ALTER TABLE exploration_checkpoints ADD COLUMN observed_harness_version TEXT;
