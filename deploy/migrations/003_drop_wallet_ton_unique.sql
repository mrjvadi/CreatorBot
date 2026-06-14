-- Migration 003: حذف unique constraint از wallets.ton_address
-- چون همه کاربرها یک masterAddr مشترک دارند

BEGIN;

DROP INDEX IF EXISTS idx_wallets_ton_address;
CREATE INDEX IF NOT EXISTS idx_wallets_ton_address ON wallets(ton_address);

COMMIT;

SELECT 'Migration 003 completed' AS result;
