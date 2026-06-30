-- +goose Up
CREATE TABLE wishlist_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX wishlist_items_user_id_product_id_idx ON wishlist_items (user_id, product_id);

-- +goose Down
DROP TABLE IF EXISTS wishlist_items;
