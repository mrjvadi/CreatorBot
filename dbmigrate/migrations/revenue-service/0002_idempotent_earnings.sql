-- هر ref غیرخالی فقط یک earning را نمایندگی می‌کند.
CREATE UNIQUE INDEX IF NOT EXISTS uq_earnings_ref_id_nonempty
ON earnings (ref_id) WHERE ref_id IS NOT NULL AND ref_id <> '';
