-- +goose Up
CREATE TABLE product_variants (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
	price_override_amount BIGINT,
	price_override_currency TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX product_variants_product_id_idx ON product_variants (product_id);

CREATE TABLE variant_attribute_values (
	variant_id UUID NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
	attribute_value_id UUID NOT NULL REFERENCES attribute_values (id) ON DELETE CASCADE,
	PRIMARY KEY (variant_id, attribute_value_id)
);

-- +goose Down
DROP TABLE variant_attribute_values;
DROP TABLE product_variants;
