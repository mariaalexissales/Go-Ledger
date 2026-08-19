-- Rows deleted by the up migration are not recoverable; this only restores the
-- constraints.
ALTER TABLE transactions ALTER COLUMN timestamp  DROP NOT NULL;
ALTER TABLE accounts     ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE transactions ALTER COLUMN account_id DROP NOT NULL;
