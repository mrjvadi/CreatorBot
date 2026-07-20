-- idempotency برای عملیات مالی داخلی retryپذیر.
CREATE UNIQUE INDEX IF NOT EXISTS uq_transactions_wallet_service_ref_type
ON transactions (wallet_id, service_id, ref, type)
WHERE type = 'credit_add' AND ref IS NOT NULL AND ref <> '' AND service_id IS NOT NULL AND service_id <> '';
