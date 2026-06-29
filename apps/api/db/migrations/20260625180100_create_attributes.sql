-- +goose Up
CREATE TABLE attributes (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX attributes_name_idx ON attributes (lower(name));

CREATE TABLE attribute_values (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	attribute_id UUID NOT NULL REFERENCES attributes (id) ON DELETE CASCADE,
	value TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX attribute_values_attribute_id_value_idx ON attribute_values (attribute_id, lower(value));

-- +goose Down
DROP TABLE attribute_values;
DROP TABLE attributes;
