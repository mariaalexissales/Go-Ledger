-- Every column below backs a filter or sort on a list endpoint. security_events
-- in particular grows fast once the demo scenarios start firing.
CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions (account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions (timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_security_events_timestamp ON security_events (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_flag_ts ON security_events (flag_status, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_ip_ts ON security_events (ip_address, timestamp DESC);
