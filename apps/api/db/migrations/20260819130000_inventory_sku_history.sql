-- +goose Up

-- Append-only audit trail of SKU changes on an inventory item. Orders and
-- invoices snapshot product name/label/price (never the SKU), so retired SKUs
-- are not a lookup key — this table exists purely for traceability: given an
-- item you can see every SKU it ever bore, who changed it, when, and why.
CREATE TABLE inventory_sku_history (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	inventory_item_id UUID NOT NULL REFERENCES inventory_items (id) ON DELETE CASCADE,
	old_sku TEXT NOT NULL,
	new_sku TEXT NOT NULL,
	reason TEXT,
	changed_by UUID REFERENCES users (id) ON DELETE SET NULL,
	changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX inventory_sku_history_item_idx
	ON inventory_sku_history (inventory_item_id, changed_at DESC);

-- +goose Down
DROP TABLE inventory_sku_history;
