-- +goose Up
-- Reshapes the order/payment schema for the asynchronous Revolut card flow:
-- an order now lives in 'pending_payment' while the customer pays via the
-- embedded widget, and is only moved to 'paid' by the verified
-- ORDER_COMPLETED webhook. New refund states support in-app admin refunds.
-- Payments also gain the provider order id (the webhook lookup key) plus
-- captured/refunded running totals.

ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (
	status IN (
		'pending',
		'pending_payment',
		'paid',
		'payment_failed',
		'shipped',
		'delivered',
		'cancelled',
		'refunded',
		'partially_refunded'
	)
);

ALTER TABLE order_payments DROP CONSTRAINT order_payments_status_check;
ALTER TABLE order_payments ADD CONSTRAINT order_payments_status_check CHECK (
	status IN (
		'pending',
		'authorised',
		'succeeded',
		'failed',
		'cancelled',
		'refunded',
		'partially_refunded'
	)
);

ALTER TABLE order_payments
	ADD COLUMN provider_order_id TEXT,
	ADD COLUMN captured_minor BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN refunded_minor BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- provider_order_id is the Revolut order id; it uniquely identifies a payment
-- so an out-of-order or duplicated webhook resolves to exactly one row. NULL
-- for the legacy mock/non-card rows, so the uniqueness is partial.
CREATE UNIQUE INDEX order_payments_provider_order_id_idx
	ON order_payments (provider_order_id)
	WHERE provider_order_id IS NOT NULL;

-- +goose Down
DROP INDEX order_payments_provider_order_id_idx;

ALTER TABLE order_payments
	DROP COLUMN provider_order_id,
	DROP COLUMN captured_minor,
	DROP COLUMN refunded_minor,
	DROP COLUMN updated_at;

ALTER TABLE order_payments DROP CONSTRAINT order_payments_status_check;
ALTER TABLE order_payments ADD CONSTRAINT order_payments_status_check CHECK (
	status IN ('succeeded', 'failed')
);

ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (
	status IN ('pending', 'paid', 'shipped', 'delivered', 'cancelled')
);
