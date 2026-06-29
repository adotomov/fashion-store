-- +goose Up
CREATE TABLE inventory_items (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	variant_id UUID NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
	sku TEXT NOT NULL,
	quantity_on_hand INT NOT NULL DEFAULT 0,
	quantity_reserved INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX inventory_items_variant_id_idx ON inventory_items (variant_id);
CREATE UNIQUE INDEX inventory_items_sku_idx ON inventory_items (sku);

-- +goose Down
DROP TABLE inventory_items;
