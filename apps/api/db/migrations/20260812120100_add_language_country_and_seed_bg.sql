-- +goose Up
-- country_code maps a store language to the ISO country whose visitors should
-- see it (via the load balancer's geo region). Empty = not geo-targeted.
ALTER TABLE store_languages ADD COLUMN country_code TEXT NOT NULL DEFAULT '';

-- Register Bulgarian as a store language: enabled and geo-mapped to Bulgaria, so
-- visitors from BG get Bulgarian while English stays the default/base for
-- everyone else. Not made is_default here — English remains the display default
-- until an admin flips it (Set as default). bg UI strings and email templates
-- already exist; this makes the language selectable. Idempotent.
INSERT INTO store_languages (code, name, is_default, enabled, country_code)
VALUES ('bg', 'Български', false, true, 'BG')
ON CONFLICT (code) DO UPDATE SET enabled = true, country_code = EXCLUDED.country_code;

-- +goose Down
-- Only drop the column; leave the bg row so any bg translations created since
-- are not cascade-deleted.
ALTER TABLE store_languages DROP COLUMN country_code;
