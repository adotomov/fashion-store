-- +goose Up

-- About Us cover photo and Contact Us store photo, plus free-text opening
-- hours — all singleton store-settings fields backing the new public About and
-- Contact pages. Images mirror the existing logo columns (nullable file
-- references served through an admin/storefront proxy endpoint); opening_hours
-- is plain multi-line text the admin can write in any language.
ALTER TABLE store_settings
    ADD COLUMN opening_hours          TEXT,
    ADD COLUMN about_cover_bucket       TEXT,
    ADD COLUMN about_cover_object_key   TEXT,
    ADD COLUMN about_cover_content_type TEXT,
    ADD COLUMN about_cover_size_bytes   BIGINT,
    ADD COLUMN store_image_bucket       TEXT,
    ADD COLUMN store_image_object_key   TEXT,
    ADD COLUMN store_image_content_type TEXT,
    ADD COLUMN store_image_size_bytes   BIGINT;

-- +goose Down

ALTER TABLE store_settings
    DROP COLUMN opening_hours,
    DROP COLUMN about_cover_bucket,
    DROP COLUMN about_cover_object_key,
    DROP COLUMN about_cover_content_type,
    DROP COLUMN about_cover_size_bytes,
    DROP COLUMN store_image_bucket,
    DROP COLUMN store_image_object_key,
    DROP COLUMN store_image_content_type,
    DROP COLUMN store_image_size_bytes;
