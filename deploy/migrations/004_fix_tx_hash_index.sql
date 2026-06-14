-- Migration 004: partial unique index برای tx_hash
-- فقط وقتی tx_hash خالی نیست unique باشد

BEGIN;

DROP INDEX IF EXISTS idx_transactions_tx_hash;

-- partial index: فقط ردیف‌هایی که tx_hash مقدار دارند
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_tx_hash_nonempty
    ON transactions(tx_hash)
    WHERE tx_hash IS NOT NULL AND tx_hash != '';

-- index عادی برای جستجو
CREATE INDEX IF NOT EXISTS idx_transactions_tx_hash
    ON transactions(tx_hash);

COMMIT;

SELECT 'Migration 004 completed' AS result;
