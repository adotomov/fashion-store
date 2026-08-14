-- +goose Up
-- Physical shipping attributes per SKU (product variant): weight in grams,
-- dimensions in centimetres. Used to compute a parcel's weight when asking
-- Speedy for shipping costs.
ALTER TABLE product_variants
	ADD COLUMN weight_grams INT NOT NULL DEFAULT 0,
	ADD COLUMN length_cm INT NOT NULL DEFAULT 0,
	ADD COLUMN width_cm INT NOT NULL DEFAULT 0,
	ADD COLUMN height_cm INT NOT NULL DEFAULT 0;

-- The parcel weight (grams) used to price/ship the order, summed from its SKUs
-- at order time. Kept on the order so the card-settlement path (which runs
-- after the cart is gone) can still price and ship with the real weight.
ALTER TABLE orders ADD COLUMN parcel_weight_grams INT NOT NULL DEFAULT 0;

-- Speedy shipping-cost quotes captured per order, one row per delivery method
-- priced. Write-only reference data: never shown to shoppers or admins.
CREATE TABLE order_shipping_quotes (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
	delivery_method TEXT NOT NULL,
	service_id TEXT NOT NULL,
	amount_minor BIGINT NOT NULL,
	currency TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX order_shipping_quotes_order_id_idx ON order_shipping_quotes (order_id);

-- +goose Down
DROP TABLE order_shipping_quotes;

ALTER TABLE orders DROP COLUMN parcel_weight_grams;

ALTER TABLE product_variants
	DROP COLUMN weight_grams,
	DROP COLUMN length_cm,
	DROP COLUMN width_cm,
	DROP COLUMN height_cm;
