-- +goose Up
CREATE TABLE orders (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	order_number TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'cancelled')),
	total_amount BIGINT NOT NULL,
	total_currency TEXT NOT NULL,
	placed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX orders_order_number_idx ON orders (order_number);
CREATE INDEX orders_user_id_idx ON orders (user_id);

-- order_items snapshots product name/variant/price at order time rather
-- than referencing live catalog rows, since those can change or be deleted
-- after the order is placed.
CREATE TABLE order_items (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
	product_name TEXT NOT NULL,
	variant_label TEXT,
	quantity INT NOT NULL DEFAULT 1,
	unit_price_amount BIGINT NOT NULL,
	unit_price_currency TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX order_items_order_id_idx ON order_items (order_id);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;
