-- +goose Up
-- payment_transactions is an append-only audit ledger of every payment
-- lifecycle event: a Revolut order opened (initiated), a completed payment
-- captured, a payment failed/cancelled, or a refund. Each row is written in the
-- SAME transaction as the order_payments state change that produced it, so the
-- ledger can never drift from the mutable payment row. Rows are never updated
-- or deleted — history is immutable.
CREATE TABLE payment_transactions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
	provider TEXT NOT NULL,
	provider_order_id TEXT,
	provider_reference TEXT,
	type TEXT NOT NULL CHECK (type IN ('initiated', 'captured', 'failed', 'refunded')),
	status TEXT,
	amount_minor BIGINT NOT NULL,
	currency TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX payment_transactions_order_id_idx ON payment_transactions (order_id);
CREATE INDEX payment_transactions_provider_order_id_idx ON payment_transactions (provider_order_id);

-- +goose Down
DROP TABLE payment_transactions;
