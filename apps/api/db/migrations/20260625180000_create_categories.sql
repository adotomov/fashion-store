-- +goose Up
CREATE TABLE categories (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL,
	slug TEXT NOT NULL,
	parent_id UUID REFERENCES categories (id) ON DELETE SET NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX categories_slug_idx ON categories (slug);
CREATE INDEX categories_parent_id_idx ON categories (parent_id);

-- +goose Down
DROP TABLE categories;
