-- +goose Up
CREATE TABLE inventory_reservations (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	status TEXT NOT NULL CHECK (status IN ('pending', 'committed', 'released', 'expired')),
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE inventory_reservation_items (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	reservation_id UUID NOT NULL REFERENCES inventory_reservations (id) ON DELETE CASCADE,
	inventory_item_id UUID NOT NULL REFERENCES inventory_items (id) ON DELETE CASCADE,
	quantity INT NOT NULL
);

CREATE INDEX inventory_reservation_items_reservation_id_idx ON inventory_reservation_items (reservation_id);
CREATE INDEX inventory_reservation_items_inventory_item_id_idx ON inventory_reservation_items (inventory_item_id);

-- +goose Down
DROP TABLE inventory_reservation_items;
DROP TABLE inventory_reservations;
