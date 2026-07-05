-- +goose Up
ALTER TABLE products ADD COLUMN nks_code TEXT;

-- +goose Down
ALTER TABLE products DROP COLUMN nks_code;
