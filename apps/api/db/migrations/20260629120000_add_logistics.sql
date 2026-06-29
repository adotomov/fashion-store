-- +goose Up
CREATE TABLE logistics_provider_settings (
    provider   TEXT PRIMARY KEY,
    enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    config     JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE orders ADD COLUMN speedy_shipment_id TEXT;
ALTER TABLE orders ADD COLUMN delivery_office_id TEXT;

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS delivery_office_id;
ALTER TABLE orders DROP COLUMN IF EXISTS speedy_shipment_id;
DROP TABLE IF EXISTS logistics_provider_settings;
