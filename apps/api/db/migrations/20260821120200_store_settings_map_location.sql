-- +goose Up

-- Optional precise map pin for the public Contact page. Free text: normally
-- "lat,lng" (e.g. "42.6977,23.3219") but any Google Maps place query works.
-- When set it overrides geocoding the store address, so the map lands exactly.
ALTER TABLE store_settings ADD COLUMN map_location TEXT;

-- +goose Down

ALTER TABLE store_settings DROP COLUMN map_location;
