-- +goose Up
ALTER TABLE hero_settings
    ADD COLUMN background_image_bucket       TEXT,
    ADD COLUMN background_image_object_key   TEXT,
    ADD COLUMN background_image_content_type TEXT,
    ADD COLUMN background_image_size_bytes   BIGINT;

-- +goose Down
ALTER TABLE hero_settings
    DROP COLUMN background_image_bucket,
    DROP COLUMN background_image_object_key,
    DROP COLUMN background_image_content_type,
    DROP COLUMN background_image_size_bytes;
