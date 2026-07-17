-- +goose Up
-- cart_guest_token records the guest cart token a card order was placed from, so
-- the payment webhook (which runs server-to-server, without the shopper's cart
-- cookie) can clear the right cart once the payment settles. Null for orders
-- placed by a signed-in user (their cart is keyed by user_id) and for
-- pay-on-delivery orders (whose cart is cleared synchronously at placement).
ALTER TABLE orders ADD COLUMN cart_guest_token UUID;

-- +goose Down
ALTER TABLE orders DROP COLUMN cart_guest_token;
