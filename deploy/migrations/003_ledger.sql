-- Migration 003: Double-Entry Ledger برای botpay

CREATE TABLE IF NOT EXISTS ledger_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    transaction_id  UUID NOT NULL,
    wallet_id       UUID NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('debit', 'credit')),
    amount_nano     BIGINT NOT NULL CHECK (amount_nano > 0),
    balance_after   BIGINT NOT NULL,
    ref             TEXT,
    note            TEXT
);

CREATE INDEX IF NOT EXISTS idx_ledger_wallet    ON ledger_entries(wallet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_tx        ON ledger_entries(transaction_id);
CREATE INDEX IF NOT EXISTS idx_ledger_type      ON ledger_entries(type);

-- Audit log
CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    actor_id    UUID NOT NULL,
    actor_role  TEXT,
    target_id   TEXT,
    target_type TEXT,
    action      TEXT NOT NULL,
    old_value   JSONB,
    new_value   JSONB,
    ip_addr     TEXT,
    meta        JSONB
);

CREATE INDEX IF NOT EXISTS idx_audit_actor   ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_action  ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_target  ON audit_logs(target_id, target_type);
