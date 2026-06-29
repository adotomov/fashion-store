-- +goose Up
-- Stores only display metadata for a saved payment method — brand, last 4
-- digits, and expiry. Never the full card number or CVV; a real integration
-- would store a processor token (e.g. Stripe payment_method_id) here instead.
CREATE TABLE payment_methods (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	brand TEXT NOT NULL,
	last4 TEXT NOT NULL,
	exp_month INT NOT NULL,
	exp_year INT NOT NULL,
	is_default BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX payment_methods_user_id_idx ON payment_methods (user_id);

-- +goose Down
DROP TABLE payment_methods;
