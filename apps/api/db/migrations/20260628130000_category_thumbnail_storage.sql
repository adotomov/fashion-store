-- +goose Up
ALTER TABLE categories DROP COLUMN image_url;
ALTER TABLE categories ADD COLUMN thumbnail_bucket TEXT;
ALTER TABLE categories ADD COLUMN thumbnail_object_key TEXT;
ALTER TABLE categories ADD COLUMN thumbnail_content_type TEXT;
ALTER TABLE categories ADD COLUMN thumbnail_size_bytes BIGINT;

-- +goose Down
ALTER TABLE categories DROP COLUMN thumbnail_bucket;
ALTER TABLE categories DROP COLUMN thumbnail_object_key;
ALTER TABLE categories DROP COLUMN thumbnail_content_type;
ALTER TABLE categories DROP COLUMN thumbnail_size_bytes;
ALTER TABLE categories ADD COLUMN image_url TEXT;
