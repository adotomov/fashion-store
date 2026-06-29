-- +goose Up
CREATE TABLE product_types (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL,
	slug TEXT NOT NULL,
	position INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX product_types_slug_idx ON product_types (slug);

INSERT INTO product_types (name, slug) VALUES ('Uncategorized', 'uncategorized');

ALTER TABLE categories ADD COLUMN product_type_id UUID REFERENCES product_types (id) ON DELETE RESTRICT;
UPDATE categories SET product_type_id = (SELECT id FROM product_types WHERE slug = 'uncategorized');
ALTER TABLE categories ALTER COLUMN product_type_id SET NOT NULL;

CREATE INDEX categories_product_type_id_idx ON categories (product_type_id);

-- +goose Down
ALTER TABLE categories DROP COLUMN product_type_id;
DROP TABLE product_types;
