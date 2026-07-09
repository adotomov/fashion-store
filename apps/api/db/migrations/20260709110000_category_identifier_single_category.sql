-- +goose Up

-- Internal identifier per category (e.g. DR-01, GE-10). Used as the fixed
-- prefix when composing variant SKUs. Unique when set; NULL allowed for
-- categories that don't have one yet.
ALTER TABLE categories ADD COLUMN internal_identifier TEXT;
CREATE UNIQUE INDEX categories_internal_identifier_key
    ON categories (internal_identifier)
    WHERE internal_identifier IS NOT NULL;

-- A product now belongs to exactly one category. Collapse any existing
-- multi-category rows to a single deterministic one, then enforce it.
DELETE FROM product_categories pc
USING (
    SELECT product_id, MIN(category_id::text) AS keep
    FROM product_categories
    GROUP BY product_id
) d
WHERE pc.product_id = d.product_id
  AND pc.category_id::text <> d.keep;

CREATE UNIQUE INDEX product_categories_product_id_key
    ON product_categories (product_id);

-- +goose Down
DROP INDEX IF EXISTS product_categories_product_id_key;
DROP INDEX IF EXISTS categories_internal_identifier_key;
ALTER TABLE categories DROP COLUMN internal_identifier;
