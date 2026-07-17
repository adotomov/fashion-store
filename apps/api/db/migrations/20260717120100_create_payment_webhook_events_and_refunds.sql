-- +goose Up
-- payment_webhook_events is the idempotency ledger for inbound Revolut
-- webhooks. The handler inserts the event id first (ON CONFLICT DO NOTHING);
-- a conflict means we've already processed it and can skip finalization, so a
-- redelivered or duplicated webhook can never double-commit stock or
-- double-book a shipment.
CREATE TABLE payment_webhook_events (
	id TEXT PRIMARY KEY, -- Revolut event id
	event_type TEXT NOT NULL,
	provider_order_id TEXT,
	payload JSONB NOT NULL,
	received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX payment_webhook_events_provider_order_id_idx
	ON payment_webhook_events (provider_order_id);

-- order_refunds records each admin-initiated refund against an order. state is
-- driven by the refund webhook: 'pending' when we call Revolut, 'completed'
-- once confirmed. amount_minor supports partial refunds; the order's rolled-up
-- state (refunded / partially_refunded) is derived from the sum of completed
-- refunds vs the captured total.
CREATE TABLE order_refunds (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
	provider_refund_id TEXT UNIQUE,
	amount_minor BIGINT NOT NULL,
	currency TEXT NOT NULL,
	reason TEXT,
	state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'completed', 'failed')),
	created_by UUID REFERENCES users (id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX order_refunds_order_id_idx ON order_refunds (order_id);

-- +goose Down
DROP TABLE order_refunds;
DROP TABLE payment_webhook_events;
