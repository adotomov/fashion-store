-- +goose Up
CREATE TABLE inventory_movements (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	inventory_item_id UUID NOT NULL REFERENCES inventory_items (id) ON DELETE CASCADE,
	type TEXT NOT NULL CHECK (type IN (
		'initial_stock', 'admin_adjustment', 'reservation', 'reservation_release',
		'sale_committed', 'return', 'manual_correction'
	)),
	quantity_delta INT NOT NULL,
	note TEXT,
	created_by UUID REFERENCES users (id) ON DELETE SET NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX inventory_movements_inventory_item_id_idx ON inventory_movements (inventory_item_id, created_at DESC);

-- +goose Down
DROP TABLE inventory_movements;
