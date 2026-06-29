-- +goose Up
-- Adds everything the checkout flow needs on top of the original
-- read/list-only orders table: who to contact, where to ship/bill,
-- which mocked delivery/payment method was chosen, fulfillment/tracking
-- state for admin, and a link back to the inventory reservation that was
-- committed when the order was placed.
ALTER TABLE orders
	ADD COLUMN contact_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN contact_email TEXT NOT NULL DEFAULT '',
	ADD COLUMN contact_phone TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_recipient_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_phone TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_line1 TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_line2 TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_city TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_region TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_postal_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_country_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_recipient_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_phone TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_line1 TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_line2 TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_city TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_region TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_postal_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_country_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN delivery_method TEXT NOT NULL DEFAULT 'speedy' CHECK (delivery_method IN ('speedy', 'easybox')),
	ADD COLUMN delivery_fee_amount BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN delivery_fee_currency TEXT NOT NULL DEFAULT 'EUR',
	ADD COLUMN payment_method TEXT NOT NULL DEFAULT 'cash_on_delivery'
		CHECK (payment_method IN ('cash_on_delivery', 'card_on_easybox', 'card_online')),
	ADD COLUMN carrier TEXT,
	ADD COLUMN tracking_number TEXT,
	ADD COLUMN shipment_status TEXT,
	ADD COLUMN viewed_by_admin_at TIMESTAMPTZ,
	ADD COLUMN reservation_id UUID REFERENCES inventory_reservations (id);

CREATE INDEX orders_viewed_by_admin_at_idx ON orders (viewed_by_admin_at);

-- order_payments records the outcome of a (mocked, Revolut-shaped) charge
-- attempt for card_online orders. provider_reference mirrors the field a
-- real Revolut order/payment id would occupy, so swapping the mock
-- gateway for a real one later doesn't change this schema.
CREATE TABLE order_payments (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
	provider TEXT NOT NULL,
	provider_reference TEXT,
	status TEXT NOT NULL CHECK (status IN ('succeeded', 'failed')),
	amount_minor BIGINT NOT NULL,
	currency TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX order_payments_order_id_idx ON order_payments (order_id);

-- +goose Down
DROP TABLE order_payments;

DROP INDEX orders_viewed_by_admin_at_idx;

ALTER TABLE orders
	DROP COLUMN contact_name,
	DROP COLUMN contact_email,
	DROP COLUMN contact_phone,
	DROP COLUMN shipping_recipient_name,
	DROP COLUMN shipping_phone,
	DROP COLUMN shipping_line1,
	DROP COLUMN shipping_line2,
	DROP COLUMN shipping_city,
	DROP COLUMN shipping_region,
	DROP COLUMN shipping_postal_code,
	DROP COLUMN shipping_country_code,
	DROP COLUMN billing_recipient_name,
	DROP COLUMN billing_phone,
	DROP COLUMN billing_line1,
	DROP COLUMN billing_line2,
	DROP COLUMN billing_city,
	DROP COLUMN billing_region,
	DROP COLUMN billing_postal_code,
	DROP COLUMN billing_country_code,
	DROP COLUMN delivery_method,
	DROP COLUMN delivery_fee_amount,
	DROP COLUMN delivery_fee_currency,
	DROP COLUMN payment_method,
	DROP COLUMN carrier,
	DROP COLUMN tracking_number,
	DROP COLUMN shipment_status,
	DROP COLUMN viewed_by_admin_at,
	DROP COLUMN reservation_id;
