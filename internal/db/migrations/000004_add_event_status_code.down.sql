DROP INDEX IF EXISTS idx_security_events_status_ts;

ALTER TABLE security_events DROP COLUMN IF EXISTS status_code;
