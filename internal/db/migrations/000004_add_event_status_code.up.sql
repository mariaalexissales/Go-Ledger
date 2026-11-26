-- Nullable: rows from before this migration have no status to backfill.
ALTER TABLE security_events ADD COLUMN IF NOT EXISTS status_code SMALLINT;

CREATE INDEX IF NOT EXISTS idx_security_events_status_ts ON security_events (status_code, timestamp DESC);
