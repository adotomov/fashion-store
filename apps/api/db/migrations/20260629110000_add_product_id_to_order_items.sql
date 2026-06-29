-- +goose Up
ALTER TABLE order_items ADD COLUMN product_id UUID REFERENCES products(id) ON DELETE SET NULL;
CREATE INDEX idx_order_items_product_id ON order_items (product_id);

-- +goose Down
DROP INDEX IF EXISTS idx_order_items_product_id;
ALTER TABLE order_items DROP COLUMN IF EXISTS product_id;
