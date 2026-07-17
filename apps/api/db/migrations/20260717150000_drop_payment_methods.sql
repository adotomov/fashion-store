-- +goose Up
-- The saved-cards feature is removed: card payments now go through Revolut's
-- embedded widget (no PAN or card metadata ever stored on our side), so the
-- payment_methods vanity table (brand/last4/expiry only) has no remaining use.
DROP TABLE IF EXISTS payment_methods;

-- +goose Down
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
