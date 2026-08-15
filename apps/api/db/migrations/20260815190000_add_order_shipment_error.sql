-- +goose Up
-- Records why a Speedy shipment booking failed so the admin order view can
-- surface the reason and offer a retry. Empty/NULL means no failure recorded
-- (never attempted, or the last attempt succeeded).
ALTER TABLE orders ADD COLUMN shipment_error TEXT;

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS shipment_error;
