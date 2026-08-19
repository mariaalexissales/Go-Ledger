-- Three columns were created nullable but are scanned into non-pointer Go
-- fields (api.Account.CreatedAt, api.Transaction.AccountID/Timestamp), so a
-- NULL row fails at read time instead of write time.
--
-- Backfill first. SET NOT NULL fails outright if any row violates it, and a
-- failed migration leaves golang-migrate's schema_migrations row marked dirty,
-- which blocks every later boot until someone clears it by hand.

-- The one destructive statement in the migration set. A transaction with no
-- account_id references nothing and cannot be repaired, only dropped. The
-- foreign key has always been there, so this is empty on any database that was
-- only ever written to through the API.
DELETE FROM transactions WHERE account_id IS NULL;

UPDATE accounts     SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL;
UPDATE transactions SET timestamp  = CURRENT_TIMESTAMP WHERE timestamp  IS NULL;

ALTER TABLE transactions ALTER COLUMN account_id SET NOT NULL;
ALTER TABLE accounts     ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE transactions ALTER COLUMN timestamp  SET NOT NULL;
