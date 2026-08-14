-- +goose Up
-- Restructure saved and order-snapshot addresses to Speedy's code-based model:
-- city/neighbourhood(complex)/street are stored as Speedy location IDs plus
-- their display names, so shipments carry resolved codes instead of free text.
-- The database is empty at rollout, so free-text columns are dropped outright.

-- user_addresses: drop the free-text lines, keep recipient/phone/city/country,
-- add the structured Speedy fields.
ALTER TABLE user_addresses
	DROP COLUMN line1,
	DROP COLUMN line2,
	DROP COLUMN region,
	DROP COLUMN postal_code;

ALTER TABLE user_addresses
	ALTER COLUMN country_code SET DEFAULT 'BG',
	ADD COLUMN country_id INT NOT NULL DEFAULT 100,
	ADD COLUMN site_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN post_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN complex_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN complex_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN street_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN street_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN street_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN block_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN entrance_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN floor_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN apartment_no TEXT NOT NULL DEFAULT '';

-- orders: same restructure for both the shipping and billing snapshots.
ALTER TABLE orders
	DROP COLUMN shipping_line1,
	DROP COLUMN shipping_line2,
	DROP COLUMN shipping_region,
	DROP COLUMN shipping_postal_code,
	DROP COLUMN billing_line1,
	DROP COLUMN billing_line2,
	DROP COLUMN billing_region,
	DROP COLUMN billing_postal_code;

ALTER TABLE orders
	ALTER COLUMN shipping_country_code SET DEFAULT 'BG',
	ADD COLUMN shipping_country_id INT NOT NULL DEFAULT 100,
	ADD COLUMN shipping_site_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN shipping_post_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_complex_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN shipping_complex_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_street_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN shipping_street_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_street_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_block_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_entrance_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_floor_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_apartment_no TEXT NOT NULL DEFAULT '',
	ALTER COLUMN billing_country_code SET DEFAULT 'BG',
	ADD COLUMN billing_country_id INT NOT NULL DEFAULT 100,
	ADD COLUMN billing_site_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN billing_post_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_complex_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN billing_complex_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_street_id BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN billing_street_name TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_street_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_block_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_entrance_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_floor_no TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_apartment_no TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE orders
	DROP COLUMN shipping_country_id,
	DROP COLUMN shipping_site_id,
	DROP COLUMN shipping_post_code,
	DROP COLUMN shipping_complex_id,
	DROP COLUMN shipping_complex_name,
	DROP COLUMN shipping_street_id,
	DROP COLUMN shipping_street_name,
	DROP COLUMN shipping_street_no,
	DROP COLUMN shipping_block_no,
	DROP COLUMN shipping_entrance_no,
	DROP COLUMN shipping_floor_no,
	DROP COLUMN shipping_apartment_no,
	DROP COLUMN billing_country_id,
	DROP COLUMN billing_site_id,
	DROP COLUMN billing_post_code,
	DROP COLUMN billing_complex_id,
	DROP COLUMN billing_complex_name,
	DROP COLUMN billing_street_id,
	DROP COLUMN billing_street_name,
	DROP COLUMN billing_street_no,
	DROP COLUMN billing_block_no,
	DROP COLUMN billing_entrance_no,
	DROP COLUMN billing_floor_no,
	DROP COLUMN billing_apartment_no,
	ALTER COLUMN shipping_country_code DROP DEFAULT,
	ALTER COLUMN billing_country_code DROP DEFAULT,
	ADD COLUMN shipping_line1 TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_line2 TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_region TEXT NOT NULL DEFAULT '',
	ADD COLUMN shipping_postal_code TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_line1 TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_line2 TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_region TEXT NOT NULL DEFAULT '',
	ADD COLUMN billing_postal_code TEXT NOT NULL DEFAULT '';

ALTER TABLE user_addresses
	DROP COLUMN country_id,
	DROP COLUMN site_id,
	DROP COLUMN post_code,
	DROP COLUMN complex_id,
	DROP COLUMN complex_name,
	DROP COLUMN street_id,
	DROP COLUMN street_name,
	DROP COLUMN street_no,
	DROP COLUMN block_no,
	DROP COLUMN entrance_no,
	DROP COLUMN floor_no,
	DROP COLUMN apartment_no,
	ALTER COLUMN country_code DROP DEFAULT,
	ADD COLUMN line1 TEXT NOT NULL DEFAULT '',
	ADD COLUMN line2 TEXT,
	ADD COLUMN region TEXT,
	ADD COLUMN postal_code TEXT NOT NULL DEFAULT '';
