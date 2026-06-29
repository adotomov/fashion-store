-- +goose Up
CREATE TABLE product_media (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
	bucket TEXT NOT NULL,
	object_key TEXT NOT NULL,
	content_type TEXT,
	size_bytes BIGINT,
	position INT NOT NULL DEFAULT 0,
	alt_text TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX product_media_product_id_idx ON product_media (product_id, position);

-- +goose Down
DROP TABLE product_media;
