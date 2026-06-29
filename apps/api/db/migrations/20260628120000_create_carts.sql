-- +goose Up
CREATE TABLE carts (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID REFERENCES users (id) ON DELETE CASCADE,
	guest_token UUID,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX carts_user_id_idx ON carts (user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX carts_guest_token_idx ON carts (guest_token) WHERE guest_token IS NOT NULL;

CREATE TABLE cart_items (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	cart_id UUID NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
	variant_id UUID NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
	quantity INT NOT NULL CHECK (quantity > 0),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX cart_items_cart_id_variant_id_idx ON cart_items (cart_id, variant_id);

-- +goose Down
DROP TABLE cart_items;
DROP TABLE carts;
