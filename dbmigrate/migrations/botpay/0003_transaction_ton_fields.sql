-- داده‌ی خامِ on-chain برای واریزهای TON: logical time و unix time تراکنش.
-- (fee/from/to از قبل روی جدول transactions موجودند.) برای تراکنش‌های داخلی ۰ می‌مانند.
-- AutoMigrate هم این ستون‌ها را additive می‌سازد؛ این migration نسخه‌دارِ رسمی است.
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS tx_lt BIGINT NOT NULL DEFAULT 0;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS tx_utime BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_transactions_tx_lt ON transactions (tx_lt);
