-- +goose Up
CREATE TABLE product_attributes (
	product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
	attribute_id UUID NOT NULL REFERENCES attributes (id) ON DELETE CASCADE,
	PRIMARY KEY (product_id, attribute_id)
);

DROP TABLE product_attribute_values;

-- +goose Down
CREATE TABLE product_attribute_values (
	product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
	attribute_value_id UUID NOT NULL REFERENCES attribute_values (id) ON DELETE CASCADE,
	PRIMARY KEY (product_id, attribute_value_id)
);

DROP TABLE product_attributes;
