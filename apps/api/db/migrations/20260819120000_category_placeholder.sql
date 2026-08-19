-- +goose Up

-- A placeholder category is a pure grouping node in the nav hierarchy (e.g.
-- "Men", "Women") that products may never be assigned to directly — only its
-- leaf descendants hold products. Defaults to FALSE so existing categories
-- stay assignable.
ALTER TABLE categories ADD COLUMN is_placeholder BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE categories DROP COLUMN is_placeholder;
