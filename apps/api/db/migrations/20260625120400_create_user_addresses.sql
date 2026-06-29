-- +goose Up
CREATE TABLE user_addresses (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	label TEXT,
	recipient_name TEXT NOT NULL,
	phone TEXT,
	line1 TEXT NOT NULL,
	line2 TEXT,
	city TEXT NOT NULL,
	region TEXT,
	postal_code TEXT NOT NULL,
	country_code TEXT NOT NULL,
	is_default BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX user_addresses_user_id_idx ON user_addresses (user_id);

-- +goose Down
DROP TABLE user_addresses;
