-- +goose Up
-- The cart row doubles as the checkout session: while a shopper is checking out
-- we hold a single stock reservation against their cart (acquired when checkout
-- starts, reused across payment-method switches, committed on settlement, and
-- reclaimed by a TTL sweeper on abandonment). These columns track that hold.
ALTER TABLE carts
	ADD COLUMN reservation_id UUID,
	ADD COLUMN reservation_expires_at TIMESTAMPTZ;

-- Lets the sweeper find abandoned holds without scanning every cart.
CREATE INDEX carts_reservation_expires_at_idx ON carts (reservation_expires_at)
	WHERE reservation_expires_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS carts_reservation_expires_at_idx;
ALTER TABLE carts
	DROP COLUMN reservation_id,
	DROP COLUMN reservation_expires_at;
