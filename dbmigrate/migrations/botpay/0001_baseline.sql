-- 0001_baseline — schema کامل سرویس botpay (دیتابیس: botpay)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service botpay -version 1


-- TABLE: invoices

CREATE TABLE public.invoices (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    wallet_id text NOT NULL,
    code text NOT NULL,
    amount bigint NOT NULL,
    received_nano bigint DEFAULT 0,
    status text DEFAULT 'pending'::text,
    service_id text,
    ref text,
    metadata text,
    expires_at timestamp with time zone,
    paid_at timestamp with time zone,
    tx_hash text
);

-- TABLE: ledger_entries

CREATE TABLE public.ledger_entries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    transaction_id text NOT NULL,
    wallet_id text NOT NULL,
    type text NOT NULL,
    amount_nano bigint NOT NULL,
    balance_after bigint,
    seq bigint NOT NULL,
    prev_hash text,
    hash text,
    ref text,
    note text
);

-- SEQUENCE: ledger_entries_seq_seq

CREATE SEQUENCE public.ledger_entries_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- SEQUENCE: ledger_entries_seq_seq

ALTER SEQUENCE public.ledger_entries_seq_seq OWNED BY public.ledger_entries.seq;

-- TABLE: transactions

CREATE TABLE public.transactions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    wallet_id text NOT NULL,
    type text NOT NULL,
    status text DEFAULT 'pending'::text,
    amount bigint NOT NULL,
    fee bigint DEFAULT 0,
    tx_hash text,
    from_address text,
    to_address text,
    service_id text,
    ref text,
    description text,
    metadata text,
    confirmed_at timestamp with time zone
);

-- TABLE: wallets

CREATE TABLE public.wallets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    telegram_id bigint NOT NULL,
    pay_handle text,
    ton_balance bigint DEFAULT 0,
    credit bigint DEFAULT 0,
    ton_address text,
    frozen bigint DEFAULT 0,
    is_active boolean DEFAULT true,
    lang character varying(8)
);

-- TABLE: withdraw_requests

CREATE TABLE public.withdraw_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    wallet_id text NOT NULL,
    to_address text NOT NULL,
    amount bigint NOT NULL,
    fee bigint,
    status text DEFAULT 'pending'::text,
    tx_hash text,
    note text,
    admin_note text,
    processed_at timestamp with time zone
);

-- DEFAULT: ledger_entries seq

ALTER TABLE ONLY public.ledger_entries ALTER COLUMN seq SET DEFAULT nextval('public.ledger_entries_seq_seq'::regclass);

-- CONSTRAINT: invoices invoices_pkey

ALTER TABLE ONLY public.invoices
    ADD CONSTRAINT invoices_pkey PRIMARY KEY (id);

-- CONSTRAINT: ledger_entries ledger_entries_pkey

ALTER TABLE ONLY public.ledger_entries
    ADD CONSTRAINT ledger_entries_pkey PRIMARY KEY (id);

-- CONSTRAINT: transactions transactions_pkey

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);

-- CONSTRAINT: wallets wallets_pkey

ALTER TABLE ONLY public.wallets
    ADD CONSTRAINT wallets_pkey PRIMARY KEY (id);

-- CONSTRAINT: withdraw_requests withdraw_requests_pkey

ALTER TABLE ONLY public.withdraw_requests
    ADD CONSTRAINT withdraw_requests_pkey PRIMARY KEY (id);

-- INDEX: idx_invoices_code

CREATE UNIQUE INDEX idx_invoices_code ON public.invoices USING btree (code);

-- INDEX: idx_invoices_status

CREATE INDEX idx_invoices_status ON public.invoices USING btree (status);

-- INDEX: idx_invoices_wallet_id

CREATE INDEX idx_invoices_wallet_id ON public.invoices USING btree (wallet_id);

-- INDEX: idx_ledger_entries_hash

CREATE INDEX idx_ledger_entries_hash ON public.ledger_entries USING btree (hash);

-- INDEX: idx_ledger_entries_prev_hash

CREATE INDEX idx_ledger_entries_prev_hash ON public.ledger_entries USING btree (prev_hash);

-- INDEX: idx_ledger_entries_seq

CREATE UNIQUE INDEX idx_ledger_entries_seq ON public.ledger_entries USING btree (seq);

-- INDEX: idx_ledger_entries_transaction_id

CREATE INDEX idx_ledger_entries_transaction_id ON public.ledger_entries USING btree (transaction_id);

-- INDEX: idx_ledger_entries_wallet_id

CREATE INDEX idx_ledger_entries_wallet_id ON public.ledger_entries USING btree (wallet_id);

-- INDEX: idx_transactions_status

CREATE INDEX idx_transactions_status ON public.transactions USING btree (status);

-- INDEX: idx_transactions_tx_hash

CREATE INDEX idx_transactions_tx_hash ON public.transactions USING btree (tx_hash);

-- INDEX: idx_transactions_type

CREATE INDEX idx_transactions_type ON public.transactions USING btree (type);

-- INDEX: idx_transactions_wallet_id

CREATE INDEX idx_transactions_wallet_id ON public.transactions USING btree (wallet_id);

-- INDEX: idx_wallets_pay_handle

CREATE UNIQUE INDEX idx_wallets_pay_handle ON public.wallets USING btree (pay_handle);

-- INDEX: idx_wallets_telegram_id

CREATE UNIQUE INDEX idx_wallets_telegram_id ON public.wallets USING btree (telegram_id);

-- INDEX: idx_wallets_ton_address

CREATE INDEX idx_wallets_ton_address ON public.wallets USING btree (ton_address);

-- INDEX: idx_withdraw_requests_status

CREATE INDEX idx_withdraw_requests_status ON public.withdraw_requests USING btree (status);

-- INDEX: idx_withdraw_requests_wallet_id

CREATE INDEX idx_withdraw_requests_wallet_id ON public.withdraw_requests USING btree (wallet_id);

-- PostgreSQL database dump complete

