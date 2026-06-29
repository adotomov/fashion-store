-- +goose Up
CREATE TABLE products (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL,
	slug TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'archived')),
	base_price_amount BIGINT NOT NULL,
	base_price_currency TEXT NOT NULL DEFAULT 'EUR',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX products_slug_idx ON products (slug);

-- +goose Down
DROP TABLE products;
