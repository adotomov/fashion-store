-- +goose Up
CREATE TABLE product_categories (
	product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
	category_id UUID NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
	PRIMARY KEY (product_id, category_id)
);

CREATE TABLE catalog_products (
	catalog_id UUID NOT NULL REFERENCES catalogs (id) ON DELETE CASCADE,
	product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
	PRIMARY KEY (catalog_id, product_id)
);

CREATE TABLE product_attribute_values (
	product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
	attribute_value_id UUID NOT NULL REFERENCES attribute_values (id) ON DELETE CASCADE,
	PRIMARY KEY (product_id, attribute_value_id)
);

-- +goose Down
DROP TABLE product_attribute_values;
DROP TABLE catalog_products;
DROP TABLE product_categories;
