-- +goose Up
ALTER TABLE categories ADD COLUMN image_url TEXT;

ALTER TABLE products ADD COLUMN compare_at_price_amount BIGINT;
ALTER TABLE products ADD COLUMN compare_at_price_currency TEXT;

-- +goose Down
ALTER TABLE products DROP COLUMN compare_at_price_amount;
ALTER TABLE products DROP COLUMN compare_at_price_currency;

ALTER TABLE categories DROP COLUMN image_url;
